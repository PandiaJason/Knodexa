package experiment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Fixed current-release comparison. These are two scenarios, not enough samples
// for statistical superiority. Normal checks do not consume model usage.
func TestLiveReleaseComparison(t *testing.T) {
	if os.Getenv("KNODEXA_LIVE_RELEASE") != "1" {
		t.Skip("opt-in authenticated two-pair evaluation")
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(repo, "audit", "release-evaluation-v1")
	if override := os.Getenv("KNODEXA_RELEASE_OUTPUT"); override != "" {
		if filepath.IsAbs(override) {
			base = override
		} else {
			base = filepath.Join(repo, override)
		}
	}
	if _, err := os.Stat(base); err == nil {
		t.Fatal("preserve existing release evidence before another run")
	}
	if err := os.MkdirAll(base, 0755); err != nil {
		t.Fatal(err)
	}
	plan := map[string]interface{}{"pairs": 2, "cases": []string{"same-conversation-return", "fresh-context-return"}, "orders": []string{"native first", "knodexa first"}, "model": "gpt-6.1-sol", "reasoning_effort": "low", "baseline": "Codex with scoped native memory files and project instructions", "knodexa": "current Go binary with verified recording and compact restoration", "primary": "independent return behavior, reference preservation and completed source memory preservation", "phases": "establish correct handler, unrelated image normalization, return to handler work", "stop_rule": "all declared cases, no selective reruns; incomplete responses reported separately; absent artifacts unscored", "limits": "one pair per scenario, no forced compaction, not an all-tool comparison"}
	write := func(name string, value interface{}) {
		t.Helper()
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, name), append(data, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("plan.json", plan)
	summaries := []map[string]interface{}{}
	for i, name := range []string{"same-conversation-return", "fresh-context-return"} {
		out := filepath.Join(base, name)
		t.Run(name, func(t *testing.T) {
			t.Setenv("KNODEXA_LIVE_COMPARISON", "1")
			t.Setenv("KNODEXA_COMPARISON_OUTPUT", out)
			t.Setenv("KNODEXA_OPTIMIZED_FLOW", "1")
			t.Setenv("KNODEXA_FRESH_RETURN", "0")
			t.Setenv("KNODEXA_FIRST_ARM", "native")
			if i == 1 {
				t.Setenv("KNODEXA_FRESH_RETURN", "1")
				t.Setenv("KNODEXA_FIRST_ARM", "knodexa")
			}
			TestLiveIdentityComparison(t)
		})
		raw, err := os.ReadFile(filepath.Join(out, "results.json"))
		summary := map[string]interface{}{"case": name}
		if err != nil {
			summary["paired_quality_outcome"] = "inconclusive"
			summary["error"] = err.Error()
		} else {
			if err := json.Unmarshal(raw, &summary); err != nil {
				t.Fatal(err)
			}
			summary["case"] = name
		}
		summaries = append(summaries, summary)
		write("results.json", map[string]interface{}{"plan": plan, "pairs": summaries})
	}
}
