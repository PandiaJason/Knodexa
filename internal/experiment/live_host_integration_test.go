package experiment

import (
	"bufio"
	"context"
	"encoding/json"
	"knodexa/internal/approach"
	"knodexa/internal/index"
	"knodexa/internal/integration"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Actual host discovery, with no Knodexa invocation in the coding request.
// Three predeclared fresh ephemeral Codex contexts; no selective model reruns.
func TestLiveHostIntegration(t *testing.T) {
	if os.Getenv("KNODEXA_LIVE_HOST") != "1" {
		t.Skip("set KNODEXA_LIVE_HOST=1; consumes authenticated agent usage")
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(repo, "audit", "host-and-concurrency-v1", "host")
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "results.json")); err == nil {
		t.Fatal("existing audit retained; use a separate output directory for another run")
	}
	tools := t.TempDir()
	binary := filepath.Join(tools, "knodexa")
	build := exec.Command("go", "build", "-o", binary, "./cmd/knodexa")
	build.Dir = repo
	if data, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, data)
	}
	results := []map[string]interface{}{}
	plan := map[string]interface{}{"cases": []string{"agents-md", "skill-discovery-path", "skill-distribution-path"}, "model": "gpt-6.1-sol", "reasoning_effort": "low", "timeout_seconds": 180, "prompt": "Add ForwardWebhook(value any, service Transformer) (any, error) in handlers/webhook.go for production webhook delivery. Add relevant tests and verify the implementation.", "scope": "Fresh ephemeral Codex CLI, AGENTS.md versus implicit skill discovery versus distribution-path negative control. No direct Knodexa or skill invocation in prompt; not a quality comparison or all-host test."}
	saveJSON := func(name string, value interface{}) {
		t.Helper()
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(out, name), append(data, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	saveJSON("plan.json", plan)
	for _, mode := range []string{"agents-md", "skill-discovery-path", "skill-distribution-path"} {
		root := t.TempDir()
		write := func(rel, body string) {
			t.Helper()
			path := filepath.Join(root, rel)
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(body), 0644); err != nil {
				t.Fatal(err)
			}
		}
		write("go.mod", "module hostsample\n\ngo 1.26.0\n")
		write("handlers/reference.go", "package handlers\n\ntype Transformer interface { Transform(any) (any,error) }\nfunc Forward(value any, service Transformer) (any,error) { return service.Transform(value) }\n")
		write(".gitignore", "bin/knodexa\n.knodexa/cache.db*\n.knodexa/focus.json\n")
		data, err := os.ReadFile(binary)
		if err != nil {
			t.Fatal(err)
		}
		os.MkdirAll(filepath.Join(root, "bin"), 0755)
		if err := os.WriteFile(filepath.Join(root, "bin", "knodexa"), data, 0755); err != nil {
			t.Fatal(err)
		}
		store := storage.NewStorage(root)
		if _, err := store.Init("Host discovery audit", ""); err != nil {
			t.Fatal(err)
		}
		if err := store.SaveTask(&model.TaskData{ID: "production", Title: "Production webhook delivery", Aliases: []string{"production webhook delivery"}, Status: model.TaskStatusCompleted, Constraints: []string{"Delegate raw input exactly once; preserve result and error identity"}, RequiredVerification: []string{"Service substitution and raw-value/error identity"}, TouchesFiles: []string{"handlers/reference.go"}}); err != nil {
			t.Fatal(err)
		}
		db, err := index.Open(filepath.Join(store.KnodexaDir(), "cache.db"))
		if err != nil {
			t.Fatal(err)
		}
		idx := index.NewIndexer(db, store)
		if err := approach.Set(store, db, idx, "production", model.TaskApproach{Summary: "Thin service delegation", Rules: []string{"Do not normalize, coerce or replace raw values"}, Scopes: []string{"handlers/"}, References: []model.ApproachReference{{Path: "handlers/reference.go"}}, Checks: []model.ApproachCheck{{Path: "@touched", ASTKind: "forbid-import", Symbol: "strings"}}}); err != nil {
			t.Fatal(err)
		}
		db.Close()
		if mode == "agents-md" {
			if _, err := integration.Install(root, "agents", ""); err != nil {
				t.Fatal(err)
			}
		} else {
			rel := ""
			if mode == "skill-discovery-path" {
				rel = ".agents/skills/knodexa-continuity/SKILL.md"
			}
			if _, err := integration.Install(root, "skill", rel); err != nil {
				t.Fatal(err)
			}
		}
		for _, args := range [][]string{{"git", "init", "-b", "main"}, {"git", "config", "user.name", "Host audit"}, {"git", "config", "user.email", "audit@example.test"}, {"git", "add", "."}, {"git", "commit", "-m", "Host discovery fixture"}} {
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Dir = root
			if data, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("setup: %v %s", err, data)
			}
		}
		parentPath := filepath.Join(store.KnodexaDir(), "tasks", "production", "production.yaml")
		before, err := os.ReadFile(parentPath)
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(out, mode+"-memory-before.yaml"), before, 0644)
		args := []string{"exec", "--ephemeral", "--json", "--skip-git-repo-check", "--ignore-user-config", "-m", "gpt-6.1-sol", "-c", `model_reasoning_effort="low"`, "-c", `features.memories=false`, "-c", `shell_environment_policy.inherit="all"`, "-s", "workspace-write", "-C", root, "-o", filepath.Join(out, mode+".final.txt"), plan["prompt"].(string)}
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		cmd := exec.CommandContext(ctx, "codex", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "PATH="+tools+string(os.PathListSeparator)+os.Getenv("PATH"))
		stdout, err := os.Create(filepath.Join(out, mode+".jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		stderr, err := os.Create(filepath.Join(out, mode+".stderr"))
		if err != nil {
			t.Fatal(err)
		}
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		started := time.Now()
		runErr := cmd.Run()
		seconds := time.Since(started).Seconds()
		timedOut := ctx.Err() != nil
		cancel()
		stdout.Close()
		stderr.Close()
		trace := readHostTrace(t, filepath.Join(out, mode+".jsonl"))
		write("handlers/host_independent_test.go", hostIntegrationGrader)
		grade := exec.Command("go", "test", "./...")
		grade.Dir = root
		graded, gradeErr := grade.CombinedOutput()
		os.WriteFile(filepath.Join(out, mode+"-independent-check.log"), graded, 0644)
		after, _ := os.ReadFile(parentPath)
		os.WriteFile(filepath.Join(out, mode+"-memory-after.yaml"), after, 0644)
		source, sourceErr := os.ReadFile(filepath.Join(root, "handlers", "webhook.go"))
		if sourceErr == nil {
			os.WriteFile(filepath.Join(out, mode+"-webhook.go.txt"), source, 0644)
		}
		scored := runErr == nil && trace.Completed
		result := map[string]interface{}{"case": mode, "scored": scored, "seconds": seconds, "timeout": timedOut, "completed": trace.Completed, "commands": trace.Commands, "used_knodexa": trace.UsedKnodexa, "restored_before_edit": trace.RestoredBeforeEdit, "checked_agreement": trace.CheckedAgreement, "read_skill": trace.ReadSkill, "independent_behavior_passed": gradeErr == nil, "completed_parent_unchanged": string(before) == string(after), "input_tokens": trace.InputTokens, "output_tokens": trace.OutputTokens, "thread_id": trace.ThreadID, "ephemeral": true, "integration_pass": scored && trace.RestoredBeforeEdit && trace.CheckedAgreement && gradeErr == nil && string(before) == string(after)}
		if runErr != nil {
			result["execution_error"] = runErr.Error()
		}
		results = append(results, result)
		saveJSON("results.json", map[string]interface{}{"plan": plan, "results": results})
		t.Logf("%s: scored=%v restored=%v checked=%v behavior=%v parent=%v seconds=%.1f", mode, scored, trace.RestoredBeforeEdit, trace.CheckedAgreement, gradeErr == nil, string(before) == string(after), seconds)
	}
}

