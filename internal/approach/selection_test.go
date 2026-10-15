package approach

import (
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"path/filepath"
	"testing"
)

func TestScopeRoutingIsIndependentOfNamesAndRecency(t *testing.T) {
	store := storage.NewStorage(t.TempDir())
	if _, err := store.Init("scope", ""); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := index.NewIndexer(db, store)
	for _, task := range []*model.TaskData{
		{ID: "service-design", Approach: &model.TaskApproach{Summary: "inject service", Rules: []string{"delegate"}, Scopes: []string{"handlers/"}}},
		{ID: "recent-image-work", Approach: &model.TaskApproach{Summary: "embed transformation", Rules: []string{"uppercase directly"}, Scopes: []string{"images/"}}},
		{ID: "unrelated-name", TouchesFiles: []string{"handlers/webhook.py"}},
		{ID: "mixed", TouchesFiles: []string{"handlers/webhook.py", "images/thumbnail.py"}},
	} {
		if err := store.SaveTask(task); err != nil {
			t.Fatal(err)
		}
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	contracts, selection, err := Select(db, "unrelated-name")
	if err != nil {
		t.Fatal(err)
	}
	if selection.Mode != "file_scope" || len(contracts) != 1 || contracts[0].SourceTask != "service-design" || len(selection.ExcludedSources) != 1 {
		t.Fatalf("wrong selection: %+v %+v", selection, contracts)
	}
	_, mixed, err := Select(db, "mixed")
	if err != nil {
		t.Fatal(err)
	}
	if mixed.Mode != "missing" {
		t.Fatal("partial overlap became authority")
	}
	if err := store.SaveTask(&model.TaskData{ID: "conflicting-design", Approach: &model.TaskApproach{Summary: "different", Rules: []string{"embed"}, Scopes: []string{"handlers/*.py"}}}); err != nil {
		t.Fatal(err)
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	contracts, selection, err = Select(db, "unrelated-name")
	if err != nil {
		t.Fatal(err)
	}
	if selection.Mode != "ambiguous" || len(contracts) != 0 || !selection.ReviewRequired {
		t.Fatal("ambiguous conventions were silently merged")
	}
	if err := Reuse(store, db, idx, "unrelated-name", []string{"service-design"}); err != nil {
		t.Fatal(err)
	}
	contracts, selection, err = Select(db, "unrelated-name")
	if err != nil {
		t.Fatal(err)
	}
	if selection.Mode != "explicit" || len(contracts) != 1 {
		t.Fatal("explicit agreement did not resolve ambiguity")
	}
	for _, scope := range []string{"../escape/", "/absolute/", "[broken"} {
		if err := Set(store, db, idx, "service-design", model.TaskApproach{Summary: "x", Rules: []string{"x"}, Scopes: []string{scope}}); err == nil {
			t.Fatalf("accepted invalid scope %q", scope)
		}
	}
}
