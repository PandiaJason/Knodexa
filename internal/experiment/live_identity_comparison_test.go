package experiment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"knodexa/internal/model"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A paired opt-in comparison. Timeouts remain inconclusive, not quality failures.
func TestLiveIdentityComparison(t *testing.T) {
	if os.Getenv("KNODEXA_LIVE_COMPARISON") != "1" {
		t.Skip("set KNODEXA_LIVE_COMPARISON=1 for the paid paired trial")
	}
	repo, _ := filepath.Abs("../..")
	out := identityComparisonOutput(repo)
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	plan := map[string]interface{}{"arms": []string{"native", "knodexa"}, "order": "native first", "pairs": 1, "primary": "independent return behavior passes and source reference/memory preserved", "model": "gpt-6.1-sol", "reasoning_effort": "low", "background_memories_enabled": false, "return_supplies_task_id_or_path": false}
	plan["optimized_record_and_text"] = os.Getenv("KNODEXA_OPTIMIZED_FLOW") == "1"
	plan["fresh_return_context"] = os.Getenv("KNODEXA_FRESH_RETURN") == "1"
	data, _ := json.MarshalIndent(plan, "", "  ")
	os.WriteFile(filepath.Join(out, "plan.json"), data, 0644)
	runKnodexa := func(t *testing.T) { t.Setenv("KNODEXA_LIVE_IDENTITY", "1"); TestLiveRequestIdentity(t) }
	if os.Getenv("KNODEXA_FIRST_ARM") == "knodexa" {
		plan["order"] = "knodexa first"
		data, _ = json.MarshalIndent(plan, "", "  ")
		os.WriteFile(filepath.Join(out, "plan.json"), data, 0644)
		t.Run("knodexa", runKnodexa)
		t.Run("native", runNativeIdentity)
	} else {
		t.Run("native", runNativeIdentity)
		t.Run("knodexa", runKnodexa)
	}
	summary := map[string]interface{}{"plan": plan, "arms": map[string]interface{}{}}
	arms := summary["arms"].(map[string]interface{})
	for _, arm := range []string{"native", "knodexa"} {
		raw, err := os.ReadFile(filepath.Join(out, arm, "results.json"))
		if err != nil {
			arms[arm] = map[string]interface{}{"status": "inconclusive", "error": err.Error()}
			continue
		}
		var result map[string]interface{}
		if err := json.Unmarshal(raw, &result); err != nil {
			t.Fatal(err)
		}
		arms[arm] = result
	}
	data, _ = json.MarshalIndent(summary, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "results.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	writeIdentityComparisonReport(t, out, summary)
}

func writeIdentityComparisonReport(t *testing.T, out string, summary map[string]interface{}) {
	t.Helper()
	arms := summary["arms"].(map[string]interface{})
	metrics := map[string]map[string]interface{}{}
	rows := []string{}
	for _, arm := range []string{"native", "knodexa"} {
		result := arms[arm].(map[string]interface{})
		_, scored := result["return_independent_checks_passed"]
		quality := result["return_independent_checks_passed"] == true && result["completed_memory_preserved"] == true && result["reference_preserved"] == true
		var seconds, input, cached, items float64
		for _, label := range []string{"establish", "switch", "return"} {
			phase, ok := result[label].(map[string]interface{})
			if !ok {
				continue
			}
			if value, ok := phase["seconds"].(float64); ok {
				seconds += value
			}
			if value, ok := phase["tool_items"].(float64); ok {
				items += value
			}
			if usage, ok := phase["usage"].([]interface{}); ok {
				for _, u := range usage {
					entry, ok := u.(map[string]interface{})
					if !ok {
						continue
					}
					if v, ok := entry["input_tokens"].(float64); ok {
						input += v
					}
					if v, ok := entry["cached_input_tokens"].(float64); ok {
						cached += v
					}
				}
			}
		}
		metrics[arm] = map[string]interface{}{"quality_scored": scored, "common_quality_passed": quality, "response_completed": result["response_completed"], "workflow_seconds": seconds, "tool_items": items, "input_tokens": input, "cached_input_tokens": cached}
		qualityText := "Unscored"
		if scored {
			qualityText = "Failed"
			if quality {
				qualityText = "Passed"
			}
		}
		rows = append(rows, fmt.Sprintf("| %s | %s | %.2f | %.0f | %.0f |", arm, qualityText, seconds, items, input))
	}
	outcome := "inconclusive"
	if metrics["native"]["quality_scored"] == true && metrics["knodexa"]["quality_scored"] == true {
		n := metrics["native"]["common_quality_passed"] == true
		k := metrics["knodexa"]["common_quality_passed"] == true
		outcome = "tie"
		if n != k {
			outcome = "native_win_this_trial"
			if k {
				outcome = "knodexa_win_this_trial"
			}
		}
	}
	summary["metrics"] = metrics
	summary["paired_quality_outcome"] = outcome
	data, _ := json.MarshalIndent(summary, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "results.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	report := "# Paired real-agent work identity comparison\n\nOutcome: **" + outcome + "**.\n\nBoth arms use the same Go requirements, actual producer memory capture, unrelated image work, and a same-conversation return with no task ID or path. The native arm uses an index and modular JSON memory files with native file tools; Knodexa uses its request resolver and canonical task records.\n\n| Arm | Common quality checks | Workflow seconds | Command/file items | Input tokens |\n|---|---|---:|---:|---:|\n" + strings.Join(rows, "\n") + "\n\nCommon quality means independent returned behavior passes and original reference and completed memory are preserved. Native memory retrieval does not require a Knodexa tool call. Knodexa-specific routing/baseline checks are reported separately, not used to make the native quality criteria stricter.\n\nOne pair, unrandomized arm order, a controlled Go repository, one recorded phrase, different generated implementations and histories, no forced compaction, and generated background memories disabled. This does not establish general superiority over Codex or compare all desktop tools. Timeouts remain unscored. Workflow timings include the three agent phases, exclude harness setup/grading, and depend on server scheduling. Tokens are cumulative tool-call usage, not active context-window occupancy. Do not treat a tie as a quality win.\n\nRaw traces, final source snapshots, producer memories, and independent checks remain in each arm directory. Plan and aggregate metrics are in plan.json and results.json.\n"
	if summary["harness_interruption"] != nil {
		report += "\nHarness interruption: the initial native producer passed behavior checks, but a typed parser rejected its valid prose approach notes. Its interrupted traces remain in native-setup-error. The parser was corrected; the fresh Knodexa arm completed, then a fresh native arm was attempted. Its return phase timed out after writing code and passing agent-authored tests; independent return checks were not reached. The interrupted setup attempt was not a scored quality failure, and its usage is excluded from the paired workflow figures.\n"
	}
	if outcome == "inconclusive" {
		report += "\nUnscored-arm usage and tool counts may be partial: a timed-out return has no completed-turn token usage. Do not compare its displayed token total with a completed arm or interpret the timeout as a demonstrated continuity error.\n"
	}
	report = strings.ReplaceAll(report, "Timeouts remain unscored.", "Generated artifacts are independently graded even after a return-response timeout; timeouts without artifacts remain unscored. Response completion is a separate metric.")
	report += "\nResponse completion: native=" + fmt.Sprint(metrics["native"]["response_completed"]) + ", Knodexa=" + fmt.Sprint(metrics["knodexa"]["response_completed"]) + ". Token usage may be incomplete for any timed-out response even when its artifact is graded.\n"
	if os.Getenv("KNODEXA_OPTIMIZED_FLOW") == "1" {
		report += "\nThis run uses the verified one-command record operation and compact text prepare output.\n"
	}
	if os.Getenv("KNODEXA_FRESH_RETURN") == "1" {
		report = strings.ReplaceAll(report, "same-conversation return", "fresh-context return")
		report = strings.ReplaceAll(report, "Same-conversation identity return", "Fresh-context identity return")
		report += "\nReturn requests start a fresh agent session for both arms, with the same repository and preserved memories but without earlier conversation history. This tests cross-context retrieval, not forced compaction within a conversation.\n"
	}
	if err := os.WriteFile(filepath.Join(out, "REPORT.md"), []byte(report), 0644); err != nil {
		t.Fatal(err)
	}
}

func runNativeIdentity(t *testing.T) {
	repo, _ := filepath.Abs("../..")
	out := filepath.Join(identityComparisonOutput(repo), "native")
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "memory"), 0755); err != nil {
		t.Fatal(err)
	}
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	shell := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = root
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %v %s", args, err, data)
		}
		return strings.TrimSpace(string(data))
	}
	write(filepath.Join(root, "go.mod"), "module identitysample\n\ngo 1.26.0\n")
	for _, args := range [][]string{{"git", "init", "-b", "main"}, {"git", "config", "user.name", "Identity trial"}, {"git", "config", "user.email", "trial@example.test"}} {
		shell(args...)
	}
	write(filepath.Join(root, "memory", "index.md"), "# Work identities\n- work-17: Event forwarding adapters; memory/work-17.json\n- image-42: Image label normalization; memory/image-42.json\nResolve requests by recorded titles, aliases, goals, and scopes, not recency.\n")
	for _, task := range []*model.TaskData{
		{ID: "work-17", Title: "Event forwarding adapters", Goal: "Deliver raw events through injected services with observer ordering", Status: model.TaskStatusActive, TouchesFiles: []string{"handlers/reference.go"}, Constraints: []string{"Delegate raw values exactly once; do not normalize or coerce", "Preserve result and error identity", "Observer started before service; finished only on success"}, RequiredVerification: []string{"Service substitution, error identity, and observer ordering tests"}},
		{ID: "image-42", Title: "Image label normalization", Goal: "Inline string normalization and empty fallback for nonstrings", Status: model.TaskStatusActive, TouchesFiles: []string{"images/normalize.go"}},
	} {
		data, _ := json.MarshalIndent(task, "", "  ")
		write(filepath.Join(root, "memory", task.ID+".json"), string(data)+"\n")
	}
	write(filepath.Join(root, "AGENTS.md"), `Use Go standard library and native file tools for continuity.
Read memory/index.md and the selected work JSON before coding. For requests
without an ID, match recorded titles, aliases, goals, and scopes before coding
or creating work. Continue unique matches; uncertain matches need clarification.
Read selected references. Do not reopen completed work or rewrite its source
memory on return. After initially implementing known work, record approach,
explicit rules, references [{"path":"..."}], scopes, and aliases in its JSON.
Capture the phrase users use for that work. Add and run Go tests. Compare saved
baseline_commit with git rev-parse HEAD on returning to existing work.
`)
	shell("git", "add", ".")
	shell("git", "commit", "-m", "Initial trial")
	report := map[string]interface{}{"arm": "native", "status": "running", "model": "gpt-6.1-sol", "reasoning_effort": "low", "background_memories_enabled": false, "scope": "Same-conversation identity return; modular file memory, actual producer capture, no task ID or file path on return"}
	save := func() {
		data, _ := json.MarshalIndent(report, "", "  ")
		write(filepath.Join(out, "results.json"), string(data)+"\n")
	}
	run := func(label, prompt, resume string) livePhase {
		t.Helper()
		phase := executeLiveIdentity(root, out, label, prompt, resume)
		report[label] = phase
		save()
		t.Logf("%s: exit=%d time=%.1fs", label, phase.ExitCode, phase.Seconds)
		if phase.ExitCode != 0 || phase.ThreadID == "" {
			report["status"] = "inconclusive"
			// Retain code even when a completed edit is followed by a stalled
			// final model response. Missing artifacts must not look like bad code.
			paths, _ := filepath.Glob(filepath.Join(root, "handlers", "*.go"))
			for _, path := range paths {
				data, _ := os.ReadFile(path)
				write(filepath.Join(out, filepath.Base(path)+".txt"), string(data))
			}
			save()
			if label == "return" && phase.ThreadID != "" {
				return phase
			}
			t.Fatalf("%s did not complete; no quality score", label)
		}
		return phase
	}
	seed := run("establish", `Current work ID work-17. Build a webhook handler in package handlers, in handlers/reference.go. Define Transformer.Transform(value any) (any,error), Observer.Started(value any), Observer.Finished(result any), and Handle(value any,service Transformer,observer Observer) (any,error). Delegate the raw input exactly once and return both original result and original error unchanged. Call Started before Transform for a non-nil observer; call Finished only if there is no error. Do not trim, uppercase, coerce, retry, recover panics, or replace values. Add and run tests. Record this established approach and reference in the work's memory JSON, and capture the phrase "webhook handler" as a work alias. Do not create another task.`, "")
	grade := func(label, target string) bool {
		t.Helper()
		source := strings.ReplaceAll(liveIdentityGrader, "TARGET", target)
		path := filepath.Join(root, "handlers", "identity_independent_test.go")
		write(path, source)
		cmd := exec.Command("go", "test", "./...")
		cmd.Dir = root
		data, err := cmd.CombinedOutput()
		os.Remove(path)
		write(filepath.Join(out, label+"-independent-checks.log"), string(data))
		report[label+"_independent_checks_passed"] = err == nil
		save()
		return err == nil
	}
	if !grade("establish", "Handle") {
		report["status"] = "producer_failed"
		save()
		t.Fatal("producer reference failed")
	}
	sourcePath := filepath.Join(root, "memory", "work-17.json")
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	var source model.TaskData
	write(filepath.Join(out, "producer-memory-initial.json"), string(raw))
	var document map[string]interface{}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	// Native notes may use prose rather than Knodexa's typed approach schema.
	// Preserve that representation instead of requiring the competing format.
	if !strings.Contains(strings.ToLower(string(raw)), "webhook handler") || document["approach"] == nil {
		report["status"] = "memory_capture_failed"
		save()
		t.Fatal("producer failed to record memory")
	}
	shell("git", "add", ".")
	shell("git", "commit", "-m", "Correct reference and recorded memory")
	source.BaselineCommit = shell("git", "rev-parse", "HEAD")
	document["status"] = "COMPLETED"
	document["baseline_commit"] = source.BaselineCommit
	protected, _ := json.MarshalIndent(document, "", "  ")
	protected = append(protected, '\n')
	write(sourcePath, string(protected))
	original, err := os.ReadFile(filepath.Join(root, "handlers", "reference.go"))
	if err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(out, "producer-memory.json"), string(protected))
	write(filepath.Join(out, "established-reference.go.txt"), string(original))
	run("switch", `Switch focus to known work image-42. Create package images with NormalizeLabel(value any) string. In this image utility, normalize directly: trim strings, collapse internal whitespace, then uppercase; nonstrings return an empty string. No service injection or observer hooks. Add tests covering malformed values and whitespace. Record the image scope approach and reference in its memory JSON, run tests, and preserve handler work and its memory.`, seed.ThreadID)
	shell("git", "add", ".")
	shell("git", "commit", "-m", "Unrelated image work")
	returnThread := seed.ThreadID
	if os.Getenv("KNODEXA_FRESH_RETURN") == "1" {
		returnThread = ""
	}
	returned := run("return", `Add another webhook handler for delivery events. Expose HandleWebhook(value any, service Transformer, observer Observer) (any,error). Keep our established layout and behavior. Add and run relevant tests.`, returnThread)
	report["response_completed"] = returned.ExitCode == 0
	if !canGradeLiveReturn(root, returned) {
		report["status"] = "inconclusive"
		save()
		t.Fatal("no returned artifact to grade after timeout")
	}
	passed := grade("return", "HandleWebhook")
	current, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := os.ReadFile(filepath.Join(root, "handlers", "reference.go"))
	if err != nil {
		t.Fatal(err)
	}
	report["completed_memory_preserved"] = bytes.Equal(protected, current)
	report["reference_preserved"] = bytes.Equal(original, reference)
	report["baseline_drift_expected"] = source.BaselineCommit != shell("git", "rev-parse", "HEAD")
	files, _ := filepath.Glob(filepath.Join(root, "handlers", "*.go"))
	for _, path := range files {
		data, _ := os.ReadFile(path)
		write(filepath.Join(out, filepath.Base(path)+".txt"), string(data))
	}
	success := passed && bytes.Equal(protected, current) && bytes.Equal(original, reference)
	report["status"] = "passed"
	if returned.ExitCode != 0 {
		report["status"] = "artifact_passed_response_incomplete"
	}
	if !success {
		report["status"] = "failed"
	}
	save()
	if !success {
		t.Fatal("native continuity checks failed")
	}
}

