package index

import (
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"path/filepath"
	"testing"
)

func TestLessonRelevanceOutranksConfidenceAndSurvivesRebuild(t *testing.T) {
	store := storage.NewStorage(t.TempDir())
	if _, err := store.Init("relevance", ""); err != nil {
		t.Fatal(err)
	}
	db, err := Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := NewIndexer(db, store)
	for _, item := range []struct {
		id, source  string
		confidence  float64
		files, tags []string
		status      model.KnowledgeStatus
	}{
		{"L-own", "target", 0.15, nil, nil, model.StatusVerified},
		{"L-source", "agreement", 0.1, nil, nil, model.StatusVerified},
		{"L-link", "elsewhere", 0.2, nil, nil, model.StatusVerified},
		{"L-file", "elsewhere", 0.3, []string{"handler.go"}, nil, model.StatusVerified},
		{"L-tag", "elsewhere", 1, nil, []string{"http"}, model.StatusVerified},
		{"L-noise", "elsewhere", 1, nil, nil, model.StatusVerified},
		{"L-retired", "agreement", 1, []string{"handler.go"}, nil, model.StatusSuperseded},
		{"L-unverified", "agreement", 1, nil, nil, model.StatusProposed},
	} {
		if err := store.SaveLesson(&model.LessonDocument{Frontmatter: model.LessonFrontmatter{ID: item.id, SourceTaskID: item.source, Confidence: item.confidence, RelatedFiles: item.files, Tags: item.tags, Status: item.status}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.AppendLink(&model.Edge{FromID: "L-link", ToID: "target", Type: model.EdgeRelevantTo}); err != nil {
		t.Fatal(err)
	}
	for _, sync := range []func() error{idx.Sync, idx.Rebuild} {
		if err := sync(); err != nil {
			t.Fatal(err)
		}
		lessons, err := db.GetRelatedLessonsForSources("target", []string{"agreement", "agreement"}, []string{"handler.go"}, []string{"http"}, 5)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"L-own", "L-source", "L-link", "L-file", "L-tag"}
		if len(lessons) != len(want) {
			t.Fatalf("bad admission: %+v", lessons)
		}
		for i, id := range want {
			if lessons[i].ID != id || lessons[i].MatchReason == "" {
				t.Fatalf("confidence crowded out relevance: %+v", lessons)
			}
		}
		limited, err := db.GetRelatedLessonsForSources("target", []string{"agreement"}, []string{"handler.go"}, []string{"http"}, 2)
		if err != nil || len(limited) != 2 || limited[1].ID != "L-source" {
			t.Fatalf("limit lost selected memory: %+v %v", limited, err)
		}
	}
}
