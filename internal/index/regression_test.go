package index

import (
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncReconcilesAndRollsBack(t *testing.T) {
	store := storage.NewStorage(t.TempDir())
	if _, err := store.Init("Snapshot", "Regression"); err != nil {
		t.Fatal(err)
	}
	db, err := Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := NewIndexer(db, store)
	task := &model.TaskData{ID: "T", Constraints: []string{"OLD"}, TouchesFiles: []string{"old.go"}}
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	lesson := &model.LessonDocument{Frontmatter: model.LessonFrontmatter{ID: "L-test", SourceTaskID: "source", Status: model.StatusVerified, Tags: []string{"old-tag"}, RelatedFiles: []string{"old.go"}}, Body: "Guide"}
	if err := store.SaveLesson(lesson); err != nil {
		t.Fatal(err)
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	task.Constraints = []string{"NEW"}
	task.TouchesFiles = []string{"new.go"}
	lesson.Frontmatter.Tags = []string{"new-tag"}
	lesson.Frontmatter.RelatedFiles = []string{"new.go"}
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveLesson(lesson); err != nil {
		t.Fatal(err)
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"SELECT count(*) FROM node_constraints WHERE constraint_text='OLD'", "SELECT count(*) FROM node_files WHERE file_path='old.go'", "SELECT count(*) FROM node_tags WHERE tag='old-tag'"} {
		var count int
		if err := db.SqlDB().QueryRow(query).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("stale rows for %s", query)
		}
	}
	// A malformed canonical record must preserve the previous committed snapshot.
	broken := filepath.Join(store.KnodexaDir(), "lessons", "L-broken.md")
	if err := os.WriteFile(broken, []byte("broken frontmatter"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := idx.Sync(); err == nil {
		t.Fatal("malformed record accepted")
	}
	if _, err := db.GetNode("T"); err != nil {
		t.Fatalf("failed sync erased snapshot: %v", err)
	}
	if err := os.Remove(broken); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(store.KnodexaDir(), "tasks", "T", "T.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(store.KnodexaDir(), "lessons", "L-test.md")); err != nil {
		t.Fatal(err)
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.GetNode("T"); err == nil {
		t.Fatal("deleted task retained")
	}
	if _, err := db.GetNode("L-test"); err == nil {
		t.Fatal("deleted lesson retained")
	}
}

func TestLessonMatchingUsesTagsFilesAndLinks(t *testing.T) {
	store := storage.NewStorage(t.TempDir())
	if _, err := store.Init("Matches", ""); err != nil {
		t.Fatal(err)
	}
	db, err := Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := NewIndexer(db, store)
	for _, id := range []string{"L-b", "L-a"} {
		if err := store.SaveLesson(&model.LessonDocument{Frontmatter: model.LessonFrontmatter{ID: id, SourceTaskID: "source", Status: model.StatusVerified, Confidence: 0.9, Tags: []string{"auth"}, RelatedFiles: []string{"auth.go"}}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SaveTask(&model.TaskData{ID: "task"}); err != nil {
		t.Fatal(err)
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	none, err := db.GetRelatedLessons("task", []string{"image.png"}, []string{"images"}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(none) != 0 {
		t.Fatalf("noise: %+v", none)
	}
	for _, args := range []struct{ files, tags []string }{{[]string{"auth.go"}, nil}, {nil, []string{"auth"}}} {
		matches, err := db.GetRelatedLessons("task", args.files, args.tags, 5)
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) != 2 || matches[0].ID != "L-a" || len(matches[0].TouchesFiles) != 1 {
			t.Fatalf("bad matching/order: %+v", matches)
		}
	}
	if err := store.AppendLink(&model.Edge{FromID: "L-b", ToID: "task", Type: model.EdgeRelevantTo}); err != nil {
		t.Fatal(err)
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	linked, err := db.GetRelatedLessons("task", nil, nil, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(linked) != 1 || linked[0].ID != "L-b" {
		t.Fatalf("explicit link lost: %+v", linked)
	}
}
