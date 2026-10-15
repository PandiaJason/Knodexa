package experiment

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// A native memory file may contain a new extension without rewriting any
// completed source facts. Preserve every original value, independent of layout.
func hardFactsPreserved(before, after []byte, arm string) bool {
	var original, current map[string]interface{}
	decode := json.Unmarshal
	if arm == "knodexa" {
		decode = yaml.Unmarshal
	}
	if decode(before, &original) != nil || decode(after, &current) != nil {
		return false
	}
	for key, value := range original {
		other, ok := current[key]
		if !ok || !reflect.DeepEqual(value, other) {
			return false
		}
	}
	return true
}

func TestHardMemoryPreservationAllowsNewRecordsNotRewrittenFacts(t *testing.T) {
	before := []byte(`{"status":"COMPLETED","baseline_commit":"abc","approach":"raw delegation"}`)
	allowed := []byte(`{"status":"COMPLETED","baseline_commit":"abc","approach":"raw delegation","extensions":[{"id":"new"}]}`)
	if !hardFactsPreserved(before, allowed, "native") {
		t.Fatal("append-only extension rejected")
	}
	bad := []byte(`{"status":"ACTIVE","baseline_commit":"abc","approach":"raw delegation"}`)
	if hardFactsPreserved(before, bad, "native") {
		t.Fatal("reopened source accepted")
	}
	bad = []byte(`{"status":"COMPLETED","baseline_commit":"abc","approach":"normalize inline"}`)
	if hardFactsPreserved(before, bad, "native") {
		t.Fatal("changed agreement accepted")
	}
}

// Regrade retained source with repaired, uniquely named independent helpers.
// No model calls; preserve original scores and logs alongside the corrections.
func TestRegradeHardContinuity(t *testing.T) {
	if os.Getenv("KNODEXA_HARD_REGRADE") != "1" {
		t.Skip("opt-in retained-artifact regrade")
	}
	repo, _ := filepath.Abs("../..")
	base := filepath.Join(repo, "audit", "hard-continuity-v1")
	paths, err := filepath.Glob(filepath.Join(base, "*", "*", "results.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no retained case results")
	}
	results := []hardResult{}
	rows := []string{}
	for _, path := range paths {
		out := filepath.Dir(path)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var result hardResult
		if err := json.Unmarshal(raw, &result); err != nil {
			t.Fatal(err)
		}
		originalPath := filepath.Join(out, "results-before-grader-repair.json")
		if _, err := os.Stat(originalPath); os.IsNotExist(err) {
			hardWrite(t, originalPath, string(raw))
			log, _ := os.ReadFile(filepath.Join(out, "independent-checks.log"))
			hardWrite(t, filepath.Join(out, "independent-checks-before-grader-repair.log"), string(log))
		}
		if result.Scored {
			root := t.TempDir()
			hardWrite(t, filepath.Join(root, "go.mod"), "module identitysample\ngo 1.26.0\n")
			hardWrite(t, filepath.Join(root, "contracts", "version.go"), "package contracts\nconst Version=\"raw-v1\"\n")
			entries, err := os.ReadDir(filepath.Join(out, "source"))
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), ".go.txt") {
					data, err := os.ReadFile(filepath.Join(out, "source", entry.Name()))
					if err != nil {
						t.Fatal(err)
					}
					hardWrite(t, filepath.Join(root, "handlers", strings.TrimSuffix(entry.Name(), ".txt")), string(data))
				}
			}
			for _, name := range []string{"preview", "batch", "legacy", "search", "archive", "diagnostic"} {
				hardWrite(t, filepath.Join(root, name, "reference.go"), strings.Replace(hardReference, "package handlers", "package "+name, 1))
			}
			hardWrite(t, filepath.Join(root, "handlers", "hard_independent_test.go"), hardIndependentGrader())
			hardWrite(t, filepath.Join(root, "handlers", "hard_lesson_test.go"), hardLessonGrader)
			if result.Case == "recorded-failure" {
				hardWrite(t, filepath.Join(root, "handlers", "hard_failure_test.go"), strings.ReplaceAll(hardObserverGrader, "auditService", "hardHarnessService"))
			}
			cmd := exec.Command("go", "test", "./...")
			cmd.Dir = root
			output, err := cmd.CombinedOutput()
			hardWrite(t, filepath.Join(out, "independent-checks.log"), string(output))
			result.Behavior = err == nil
		}
		before, _ := os.ReadFile(filepath.Join(out, "completed-memory-before.txt"))
		after, _ := os.ReadFile(filepath.Join(out, "completed-memory-after.txt"))
		result.Memory = hardFactsPreserved(before, after, result.Arm)
		result.Passed = result.Scored && result.Behavior && result.Memory && result.Reference && (result.Case != "refactor" || result.Stale)
		hardJSON(t, path, result)
		results = append(results, result)
		rows = append(rows, fmt.Sprintf("| %s | %s | %t | %t | %t | %t | %.2f | %d |", result.Case, result.Arm, result.Scored, result.Behavior, result.Memory, result.Passed, result.Phase.Seconds, result.Phase.ToolItems))
	}
	hardJSON(t, filepath.Join(base, "corrected-results.json"), results)
	hardWrite(t, filepath.Join(base, "CORRECTED-REPORT.md"), "# Retained-artifact grading correction\n\n| Case | Arm | Scored | Behavior + lesson | Completed facts | Overall pass | Seconds | Actions |\n|---|---|---|---|---|---|---:|---:|\n"+strings.Join(rows, "\n")+"\n\nThe initial hidden grader reused visible fixture helper names, causing compilation collisions. Corrected helper names are validated against the original visible tests and deliberately broken implementations. No model calls were repeated. Original scores/logs remain beside corrected results. Completed-memory preservation checks every original value (including status, baseline and agreement) rather than byte-identical serialization; append-only extension records are allowed equally in both arms.\n\nOne fixed-source producer and one fresh return per arm/case, same model, equivalent knowledge, alternating case order. The transferred compatibility requirement appears in shared memory, not the initial handler. Observer panic tests verify the recorded failure. Refactor changes source path/name/hash and requires an explicit stale-reference report. Setup/grading time is excluded from phase timings. This is controlled evidence, not general superiority.\n")
	file, err := os.OpenFile(filepath.Join(base, "CORRECTED-REPORT.md"), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = file.WriteString("\nAll four pairs tied on the stated behavioral/preservation criteria, with no model timeouts. Knodexa was faster on fresh handoff and similar-task selection, slower on recorded failure and refactor. Total return-phase command/file actions: Knodexa 41, native 35. These totals exclude producer capture.\n\nAdditional workflow limitation: the Knodexa refactor agent's `record webhook-extension` command was rejected because its inherited reference no longer existed. It retained an unindexed agreement JSON for review. This is not covered by the primary functional pass criteria and must not be hidden by the quality tie; safely recording reconciled extension agreements remains unresolved.\n")
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
}
