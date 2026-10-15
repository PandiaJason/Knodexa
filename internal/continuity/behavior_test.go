package continuity

import (
	"bytes"
	"gopkg.in/yaml.v3"
	"knodexa/internal/approach"
	ctxpkg "knodexa/internal/context"
	"knodexa/internal/git"
	"knodexa/internal/identity"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const qualityReference = `package sample
type Transformer interface { Transform(any)(any,error) }
type Observer interface { Started(any); Finished(any) }
func Reference(v any,s Transformer,o Observer)(any,error){
 if o!=nil{o.Started(v)};r,e:=s.Transform(v);if e==nil&&o!=nil{o.Finished(r)};return r,e
}

`
const qualityCorrect = `package sample
func Extension(v any,s Transformer,o Observer)(any,error){return Reference(v,s,o)}
`
const qualityAcceptance = `package sample
import("testing";"reflect";"errors")
type acceptanceService struct{received any;calls int;events *[]string; result any;err error}
func(s *acceptanceService)Transform(v any)(any,error){s.calls++;s.received=v;*s.events=append(*s.events,"service");return s.result,s.err}
type acceptanceObserver struct{events *[]string}
func(o acceptanceObserver)Started(any){*o.events=append(*o.events,"started")}
func(o acceptanceObserver)Finished(any){*o.events=append(*o.events,"finished")}
func TestAcceptance(t *testing.T){for _,fail:=range []bool{false,true}{
 events:=[]string{};result:=&struct{N int}{7};sentinel:=errors.New("sentinel");var expected error;if fail{expected=sentinel}
 s:=&acceptanceService{events:&events,result:result,err:expected};v:="  MiXeD  "
 got,err:=Extension(v,s,acceptanceObserver{&events});want:=[]string{"started","service"};if !fail{want=append(want,"finished")}
 if s.received!=v||s.calls!=1||got!=result||err!=expected||!reflect.DeepEqual(events,want){t.Fatalf("behavior changed: raw=%v calls=%d result=%v error=%v events=%v",s.received,s.calls,got,err,events)}
}}
func TestAgentSmoke(t *testing.T){}
`

type qualityFixture struct {
	root    string
	store   *storage.Storage
	service *Service
	input   RecordInput
	parent  []byte
}

func newQualityFixture(t *testing.T) qualityFixture {
	t.Helper()
	root := t.TempDir()
	store := storage.NewStorage(root)
	if _, err := store.Init("Quality fixture", ""); err != nil {
		t.Fatal(err)
	}
	db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	idx := index.NewIndexer(db, store)
	gi := git.NewInspector(root)
	service := NewService(store, db, idx, ctxpkg.NewBuilder(db, store, gi), gi)
	for path, body := range map[string]string{"go.mod": "module sample\ngo 1.26.0\n", "reference.go": qualityReference, "extension.go": qualityCorrect, "acceptance_test.go": qualityAcceptance} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SaveTask(&model.TaskData{ID: "parent", Status: model.TaskStatusCompleted, BaselineCommit: "original-baseline", Constraints: []string{"Preserve raw delegation"}}); err != nil {
		t.Fatal(err)
	}
	value := model.TaskApproach{Summary: "Raw delegation", Rules: []string{"Keep raw identity and observer order"}, References: []model.ApproachReference{{Path: "reference.go"}}, BehaviorChecks: []model.BehaviorCheck{{Name: "raw delegation acceptance", Command: "go test -count=1 -run '^TestAcceptance$' ./...", ProtectedFiles: []model.ApproachReference{{Path: "acceptance_test.go"}}}}}
	if err := approach.Set(store, db, idx, "parent", value); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTask(&model.TaskData{ID: "child", ParentID: "parent", Aliases: []string{"delivery extension"}, TouchesFiles: []string{"extension.go"}}); err != nil {
		t.Fatal(err)
	}
	if err := idx.Sync(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(store.KnodexaDir(), "tasks", "parent", "parent.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return qualityFixture{root, store, service, RecordInput{Files: []string{"extension.go"}, Approach: model.TaskApproach{Summary: "Extension", Rules: []string{"Preserve parent contract"}, References: []model.ApproachReference{{Path: "reference.go"}}}}, before}
}

func (f qualityFixture) unchanged(t *testing.T) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(f.store.KnodexaDir(), "tasks", "parent", "parent.yaml"))
	if err != nil || !bytes.Equal(data, f.parent) {
		t.Fatal("completed parent changed", err)
	}
}

func TestBehaviorContractsRejectCompilableRegressions(t *testing.T) {
	mutants := map[string]string{
		"normalization":    `package sample;import "strings";func Extension(v any,s Transformer,o Observer)(any,error){if x,ok:=v.(string);ok{v=strings.TrimSpace(x)};return Reference(v,s,o)}`,
		"error-swallowing": `package sample;func Extension(v any,s Transformer,o Observer)(any,error){r,_:=Reference(v,s,o);return r,nil}`,
		"retry":            `package sample;func Extension(v any,s Transformer,o Observer)(any,error){Reference(v,s,o);return Reference(v,s,o)}`,
		"observer-order":   `package sample;func Extension(v any,s Transformer,o Observer)(any,error){r,e:=s.Transform(v);if o!=nil{o.Started(v);o.Finished(r)};return r,e}`,
	}
	for name, body := range mutants {
		t.Run(name, func(t *testing.T) {
			f := newQualityFixture(t)
			os.WriteFile(filepath.Join(f.root, "extension.go"), []byte(body), 0644)
			result, err := f.service.Record("child", f.input, []string{"go test -run '^TestAgentSmoke$' ./..."}, 0)
			if err == nil || result.Recorded || len(result.BehaviorChecks) != 1 || result.BehaviorChecks[0].ExitCode == 0 {
				t.Fatalf("regression accepted: %v %+v", err, result)
			}
			if len(result.Verification) != 1 || result.Verification[0].ExitCode != 0 {
				t.Fatal("mutant must compile and pass the weak caller test")
			}
			saved, _ := f.store.ReadTask("child")
			if saved.Approach != nil {
				t.Fatal("failed quality installed partial memory")
			}
			f.unchanged(t)
		})
	}
}

func TestBehaviorContractsCaptureCorrectWorkAndStayInSmallContext(t *testing.T) {
	f := newQualityFixture(t)
	result, err := f.service.Record("child", f.input, []string{"go test -run '^TestAgentSmoke$' ./..."}, 0)
	if err != nil || !result.Recorded || len(result.BehaviorChecks) != 1 || result.BehaviorChecks[0].ExitCode != 0 {
		t.Fatalf("correct work rejected: %v %+v", err, result)
	}
	f.unchanged(t)
	prepared, err := f.service.Prepare(identity.Request{SelectTask: "child", Text: "delivery extension"}, 1, ResumeOptions{})
	if err != nil || !strings.Contains(prepared.FormatHuman(), "raw delegation acceptance") || !strings.Contains(prepared.FormatHuman(), "acceptance_test.go") {
		t.Fatal("small context dropped mandatory behavior checks", err)
	}
}

func TestBehaviorContractsRejectWeakenedTestsBeforeVerification(t *testing.T) {
	f := newQualityFixture(t)
	os.WriteFile(filepath.Join(f.root, "acceptance_test.go"), []byte("package sample\n"), 0644)
	result, err := f.service.Record("child", f.input, []string{"touch verification-ran"}, 0)
	if err == nil || result.Recorded || !strings.Contains(err.Error(), "protected behavior input") {
		t.Fatal("changed acceptance test accepted", err)
	}
	if _, err := os.Stat(filepath.Join(f.root, "verification-ran")); !os.IsNotExist(err) {
		t.Fatal("caller command ran after protected test tampering")
	}
	prepared, err := f.service.Prepare(identity.Request{SelectTask: "child", Text: "delivery extension"}, 100, ResumeOptions{})
	if err != nil || !prepared.NeedsReview {
		t.Fatal("prepare hid test drift", err)
	}
	f.unchanged(t)
}

func TestRunEnforcesRestoredBehaviorAndPinsAfterAgentExecution(t *testing.T) {
	for _, tamper := range []bool{false, true} {
		t.Run(map[bool]string{false: "bad-code", true: "changed-tests"}[tamper], func(t *testing.T) {
			f := newQualityFixture(t)
			command := "cat > extension.go <<'EOF'\npackage sample\nfunc Extension(v any,s Transformer,o Observer)(any,error){return nil,nil}\nEOF\n"
			if tamper {
				command = "printf '\\n// changed by invoked agent\\n' >> acceptance_test.go"
			}
			result, err := f.service.Run(identity.Request{SelectTask: "child", Text: "delivery extension"}, 3000, RunOptions{Command: []string{"/bin/sh", "-c", command}, Verify: []string{"true"}})
			if err == nil || result.Captured {
				t.Fatalf("runner accepted regression/tampering: %v %+v", err, result)
			}
			if !tamper && (len(result.BehaviorChecks) != 1 || result.BehaviorChecks[0].ExitCode == 0) {
				t.Fatal("recorded behavior command not executed")
			}
			f.unchanged(t)
		})
	}
}

func TestRecordCannotDropAnExistingBehaviorCheck(t *testing.T) {
	f := newQualityFixture(t)
	os.WriteFile(filepath.Join(f.root, "extension.go"), []byte("package sample\nfunc Extension(v any,s Transformer,o Observer)(any,error){return nil,nil}\n"), 0644)
	input := f.input
	input.Approach.ChangeReason = "Review changed agreement, not permission to skip acceptance"
	result, err := f.service.Record("parent", input, []string{"true"}, 0)
	if err == nil || result.Recorded || len(result.BehaviorChecks) != 1 {
		t.Fatalf("omitting checks bypassed old behavior: %v %+v", err, result)
	}
	f.unchanged(t)
}

func TestSuccessfulRecordRetainsOmittedBehaviorChecks(t *testing.T) {
	f := newQualityFixture(t)
	input := f.input
	input.Approach.ChangeReason = "Refresh verified work"
	result, err := f.service.Record("parent", input, []string{"true"}, 0)
	if err != nil || !result.Recorded {
		t.Fatalf("correct refresh failed: %v %+v", err, result)
	}
	parent, _ := f.store.ReadTask("parent")
	if len(parent.Approach.BehaviorChecks) != 1 || parent.Approach.BehaviorChecks[0].Name != "raw delegation acceptance" {
		t.Fatal("successful refresh erased future acceptance")
	}
	if parent.Status != model.TaskStatusCompleted || parent.BaselineCommit != "original-baseline" {
		t.Fatal("refresh reopened or rebaselined source")
	}
}

func TestRecordCannotReplaceBehaviorWithWeakerCommand(t *testing.T) {
	f := newQualityFixture(t)
	parent, _ := f.store.ReadTask("parent")
	input := f.input
	input.Approach.ChangeReason = "Refresh work"
	check := parent.Approach.BehaviorChecks[0]
	check.Command = "true"
	input.Approach.BehaviorChecks = []model.BehaviorCheck{check}
	result, err := f.service.Record("parent", input, []string{"touch verification-ran"}, 0)
	if err == nil || result.Recorded || !strings.Contains(err.Error(), "approach set") {
		t.Fatalf("weaker command accepted: %v %+v", err, result)
	}
	if _, err := os.Stat(filepath.Join(f.root, "verification-ran")); !os.IsNotExist(err) {
		t.Fatal("unreviewed replacement ran verification")
	}
	f.unchanged(t)
}

func TestRecordRejectsBehaviorAgreementChangedByVerification(t *testing.T) {
	f := newQualityFixture(t)
	parent, err := f.store.ReadTask("parent")
	if err != nil {
		t.Fatal(err)
	}
	parent.Approach.BehaviorChecks = nil
	data, err := yaml.Marshal(parent)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(f.root, "weakened-parent.yaml"), data, 0644)
	result, err := f.service.Record("child", f.input, []string{"cp weakened-parent.yaml .knodexa/tasks/parent/parent.yaml"}, 0)
	if err == nil || result.Recorded || !strings.Contains(err.Error(), "behavior agreement changed") {
		t.Fatalf("changed obligations accepted: %v %+v", err, result)
	}
	child, _ := f.store.ReadTask("child")
	if child.Approach != nil {
		t.Fatal("captured against weakened acceptance")
	}
	// Invoked commands own their file edits; rejection does not roll them back.
}