// Regenerate a saved comparison report without making any model calls.
func TestSummarizeIdentityComparison(t *testing.T) {
	if os.Getenv("KNODEXA_IDENTITY_REPORT") != "1" {
		t.Skip("set KNODEXA_IDENTITY_REPORT=1 to summarize retained results")
	}
	repo, _ := filepath.Abs("../..")
	out := filepath.Join(repo, "audit", "identity-comparison")
	data, err := os.ReadFile(filepath.Join(out, "results.json"))
	if err != nil {
		t.Fatal(err)
	}
	var summary map[string]interface{}
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatal(err)
	}
	arms := map[string]interface{}{}
	for _, arm := range []string{"native", "knodexa"} {
		raw, err := os.ReadFile(filepath.Join(out, arm, "results.json"))
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]interface{}
		if err := json.Unmarshal(raw, &result); err != nil {
			t.Fatal(err)
		}
		arms[arm] = result
	}
	summary["arms"] = arms
	if _, err := os.Stat(filepath.Join(out, "native-setup-error")); err == nil {
		summary["harness_interruption"] = map[string]interface{}{"arm": "native", "path": "native-setup-error", "reason": "Native producer behavior passed, but the harness incorrectly required Knodexa's typed approach structure for valid prose memory. Corrected schema handling; retained interrupted attempt; completed a fresh native arm without replacing a scored quality outcome."}
	}
	writeIdentityComparisonReport(t, out, summary)
}

// Complete a native arm interrupted by a harness-format error, without
// replacing any scored quality result or rerunning the completed Knodexa arm.
func TestLiveNativeIdentity(t *testing.T) {
	if os.Getenv("KNODEXA_LIVE_NATIVE_RETRY") != "1" {
		t.Skip("opt-in native arm recovery")
	}
	runNativeIdentity(t)
}
