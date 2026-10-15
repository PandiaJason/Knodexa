package index

import (
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"os"
	"path/filepath"
	"testing"
)

func TestIncrementalNoOpContentEditsAndMoves(t *testing.T) {
	store := storage.NewStorage(t.TempDir())
	if _, err := store.Init("Incremental", ""); err != nil {
		t.Fatal(err)
	}
	db, err := Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := NewIndexer(db, store)
	task := &model.TaskData{ID: "T", Title: "before", Constraints: []string{"OLD"}}
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	initial, err := idx.SyncWithStats()
	if err != nil {
		t.Fatal(err)
	}
	if initial.UpdatedRecords != 3 {
		t.Fatalf("unexpected initial stats: %+v", initial)
	}
	var before, after int64
	if err := db.sqlDB.QueryRow("SELECT total_changes()").Scan(&before); err != nil {
		t.Fatal(err)
	}
	steady, err := idx.SyncWithStats()
	if err != nil {
		t.Fatal(err)
	}
	if err := db.sqlDB.QueryRow("SELECT total_changes()").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if steady.UpdatedRecords != 0 || steady.DeletedRecords != 0 || before != after {
		t.Fatalf("unchanged sync wrote rows: %+v, %d->%d", steady, before, after)
	}
	path := filepath.Join(store.KnodexaDir(), "tasks", "T", "T.yaml")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	task.Title = "edited"
	task.Constraints = []string{"NEW"}
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	edited, err := idx.SyncWithStats()
	if err != nil {
		t.Fatal(err)
	}
	if edited.UpdatedRecords != 1 {
		t.Fatalf("single edit touched other records: %+v", edited)
	}
	got, err := db.GetTaskData("T")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "edited" || len(got.Constraints) != 1 || got.Constraints[0] != "NEW" {
		t.Fatalf("same-mtime content edit missed: %+v", got)
	}
	task.ParentID = "parent"
	if err := store.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	moved, err := idx.SyncWithStats()
	if err != nil {
		t.Fatal(err)
	}
	if moved.UpdatedRecords != 1 || moved.DeletedRecords != 1 {
		t.Fatalf("move not reconciled: %+v", moved)
	}
	got, err = db.GetTaskData("T")
	if err != nil {
		t.Fatal(err)
	}
	if got.ParentID != "parent" {
		t.Fatal("moved task metadata lost")
	}
	// Reopening the cache preserves the fingerprints across CLI sessions.
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	reopened, err := NewIndexer(db, store).SyncWithStats()
	if err != nil {
		t.Fatal(err)
	}
	if reopened.UpdatedRecords != 0 {
		t.Fatalf("reopening rebuilt records: %+v", reopened)
	}
}

func TestIncrementalSharedEdgeOwnershipAndDuplicateRollback(t *testing.T) {
	store := storage.NewStorage(t.TempDir())
	if _, err := store.Init("Owners", ""); err != nil {
		t.Fatal(err)
	}
	db, err := Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := NewIndexer(db, store)
	d := &model.DecisionDocument{Frontmatter: model.DecisionFrontmatter{ID: "D-test", Status: model.StatusVerified, EvidenceIDs: []string{"E-test"}}}
	if err := store.SaveDecision(d); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveEvidence(&model.EvidenceData{ID: "E-test", NodeID: "D-test", EvidenceType: model.EvidenceTestRun}); err != nil {
		t.Fatal(err)
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	d.Frontmatter.EvidenceIDs = nil
	if err := store.SaveDecision(d); err != nil {
		t.Fatal(err)
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	edges, err := db.GetEdgesFrom("D-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 1 || edges[0].ToID != "E-test" {
		t.Fatal("editing one owner erased the other owner's edge")
	}
	if err := os.Remove(filepath.Join(store.KnodexaDir(), "evidence", "E-test.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	edges, err = db.GetEdgesFrom("D-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 0 {
		t.Fatal("unowned edge retained")
	}
	if err := store.SaveTask(&model.TaskData{ID: "T", Title: "original"}); err != nil {
		t.Fatal(err)
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	// Simulate an external malformed duplicate; SaveTask now refuses blind writes.
	duplicatePath := filepath.Join(store.KnodexaDir(), "tasks", "other", "T.yaml")
	if err := os.MkdirAll(filepath.Dir(duplicatePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(duplicatePath, []byte("id: T\nparent_id: other\ntitle: conflicting\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := idx.Sync(); err == nil {
		t.Fatal("duplicate canonical ID accepted")
	}
	n, err := db.GetNode("T")
	if err != nil {
		t.Fatal(err)
	}
	if n.Title != "original" {
		t.Fatal("duplicate error failed to roll back")
	}
	if err := idx.Rebuild(); err == nil {
		t.Fatal("rebuild accepted duplicate")
	}
	n, err = db.GetNode("T")
	if err != nil || n.Title != "original" {
		t.Fatal("failed rebuild erased previous snapshot")
	}
}
