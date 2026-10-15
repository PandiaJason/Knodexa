package approach

import (
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReturnToBranchRestoresApproachAndDetectsDrift(t *testing.T) {
	root := t.TempDir()
	store := storage.NewStorage(root)
	if _, err := store.Init("Continuity", ""); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := index.NewIndexer(db, store)
	for _, task := range []*model.TaskData{{ID: "A"}, {ID: "A.1", ParentID: "A"}, {ID: "A.2", ParentID: "A"}, {ID: "B"}, {ID: "B.1", ParentID: "B"}} {
		if err := store.SaveTask(task); err != nil {
			t.Fatal(err)
		}
	}
	reference := "func NewHandler(service Service) {}"
	if err := os.WriteFile(filepath.Join(root, "reference.go"), []byte(reference), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "candidate.go"), []byte(reference), 0644); err != nil {
		t.Fatal(err)
	}
	a := model.TaskApproach{Summary: "Use injected services for handlers", Rules: []string{"Preserve dependency injection and error wrapping"}, References: []model.ApproachReference{{Path: "reference.go"}}, Checks: []model.ApproachCheck{{Path: "candidate.go", MustContain: "NewHandler(service Service)", MustNotContain: "sql.Open"}}}
	if err := Set(store, db, idx, "A", a); err != nil {
		t.Fatal(err)
	}
	if err := Set(store, db, idx, "B", model.TaskApproach{Summary: "B uses a different architecture", Rules: []string{"B-specific rule"}}); err != nil {
		t.Fatal(err)
	}
	if err := Set(store, db, idx, "A.1", model.TaskApproach{Summary: "A.1 error handling", Rules: []string{"Use typed errors"}}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"A", "A.1", "A.2"} {
		contracts, err := Resolve(db, id)
		if err != nil {
			t.Fatal(err)
		}
		if len(contracts) == 0 || contracts[0].SourceTask != "A" {
			t.Fatal("A contract lost")
		}
		for _, c := range contracts {
			if c.SourceTask == "B" {
				t.Fatal("unrelated B approach contaminated A")
			}
		}
	}
	contracts, err := Resolve(db, "A.1")
	if err != nil {
		t.Fatal(err)
	}
	if len(contracts) != 2 || contracts[1].SourceTask != "A.1" {
		t.Fatal("child approach failed to extend parent")
	}
	report, err := Check(store, db, "A.2")
	if err != nil {
		t.Fatal(err)
	}
	if !report.StructuralChecksPassed || !report.SemanticReviewRequired {
		t.Fatal("checker made an unsupported semantic claim")
	}
	if err := os.WriteFile(filepath.Join(root, "candidate.go"), []byte("func NewHandler() { sql.Open(\"db\") }"), 0644); err != nil {
		t.Fatal(err)
	}
	report, err = Check(store, db, "A.2")
	if err != nil {
		t.Fatal(err)
	}
	if report.StructuralChecksPassed {
		t.Fatal("B-like implementation drift not detected")
	}
	if err := os.WriteFile(filepath.Join(root, "reference.go"), []byte("changed reference"), 0644); err != nil {
		t.Fatal(err)
	}
	report, err = Check(store, db, "A.1")
	if err != nil {
		t.Fatal(err)
	}
	if report.Results[0].Passed {
		t.Fatal("reference drift was hidden")
	}
	tree, err := Export(store, db)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(tree)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "flowchart TD") {
		t.Fatal("no visible memory tree")
	}
	module, err := os.ReadFile(filepath.Join(store.KnodexaDir(), "exports", "tasks", "A.2", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(module), "Use injected services") || strings.Contains(string(module), "B-specific rule") {
		t.Fatal("exported module has wrong branch memory")
	}
	if _, err := os.Stat(filepath.Join(store.KnodexaDir(), "exports", "tasks", "A.2", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func TestApproachRejectsEscapingReferencesAndExportAliases(t *testing.T) {
	root := t.TempDir()
	store := storage.NewStorage(root)
	if _, err := store.Init("Paths", ""); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := index.NewIndexer(db, store)
	if err := store.SaveTask(&model.TaskData{ID: "A"}); err != nil {
		t.Fatal(err)
	}
	value := model.TaskApproach{Summary: "Pattern", Rules: []string{"Rule"}, References: []model.ApproachReference{{Path: "../outside"}}}
	if err := Set(store, db, idx, "A", value); err == nil {
		t.Fatal("escaping reference accepted")
	}
	exports := filepath.Join(store.KnodexaDir(), "exports")
	if err := os.Symlink(root, exports); err != nil {
		t.Fatal(err)
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	if _, err := Export(store, db); err == nil {
		t.Fatal("export followed an alias into user files")
	}
}

func TestReuseDoesNotDependOnTaskNamesOrTreePosition(t *testing.T) {
	store := storage.NewStorage(t.TempDir())
	if _, err := store.Init("Generic", ""); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := index.NewIndexer(db, store)
	for _, id := range []string{"payments-contract", "webhook-delivery", "search-index"} {
		files := []string{}
		if id == "webhook-delivery" {
			files = []string{"webhook.go"}
		}
		if err := store.SaveTask(&model.TaskData{ID: id, TouchesFiles: files}); err != nil {
			t.Fatal(err)
		}
	}
	if err := Set(store, db, idx, "payments-contract", model.TaskApproach{Summary: "Use idempotent service adapters", Rules: []string{"Inject service dependencies"}, Checks: []model.ApproachCheck{{Path: "@touched", MustContain: "service.Process"}}}); err != nil {
		t.Fatal(err)
	}
	if err := Set(store, db, idx, "search-index", model.TaskApproach{Summary: "Search batching", Rules: []string{"Search-only batching rule"}}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.RootDir(), "webhook.go"), []byte("service.Process(payload)"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Reuse(store, db, idx, "webhook-delivery", []string{"payments-contract"}); err != nil {
		t.Fatal(err)
	}
	contracts, err := Resolve(db, "webhook-delivery")
	if err != nil {
		t.Fatal(err)
	}
	if len(contracts) != 1 || contracts[0].SourceTask != "payments-contract" {
		t.Fatal("explicit reuse across unrelated IDs failed")
	}
	report, err := Check(store, db, "webhook-delivery")
	if err != nil {
		t.Fatal(err)
	}
	if !report.StructuralChecksPassed {
		t.Fatal("generic target binding failed")
	}
	if err := os.WriteFile(filepath.Join(store.RootDir(), "webhook.go"), []byte("direct_database_write(payload)"), 0644); err != nil {
		t.Fatal(err)
	}
	report, err = Check(store, db, "webhook-delivery")
	if err != nil {
		t.Fatal(err)
	}
	if report.StructuralChecksPassed {
		t.Fatal("checker inspected source files instead of new work targets")
	}
	if err := Reuse(store, db, idx, "payments-contract", []string{"webhook-delivery"}); err == nil {
		t.Fatal("reuse cycle accepted")
	}
	original, err := db.GetTaskData("payments-contract")
	if err != nil {
		t.Fatal(err)
	}
	if len(original.ApproachRefs) != 0 {
		t.Fatal("failed link mutated canonical graph")
	}
	if err := Reuse(store, db, idx, "webhook-delivery", []string{"unknown"}); err == nil {
		t.Fatal("unknown source accepted")
	}
}
