package experiment

import (
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"os"
	"path/filepath"
	"testing"
)

// A workflow acceptance trial, not a superiority comparison. Independent code
// grading and successful new-memory capture are separate required endpoints.
func TestLiveBehavioralQualityWorkflow(t *testing.T) {
	if os.Getenv("KNODEXA_LIVE_QUALITY") != "1" {
		t.Skip("opt-in authenticated refactor and acceptance workflow")
	}
	repo, _ := filepath.Abs("../..")
	out := filepath.Join(repo, "audit", "behavioral-quality-v1")
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("preserve prior evidence; output must not exist")
	}
	hardJSON(t, filepath.Join(out, "plan.json"), map[string]interface{}{
		"trial": "fresh-agent authorized refactor with executable acceptance and extension capture",
		"model": "gpt-6.1-sol", "reasoning_effort": "low", "baseline_commit": hardCommand(t, repo, "git", "rev-parse", "HEAD"),
		"endpoints": []string{"independent behavior and SDK lesson", "unchanged completed parent facts and current source", "explicit stale reference report", "persisted child agreement with current code identity and valid inherited reference reconciliation"},
		"checks":    "owner-provided independent Go tests, pinned as template inputs before return; command installs them and runs go test ./...",
		"scope":     "one real producer and fresh return agent, no native arm; implementation/workflow acceptance, never a coding quality superiority claim",
		"stop_rule": "one attempt, preserve all errors; no scored retries",
	})
	f := setupHardProducer(t, repo, out, "knodexa", t.TempDir())
	hardWrite(t, filepath.Join(f.root, "quality", "identity.go.txt"), hardIndependentGrader())
	hardWrite(t, filepath.Join(f.root, "quality", "lesson.go.txt"), hardLessonGrader)
	store := storage.NewStorage(f.root)
	parent, err := store.ReadTask("work-prod")
	if err != nil {
		t.Fatal(err)
	}
	value := *parent.Approach
	value.ChangeReason = "Owner attaches explicit protected acceptance templates for future production handler extensions"
	value.BehaviorChecks = []model.BehaviorCheck{{Name: "production webhook acceptance", Command: "cp quality/identity.go.txt handlers/hard_independent_test.go && cp quality/lesson.go.txt handlers/hard_lesson_test.go && go test -count=1 ./...", ProtectedFiles: []model.ApproachReference{{Path: "quality/identity.go.txt"}, {Path: "quality/lesson.go.txt"}}}}
	// Owner-defined obligations are recorded before the candidate exists. Setting
	// an agreement defines expectations; it does not claim the future code passed.
	hardJSON(t, filepath.Join(f.root, "owner-approach.json"), value)
	hardCommand(t, f.root, "./knodexa", "approach", "set", "work-prod", "--file=owner-approach.json")
	hardCommand(t, f.root, "./knodexa", "agents", "install")
	f.protected, err = os.ReadFile(hardMemory(f))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("KNODEXA_GRADE_EXTENSION_CAPTURE", "1")
	result := runHardCase(t, f, filepath.Join(out, "refactor", "knodexa"), "refactor")
	hardJSON(t, filepath.Join(out, "results.json"), map[string]interface{}{"result": result, "all_endpoints_passed": result.Passed && result.ExtensionCaptured, "not_a_superiority_comparison": true})
	if !result.Passed || !result.ExtensionCaptured {
		t.Fatalf("workflow acceptance failed: %+v", result)
	}
}
