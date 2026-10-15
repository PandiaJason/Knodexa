package context

import (
	"knodexa/internal/approach"
	"knodexa/internal/git"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSelectedSourceLessonsRemainDiscoverableUnderBudget(t *testing.T) {
	root := t.TempDir()
	store := storage.NewStorage(root)
	if _, err := store.Init("tiers", ""); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := index.NewIndexer(db, store)
	store.SaveTask(&model.TaskData{ID: "source", Constraints: []string{"Preserve raw results"}, RequiredVerification: []string{"Test service delegation"}})
	store.SaveTask(&model.TaskData{ID: "child", ParentID: "source"})
	if err := approach.Set(store, db, idx, "source", model.TaskApproach{Summary: "Delegate", Rules: []string{"Keep injected service"}}); err != nil {
		t.Fatal(err)
	}
	store.SaveLesson(&model.LessonDocument{Frontmatter: model.LessonFrontmatter{ID: "L-source", SourceTaskID: "source", Status: model.StatusVerified, Applicability: "When extending adapters", EvidenceIDs: []string{"E-fixture"}}, Body: "Reuse the established call order."})
	store.SaveFailure(&model.FailureData{ID: "F-source", TaskID: "source", Summary: "Do not swallow observer failures"})
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	builder := NewBuilder(db, store, git.NewInspector(root))
	full, err := builder.BuildCompact("child", 3000)
	if err != nil {
		t.Fatal(err)
	}
	if len(full.RelevantLessons) != 1 || full.RelevantLessons[0].MatchReason != "selected_source" || len(full.RelevantLessons[0].EvidenceIDs) != 1 {
		t.Fatalf("selected provenance missing: %+v", full.RelevantLessons)
	}
	tiny, err := builder.BuildCompact("child", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tiny.AcceptedConstraints, full.AcceptedConstraints) || len(tiny.RequiredVerification) != 1 || len(tiny.EstablishedApproach) != 1 || len(tiny.PreviousFailures) != 1 || !tiny.BudgetExceeded {
		t.Fatalf("essential memory pruned: %+v", tiny)
	}
	if tiny.DeferredCount != 1 || len(tiny.DeferredMemory) != 1 || tiny.DeferredMemory[0].ID != "L-source" {
		t.Fatalf("pruned detail became undiscoverable: %+v", tiny)
	}
	detail, err := builder.GetDetail(tiny.DeferredMemory[0].ID)
	if err != nil || strings.TrimSpace(detail.(*model.LessonDocument).Body) != "Reuse the established call order." {
		t.Fatal("deferred lesson cannot be expanded")
	}
}
