package continuity

import (
	"bytes"
	"encoding/json"
	"knodexa/internal/approach"
	"knodexa/internal/context"
	"knodexa/internal/git"
	"knodexa/internal/identity"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"os"
	"path/filepath"
	"testing"
)

func TestRecordVerifiesBeforeSingleMemoryUpdate(t *testing.T) {
	root := t.TempDir()
	store := storage.NewStorage(root)
	if _, err := store.Init("record", ""); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := index.NewIndexer(db, store)
	inspector := git.NewInspector(root)
	service := NewService(store, db, idx, context.NewBuilder(db, store, inspector), inspector)
	os.WriteFile(filepath.Join(root, "reference.go"), []byte("package demo\nfunc Original(){}\n"), 0644)
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module demo\ngo 1.26.0\n"), 0644)
	task := &model.TaskData{ID: "known", Status: model.TaskStatusCompleted, BaselineCommit: "saved-baseline", Constraints: []string{"Never normalize"}, RequiredVerification: []string{"Preserve result identity"}}
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	original, _ := store.ReadTask("known")
	before, _ := json.Marshal(original)
	input := RecordInput{Files: []string{"reference.go"}, Aliases: []string{"callback work"}, Approach: model.TaskApproach{Summary: "Delegate", Rules: []string{"Never normalize"}, References: []model.ApproachReference{{Path: "reference.go"}}, Checks: []model.ApproachCheck{{Path: "reference.go", ASTKind: "require-function", Symbol: "Original"}}}}
	result, err := service.Record("known", input, []string{"exit 1"}, 0)
	if err == nil || result.Recorded {
		t.Fatal("failed verification accepted")
	}
	after, _ := store.ReadTask("known")
	encoded, _ := json.Marshal(after)
	if !bytes.Equal(before, encoded) {
		t.Fatal("partial memory after failed test")
	}
	input.Approach.Checks[0].ASTKind = "require-call"
	result, err = service.Record("known", input, []string{"go test ./..."}, 0)
	if err == nil || result.Recorded {
		t.Fatal("failed structural check accepted")
	}
	after, _ = store.ReadTask("known")
	encoded, _ = json.Marshal(after)
	if !bytes.Equal(before, encoded) {
		t.Fatal("partial memory after failed checks")
	}
	input.Approach.Checks[0].ASTKind = "require-function"
	input.Approach.Checks[0].Path = "@touched"
	result, err = service.Record("known", input, []string{"go test ./..."}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Recorded || result.WorkID != "known" || result.Version != 1 {
		t.Fatalf("incomplete recording: %+v", result)
	}
	saved, _ := store.ReadTask("known")
	if saved.Status != task.Status || saved.BaselineCommit != task.BaselineCommit || len(saved.CodeAnchors) != 1 || len(saved.Aliases) != 1 {
		t.Fatal("record lost metadata")
	}
	input.Approach.ChangeReason = "Explicit reviewed replacement"
	result, err = service.Record("known", input, []string{"go test ./..."}, 0)
	if err != nil {
		t.Fatal(err)
	}
	saved, _ = store.ReadTask("known")
	if result.Version != 2 || len(saved.ApproachHistory) != 1 {
		t.Fatal("replacement history lost")
	}
	prepared, err := service.Prepare(identity.Request{Text: "Extend callback work"}, 3000, ResumeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	text := prepared.FormatHuman()
	raw, _ := json.Marshal(prepared)
	if !bytes.Contains([]byte(text), []byte("Never normalize")) || !bytes.Contains([]byte(text), []byte("Preserve result identity")) || !bytes.Contains([]byte(text), []byte("func Original()")) {
		t.Fatal("compact handoff lost essential context")
	}
	t.Logf("handoff bytes: text=%d JSON=%d", len(text), len(raw))
	if len(text) >= len(raw) {
		t.Fatal("text handoff did not reduce payload")
	}
	if report, err := approach.Check(store, db, "known"); err != nil || !report.StructuralChecksPassed {
		t.Fatal("recorded approach not usable")
	}
}
