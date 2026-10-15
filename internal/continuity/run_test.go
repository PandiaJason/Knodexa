package continuity

import (
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

func TestRunRestoresBeforeExecutionAndCapturesOnlyVerifiedWork(t *testing.T) {
	root := t.TempDir()
	store := storage.NewStorage(root)
	if _, err := store.Init("run", ""); err != nil {
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
	os.Mkdir(filepath.Join(root, "handlers"), 0755)
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module runnerexample\ngo 1.26.0\n"), 0644)
	os.WriteFile(filepath.Join(root, "handlers", "reference.go"), []byte("package handlers\nfunc Original() {}\n"), 0644)
	if err := store.SaveTask(&model.TaskData{ID: "known", Title: "Callback adapters", Aliases: []string{"callback adapter"}, Status: model.TaskStatusCompleted, TouchesFiles: []string{"handlers/reference.go"}}); err != nil {
		t.Fatal(err)
	}
	if err := approach.Set(store, db, idx, "known", model.TaskApproach{Summary: "Reuse services", Rules: []string{"Preserve delegation"}, Scopes: []string{"handlers/"}, References: []model.ApproachReference{{Path: "handlers/reference.go"}}}); err != nil {
		t.Fatal(err)
	}
	result, err := service.Run(identity.Request{Text: "Extend callback adapter"}, 3000, RunOptions{Command: []string{os.Args[0], "-test.run=^TestContinuityCommandHelper$"}, Verify: []string{"go test ./..."}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Captured || len(result.Prepared.References) != 1 {
		t.Fatalf("no bundled restoration/capture: %+v", result)
	}
	task, _ := store.ReadTask("known")
	if len(task.CodeAnchors) != 2 || task.Status != model.TaskStatusCompleted || task.WorkID != "known" {
		t.Fatal("capture lost continuity or reopened completed task")
	}
	old := task.UpdatedAt
	_, err = service.Run(identity.Request{Text: "Extend callback adapter"}, 3000, RunOptions{Command: []string{"/bin/sh", "-c", "true"}, Verify: []string{"exit 1"}})
	if err == nil {
		t.Fatal("failed proof accepted")
	}
	task, _ = store.ReadTask("known")
	if !task.UpdatedAt.Equal(old) {
		t.Fatal("failed verification changed memory")
	}
	// The implementation still compiles, but violates the restored convention.
	// Even rewriting canonical checks during execution cannot bless that drift.
	task.Approach.Checks = []model.ApproachCheck{{Path: "@touched", ASTKind: "forbid-call", Symbol: "println"}}
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	blockedDrift, err := service.Run(identity.Request{Text: "Extend callback adapter"}, 3000, RunOptions{Command: []string{os.Args[0], "-test.run=^TestContinuityDriftHelper$"}, Verify: []string{"go test ./..."}})
	if err == nil || blockedDrift.Captured || blockedDrift.Checks == nil || blockedDrift.Checks.StructuralChecksPassed {
		t.Fatalf("weakened agreement bypassed restored guard: %+v, %v", blockedDrift, err)
	}
	if len(blockedDrift.Commands) != 2 || blockedDrift.Commands[1].ExitCode != 0 {
		t.Fatal("fixture did not pass ordinary Go tests")
	}
	os.Remove(filepath.Join(root, "handlers", "drift.go"))
	if err := store.UpdateTask("known", func(current *model.TaskData) error { current.Approach = task.Approach; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "handlers", "reference.go"), []byte("package handlers\nfunc Original() { println(1) }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	prepared, err := service.Prepare(identity.Request{Text: "Extend callback adapter"}, 3000, ResumeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !prepared.NeedsReview || len(prepared.ReviewReasons) == 0 {
		t.Fatal("changed reference silently accepted")
	}
	blocked, err := service.Run(identity.Request{Text: "Extend callback adapter"}, 3000, RunOptions{Command: []string{"/bin/sh", "-c", "true"}, Verify: []string{"true"}})
	if err == nil || len(blocked.Commands) != 0 || blocked.Captured {
		t.Fatal("stale agreement did not block execution")
	}
}
func TestContinuityDriftHelper(t *testing.T) {
	if os.Getenv("KNODEXA_CONTEXT_FILE") == "" {
		t.Skip("command helper")
	}
	if err := os.WriteFile(filepath.Join("handlers", "drift.go"), []byte("package handlers\n// Original()\nfunc Drift() { println(1) }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	store := storage.NewStorage(root)
	task, err := store.ReadTask("known")
	if err != nil {
		t.Fatal(err)
	}
	task.Approach.Checks = nil
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
}
func TestContinuityCommandHelper(t *testing.T) {
	path := os.Getenv("KNODEXA_CONTEXT_FILE")
	if path == "" {
		t.Skip("command helper")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var prepared PreparedWork
	if err := json.Unmarshal(data, &prepared); err != nil {
		t.Fatal(err)
	}
	if prepared.SelectedTask != "known" || len(prepared.References) != 1 {
		t.Fatal("context missing before command")
	}
	if err := os.WriteFile(filepath.Join("handlers", "extension.go"), []byte("package handlers\nfunc Extension() { Original() }\n"), 0644); err != nil {
		t.Fatal(err)
	}
}
