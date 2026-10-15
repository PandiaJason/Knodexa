package mcp

import (
	"knodexa/internal/continuity"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"os"
	"path/filepath"
	"testing"
)

func TestMCPRecordExecutesBehaviorContract(t *testing.T) {
	root := t.TempDir()
	store := storage.NewStorage(root)
	if _, err := store.Init("Behavior MCP", ""); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTask(&model.TaskData{ID: "quality"}); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "source.go"), []byte("package sample\nfunc Original(){}\n"), 0644)
	os.WriteFile(filepath.Join(root, "spec.txt"), []byte("Acceptance"), 0644)
	server, err := NewServer(root)
	if err != nil {
		t.Fatal(err)
	}
	defer server.handler.db.Close()
	result, err := server.handler.HandleTool("knodexa_record_work", map[string]interface{}{
		"task_id": "quality", "verify": []string{"true"},
		"agreement": map[string]interface{}{"files": []string{"source.go"}, "approach": map[string]interface{}{
			"summary": "Quality", "rules": []string{"Preserve contract"}, "behavior_checks": []interface{}{map[string]interface{}{"name": "acceptance", "command": "exit 7", "protected_files": []interface{}{map[string]interface{}{"path": "spec.txt"}}}},
		}},
	})
	if err == nil {
		t.Fatal("MCP ignored failing behavior check")
	}
	r := result.(*continuity.RecordResult)
	if r.Recorded || len(r.BehaviorChecks) != 1 || r.BehaviorChecks[0].ExitCode != 7 {
		t.Fatalf("behavior evidence missing: %+v", r)
	}
	task, _ := store.ReadTask("quality")
	if task.Approach != nil {
		t.Fatal("failed MCP capture installed memory")
	}
}
