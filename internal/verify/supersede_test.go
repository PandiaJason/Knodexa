package verify

import (
	"bytes"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"os"
	"path/filepath"
	"testing"
)

func TestSupersessionValidatesReplacementAndPreservesHistoricalLesson(t *testing.T) {
	store := storage.NewStorage(t.TempDir())
	if _, err := store.Init("retirement", ""); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := index.NewIndexer(db, store)
	engine := NewEngine(store, idx, db)
	old := &model.LessonDocument{Frontmatter: model.LessonFrontmatter{ID: "L-old", SourceTaskID: "task", Status: model.StatusVerified}, Body: "Historical design and evidence remain inspectable."}
	store.SaveLesson(old)
	store.SaveLesson(&model.LessonDocument{Frontmatter: model.LessonFrontmatter{ID: "L-proposal", Status: model.StatusProposed}})
	store.SaveLesson(&model.LessonDocument{Frontmatter: model.LessonFrontmatter{ID: "L-new", SourceTaskID: "task", Status: model.StatusVerified}, Body: "New agreed design."})
	store.SaveDecision(&model.DecisionDocument{Frontmatter: model.DecisionFrontmatter{ID: "D-new", Status: model.StatusVerified}})
	path := filepath.Join(store.KnodexaDir(), "lessons", "L-old.md")
	before, _ := os.ReadFile(path)
	original, err := store.ReadLesson("L-old")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ replacement, reason string }{{"L-old", "self"}, {"L-missing", "missing"}, {"L-proposal", "unverified"}, {"D-new", "wrong kind"}, {"L-new", " "}} {
		if err := engine.SupersedeNode("L-old", test.replacement, test.reason, "test"); err == nil {
			t.Fatalf("invalid replacement accepted: %+v", test)
		}
		after, _ := os.ReadFile(path)
		if !bytes.Equal(before, after) {
			t.Fatal("invalid replacement changed canonical memory")
		}
	}
	if err := engine.SupersedeNode("L-old", "L-new", "Reviewed replacement after interface migration", "test"); err != nil {
		t.Fatal(err)
	}
	saved, err := store.ReadLesson("L-old")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Body != original.Body || saved.Frontmatter.SupersededBy != "L-new" || saved.Frontmatter.InvalidatedAt == nil || saved.Frontmatter.SupersessionReason == "" {
		t.Fatalf("history/provenance lost: %+v", saved)
	}
	if err := engine.SupersedeNode("L-old", "L-new", "overwrite retirement history", "test"); err == nil {
		t.Fatal("retirement rewritten")
	}
	if err := idx.Rebuild(); err != nil {
		t.Fatal(err)
	}
	active, err := db.GetRelatedLessons("task", nil, nil, 5)
	if err != nil || len(active) != 1 || active[0].ID != "L-new" {
		t.Fatalf("obsolete lesson injected: %+v %v", active, err)
	}
}
