package continuity

import (
	"bytes"
	"knodexa/internal/context"
	"knodexa/internal/git"
	"knodexa/internal/identity"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"os"
	"path/filepath"
	"testing"
)

func TestRequestRestoresCompletedWorkWithoutCreatingOrReopeningTask(t *testing.T) {
	root := t.TempDir()
	store := storage.NewStorage(root)
	if _, err := store.Init("requests", ""); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := index.NewIndexer(db, store)
	inspector := git.NewInspector(root)
	service := NewService(store, db, idx, context.NewBuilder(db, store, inspector), inspector)
	for _, task := range []*model.TaskData{
		{ID: "callbacks", Title: "Inbound event delivery", Aliases: []string{"webhook delivery"}, Goal: "Use established service adapters", Status: model.TaskStatusCompleted, Constraints: []string{"Keep service delegation"}, RequiredVerification: []string{"Observer ordering"}, Approach: &model.TaskApproach{Summary: "Injected services", Rules: []string{"Preserve service result identity"}}},
		{ID: "images", Title: "Image normalization", Status: model.TaskStatusActive, Approach: &model.TaskApproach{Summary: "Inline transforms", Rules: []string{"Uppercase inline"}}},
	} {
		if err := store.SaveTask(task); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SaveFailure(&model.FailureData{ID: "F-retry", TaskID: "callbacks", Summary: "Do not retry handler calls"}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(store.KnodexaDir(), "tasks", "callbacks", "callbacks.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.ResolveRequest(identity.Request{Text: "Extend webhook delivery"}, 1, ResumeOptions{PreviousTaskID: "images"})
	if err != nil {
		t.Fatal(err)
	}
	if result.SelectedTask != "callbacks" || result.Context == nil || result.Context.Task.Status != "COMPLETED" || !result.Context.BudgetExceeded || len(result.Context.RequiredVerification) != 1 || len(result.Context.PreviousFailures) != 1 {
		t.Fatalf("incomplete identity restoration: %+v %+v", result, result.Context)
	}
	if len(result.Context.ContextBoundary.DiscardSources) != 1 || result.Context.ContextBoundary.DiscardSources[0] != "images" {
		t.Fatal("missing history boundary")
	}
	after, err := os.ReadFile(filepath.Join(store.KnodexaDir(), "tasks", "callbacks", "callbacks.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("request resolution mutated prior work")
	}
	tasks, err := store.ListTasks()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Fatal("resolution generated a new task")
	}
	if err := store.SaveFocus("images"); err != nil {
		t.Fatal(err)
	}
	automatic, err := service.ResolveRequest(identity.Request{Text: "Extend webhook delivery"}, 1500, ResumeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if automatic.SelectedTask != "callbacks" || automatic.Context.ContextBoundary.PreviousTask != "images" || len(automatic.Context.ContextBoundary.DiscardSources) != 1 {
		t.Fatal("advisory focus did not restore exclusion boundary")
	}
	if err := store.SaveFocus("missing"); err != nil {
		t.Fatal(err)
	}
	stale, err := service.ResolveRequest(identity.Request{Text: "Extend webhook delivery"}, 1500, ResumeOptions{})
	if err != nil || stale.SelectedTask != "callbacks" {
		t.Fatal("stale focus blocked target identity")
	}
	if err := store.SaveFocus("images"); err != nil {
		t.Fatal(err)
	}
	unknown, err := service.ResolveRequest(identity.Request{Text: "Orbital telescope scheduling"}, 1500, ResumeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if unknown.Context != nil || unknown.CreateNewWorkAllowed {
		t.Fatal("unmatched request received invented work context")
	}
	if unknown.SelectedTask != "" {
		t.Fatal("focus was used as identity authority")
	}
}