type hostTrace struct {
	Completed, UsedKnodexa, RestoredBeforeEdit, CheckedAgreement, ReadSkill bool
	Commands, InputTokens, OutputTokens                                     int
	ThreadID                                                                string
}

func readHostTrace(t *testing.T, path string) hostTrace {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 10*1024*1024)
	trace := hostTrace{}
	edited := false
	for scanner.Scan() {
		var event struct {
			Type     string `json:"type"`
			ThreadID string `json:"thread_id"`
			Usage    struct {
				Input  int `json:"input_tokens"`
				Output int `json:"output_tokens"`
			} `json:"usage"`
			Item struct {
				Type     string `json:"type"`
				Command  string `json:"command"`
				Status   string `json:"status"`
				ExitCode int    `json:"exit_code"`
				Output   string `json:"aggregated_output"`
				Changes  []struct {
					Path string `json:"path"`
				} `json:"changes"`
			} `json:"item"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		if event.ThreadID != "" {
			trace.ThreadID = event.ThreadID
		}
		if event.Type == "turn.completed" {
			trace.Completed = true
			trace.InputTokens += event.Usage.Input
			trace.OutputTokens += event.Usage.Output
		}
		if event.Type == "item.completed" && event.Item.Type == "file_change" {
			for _, change := range event.Item.Changes {
				if strings.HasSuffix(change.Path, ".go") {
					edited = true
				}
			}
		}
		if event.Type != "item.completed" || event.Item.Type != "command_execution" {
			continue
		}
		trace.Commands++
		command := event.Item.Command
		if strings.Contains(command, "SKILL.md") && event.Item.ExitCode == 0 {
			trace.ReadSkill = true
		}
		if strings.Contains(command, "knodexa") {
			trace.UsedKnodexa = true
			resolved := strings.Contains(event.Item.Output, "selected_task: production") || strings.Contains(event.Item.Output, `"selected_task":"production"`) || strings.Contains(event.Item.Output, `"selected_task": "production"`) || strings.Contains(event.Item.Output, `"task":{"id":"production"`)
			if (strings.Contains(command, " prepare ") || strings.Contains(command, " resume ")) && event.Item.ExitCode == 0 && !edited && resolved {
				trace.RestoredBeforeEdit = true
			}
			if (strings.Contains(command, "approach check") || strings.Contains(command, " record ")) && event.Item.ExitCode == 0 {
				trace.CheckedAgreement = true
			}
		}
		if strings.Contains(command, "webhook.go") && (strings.Contains(command, "> ") || strings.Contains(command, "write") || strings.Contains(command, "tee ")) {
			edited = true
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return trace
}

const hostIntegrationGrader = `package handlers
import ("errors";"testing")
type hostAuditService struct { calls int; input any; result any; err error }
func(s *hostAuditService) Transform(v any)(any,error){s.calls++;s.input=v;return s.result,s.err}
func TestHostAuditRawDelegation(t *testing.T){
 value:=&struct{Text string}{" raw "};result:=&struct{ID int}{17};failure:=errors.New("sentinel")
 for _,input:=range []any{value," raw ",42,nil}{for _,failure:=range []error{nil,failure}{s:=&hostAuditService{result:result,err:failure};got,err:=ForwardWebhook(input,s);if s.calls!=1||s.input!=input||got!=result||err!=failure{t.Fatalf("raw delegation changed: %+v %v %v",s,got,err)}}}
}
`
