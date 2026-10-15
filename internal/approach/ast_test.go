package approach

import (
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"path/filepath"
	"testing"
)

func TestASTChecksIgnoreCommentsAndNormalizeImportAliases(t *testing.T) {
	source := []byte("package demo\nimport db \"database/sql\"\n// Original()\nfunc Wrapper(){ db.Open(\"x\",\"y\") }\n")
	for _, test := range []struct {
		kind, symbol string
		want         bool
	}{
		{"require-call", "Original", false}, {"require-call", "database/sql.Open", true},
		{"forbid-call", "database/sql.Open", false}, {"forbid-import", "database/sql", false},
		{"require-function", "Wrapper", true},
	} {
		got, _ := checkAST("demo.go", source, model.ApproachCheck{ASTKind: test.kind, Symbol: test.symbol})
		if got != test.want {
			t.Fatalf("%s %s: got %t", test.kind, test.symbol, got)
		}
	}
	if got, _ := checkAST("demo.go", []byte("broken"), model.ApproachCheck{ASTKind: "require-call", Symbol: "Original"}); got {
		t.Fatal("invalid Go accepted")
	}
}

func TestApproachReplacementRetainsVersionedAgreement(t *testing.T) {
	store := storage.NewStorage(t.TempDir())
	if _, err := store.Init("versions", ""); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := index.NewIndexer(db, store)
	if err := store.SaveTask(&model.TaskData{ID: "work"}); err != nil {
		t.Fatal(err)
	}
	original := model.TaskApproach{Summary: "original", Rules: []string{"Preserve injection"}}
	if err := Set(store, db, idx, "work", original); err != nil {
		t.Fatal(err)
	}
	replacement := model.TaskApproach{Summary: "replacement", Rules: []string{"Use updated interface"}, ChangeReason: "Interface intentionally migrated"}
	unexplained := replacement
	unexplained.ChangeReason = "  "
	if err := Set(store, db, idx, "work", unexplained); err == nil {
		t.Fatal("agreement replacement accepted without an actual reason")
	}
	if err := Set(store, db, idx, "work", replacement); err != nil {
		t.Fatal(err)
	}
	task, err := db.GetTaskData("work")
	if err != nil {
		t.Fatal(err)
	}
	if task.Approach.Version != 2 || len(task.ApproachHistory) != 1 || task.ApproachHistory[0].Version != 1 || task.ApproachHistory[0].Rules[0] != "Preserve injection" || task.Approach.ChangeReason != replacement.ChangeReason {
		t.Fatalf("history lost: %+v", task)
	}
}

func TestOpposingInheritedChecksAreReported(t *testing.T) {
	contracts := []Contract{{SourceTask: "original", Approach: model.TaskApproach{Checks: []model.ApproachCheck{{Path: "@touched", ASTKind: "require-call", Symbol: "Original"}}}}, {SourceTask: "other", Approach: model.TaskApproach{Checks: []model.ApproachCheck{{Path: "@touched", ASTKind: "forbid-call", Symbol: "Original"}}}}}
	results := structuralConflicts(contracts)
	if len(results) != 1 || results[0].Passed || results[0].Kind != "agreement-conflict" {
		t.Fatalf("conflict ignored: %+v", results)
	}
}
