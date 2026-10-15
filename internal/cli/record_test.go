package cli

import (
	"knodexa/internal/model"
	"os"
	"path/filepath"
	"testing"
)

func TestRecordCLIUsesOneVerifiedOperation(t *testing.T) {
	root := t.TempDir()
	app, err := NewApp(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"init", "--name=Record"}); err != nil {
		t.Fatal(err)
	}
	app, err = NewApp(root)
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	if err := app.store.SaveTask(&model.TaskData{ID: "known", Status: model.TaskStatusCompleted}); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "source.go"), []byte("package demo\nfunc Original(){}\n"), 0644)
	input := filepath.Join(root, "agreement.json")
	os.WriteFile(input, []byte(`{"aliases":["callback work"],"files":["source.go"],"approach":{"summary":"Delegate","rules":["Keep raw values"],"references":[{"path":"source.go"}]}}`), 0644)
	if err := app.Run([]string{"record", "known", "--file=" + input, "--verify=true"}); err != nil {
		t.Fatal(err)
	}
	task, err := app.store.ReadTask("known")
	if err != nil {
		t.Fatal(err)
	}
	if task.WorkID != "known" || task.Status != model.TaskStatusCompleted || task.Approach == nil || len(task.CodeAnchors) != 1 || len(task.Aliases) != 1 {
		t.Fatalf("incomplete command: %+v", task)
	}
	if err := app.Run([]string{"prepare", "Extend callback work", "--text"}); err != nil {
		t.Fatal(err)
	}
}
