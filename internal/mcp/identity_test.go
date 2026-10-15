package mcp

import (
	"knodexa/internal/continuity"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"testing"
)

func TestMCPResolvesNaturalRequestWithoutTaskID(t *testing.T) {
	root := t.TempDir()
	store := storage.NewStorage(root)
	if _, err := store.Init("MCP identity", ""); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTask(&model.TaskData{ID: "known-work", Title: "Inbound event delivery", Status: model.TaskStatusCompleted}); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(root)
	if err != nil {
		t.Fatal(err)
	}
	defer server.handler.db.Close()
	if _, err := server.handler.HandleTool("knodexa_add_work_aliases", map[string]interface{}{"task_id": "known-work", "aliases": []string{"webhook callbacks"}}); err != nil {
		t.Fatal(err)
	}
	value, err := server.handler.HandleTool("knodexa_resolve_request", map[string]interface{}{"request": "Extend webhook callbacks"})
	if err != nil {
		t.Fatal(err)
	}
	result, ok := value.(*continuity.ResolvedRequest)
	if !ok || result.SelectedTask != "known-work" || result.Context.Task.Status != "COMPLETED" {
		t.Fatalf("MCP lost identity: %+v", value)
	}
	preparedValue, err := server.handler.HandleTool("knodexa_prepare_work", map[string]interface{}{"request": "Extend webhook callbacks"})
	if err != nil {
		t.Fatal(err)
	}
	prepared, ok := preparedValue.(*continuity.PreparedWork)
	if !ok || prepared.SelectedTask != "known-work" || !prepared.NeedsReview {
		t.Fatalf("missing agreement must require review: %+v", preparedValue)
	}
	for _, args := range []map[string]interface{}{{"request": 7}, {"request": "x", "files": "not an array"}, {"request": "x", "previous_task_id": 7}, {"request": "x", "token_budget": 0}} {
		if _, err := server.handler.HandleTool("knodexa_resolve_request", args); err == nil {
			t.Fatalf("accepted invalid MCP request: %+v", args)
		}
	}
}
