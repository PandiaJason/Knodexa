package experiment

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"knodexa/internal/approach"
	"knodexa/internal/codegraph"
	"knodexa/internal/identity"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const hardReference = `package handlers
type Transformer interface { Transform(any)(any,error) }
type Observer interface { Started(any); Finished(any) }
func Handle(value any, service Transformer, observer Observer)(any,error){
 if observer!=nil {observer.Started(value)}
 result,err:=service.Transform(value)
 if err==nil&&observer!=nil {observer.Finished(result)}
 return result,err
}
`
const hardLesson = "When adding a new production handler entrypoint, expose HandlerContractVersion() string returning raw-v1 for integration compatibility. This requirement comes from the SDK study, not the older unversioned reference handler."
const hardFailure = "F-observer-panic: recovering observer panics hid integration bugs. Started panic must propagate unchanged before service invocation; Finished panic must propagate unchanged after one successful service invocation. Never retry Transform."

type hardFixture struct {
	root, arm, baseline string
	protected           []byte
}
type hardResult struct {
	ExtensionCaptured bool      `json:"extension_captured,omitempty"`
	Case              string    `json:"case"`
	Arm               string    `json:"arm"`
	Phase             livePhase `json:"phase"`
	Behavior          bool      `json:"behavior_passed"`
	Memory            bool      `json:"completed_memory_preserved"`
	Reference         bool      `json:"reference_preserved"`
	Stale             bool      `json:"stale_agreement_reported"`
	Scored            bool      `json:"scored"`
	Passed            bool      `json:"passed"`
	Error             string    `json:"error,omitempty"`
}

func hardWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}
func hardJSON(t *testing.T, path string, value interface{}) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	hardWrite(t, path, string(data)+"\n")
}
func hardCommand(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = root
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %v %s", args, err, data)
	}
	return strings.TrimSpace(string(data))
}
func hardMemory(f hardFixture) string {
	if f.arm == "native" {
		return filepath.Join(f.root, "memory", "work-prod.json")
	}
	return filepath.Join(f.root, ".knodexa", "tasks", "work-prod", "work-prod.yaml")
}
func hardCopy(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(to, rel)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("unexpected fixture symlink %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(dest, data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatal(err)
	}
}

