package experiment

import (
	"bytes"
	"gopkg.in/yaml.v3"
	"knodexa/internal/codegraph"
	"knodexa/internal/context"
	"knodexa/internal/continuity"
	"knodexa/internal/git"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Replay the real failed capture using retained code and source agreement. This
// checks the service fix without making a fresh agent or superiority claim.
func TestReplayRefactoredExtensionCapture(t *testing.T) {
	repo, _ := filepath.Abs("../..")
	artifact := filepath.Join(repo, "audit", "hard-continuity-v1", "refactor", "knodexa")
	original, err := os.ReadFile(filepath.Join(artifact, "completed-memory-before.txt"))
	if os.IsNotExist(err) {
		t.Skip("retained live fixture not present")
	}
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	hardWrite(t, filepath.Join(root, "go.mod"), "module identitysample\ngo 1.26.0\n")
	hardWrite(t, filepath.Join(root, "contracts", "version.go"), "package contracts\nconst Version=\"raw-v1\"\n")
	entries, err := os.ReadDir(filepath.Join(artifact, "source"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".go.txt") && !strings.HasPrefix(entry.Name(), "hard_") {
			data, err := os.ReadFile(filepath.Join(artifact, "source", entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			hardWrite(t, filepath.Join(root, "handlers", strings.TrimSuffix(entry.Name(), ".txt")), string(data))
		}
	}
	hardWrite(t, filepath.Join(root, "handlers", "replay_independent_test.go"), hardIndependentGrader())
	hardWrite(t, filepath.Join(root, "handlers", "replay_lesson_test.go"), hardLessonGrader)
	store := storage.NewStorage(root)
	if _, err := store.Init("replay", ""); err != nil {
		t.Fatal(err)
	}
	var parent model.TaskData
	if err := yaml.Unmarshal(original, &parent); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTask(&parent); err != nil {
		t.Fatal(err)
	}
	parentPath := filepath.Join(store.KnodexaDir(), "tasks", parent.ID, parent.ID+".yaml")
	before, err := os.ReadFile(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTask(&model.TaskData{ID: "extension", ParentID: parent.ID, Status: model.TaskStatusActive, TouchesFiles: []string{"handlers/webhook.go"}}); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := index.NewIndexer(db, store)
	gi := git.NewInspector(root)
	service := continuity.NewService(store, db, idx, context.NewBuilder(db, store, gi), gi)
	anchor, err := codegraph.Read(root, "handlers/transport.go")
	if err != nil {
		t.Fatal(err)
	}
	version := parent.Approach.Version
	if version == 0 {
		version = 1
	}
	input := continuity.RecordInput{Aliases: []string{"additional production webhook handler"}, Files: []string{"handlers/webhook.go"}, Approach: model.TaskApproach{Summary: "Delegate to the reviewed current production dispatch", Rules: []string{"Retain raw delegation, observer ordering, exact error/result identity and SDK compatibility"}, References: []model.ApproachReference{{Path: "handlers/transport.go"}}, ReferenceReconciliations: []model.ReferenceReconciliation{{SourceTask: parent.ID, SourceVersion: version, OriginalPath: parent.Approach.References[0].Path, OriginalSHA256: parent.Approach.References[0].SHA256, CurrentPath: "handlers/transport.go", CurrentSHA256: anchor.SHA256, Reason: "Reviewed authorized relocation and function refactor against retained independent behavior and compatibility checks"}}}}
	result, err := service.Record("extension", input, []string{"go test ./..."}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Recorded {
		t.Fatal("real artifact capture remains blocked")
	}
	after, err := os.ReadFile(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("replay changed completed source agreement")
	}
	t.Logf("Retained refactor extension indexed: recorded=%t, version=%d; parent unchanged", result.Recorded, result.Version)
}
