package experiment

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"knodexa/internal/continuity"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// This opt-in test consumes authenticated Codex usage. Ordinary checks skip it.
func TestLiveRequestIdentity(t *testing.T) {
	if os.Getenv("KNODEXA_LIVE_IDENTITY") != "1" {
		t.Skip("set KNODEXA_LIVE_IDENTITY=1 to run the live agent trial")
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(repo, "audit", "identity-live")
	if os.Getenv("KNODEXA_LIVE_COMPARISON") == "1" {
		out = filepath.Join(identityComparisonOutput(repo), "knodexa")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
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
			t.Fatalf("%v: %v\n%s", args, err, data)
		}
		return strings.TrimSpace(string(data))
	}
	build := exec.Command("go", "build", "-o", filepath.Join(root, "knodexa"), "./cmd/knodexa")
	build.Dir = repo
	if data, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, data)
	}
	write(filepath.Join(root, "go.mod"), "module identitysample\n\ngo 1.26.0\n")
	write(filepath.Join(root, ".gitignore"), "knodexa\n.knodexa/cache.db*\n.knodexa/focus.json\n")
	shell("git", "init", "-b", "main")
	shell("git", "config", "user.name", "Identity trial")
	shell("git", "config", "user.email", "trial@example.test")
	shell("./knodexa", "init", "--name=Real identity trial")
	store := storage.NewStorage(root)
	for _, task := range []*model.TaskData{
		{ID: "work-17", Title: "Event forwarding adapters", Goal: "Deliver raw events through injected services with observer ordering", Status: model.TaskStatusActive, TouchesFiles: []string{"handlers/reference.go"}, Constraints: []string{"Delegate raw values exactly once; do not normalize or coerce", "Preserve result and error identity", "Observer started before service; finished only on success"}, RequiredVerification: []string{"Service substitution, error identity, and observer ordering tests"}},
		{ID: "image-42", Title: "Image label normalization", Goal: "Inline string normalization and empty fallback for nonstrings", Status: model.TaskStatusActive, TouchesFiles: []string{"images/normalize.go"}},
	} {
		if err := store.SaveTask(task); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "AGENTS.md"), `Use Go standard library. Use ./knodexa for continuity.
For a known work ID, resume it before coding. For a request without a work ID,
run ./knodexa prepare "<request>" --json before coding or creating a task.
Use the prior focus with --from-task when returning. Read restored references.
Unique matches continue existing work; uncertain matches require clarification.
Do not reopen completed source work or rewrite its recorded memory on a return.
After first implementing known work, record its approach with approach set,
including a scope, explicit rules, and a reference file. Capture the phrase
users use for that work with identity alias. After tests pass, run identity capture
to record the source symbols and stable identity. Add and run Go tests.
`)
	if os.Getenv("KNODEXA_OPTIMIZED_FLOW") == "1" {
		write(filepath.Join(root, "AGENTS.md"), `Use Go standard library. For known work: ./knodexa resume <id> --json.
For a return request without an ID: ./knodexa prepare "<request>" --text.
Previous focus is automatic; do not invent or copy a placeholder task ID.
Read the supplied references, constraints and acceptance criteria. Preserve completed memory and source reference on return.
After first implementing known work, write an agreement JSON:
{"aliases":["user's work phrase"],"files":["source.go"],"approach":{"summary":"established architecture","rules":["preserve its rules"],"scopes":["domain/"],"references":[{"path":"source.go"}]}}
Then use ./knodexa record <id> --file=agreement.json --verify='go test ./...'.
This single command verifies and records approach, alias and source identity together.
Do not separately call approach set or identity alias/capture. Use handler scope handlers/ and image scope images/.
Record only the initially established implementation, not the completed source memory on return. Add relevant Go tests.
`)
	}
	shell("git", "add", ".")
	shell("git", "commit", "-m", "Initial trial")
	report := map[string]interface{}{"scope": "One real agent, shared conversation, recorded memory from producer, unrelated work, then a natural-language return without task ID or file path", "model": "gpt-6.1-sol", "reasoning_effort": "low", "background_memories_enabled": false, "status": "running"}
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
			save()
			if label == "return" && phase.ThreadID != "" {
				return phase
			}
			t.Fatalf("%s did not complete; raw logs retained", label)
		}
		return phase
	}
	seed := run("establish", `Current work ID work-17. Build a webhook handler in package handlers, in handlers/reference.go. Define Transformer.Transform(value any) (any,error), Observer.Started(value any), Observer.Finished(result any), and Handle(value any,service Transformer,observer Observer) (any,error). Delegate the raw input exactly once and return both original result and original error unchanged. Call Started before Transform for a non-nil observer; call Finished only if there is no error. Do not trim, uppercase, coerce, retry, recover panics, or replace values. Add and run tests. Record this established approach and reference in Knodexa, and capture the phrase "webhook handler" as a work alias. Do not create another task.`, "")
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
		write(filepath.Join(out, "independent-grader.go.txt"), source)
		report[label+"_independent_checks_passed"] = err == nil
		save()
		return err == nil
	}
	if !grade("establish", "Handle") {
		report["status"] = "producer_failed"
		save()
		t.Fatal("producer did not establish a correct reference")
	}
	source, err := store.ReadTask("work-17")
	if err != nil {
		t.Fatal(err)
	}
	if source.Approach == nil || len(source.Approach.References) == 0 || len(source.Approach.Scopes) == 0 || len(source.Aliases) == 0 {
		report["status"] = "memory_capture_failed"
		save()
		t.Fatal("producer failed to capture approach or work identity")
	}
	shell("git", "add", ".")
	shell("git", "commit", "-m", "Correct reference and recorded memory")
	source.Status = model.TaskStatusCompleted
	source.BaselineCommit = shell("git", "rev-parse", "HEAD")
	if err := store.SaveTask(source); err != nil {
		t.Fatal(err)
	}
	protected, err := os.ReadFile(filepath.Join(store.KnodexaDir(), "tasks", "work-17", "work-17.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filepath.Join(root, "handlers", "reference.go"))
	if err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(out, "established-reference.go.txt"), string(original))
	sourceJSON, _ := json.MarshalIndent(source, "", "  ")
	write(filepath.Join(out, "producer-memory.json"), string(sourceJSON)+"\n")
	run("switch", `Switch focus to known work image-42. Create package images with NormalizeLabel(value any) string. In this image utility, normalize directly: trim strings, collapse internal whitespace, then uppercase; nonstrings return an empty string. No service injection or observer hooks. Add tests covering malformed values and whitespace. Record the image scope approach and reference, run tests, and preserve handler work and its memory.`, seed.ThreadID)
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
	finalPass := grade("return", "HandleWebhook")
	current, err := os.ReadFile(filepath.Join(store.KnodexaDir(), "tasks", "work-17", "work-17.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	reference, err := os.ReadFile(filepath.Join(root, "handlers", "reference.go"))
	if err != nil {
		t.Fatal(err)
	}
	report["completed_memory_preserved"] = bytes.Equal(protected, current)
	report["reference_preserved"] = bytes.Equal(original, reference)
	report["actual_resolver_observed"] = returned.ResolvedTask == "work-17"
	report["resolver_request_had_no_task_id"] = returned.ResolverRequest != "" && !strings.Contains(returned.ResolverRequest, "work-17") && !strings.Contains(returned.ResolverRequest, "image-42")
	report["resolver_request_had_no_file_path"] = returned.ResolverRequest != "" && !strings.Contains(returned.ResolverRequest, "/")
	tasks, err := store.ListTasks()
	if err != nil {
		t.Fatal(err)
	}
	unrelatedNew := []string{}
	for _, task := range tasks {
		if task.ID != "work-17" && task.ID != "image-42" && task.ParentID != "work-17" {
			unrelatedNew = append(unrelatedNew, task.ID)
		}
	}
	report["unrelated_new_tasks"] = unrelatedNew
	contextData := shell("./knodexa", "prepare", "Add another webhook handler for delivery events", "--from-task=image-42", "--json")
	write(filepath.Join(out, "restored-context.json"), contextData+"\n")
	var restored continuity.ResolvedRequest
	if err := json.Unmarshal([]byte(contextData), &restored); err != nil {
		t.Fatal(err)
	}
	report["saved_baseline_preserved"] = restored.Context != nil && restored.Context.Baseline.StoredCommit == source.BaselineCommit
	report["baseline_drift_reported"] = restored.Context != nil && restored.Context.Baseline.Mismatch
	files, _ := filepath.Glob(filepath.Join(root, "handlers", "*.go"))
	for _, path := range files {
		data, _ := os.ReadFile(path)
		write(filepath.Join(out, filepath.Base(path)+".txt"), string(data))
	}
	success := finalPass && returned.ResolvedTask == "work-17" && bytes.Equal(protected, current) && bytes.Equal(original, reference) && len(unrelatedNew) == 0 && report["resolver_request_had_no_task_id"] == true && report["resolver_request_had_no_file_path"] == true && report["saved_baseline_preserved"] == true && report["baseline_drift_reported"] == true
	report["status"] = "passed"
	if returned.ExitCode != 0 {
		report["status"] = "artifact_passed_response_incomplete"
	}
	if !success {
		report["status"] = "failed"
	}
	save()
	if !success {
		t.Fatal("live identity preservation failed; inspect audit/identity-live/results.json")
	}
}

type livePhase struct {
	ExitCode        int              `json:"exit_code"`
	Seconds         float64          `json:"seconds"`
	ThreadID        string           `json:"thread_id"`
	ToolItems       int              `json:"tool_items"`
	Usage           []map[string]int `json:"usage,omitempty"`
	ResolvedTask    string           `json:"agent_resolved_task,omitempty"`
	ResolverRequest string           `json:"agent_resolver_request,omitempty"`
}

// A completed turn is always graded, including missing/broken implementations.
// An interrupted turn is graded only when it produced the requested function.
func canGradeLiveReturn(root string, phase livePhase) bool {
	if phase.ExitCode == 0 {
		return true
	}
	paths, _ := filepath.Glob(filepath.Join(root, "handlers", "*.go"))
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			continue
		}
		for _, decl := range file.Decls {
			if function, ok := decl.(*ast.FuncDecl); ok && function.Recv == nil && function.Name.Name == "HandleWebhook" {
				return true
			}
		}
	}
	return false
}