func setupHardProducer(t *testing.T, repo, out, arm, root string) hardFixture {
	t.Helper()
	f := hardFixture{root: root, arm: arm}
	hardWrite(t, filepath.Join(root, "go.mod"), "module identitysample\n\ngo 1.26.0\n")
	hardWrite(t, filepath.Join(root, ".gitignore"), "knodexa\n.knodexa/cache.db*\n.knodexa/focus.json\n")
	hardWrite(t, filepath.Join(root, "handlers", "reference.go"), hardReference)
	hardWrite(t, filepath.Join(root, "handlers", "reference_test.go"), strings.ReplaceAll(liveIdentityGrader, "TARGET", "Handle"))
	hardWrite(t, filepath.Join(root, "contracts", "version.go"), "package contracts\nconst Version = \"raw-v1\"\n")
	hardWrite(t, filepath.Join(root, "contracts", "version_test.go"), "package contracts\nimport \"testing\"\nfunc TestVersion(t *testing.T){if Version!=\"raw-v1\"{t.Fatal(Version)}}\n")
	for _, args := range [][]string{{"init", "-b", "main"}, {"config", "user.name", "Continuity trial"}, {"config", "user.email", "trial@example.test"}} {
		hardCommand(t, root, append([]string{"git"}, args...)...)
	}
	task := &model.TaskData{ID: "work-prod", Title: "Production event forwarding", Goal: "Production webhook delivery through injected service", Status: model.TaskStatusActive, Aliases: []string{"production webhook delivery"}, TouchesFiles: []string{"handlers/reference.go"}, Constraints: []string{"Raw input delegated exactly once; preserve result/error identity", "Observer Started before service; Finished only on success", "No trimming, coercion, retry or panic recovery"}, RequiredVerification: []string{"Raw inputs, service substitution, observer ordering and error identity"}}
	if arm == "knodexa" {
		hardCommand(t, repo, "go", "build", "-o", filepath.Join(root, "knodexa"), "./cmd/knodexa")
		hardCommand(t, root, "./knodexa", "init", "--name=Hard continuity trial")
		store := storage.NewStorage(root)
		if err := store.SaveTask(task); err != nil {
			t.Fatal(err)
		}
		if err := store.SaveTask(&model.TaskData{ID: "sdk-study", Title: "SDK compatibility study", Status: model.TaskStatusCompleted}); err != nil {
			t.Fatal(err)
		}
		if err := store.SaveFailure(&model.FailureData{ID: "F-observer-panic", TaskID: "work-prod", Summary: hardFailure, CausalDetails: "Recovering observer panics hides caller defects; retain exact-once delegation and panic identity", AttemptedAt: time.Now().UTC()}); err != nil {
			t.Fatal(err)
		}
		if err := store.SaveLesson(&model.LessonDocument{Frontmatter: model.LessonFrontmatter{ID: "L-sdk-contract", Title: "New handler compatibility marker", SourceTaskID: "sdk-study", Status: model.StatusVerified, Summary: hardLesson, Applicability: "When adding a production handler entrypoint", RelatedFiles: []string{"handlers/reference.go"}, Tags: []string{"handlers", "compatibility"}, EvidenceIDs: []string{"E-sdk-contract"}}, Body: hardLesson}); err != nil {
			t.Fatal(err)
		}
		if err := store.SaveEvidence(&model.EvidenceData{ID: "E-sdk-contract", NodeID: "L-sdk-contract", EvidenceType: model.EvidenceType("TEST_RUN"), ExitCode: 0, Command: "go test ./...", OutputSummary: "Fixture version is raw-v1; initial reference behavior verified before producer", CreatedAt: time.Now().UTC()}); err != nil {
			t.Fatal(err)
		}
		hardWrite(t, filepath.Join(root, "AGENTS.md"), `Use Go and shell. Known initial work may use ./knodexa resume <id> --json.
For a new request use ./knodexa prepare "<request>" --text. This supplies the handoff; no second resume is needed.
Inspect identity evidence. Continue uniquely named work; review competing candidates before explicitly selecting. Do not invent new work or reactivate completed source work.
Read restored source, failures and cross-task lessons. Apply relevant lessons without overriding established constraints.
For initial capture write agreement JSON {aliases,files,approach:{summary,rules,scopes,references:[{path}]}} then ./knodexa record <id> --file=agreement.json --verify='go test ./...'.
Preserve completed source records on extensions. Report stale/missing references and changed baselines; inspect current code before extending, without silently accepting the baseline or rewriting the old agreement.
`)
	} else {
		hardJSON(t, filepath.Join(root, "memory", "work-prod.json"), task)
		hardJSON(t, filepath.Join(root, "memory", "knowledge.json"), map[string]interface{}{"failures": []interface{}{map[string]interface{}{"id": "F-observer-panic", "task_id": "work-prod", "description": hardFailure}}, "lessons": []interface{}{map[string]interface{}{"id": "L-sdk-contract", "source_task": "sdk-study", "status": "VERIFIED", "related_files": []string{"handlers/reference.go"}, "applicability": "When adding a production handler entrypoint", "description": hardLesson}}})
		hardWrite(t, filepath.Join(root, "memory", "index.md"), "# Work identities\n- work-prod: production webhook delivery; memory/work-prod.json\nCross-task lessons and failures: memory/knowledge.json; select by task and related files.\n")
		hardWrite(t, filepath.Join(root, "AGENTS.md"), `Use Go and native file tools. Before coding read memory/index.md, identify selected work by aliases, titles, goals and scopes, and read its record and source references. Read applicable failures and cross-task lessons in memory/knowledge.json.
Continue uniquely named work; uncertain matches require review, not invented tasks. Preserve completed source records on extensions. Compare saved baseline_commit to git rev-parse HEAD.
Initial producer: record approach, rules, scopes, reference paths/hashes and aliases in the selected JSON. Prose approach notes are allowed. Run Go tests.
Report stale/missing references and changed baselines; inspect current code before extending, without silently accepting the baseline or rewriting the old agreement.
`)
	}
	hardCommand(t, root, "gofmt", "-w", "handlers/reference.go", "handlers/reference_test.go", "contracts/version.go")
	hardCommand(t, root, "go", "test", "./...")
	hardCommand(t, root, "git", "add", ".")
	hardCommand(t, root, "git", "commit", "-m", "Correct reference and equivalent continuity facts")
	hardWrite(t, filepath.Join(out, arm, "producer", "PROMPT.txt"), "Known initial work work-prod. The existing production handler is already correct and its tests pass. Review it and record its established approach, handlers/ scope, source reference, raw delegation rules, and alias production webhook delivery using your documented memory mechanism. Run tests. Preserve the implementation. Do not implement the extension yet.")
	prompt, _ := os.ReadFile(filepath.Join(out, arm, "producer", "PROMPT.txt"))
	phase := executeLiveIdentity(root, filepath.Join(out, arm, "producer"), "capture", string(prompt), "")
	hardJSON(t, filepath.Join(out, arm, "producer", "phase.json"), phase)
	if phase.ExitCode != 0 {
		t.Fatal("producer did not finish; retained unscored setup logs")
	}
	hardCommand(t, root, "go", "test", "./...")
	hardCommand(t, root, "git", "add", ".")
	hardCommand(t, root, "git", "commit", "--allow-empty", "-m", "Producer captured known correct work")
	f.baseline = hardCommand(t, root, "git", "rev-parse", "HEAD")
	if arm == "knodexa" {
		store := storage.NewStorage(root)
		saved, err := store.ReadTask("work-prod")
		if err != nil {
			t.Fatal(err)
		}
		if saved.Approach == nil || len(saved.CodeAnchors) == 0 {
			t.Fatal("producer did not capture profile/source anchors")
		}
		saved.Status = model.TaskStatusCompleted
		saved.BaselineCommit = f.baseline
		if err := store.SaveTask(saved); err != nil {
			t.Fatal(err)
		}
	} else {
		raw, err := os.ReadFile(hardMemory(f))
		if err != nil {
			t.Fatal(err)
		}
		var saved map[string]interface{}
		if err := json.Unmarshal(raw, &saved); err != nil {
			t.Fatal(err)
		}
		if saved["approach"] == nil {
			t.Fatal("producer did not capture native approach")
		}
		saved["status"] = "COMPLETED"
		saved["baseline_commit"] = f.baseline
		hardJSON(t, hardMemory(f), saved)
	}
	f.protected, _ = os.ReadFile(hardMemory(f))
	return f
}

