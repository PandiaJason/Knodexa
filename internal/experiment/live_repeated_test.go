package experiment

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func identityComparisonOutput(repo string) string {
	if out := os.Getenv("KNODEXA_COMPARISON_OUTPUT"); out != "" {
		return out
	}
	return filepath.Join(repo, "audit", "identity-comparison")
}

// Fixed, predeclared pairs; do not stop or rerun based on scored outcomes.
func TestLiveRepeatedContinuityComparison(t *testing.T) {
	if os.Getenv("KNODEXA_LIVE_REPEATED") != "1" {
		t.Skip("opt-in repeated paid live comparison")
	}
	repo, _ := filepath.Abs("../..")
	base := filepath.Join(repo, "audit", "continuity-repeated-v2")
	if _, err := os.Stat(base); err == nil {
		t.Fatal("output already exists; preserve prior evidence before rerunning")
	}
	if err := os.MkdirAll(base, 0755); err != nil {
		t.Fatal(err)
	}
	plan := map[string]interface{}{"pairs": 3, "orders": []string{"native first", "knodexa first", "native first"}, "primary": "independent behavior, source and completed-memory preservation", "knodexa": "current Go binary, prepare and source capture", "baseline": "Codex CLI with scoped native file memory", "scope": "same-conversation unrelated work then return; no forced compaction", "timeouts": "grade returned artifacts; report completion separately"}
	data, _ := json.MarshalIndent(plan, "", "  ")
	os.WriteFile(filepath.Join(base, "plan.json"), data, 0644)
	summaries := []map[string]interface{}{}
	rows := []string{}
	counts := map[string]int{}
	for i := 1; i <= 3; i++ {
		out := filepath.Join(base, fmt.Sprintf("pair-%d", i))
		t.Run(fmt.Sprintf("pair-%d", i), func(t *testing.T) {
			t.Setenv("KNODEXA_LIVE_COMPARISON", "1")
			t.Setenv("KNODEXA_COMPARISON_OUTPUT", out)
			if i == 2 {
				t.Setenv("KNODEXA_FIRST_ARM", "knodexa")
			}
			TestLiveIdentityComparison(t)
		})
		raw, err := os.ReadFile(filepath.Join(out, "results.json"))
		summary := map[string]interface{}{}
		if err != nil {
			summary["paired_quality_outcome"] = "inconclusive"
			summary["error"] = err.Error()
		} else if err = json.Unmarshal(raw, &summary); err != nil {
			t.Fatal(err)
		}
		summaries = append(summaries, summary)
		outcome, _ := summary["paired_quality_outcome"].(string)
		counts[outcome]++
		rows = append(rows, fmt.Sprintf("| %d | %s |", i, outcome))
		data, _ = json.MarshalIndent(map[string]interface{}{"plan": plan, "counts": counts, "pairs": summaries}, "", "  ")
		os.WriteFile(filepath.Join(base, "results.json"), data, 0644)
	}
	report := "# Repeated real-agent continuity comparison\n\n| Pair | Independent quality outcome |\n|---|---|\n" + strings.Join(rows, "\n") + "\n\nThree fresh pairs with alternating order, identical requirements and independent behavior grading. Raw traces, tool counts, timing, token usage and memory snapshots are retained per pair. The same model runs both arms. This tests the current prepare/source-capture workflow, not automatic command hooks or all Codex desktop tools. No forced context compaction. One controlled scenario and three pairs cannot establish general superiority. Timeouts and setup failures remain visible; artifacts are graded when available.\n"
	os.WriteFile(filepath.Join(base, "REPORT.md"), []byte(report), 0644)
}