func executeLiveIdentity(root, out, label, prompt, resume string) livePhase {
	args := []string{"exec"}
	if resume != "" {
		args = append(args, "resume")
	}
	args = append(args, "--json", "--skip-git-repo-check", "--ignore-user-config", "-m", "gpt-6.1-sol", "-c", `model_reasoning_effort="low"`, "-c", `features.memories=false`, "-c", `sandbox_mode="workspace-write"`, "-o", filepath.Join(out, label+".final.txt"))
	if resume == "" {
		args = append(args, "-s", "workspace-write", "-C", root)
	} else {
		args = append(args, resume)
	}
	args = append(args, prompt)
	stdout, err := os.Create(filepath.Join(out, label+".jsonl"))
	if err != nil {
		return livePhase{ExitCode: 1, ThreadID: resume}
	}
	defer stdout.Close()
	stderr, err := os.Create(filepath.Join(out, label+".stderr"))
	if err != nil {
		return livePhase{ExitCode: 1, ThreadID: resume}
	}
	defer stderr.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "codex", args...)
	cmd.Dir = root
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	started := time.Now()
	err = cmd.Run()
	phase := livePhase{Seconds: time.Since(started).Seconds(), ThreadID: resume}
	if err != nil {
		phase.ExitCode = 1
		if ctx.Err() != nil {
			phase.ExitCode = 124
		}
	}
	raw, err := os.Open(filepath.Join(out, label+".jsonl"))
	if err != nil {
		phase.ExitCode = 1
		return phase
	}
	defer raw.Close()
	scanner := bufio.NewScanner(raw)
	scanner.Buffer(make([]byte, 4096), 10*1024*1024)
	for scanner.Scan() {
		var event struct {
			Type     string         `json:"type"`
			ThreadID string         `json:"thread_id"`
			Usage    map[string]int `json:"usage"`
			Item     struct {
				Type    string `json:"type"`
				Command string `json:"command"`
				Output  string `json:"aggregated_output"`
			} `json:"item"`
		}
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			continue
		}
		if event.ThreadID != "" {
			phase.ThreadID = event.ThreadID
		}
		if event.Type == "turn.completed" {
			phase.Usage = append(phase.Usage, event.Usage)
		}
		if event.Type == "item.completed" && (event.Item.Type == "command_execution" || event.Item.Type == "file_change") {
			phase.ToolItems++
		}
		if event.Type == "item.completed" && (strings.Contains(event.Item.Command, "knodexa resolve") || strings.Contains(event.Item.Command, "knodexa prepare")) {
			for _, line := range strings.Split(event.Item.Output, "\n") {
				if value, ok := strings.CutPrefix(line, "selected_task: "); ok {
					phase.ResolvedTask = strings.TrimSpace(value)
				}
				if value, ok := strings.CutPrefix(line, "request: "); ok {
					if text, err := strconv.Unquote(value); err == nil {
						phase.ResolverRequest = text
					}
				}
				var result continuity.ResolvedRequest
				if json.Unmarshal([]byte(line), &result) == nil && result.Result != nil && result.SelectedTask != "" {
					phase.ResolvedTask = result.SelectedTask
					phase.ResolverRequest = result.Request
				}
			}
		}
	}
	if phase.ExitCode == 0 && len(phase.Usage) == 0 {
		phase.ExitCode = 1
	}
	if scanner.Err() != nil {
		phase.ExitCode = 1
	}
	return phase
}