func hardSimilarTasks(t *testing.T, f hardFixture) {
	for _, name := range []string{"preview", "batch", "legacy", "search", "archive", "diagnostic"} {
		path := name + "/reference.go"
		source := strings.Replace(hardReference, "package handlers", "package "+name, 1)
		hardWrite(t, filepath.Join(f.root, path), source)
		task := &model.TaskData{ID: "work-" + name, Title: strings.Title(name) + " event forwarding", Goal: name + " webhook delivery with different conventions", Status: model.TaskStatusCompleted, Aliases: []string{name + " webhook delivery"}, TouchesFiles: []string{path}, Approach: &model.TaskApproach{Summary: "Distinct " + name + " workflow", Rules: []string{"Use " + name + " conventions; not production's observer semantics"}, Scopes: []string{name + "/"}, References: []model.ApproachReference{{Path: path}}}}
		if f.arm == "knodexa" {
			store := storage.NewStorage(f.root)
			// Install the initial agreement through approach.Set, not twice via
			// SaveTask then Set (which is an intentional replacement).
			initialApproach := *task.Approach
			task.Approach = nil
			if err := store.SaveTask(task); err != nil {
				t.Fatal(err)
			}
			db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
			if err != nil {
				t.Fatal(err)
			}
			idx := index.NewIndexer(db, store)
			if err := approach.Set(store, db, idx, task.ID, initialApproach); err != nil {
				db.Close()
				t.Fatal(err)
			}
			if err := identity.Capture(store, db, idx, task.ID, []string{path}); err != nil {
				db.Close()
				t.Fatal(err)
			}
			db.Close()
		} else {
			hardJSON(t, filepath.Join(f.root, "memory", task.ID+".json"), task)
			file, err := os.OpenFile(filepath.Join(f.root, "memory", "index.md"), os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(file, "- %s: %s; memory/%s.json\n", task.ID, task.Aliases[0], task.ID)
			file.Close()
		}
	}
	hardCommand(t, f.root, "git", "add", ".")
	hardCommand(t, f.root, "git", "commit", "-m", "Several similar independently recorded workflows")
}

const hardObserverGrader = `package handlers
import "testing"
type auditCrashObserver struct{ token any; started bool }
func(o *auditCrashObserver)Started(any){if o.started{panic(o.token)}}
func(o *auditCrashObserver)Finished(any){if !o.started{panic(o.token)}}
func TestRecordedObserverFailure(t *testing.T){for _,started:=range []bool{true,false}{func(){events:=[]string{};token:=&struct{N int}{123};service:=&auditService{events:&events};observer:=&auditCrashObserver{token:token,started:started};defer func(){if recover()!=token{t.Error("observer panic swallowed/replaced")};expected:=1;if started{expected=0};if service.calls!=expected{t.Error("service invocation order/retry changed")}}();HandleWebhook("x",service,observer)}()}}
`
const hardLessonGrader = `package handlers
import "testing"
func TestTransferredSDKLesson(t *testing.T){if HandlerContractVersion()!="raw-v1"{t.Fatal("cross-task compatibility marker lost")}}
`

func hardIndependentGrader() string {
	return strings.NewReplacer("auditService", "hardHarnessService", "auditObserver", "hardHarnessObserver", "TestIndependentIdentityContract", "TestHardHarnessIdentityContract", "TARGET", "HandleWebhook").Replace(liveIdentityGrader)
}

func runHardCase(t *testing.T, base hardFixture, out, name string) hardResult {
	t.Helper()
	f := base
	f.root = t.TempDir()
	hardCopy(t, base.root, f.root)
	ref := "handlers/reference.go"
	if name == "similar-tasks" {
		hardSimilarTasks(t, f)
	}
	if name == "refactor" {
		hardCommand(t, f.root, "git", "mv", ref, "handlers/transport.go")
		ref = "handlers/transport.go"
		data, _ := os.ReadFile(filepath.Join(f.root, ref))
		source := strings.Replace(string(data), "func Handle(", "func DispatchProduction(", 1) + "\nfunc TraceTag() string {return \"delivery-v2\"}\n"
		hardWrite(t, filepath.Join(f.root, ref), source)
		tests, _ := os.ReadFile(filepath.Join(f.root, "handlers", "reference_test.go"))
		hardWrite(t, filepath.Join(f.root, "handlers", "reference_test.go"), strings.ReplaceAll(string(tests), "Handle(", "DispatchProduction("))
		hardCommand(t, f.root, "gofmt", "-w", ref, "handlers/reference_test.go")
		hardCommand(t, f.root, "go", "test", "./...")
		hardCommand(t, f.root, "git", "add", ".")
		hardCommand(t, f.root, "git", "commit", "-m", "Authorized source relocation and implementation refactor")
	}
	expectedRef, _ := os.ReadFile(filepath.Join(f.root, ref))
	prompt := "Add another production webhook delivery handler in package handlers. Expose HandleWebhook(value any, service Transformer, observer Observer) (any,error). Keep the established layout and behavior and applicable cross-task conventions. Add and run tests. Preserve completed source memory."
	if name == "recorded-failure" {
		prompt += " Add resilience while respecting the recorded observer failure contract; do not repeat past failed approaches."
	}
	if name == "refactor" {
		prompt += " The repository had an authorized refactor after the saved checkpoint. Inspect the current implementation, explicitly report stale agreements/references, and preserve the original completed agreement and baseline."
	}
	hardWrite(t, filepath.Join(out, "PROMPT.txt"), prompt)
	phase := executeLiveIdentity(f.root, out, "return", prompt, "")
	result := hardResult{Case: name, Arm: f.arm, Phase: phase}
	result.Scored = canGradeLiveReturn(f.root, phase)
	if result.Scored {
		hardWrite(t, filepath.Join(f.root, "handlers", "hard_independent_test.go"), hardIndependentGrader())
		hardWrite(t, filepath.Join(f.root, "handlers", "hard_lesson_test.go"), hardLessonGrader)
		if name == "recorded-failure" {
			hardWrite(t, filepath.Join(f.root, "handlers", "hard_failure_test.go"), strings.ReplaceAll(hardObserverGrader, "auditService", "hardHarnessService"))
		}
		cmd := exec.Command("go", "test", "./...")
		cmd.Dir = f.root
		data, err := cmd.CombinedOutput()
		hardWrite(t, filepath.Join(out, "independent-checks.log"), string(data))
		result.Behavior = err == nil
	}
	current, _ := os.ReadFile(hardMemory(f))
	result.Memory = hardFactsPreserved(base.protected, current, f.arm)
	actual, _ := os.ReadFile(filepath.Join(f.root, ref))
	result.Reference = bytes.Equal(expectedRef, actual)
	final, _ := os.ReadFile(filepath.Join(out, "return.final.txt"))
	lower := strings.ToLower(string(final))
	for _, word := range []string{"stale", "outdated", "renamed", "relocated", "missing reference"} {
		result.Stale = result.Stale || strings.Contains(lower, word)
	}
	result.Passed = result.Scored && result.Behavior && result.Memory && result.Reference && (name != "refactor" || result.Stale)
	if os.Getenv("KNODEXA_GRADE_EXTENSION_CAPTURE") == "1" && f.arm == "knodexa" {
		store := storage.NewStorage(f.root)
		db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		idx := index.NewIndexer(db, store)
		if err := idx.Sync(); err != nil {
			t.Fatal(err)
		}
		tasks, err := store.ListTasks()
		if err != nil {
			t.Fatal(err)
		}
		for _, task := range tasks {
			if task.ID == "work-prod" || task.Approach == nil || len(task.CodeAnchors) == 0 {
				continue
			}
			contracts, err := approach.Resolve(db, task.ID)
			if err != nil {
				continue
			}
			inherited := false
			for _, contract := range contracts {
				if contract.SourceTask == "work-prod" {
					inherited = true
				}
			}
			if !inherited {
				continue
			}
			report, err := approach.Check(store, db, task.ID)
			if err != nil || !report.StructuralChecksPassed {
				continue
			}
			current := true
			webhook := false
			for _, saved := range task.CodeAnchors {
				anchor, err := codegraph.Read(f.root, saved.Path)
				if err != nil || anchor.SHA256 != saved.SHA256 {
					current = false
				}
				for _, symbol := range saved.Symbols {
					if symbol == "HandleWebhook" {
						webhook = true
					}
				}
			}
			if current && webhook {
				result.ExtensionCaptured = true
			}
		}
	}
	filepath.WalkDir(filepath.Join(f.root, "handlers"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			data, _ := os.ReadFile(path)
			hardWrite(t, filepath.Join(out, "source", entry.Name()+".txt"), string(data))
		}
		return nil
	})
	hardWrite(t, filepath.Join(out, "completed-memory-before.txt"), string(base.protected))
	hardWrite(t, filepath.Join(out, "completed-memory-after.txt"), string(current))
	hardJSON(t, filepath.Join(out, "results.json"), result)
	return result
}

