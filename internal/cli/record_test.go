package cli

import (
	"knodexa/internal/model"
	"os"
	"path/filepath"
	"strings"
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

func TestRecordHelpAndBehaviorAgreementFile(t *testing.T) {
	root := t.TempDir()
	app, err := NewApp(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"record", "--help"}); err != nil {
		t.Fatal("record help requires initialization or fails", err)
	}
	if err := app.Run([]string{"init", "--name=Behavior CLI"}); err != nil {
		t.Fatal(err)
	}
	app, err = NewApp(root)
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	if err := app.store.SaveTask(&model.TaskData{ID: "quality"}); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "source.go"), []byte("package sample\nfunc Original(){}\n"), 0644)
	os.WriteFile(filepath.Join(root, "spec.txt"), []byte("Owner acceptance specification"), 0644)
	input := filepath.Join(root, "agreement.json")
	os.WriteFile(input, []byte(`{"files":["source.go"],"approach":{"summary":"Quality","rules":["Preserve contract"],"behavior_checks":[{"name":"owner acceptance","command":"exit 7","protected_files":[{"path":"spec.txt"}]}]}}`), 0644)
	if err := app.Run([]string{"record", "quality", "--file=" + input, "--verify=true"}); err == nil || !strings.Contains(err.Error(), "behavior check") {
		t.Fatal("CLI dropped behavioral contract", err)
	}
	task, _ := app.store.ReadTask("quality")
	if task.Approach != nil {
		t.Fatal("failed CLI quality saved memory")
	}
	value := filepath.Join(root, "approach.json")
	os.WriteFile(value, []byte(`{"summary":"Quality","rules":["Preserve contract"],"behavior_checks":[{"name":"owner acceptance","command":"true","protected_files":[{"path":"spec.txt"}]}]}`), 0644)
	if err := app.Run([]string{"approach", "set", "quality", "--file=" + value}); err != nil {
		t.Fatal(err)
	}
	task, _ = app.store.ReadTask("quality")
	if len(task.Approach.BehaviorChecks) != 1 || task.Approach.BehaviorChecks[0].ProtectedFiles[0].SHA256 == "" {
		t.Fatal("JSON approach did not pin acceptance input")
	}
	if err := app.Run([]string{"approach", "set", "quality", "--file=" + value, "--reason=ignored"}); err == nil {
		t.Fatal("mixed inline flags silently ignored")
	}
	if err := app.Run([]string{"record", "quality", "--made-up-flag"}); err == nil {
		t.Fatal("help fix accepted unknown flags")
	}
}