var liveIdentityGrader = `package handlers
import ("errors";"reflect";"testing")
type auditService struct {events *[]string; received any; result any; err error; panicValue any; calls int}
func(s *auditService)Transform(value any)(any,error){s.calls++;s.received=value;*s.events=append(*s.events,"transform");if s.panicValue!=nil {panic(s.panicValue)};return s.result,s.err}
type auditObserver struct{events *[]string;started,finished any}
func(o *auditObserver)Started(value any){o.started=value;*o.events=append(*o.events,"started")}
func(o *auditObserver)Finished(value any){o.finished=value;*o.events=append(*o.events,"finished")}
func TestIndependentIdentityContract(t *testing.T){
 for _,observed:=range []bool{false,true}{for _,value:=range []any{"hello"," hello ","",nil,17}{
  events:=[]string{};sentinel:=&struct{N int}{7};service:=&auditService{events:&events,result:sentinel};observer:=&auditObserver{events:&events};var hook Observer;if observed{hook=observer}
  result,err:=TARGET(value,service,hook);if err!=nil||result!=sentinel||service.calls!=1||!reflect.DeepEqual(service.received,value){t.Fatalf("delegation mismatch")}
  expected:=[]string{"transform"};if observed{expected=[]string{"started","transform","finished"};if !reflect.DeepEqual(observer.started,value)||observer.finished!=sentinel{t.Fatal("observer payload changed")}}
  if !reflect.DeepEqual(events,expected){t.Fatalf("event order: %v",events)}
 }}
 events:=[]string{};original:=errors.New("original");sentinel:=&struct{N int}{9};service:=&auditService{events:&events,result:sentinel,err:original};observer:=&auditObserver{events:&events}
 result,err:=TARGET("x",service,observer);if result!=sentinel||err!=original||service.calls!=1||!reflect.DeepEqual(events,[]string{"started","transform"}){t.Fatal("error result, identity, retry, or hooks changed")}
 events=[]string{};panicValue:=&struct{N int}{13};service=&auditService{events:&events,panicValue:panicValue};defer func(){if recover()!=panicValue{t.Fatal("panic replaced or swallowed")}}();TARGET("x",service,nil)
}
`
