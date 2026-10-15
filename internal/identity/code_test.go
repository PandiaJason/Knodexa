package identity

import (
	"knodexa/internal/model"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSymbolsRecoverWorkAfterGitRenameWithoutRecordedPhrase(t *testing.T) {
	store, db, idx := fixture(t)
	root := store.RootDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = root
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v %s", err, data)
		}
		return string(data)
	}
	run("git", "init", "-b", "main")
	run("git", "config", "user.name", "test")
	run("git", "config", "user.email", "test@example.test")
	if err := os.MkdirAll(filepath.Join(root, "handlers"), 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "handlers", "reference.go")
	if err := os.WriteFile(path, []byte("package handlers\nimport \"net/http\"\nfunc DeliverWebhook(w http.ResponseWriter) {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "original")
	task, _ := store.ReadTask("work-72")
	task.BaselineCommit = run("git", "rev-parse", "HEAD")
	task.BaselineCommit = task.BaselineCommit[:len(task.BaselineCommit)-1]
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	if err := Capture(store, db, idx, "work-72", nil); err != nil {
		t.Fatal(err)
	}
	run("git", "mv", "handlers/reference.go", "handlers/transport.go")
	run("git", "commit", "-m", "rename")
	result, err := ResolveWithCode(root, db, Request{Text: "Add tracing to DeliverWebhook"})
	if err != nil {
		t.Fatal(err)
	}
	if result.SelectedTask != "work-72" || len(result.RelocatedFiles) != 1 || result.RelocatedFiles[0].Evidence != "git_rename" {
		t.Fatalf("identity lost on rename: %+v", result)
	}
	task, _ = store.ReadTask("work-72")
	if task.Status != model.TaskStatusCompleted || task.WorkID != "work-72" || task.CodeAnchors[0].Path != "handlers/reference.go" {
		t.Fatal("resolution rewrote canonical history")
	}
	result, err = ResolveWithCode(root, db, Request{Text: "Improve transport", Files: []string{"handlers/transport.go"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.SelectedTask != "work-72" {
		t.Fatal("renamed path did not retain work")
	}
	// A matching symbol in another explicitly recorded work remains ambiguous.
	if err := store.SaveTask(&model.TaskData{ID: "parallel", Title: "Different delivery", TouchesFiles: []string{"handlers/transport.go"}}); err != nil {
		t.Fatal(err)
	}
	if err := Capture(store, db, idx, "parallel", nil); err != nil {
		t.Fatal(err)
	}
	result, err = ResolveWithCode(root, db, Request{Text: "Improve DeliverWebhook"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "needs_clarification" {
		t.Fatal("common symbol silently selected work")
	}
}