func TestLiveHardContinuity(t *testing.T) {
	if os.Getenv("KNODEXA_LIVE_HARD") != "1" {
		t.Skip("opt-in four-case paid real-agent comparison")
	}
	repo, _ := filepath.Abs("../..")
	out := filepath.Join(repo, "audit", "hard-continuity-v1")
	if _, err := os.Stat(out); err == nil {
		t.Fatal("preserve existing run before rerunning")
	}
	cases := []string{"fresh-handoff", "similar-tasks", "recorded-failure", "refactor"}
	fixtureHash := sha256.Sum256([]byte(hardReference))
	hardWrite(t, filepath.Join(out, "fixture-sha256.txt"), hex.EncodeToString(fixtureHash[:])+"\n")
	hardJSON(t, filepath.Join(out, "plan.json"), map[string]interface{}{"cases": cases, "pairs_per_case": 1, "model": "gpt-6.1-sol", "reasoning_effort": "low", "primary": "independent behavior, transferred lesson, completed memory and current source preserved; refactor also reports stale agreement", "producer": "real agent captures independently verified fixed source fixture; it does not author initial source", "return": "fresh agent context, same canonical facts in both arms", "order": "alternating arms by case", "stop_rule": "run all declared cases, retain failure/timeouts; no selective reruns"})
	fixtures := map[string]hardFixture{}
	for _, arm := range []string{"native", "knodexa"} {
		root := t.TempDir()
		t.Run(arm+"-producer", func(t *testing.T) { fixtures[arm] = setupHardProducer(t, repo, out, arm, root) })
	}
	results := []hardResult{}
	rows := []string{}
	for i, name := range cases {
		order := []string{"native", "knodexa"}
		if i%2 == 1 {
			order = []string{"knodexa", "native"}
		}
		pair := map[string]hardResult{}
		for _, arm := range order {
			f, ok := fixtures[arm]
			if !ok {
				pair[arm] = hardResult{Case: name, Arm: arm, Error: "producer setup incomplete"}
				continue
			}
			t.Run(name+"-"+arm, func(t *testing.T) { pair[arm] = runHardCase(t, f, filepath.Join(out, name, arm), name) })
		}
		for _, arm := range []string{"native", "knodexa"} {
			r, ok := pair[arm]
			if !ok {
				r = hardResult{Case: name, Arm: arm, Error: "harness setup failed"}
			}
			results = append(results, r)
			rows = append(rows, fmt.Sprintf("| %s | %s | %t | %t | %t | %t | %.2f | %d |", name, arm, r.Scored, r.Behavior, r.Memory, r.Passed, r.Phase.Seconds, r.Phase.ToolItems))
		}
		hardJSON(t, filepath.Join(out, "results.json"), results)
	}
	report := "# Four harder real-agent continuity cases\n\n| Case | Arm | Scored | Behavior + lesson | Completed memory | Overall pass | Seconds | Actions |\n|---|---|---|---|---|---|---:|---:|\n" + strings.Join(rows, "\n") + "\n\nOne pair per case, same model, alternating order, fresh return contexts and equivalent canonical facts. Producers captured independently verified fixed Go source rather than authoring it. Both arms receive the same cross-task compatibility lesson and observer failure. Refactor changes the reference path, function name and source hash; the original agreement is deliberately retained as history and the agent must flag it. Agent source tests do not determine scoring: independent Go behavior and compatibility tests run after return. Raw traces and before/after memory remain per case.\n\nScored failures are retained; setup failures and interrupted returns without artifacts are unscored. This controlled suite cannot establish general superiority or forced-compaction behavior. Timings are model phases excluding setup/grading. One observed advantage needs replication on other repositories.\n"
	hardWrite(t, filepath.Join(out, "REPORT.md"), report)
}