func TestExplicitReferenceReviewCapturesRefactoredExtensionAndBehavior(t *testing.T) {
	f := newQualityFixture(t)
	os.Remove(filepath.Join(f.root, "reference.go"))
	os.WriteFile(filepath.Join(f.root, "transport.go"), []byte(strings.Replace(qualityReference, "func Reference(", "func Dispatch(", 1)), 0644)
	os.WriteFile(filepath.Join(f.root, "extension.go"), []byte(strings.Replace(qualityCorrect, "Reference(", "Dispatch(", 1)), 0644)
	input := f.input
	input.Approach.References = []model.ApproachReference{{Path: "transport.go"}}
	input.ReferenceReviews = []ReferenceReview{{SourceTask: "parent", OriginalPath: "reference.go", CurrentPath: "transport.go", Reason: "Reviewed authorized relocation/rename; original behavioral acceptance must still pass"}}
	result, err := f.service.Record("child", input, []string{"go test -run '^TestAgentSmoke$' ./..."}, 0)
	if err != nil || !result.Recorded || len(result.BehaviorChecks) != 1 {
		t.Fatalf("reviewed extension failed: %v %+v", err, result)
	}
	f.unchanged(t)
	saved, _ := f.store.ReadTask("child")
	if len(saved.Approach.ReferenceReconciliations) != 1 || saved.Approach.ReferenceReconciliations[0].SourceVersion != 1 {
		t.Fatal("review did not persist full versioned binding")
	}
	prepared, err := f.service.Prepare(identity.Request{SelectTask: "child", Text: "delivery extension"}, 3000, ResumeOptions{})
	if err != nil || prepared.NeedsReview {
		t.Fatal("saved extension cannot be restored", err)
	}
}

func TestReferenceReviewRejectsUnrelatedOrUnreviewedMappings(t *testing.T) {
	for _, review := range []ReferenceReview{
		{SourceTask: "other", OriginalPath: "reference.go", CurrentPath: "extension.go", Reason: "reviewed"},
		{SourceTask: "parent", OriginalPath: "reference.go", CurrentPath: "extension.go"},
		{SourceTask: "child", OriginalPath: "reference.go", CurrentPath: "extension.go", Reason: "reviewed"},
		{SourceTask: "parent", OriginalPath: "reference.go", CurrentPath: "../outside.go", Reason: "reviewed"},
	} {
		t.Run(review.SourceTask+review.Reason, func(t *testing.T) {
			f := newQualityFixture(t)
			input := f.input
			input.ReferenceReviews = []ReferenceReview{review}
			result, err := f.service.Record("child", input, []string{"touch verification-ran"}, 0)
			if err == nil || result.Recorded {
				t.Fatal("invalid review accepted")
			}
			if _, err := os.Stat(filepath.Join(f.root, "verification-ran")); !os.IsNotExist(err) {
				t.Fatal("invalid review ran caller verification")
			}
			f.unchanged(t)
		})
	}
}
