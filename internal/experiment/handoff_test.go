package experiment

import (
	"fmt"
	"knodexa/internal/context"
	"knodexa/internal/continuity"
	"knodexa/internal/git"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"path/filepath"
	"testing"
)

// This is a lifecycle integration test, not a model-performance comparison.
// A fresh reader must recover prior failures and another task's lessons after
// intervening edits, irrelevant tasks, and a cache rebuild.
func TestFreshAgentHandoffAfterTaskSwitches(t *testing.T) {
	root := t.TempDir()
	store := storage.NewStorage(root)
	if _, err := store.Init("Handoff", ""); err != nil {
		t.Fatal(err)
	}
	for _, task := range []*model.TaskData{{ID: "ROOT", Constraints: []string{"Keep public API"}}, {ID: "A", ParentID: "ROOT"}, {ID: "A.1", ParentID: "A", TouchesFiles: []string{"auth.go"}, RequiredVerification: []string{"Expiry boundary acceptance"}}, {ID: "B", ParentID: "ROOT", TouchesFiles: []string{"auth.go"}}} {
		if err := store.SaveTask(task); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SaveFailure(&model.FailureData{ID: "F-coerce", TaskID: "A", Summary: "Avoid int coercion", CausalDetails: "It accepts strings and truncates floats"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveLesson(&model.LessonDocument{Frontmatter: model.LessonFrontmatter{ID: "L-bool", SourceTaskID: "B", Title: "Reject booleans", Status: model.StatusVerified, Applicability: "When accepting integer timestamps", RelatedFiles: []string{"auth.go"}}, Body: "bool subclasses int; reject bool explicitly"}); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	idx := index.NewIndexer(db, store)
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	// Switch away, creating unrelated work and changing an obsolete constraint.
	for i := 0; i < 30; i++ {
		id := fmt.Sprintf("image-%d", i)
		if err := store.SaveTask(&model.TaskData{ID: id, Constraints: []string{"Image rule"}}); err != nil {
			t.Fatal(err)
		}
		if err := store.SaveLesson(&model.LessonDocument{Frontmatter: model.LessonFrontmatter{ID: "L-" + id, SourceTaskID: id, Status: model.StatusVerified, Confidence: 1, RelatedFiles: []string{"images.png"}}, Body: "PNG advice"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	// New process-equivalent index/builder objects have no in-memory history.
	db, err = index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx = index.NewIndexer(db, store)
	inspector := git.NewInspector(root)
	service := continuity.NewService(store, db, idx, context.NewBuilder(db, store, inspector), inspector)
	first, err := service.Resume("A.1", 1500, continuity.ResumeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.PreviousFailures) != 1 || first.PreviousFailures[0].ID != "F-coerce" {
		t.Fatal("prior failure not recovered")
	}
	if len(first.RelevantLessons) != 1 || first.RelevantLessons[0].SourceTask != "B" {
		t.Fatal("cross-task lesson lost or polluted by unrelated tasks")
	}
	if len(first.RequiredVerification) != 1 || first.RequiredVerification[0] != "Expiry boundary acceptance" {
		t.Fatal("acceptance lost during handoff")
	}
	if err := idx.Rebuild(); err != nil {
		t.Fatal(err)
	}
	rebuilt, err := service.Resume("A.1", 1500, continuity.ResumeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if first.FormatHuman() != rebuilt.FormatHuman() {
		t.Fatal("rebuild changed handoff context")
	}
}
