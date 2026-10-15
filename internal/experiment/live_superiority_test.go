package experiment

import (
	"encoding/json"
	"fmt"
	"knodexa/internal/storage"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// Exact one-sided paired sign test. Ties carry no directional evidence.
func superioritySignP(wins, losses int) float64 {
	n := wins + losses
	if n == 0 {
		return 1
	}
	p := 0.0
	for k := wins; k <= n; k++ {
		choose := 1.0
		for j := 1; j <= k; j++ {
			choose *= float64(n-j+1) / float64(j)
		}
		p += choose * math.Pow(0.5, float64(n))
	}
	return p
}

func TestSuperioritySignTest(t *testing.T) {
	for _, tc := range []struct {
		wins, losses int
		want         float64
	}{
		{0, 0, 1}, {1, 0, .5}, {5, 0, .03125}, {7, 1, .03515625}, {4, 4, .63671875},
	} {
		if got := superioritySignP(tc.wins, tc.losses); math.Abs(got-tc.want) > 1e-12 {
			t.Fatalf("(%d,%d): %g, want %g", tc.wins, tc.losses, got, tc.want)
		}
	}
}

func TestSimilarTaskFixtureUsesInitialAgreement(t *testing.T) {
	root := t.TempDir()
	hardCommand(t, root, "git", "init", "-b", "main")
	hardCommand(t, root, "git", "config", "user.name", "Fixture test")
	hardCommand(t, root, "git", "config", "user.email", "fixture@example.test")
	store := storage.NewStorage(root)
	if _, err := store.Init("Fixture", "Similar-task setup regression"); err != nil {
		t.Fatal(err)
	}
	hardSimilarTasks(t, hardFixture{root: root, arm: "knodexa"})
	for _, id := range []string{"preview", "batch", "legacy", "search", "archive", "diagnostic"} {
		task, err := store.ReadTask("work-" + id)
		if err != nil || task.Approach == nil || len(task.CodeAnchors) == 0 {
			t.Fatalf("missing agreement/source identity for %s: %v", id, err)
		}
	}
}

// Fixed sample and independent producers per pair; never retry scored failures.
func TestLiveSuperiorityComparison(t *testing.T) {
	if os.Getenv("KNODEXA_LIVE_SUPERIORITY") != "1" {
		t.Skip("opt-in eight-pair authenticated comparison")
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(repo, "audit", "superiority-evaluation-v1")
	if override := os.Getenv("KNODEXA_SUPERIORITY_OUTPUT"); override != "" {
		out = override
		if !filepath.IsAbs(out) {
			out = filepath.Join(repo, out)
		}
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("output must not exist; preserve all prior evidence")
	}
	cases := []string{"fresh-handoff", "similar-tasks", "recorded-failure", "refactor", "fresh-handoff", "similar-tasks", "recorded-failure", "refactor"}
	plan := map[string]interface{}{
		"pairs": 8, "cases": cases, "source_commit": hardCommand(t, repo, "git", "rev-parse", "HEAD"),
		"model": "gpt-6.1-sol", "reasoning_effort": "low", "background_memories": false,
		"baseline":       "Codex CLI native tools with organized memory, explicit retrieval guidance, identical canonical facts",
		"producer":       "separate real agent capture in each arm of every pair; independently verified fixed source, not agent-authored initial implementation",
		"return":         "fresh independent agent session without producer transcript; equivalent task request and grading",
		"order":          "native first on odd pairs; Knodexa first on even pairs, producers and returns",
		"primary":        "behavior including transferred lesson, completed-memory facts, reference preservation; refactor also reports stale agreement",
		"analysis":       "one-sided exact paired sign test on discordant quality outcomes, alpha 0.05; ties excluded, unscored pairs never counted as losses",
		"advantage_gate": "all eight pairs scored, Knodexa wins exceed losses, one-sided p < 0.05; scope limited to this fixed suite",
		"stop_rule":      "attempt all eight once; retain failures and timeouts; no outcome-based retries or optional stopping",
		"limitations":    "one Go fixture, four related scenarios, one model; no forced compaction or all-agent comparison; return timing excludes setup and producer capture",
	}
	hardJSON(t, filepath.Join(out, "plan.json"), plan)
	results := []map[string]interface{}{}
	wins, losses, ties, unscored := 0, 0, 0, 0
	for i, name := range cases {
		pairOut := filepath.Join(out, fmt.Sprintf("pair-%02d-%s", i+1, name))
		pair := map[string]hardResult{}
		order := []string{"native", "knodexa"}
		if i%2 == 1 {
			order = []string{"knodexa", "native"}
		}
		for _, arm := range order {
			t.Run(fmt.Sprintf("pair-%02d-%s-%s", i+1, name, arm), func(t *testing.T) {
				base := setupHardProducer(t, repo, pairOut, arm, t.TempDir())
				pair[arm] = runHardCase(t, base, filepath.Join(pairOut, arm, "return"), name)
			})
		}
		n, nok := pair["native"]
		k, kok := pair["knodexa"]
		outcome := "unscored"
		if !nok || !kok || !n.Scored || !k.Scored {
			unscored++
		} else if n.Passed == k.Passed {
			ties++
			outcome = "tie"
		} else if k.Passed {
			wins++
			outcome = "knodexa_win"
		} else {
			losses++
			outcome = "native_win"
		}
		results = append(results, map[string]interface{}{"pair": i + 1, "case": name, "outcome": outcome, "arms": pair})
		p := superioritySignP(wins, losses)
		hardJSON(t, filepath.Join(out, "results.json"), map[string]interface{}{
			"plan": plan, "pairs": results, "knodexa_wins": wins, "native_wins": losses, "ties": ties,
			"unscored": unscored, "one_sided_sign_p": p,
			"suite_quality_advantage_demonstrated": len(results) == 8 && unscored == 0 && wins > losses && p < .05,
		})
		t.Logf("pair %d/%d %s: %s; K wins=%d native wins=%d ties=%d unscored=%d", i+1, len(cases), name, outcome, wins, losses, ties, unscored)
	}
	// Re-read the persisted aggregate so a malformed evidence artifact fails here.
	data, err := os.ReadFile(filepath.Join(out, "results.json"))
	if err != nil || !json.Valid(data) {
		t.Fatal("invalid persisted aggregate", err)
	}
}
