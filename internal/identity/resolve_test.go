package identity

import (
	"fmt"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"path/filepath"
	"testing"
)

func fixture(t testing.TB) (*storage.Storage, *index.DB, *index.Indexer) {
	t.Helper()
	store := storage.NewStorage(t.TempDir())
	if _, err := store.Init("identity", ""); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	idx := index.NewIndexer(db, store)
	tasks := []*model.TaskData{
		{ID: "work-72", Title: "Webhook delivery", Goal: "Deliver inbound callbacks through injected services", Aliases: []string{"inbound callbacks", "callback delivery"}, Status: model.TaskStatusCompleted, TouchesFiles: []string{"handlers/reference.go"}, Approach: &model.TaskApproach{Summary: "Service delegation", Rules: []string{"Preserve original output"}, Scopes: []string{"handlers/"}}},
		{ID: "signature-verify", ParentID: "work-72", Title: "Webhook signature validation", Goal: "Authenticate incoming delivery requests", Status: model.TaskStatusCompleted, TouchesFiles: []string{"handlers/verify.go"}},
		{ID: "recent-focus", Title: "Image normalization", Goal: "Normalize image names with inline uppercase transformations", Aliases: []string{"image sizing"}, Status: model.TaskStatusActive, TouchesFiles: []string{"images/normalize.go"}, Approach: &model.TaskApproach{Summary: "Inline transforms", Rules: []string{"Uppercase inline"}, Scopes: []string{"images/"}}},
	}
	for _, task := range tasks {
		if err := store.SaveTask(task); err != nil {
			t.Fatal(err)
		}
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	return store, db, idx
}

func TestNaturalRequestRestoresIdentityWithoutTaskIDs(t *testing.T) {
	store, db, idx := fixture(t)
	for _, text := range []string{"Add error reporting to webhook delivery", "Support another endpoint for inbound callbacks", "Use callback delivery for another event"} {
		result, err := Resolve(db, Request{Text: text})
		if err != nil {
			t.Fatal(err)
		}
		if result.Status != "resolved" || result.SelectedTask != "work-72" || result.WorkID != "work-72" || result.CreateNewWorkAllowed || result.NextAction != "continue_existing_work" {
			t.Fatalf("lost existing identity: %+v", result)
		}
	}
	// Different labels and unrelated completed work do not change the result.
	for i := 0; i < 30; i++ {
		if err := store.SaveTask(&model.TaskData{ID: fmt.Sprintf("distractor-%d", i), Title: "Image processing", Goal: "Inline normalization", Status: model.TaskStatusActive}); err != nil {
			t.Fatal(err)
		}
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	result, err := Resolve(db, Request{Text: "Extend inbound callbacks"})
	if err != nil {
		t.Fatal(err)
	}
	if result.SelectedTask != "work-72" {
		t.Fatal("recent unrelated work displaced identity")
	}
	task, err := store.ReadTask("work-72")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != model.TaskStatusCompleted {
		t.Fatal("resolution reopened completed work")
	}
}

func TestAmbiguousWeakAndUnmatchedRequestsDoNotCreateWork(t *testing.T) {
	store, db, idx := fixture(t)
	result, err := Resolve(db, Request{Text: "Improve webhook handling"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "needs_clarification" || result.SelectedTask != "" || result.CreateNewWorkAllowed {
		t.Fatalf("guessed weak evidence: %+v", result)
	}
	result, err = Resolve(db, Request{Text: "Build an orbital telescope scheduler"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "unmatched" || result.CreateNewWorkAllowed {
		t.Fatal("unknown request silently became new work")
	}
	for i := 0; i < 10; i++ {
		if err := store.SaveTask(&model.TaskData{ID: fmt.Sprintf("alias-conflict-%d", i), Title: "Other implementation", Aliases: []string{"inbound callbacks"}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	result, err = Resolve(db, Request{Text: "Extend inbound callbacks"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "needs_clarification" || result.CandidateCount != 11 || len(result.Candidates) != 8 || !result.CandidatesTruncated {
		t.Fatalf("truncation hid ambiguity: %+v", result)
	}
}

func TestFilesAndExplicitIntent(t *testing.T) {
	_, db, _ := fixture(t)
	cases := []struct {
		request                  Request
		status, selected, action string
		allowNew                 bool
	}{
		{Request{Text: "Add another endpoint", Files: []string{"handlers/new.go"}}, "resolved", "work-72", "continue_existing_work", false},
		{Request{Text: "Change handler and image paths", Files: []string{"handlers/new.go", "images/new.go"}}, "needs_clarification", "", "clarify_work_identity", false},
		{Request{Text: "Extend inbound callbacks", Intent: "new"}, "new_work_requested", "", "propose_new_work", true},
		{Request{Text: "Replace webhook delivery", Intent: "replace"}, "resolved", "work-72", "review_replacement", false},
		{Request{Text: "Improve webhook handling", SelectTask: "signature-verify"}, "resolved", "signature-verify", "continue_existing_work", false},
	}
	for _, c := range cases {
		result, err := Resolve(db, c.request)
		if err != nil {
			t.Fatal(err)
		}
		if result.Status != c.status || result.SelectedTask != c.selected || result.NextAction != c.action || result.CreateNewWorkAllowed != c.allowNew {
			t.Fatalf("request %+v -> %+v", c.request, result)
		}
	}
	result, err := Resolve(db, Request{Text: "Unclear request", SelectTask: "signature-verify"})
	if err != nil {
		t.Fatal(err)
	}
	if result.WorkID != "work-72" {
		t.Fatal("lost explicit family identity")
	}
	for _, request := range []Request{{Text: " "}, {Text: "Anything", Intent: "invent"}, {Text: "Anything", SelectTask: "missing"}, {Text: "Anything", Files: []string{"../outside.go"}}, {Text: "Anything", Intent: "new", SelectTask: "work-72"}} {
		if _, err := Resolve(db, request); err == nil {
			t.Fatalf("invalid request accepted: %+v", request)
		}
	}
}

func TestAliasEditsAndDeletionsUpdateIdentityIndex(t *testing.T) {
	store, db, idx := fixture(t)
	if err := AddAliases(store, db, idx, "work-72", []string{"event notification"}); err != nil {
		t.Fatal(err)
	}
	result, err := Resolve(db, Request{Text: "Extend event notification"})
	if err != nil {
		t.Fatal(err)
	}
	if result.SelectedTask != "work-72" {
		t.Fatal("alias not indexed")
	}
	task, err := store.ReadTask("work-72")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != model.TaskStatusCompleted {
		t.Fatal("alias edit reopened work")
	}
	task.Aliases = nil
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	result, err = Resolve(db, Request{Text: "Extend event notification"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "unmatched" {
		t.Fatal("removed alias retained authority")
	}
	if err := AddAliases(store, db, idx, "work-72", []string{"another task"}); err == nil {
		t.Fatal("accepted meaningless alias")
	}
}

func BenchmarkRequestResolution(b *testing.B) {
	store, db, idx := fixture(b)
	for i := 0; i < 500; i++ {
		if err := store.SaveTask(&model.TaskData{ID: fmt.Sprintf("task-%d", i), Title: fmt.Sprintf("Image variant %d", i), Goal: "Image pipeline changes"}); err != nil {
			b.Fatal(err)
		}
	}
	if err := idx.Sync(); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := Resolve(db, Request{Text: "Extend inbound callbacks"})
		if err != nil || result.SelectedTask != "work-72" {
			b.Fatalf("bad resolution: %v", err)
		}
	}
}
