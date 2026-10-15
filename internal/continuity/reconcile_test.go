package continuity

import (
	"bytes"
	"knodexa/internal/approach"
	"knodexa/internal/codegraph"
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

func TestChildReconcilesRefactoredReferenceWithoutChangingParent(t *testing.T) {
	root := t.TempDir()
	store := storage.NewStorage(root)
	if _, err := store.Init("reconcile", ""); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	idx := index.NewIndexer(db, store)
	gi := git.NewInspector(root)
	service := NewService(store, db, idx, context.NewBuilder(db, store, gi), gi)
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, path), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module demo\ngo 1.26.0\n")
	write("reference.go", "package demo\nfunc Original() string{return \"ok\"}\n")
	if err := store.SaveTask(&model.TaskData{ID: "parent", Status: model.TaskStatusCompleted, BaselineCommit: "historical-checkpoint", Constraints: []string{"Keep original result"}, RequiredVerification: []string{"Identity acceptance"}}); err != nil {
		t.Fatal(err)
	}
	if err := approach.Set(store, db, idx, "parent", model.TaskApproach{Summary: "Delegate", Rules: []string{"Keep original result"}, References: []model.ApproachReference{{Path: "reference.go"}}, Checks: []model.ApproachCheck{{Path: "reference.go", MustNotContain: "strings.ToUpper"}}}); err != nil {
		t.Fatal(err)
	}
	parent, _ := store.ReadTask("parent")
	before, err := os.ReadFile(filepath.Join(store.KnodexaDir(), "tasks", "parent", "parent.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(filepath.Join(root, "reference.go"))
	write("transport.go", "package demo\nfunc Dispatch() string{return \"ok\"}\n")
	write("extension.go", "package demo\nfunc Extension() string{return Dispatch()}\n")
	if err := store.SaveTask(&model.TaskData{ID: "child", ParentID: "parent", Aliases: []string{"extension adapter"}}); err != nil {
		t.Fatal(err)
	}
	anchor, err := codegraph.Read(root, "transport.go")
	if err != nil {
		t.Fatal(err)
	}
	input := RecordInput{Files: []string{"extension.go"}, Approach: model.TaskApproach{Summary: "Extend delegation", Rules: []string{"Keep parent behavior"}, References: []model.ApproachReference{{Path: "transport.go"}}, ReferenceReconciliations: []model.ReferenceReconciliation{{SourceTask: "parent", SourceVersion: 1, OriginalPath: "reference.go", OriginalSHA256: parent.Approach.References[0].SHA256, CurrentPath: "transport.go", CurrentSHA256: anchor.SHA256, Reason: "Reviewed authorized file/function refactor; behavior remains equivalent"}}}}
	bad := input
	bad.Approach = input.Approach
	bad.Approach.ReferenceReconciliations = append([]model.ReferenceReconciliation{}, input.Approach.ReferenceReconciliations...)
	bad.Approach.ReferenceReconciliations[0].CurrentSHA256 = parent.Approach.References[0].SHA256
	result, err := service.Record("child", bad, []string{"go test ./..."}, 0)
	if err == nil || result.Recorded {
		t.Fatal("wrong reviewed hash accepted")
	}
	if saved, _ := store.ReadTask("child"); saved.Approach != nil {
		t.Fatal("failed mapping installed partial memory")
	}
	result, err = service.Record("child", input, []string{"go test ./..."}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Recorded {
		t.Fatal("child capture rejected")
	}
	after, _ := os.ReadFile(filepath.Join(store.KnodexaDir(), "tasks", "parent", "parent.yaml"))
	if !bytes.Equal(before, after) {
		t.Fatal("reconciliation rewrote completed parent")
	}
	report, err := approach.Check(store, db, "child")
	if err != nil || !report.StructuralChecksPassed {
		t.Fatalf("persisted mapping not applied: %v %+v", err, report)
	}
	pkg, err := service.Prepare(identity.Request{Text: "extension adapter"}, 3000, ResumeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if pkg.NeedsReview || len(pkg.Context.RequiredVerification) != 1 || len(pkg.Context.AcceptedConstraints) == 0 {
		t.Fatal("restoration lost reconciled agreement or inherited obligations")
	}
	input.Approach.ChangeReason = "Refresh child after verification"
	if _, err := service.Record("child", input, []string{"go test ./..."}, 0); err != nil {
		t.Fatalf("repeat recording lost historical binding: %v", err)
	}
	write("transport.go", "package demo\nfunc Dispatch() string{return \"changed\"}\n")
	report, err = approach.Check(store, db, "child")
	if err != nil || report.StructuralChecksPassed {
		t.Fatal("later drift was hidden by reconciliation")
	}
}
