package approach

import (
	"knodexa/internal/model"
	"strings"
	"testing"
)

func TestReconciliationRejectsStaleOrUnrelatedBindings(t *testing.T) {
	old := strings.Repeat("a", 64)
	current := strings.Repeat("b", 64)
	contracts := []Contract{{SourceTask: "parent", Approach: model.TaskApproach{Version: 2, Rules: []string{"Keep raw input"}, References: []model.ApproachReference{{Path: "old.go", SHA256: old}}, Checks: []model.ApproachCheck{{Path: "old.go", MustNotContain: "normalize"}}}}}
	binding := model.ReferenceReconciliation{SourceTask: "parent", SourceVersion: 2, OriginalPath: "old.go", OriginalSHA256: old, CurrentPath: "new.go", CurrentSHA256: current, Reason: "Reviewed rename"}
	for _, mutate := range []func(*model.ReferenceReconciliation){func(v *model.ReferenceReconciliation) { v.SourceVersion = 1 }, func(v *model.ReferenceReconciliation) { v.OriginalSHA256 = current }, func(v *model.ReferenceReconciliation) { v.SourceTask = "unrelated" }, func(v *model.ReferenceReconciliation) { v.SourceTask = "child" }, func(v *model.ReferenceReconciliation) { v.CurrentPath = "../outside.go" }, func(v *model.ReferenceReconciliation) { v.Reason = "" }} {
		bad := binding
		mutate(&bad)
		if _, err := Reconcile(contracts, "child", []model.ReferenceReconciliation{bad}); err == nil {
			t.Fatalf("invalid reconciliation accepted: %+v", bad)
		}
	}
	resolved, err := Reconcile(contracts, "child", []model.ReferenceReconciliation{binding})
	if err != nil {
		t.Fatal(err)
	}
	if resolved[0].Approach.Checks[0].Path != "new.go" || resolved[0].Approach.Checks[0].MustNotContain != "normalize" || resolved[0].Approach.Rules[0] != "Keep raw input" {
		t.Fatal("reconciliation weakened obligations")
	}
	if contracts[0].Approach.References[0].Path != "old.go" || contracts[0].Approach.Checks[0].Path != "old.go" {
		t.Fatal("reconciliation mutated source contract")
	}
}