func TestHardFixtureAndIndependentGraders(t *testing.T) {
	root := t.TempDir()
	hardWrite(t, filepath.Join(root, "go.mod"), "module fixture\ngo 1.26.0\n")
	hardWrite(t, filepath.Join(root, "reference.go"), hardReference+"\nfunc HandleWebhook(v any,s Transformer,o Observer)(any,error){return Handle(v,s,o)}\nfunc HandlerContractVersion()string{return \"raw-v1\"}\n")
	hardWrite(t, filepath.Join(root, "reference_test.go"), strings.ReplaceAll(liveIdentityGrader, "TARGET", "Handle"))
	hardWrite(t, filepath.Join(root, "independent_test.go"), hardIndependentGrader())
	hardWrite(t, filepath.Join(root, "failure_test.go"), strings.ReplaceAll(hardObserverGrader, "auditService", "hardHarnessService"))
	hardWrite(t, filepath.Join(root, "lesson_test.go"), hardLessonGrader)
	hardCommand(t, root, "go", "test", "./...")
	raw, err := os.ReadFile(filepath.Join(root, "reference.go"))
	if err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(string(raw), "return Handle(v,s,o)", "return nil,nil", 1)
	hardWrite(t, filepath.Join(root, "reference.go"), bad)
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = root
	if err := cmd.Run(); err == nil {
		t.Fatal("independent grader accepted broken delegation")
	}
	missingLesson := strings.Replace(string(raw), "func HandlerContractVersion()string{return \"raw-v1\"}", "", 1)
	hardWrite(t, filepath.Join(root, "reference.go"), missingLesson)
	cmd = exec.Command("go", "test", "./...")
	cmd.Dir = root
	if err := cmd.Run(); err == nil {
		t.Fatal("independent grader accepted missing cross-task convention")
	}
}
