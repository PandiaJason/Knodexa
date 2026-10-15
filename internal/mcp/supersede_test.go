package mcp

import (
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"testing"
)

func TestSupersessionToolValidatesAndRetainsProvenance(t *testing.T) {
	root := t.TempDir()
	store := storage.NewStorage(root)
	if _, err := store.Init("MCP replacement", ""); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"D-old", "D-new"} {
		if err := store.SaveDecision(&model.DecisionDocument{Frontmatter: model.DecisionFrontmatter{ID: id, Status: model.StatusVerified}, Body: "Design rationale"}); err != nil {
			t.Fatal(err)
		}
	}
	server, err := NewServer(root)
	if err != nil {
		t.Fatal(err)
	}
	defer server.handler.db.Close()
	if _, err := server.handler.HandleTool("knodexa_supersede_knowledge", map[string]interface{}{"node_id": "D-old", "replacement_id": "D-missing", "reason": "migration"}); err == nil {
		t.Fatal("missing replacement accepted")
	}
	if _, err := server.handler.HandleTool("knodexa_supersede_knowledge", map[string]interface{}{"node_id": "D-old", "replacement_id": "D-new", "reason": "migration"}); err != nil {
		t.Fatal(err)
	}
	detail, err := server.handler.context.GetDetail("D-old")
	if err != nil {
		t.Fatal(err)
	}
	old := detail.(*model.DecisionDocument)
	if old.Frontmatter.InvalidatedAt == nil || old.Frontmatter.SupersessionReason != "migration" || old.Frontmatter.SupersededBy != "D-new" {
		t.Fatal("MCP lost retirement provenance")
	}
}
