package continuity

import (
	"bytes"
	"knodexa/internal/context"
	"knodexa/internal/git"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestResumePreservesDriftAndReturnsAllContinuity(t *testing.T) {
	root := t.TempDir()
	gitRun := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return string(out)
	}
	gitRun("init", "-b", "main")
	gitRun("config", "user.name", "test")
	gitRun("config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun("add", ".")
	gitRun("commit", "-m", "initial")
	store := storage.NewStorage(root)
	if _, err := store.Init("Resume", ""); err != nil {
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
	old := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	if err := store.SaveTask(&model.TaskData{ID: "T", Status: model.TaskStatusBlocked, BaselineCommit: old, RequiredVerification: []string{"custom acceptance"}, TouchesFiles: []string{"auth.go"}, Constraints: []string{"Preserve API"}}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveFailure(&model.FailureData{ID: "F-1", TaskID: "T", Summary: "Avoid integer coercion"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveLesson(&model.LessonDocument{Frontmatter: model.LessonFrontmatter{ID: "L-1", SourceTaskID: "other-agent", Status: model.StatusVerified, RelatedFiles: []string{"auth.go"}, Applicability: "When validating timestamps"}, Body: "Reject boolean inputs"}); err != nil {
		t.Fatal(err)
	}
	pkg, err := service.Resume("T", 1500, ResumeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Task.Status != "ACTIVE" || !pkg.Baseline.Mismatch || pkg.Baseline.StoredCommit != old || pkg.DetailLevel != "compact" {
		t.Fatalf("resume hid drift or omitted state: %+v", pkg)
	}
	if len(pkg.PreviousFailures) != 1 || len(pkg.RelevantLessons) != 1 || len(pkg.RequiredVerification) != 1 {
		t.Fatal("combined result lost continuity")
	}
	task, err := store.ReadTask("T")
	if err != nil {
		t.Fatal(err)
	}
	if task.BaselineCommit != old {
		t.Fatal("baseline silently replaced")
	}
	stamp := task.UpdatedAt
	if _, err := service.Resume("T", 1500, ResumeOptions{}); err != nil {
		t.Fatal(err)
	}
	task, err = store.ReadTask("T")
	if err != nil {
		t.Fatal(err)
	}
	if !task.UpdatedAt.Equal(stamp) {
		t.Fatal("repeat resume rewrote unchanged task")
	}
	acknowledged, err := service.Resume("T", 1500, ResumeOptions{AcceptBaseline: true})
	if err != nil {
		t.Fatal(err)
	}
	if acknowledged.Baseline.Mismatch || acknowledged.Baseline.StoredCommit == old {
		t.Fatal("explicit baseline acceptance failed")
	}
}

func TestResumeCompletedWorkRequiresExplicitReopen(t *testing.T) {
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v %s", err, output)
		}
	}
	run("init", "-b", "main")
	run("config", "user.name", "test")
	run("config", "user.email", "test@example.test")
	os.WriteFile(filepath.Join(root, "README.md"), []byte("baseline"), 0644)
	run("add", ".")
	run("commit", "-m", "baseline")
	store := storage.NewStorage(root)
	if _, err := store.Init("completed", ""); err != nil {
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
	if err := store.SaveTask(&model.TaskData{ID: "finished", Status: model.TaskStatusCompleted, Constraints: []string{"Preserve raw identity"}}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.KnodexaDir(), "tasks", "finished", "finished.yaml")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := service.Resume("finished", 1500, ResumeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Task.Status != "COMPLETED" || !bytes.Equal(before, after) {
		t.Fatal("retrieval reopened or rewrote completed source memory")
	}
	task, _ := store.ReadTask("finished")
	if task.BaselineCommit != "" {
		t.Fatal("completed baseline initialized without explicit request")
	}
	reopened, err := service.Resume("finished", 1500, ResumeOptions{Reopen: true})
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Task.Status != "ACTIVE" {
		t.Fatal("explicit reopening failed")
	}
	task, _ = store.ReadTask("finished")
	if task.BaselineCommit == "" {
		t.Fatal("reopened active task did not capture baseline")
	}
}

func TestReturnAfterUnrelatedWorkRestoresScopeAndBoundary(t *testing.T) {
	root := t.TempDir()
	store := storage.NewStorage(root)
	if _, err := store.Init("return", ""); err != nil {
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
		{ID: "service-design", Constraints: []string{"Injected service owns behavior"}, RequiredVerification: []string{"Service substitution test"}, Approach: &model.TaskApproach{Summary: "service handlers", Rules: []string{"delegate to service"}, Scopes: []string{"handlers/"}}},
		{ID: "image-design", TouchesFiles: []string{"images/label.py"}, Approach: &model.TaskApproach{Summary: "image labels", Rules: []string{"direct string transformations"}, Scopes: []string{"images/"}}},
		{ID: "delivery", TouchesFiles: []string{"handlers/webhook.py"}, RequiredVerification: []string{"Webhook acceptance"}},
	} {
		if err := store.SaveTask(task); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SaveFailure(&model.FailureData{ID: "F-service", TaskID: "service-design", Summary: "Do not inline transforms"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Resume("image-design", 1500, ResumeOptions{}); err != nil {
		t.Fatal(err)
	}
	pkg, err := service.Resume("delivery", 1, ResumeOptions{PreviousTaskID: "image-design"})
	if err != nil {
		t.Fatal(err)
	}
	if pkg.ApproachSelection.Mode != "file_scope" || len(pkg.EstablishedApproach) != 1 || pkg.EstablishedApproach[0].SourceTask != "service-design" {
		t.Fatal("recent unrelated approach displaced original")
	}
	if len(pkg.PreviousFailures) != 1 || pkg.PreviousFailures[0].ID != "F-service" {
		t.Fatal("scope-restored failure missing")
	}
	if len(pkg.ContextBoundary.DiscardSources) != 1 || pkg.ContextBoundary.DiscardSources[0] != "image-design" {
		t.Fatal("missing isolation boundary")
	}
	if !strings.Contains(strings.Join(pkg.AcceptedConstraints, "\n"), "Injected service owns behavior") || len(pkg.RequiredVerification) != 2 || !pkg.BudgetExceeded {
		t.Fatalf("compaction lost source agreement acceptance: %+v", pkg)
	}
	if _, err := service.Resume("delivery", 1500, ResumeOptions{PreviousTaskID: "missing"}); err == nil {
		t.Fatal("unknown previous task ignored")
	}
}
