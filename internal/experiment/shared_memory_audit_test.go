package experiment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// Audit concurrent processes separately from Go's in-process race detector.
// This opt-in audit reports unsafe outcomes rather than treating bug reproduction
// as a passing safety check. Canonical snapshots and every worker output remain.
func TestSharedMemoryAudit(t *testing.T) {
	if os.Getenv("KNODEXA_SHARED_MEMORY_AUDIT") != "1" {
		t.Skip("set KNODEXA_SHARED_MEMORY_AUDIT=1")
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(repo, "audit", "host-and-concurrency-v1", "concurrency")
	if override := os.Getenv("KNODEXA_SHARED_MEMORY_AUDIT_OUT"); override != "" {
		if filepath.IsAbs(override) {
			out = override
		} else {
			out = filepath.Join(repo, override)
		}
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "results.json")); err == nil {
		t.Fatal("audit output exists; preserve it before a new run")
	}
	bin := filepath.Join(t.TempDir(), "knodexa")
	build := exec.Command("go", "build", "-o", bin, "./cmd/knodexa")
	build.Dir = repo
	if data, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, data)
	}
	results := []map[string]interface{}{}
	for _, mode := range []string{"same-task-snapshot", "same-task-atomic-updates", "different-task-snapshots", "public-alias-commands"} {
		root := t.TempDir()
		store := storage.NewStorage(root)
		if _, err := store.Init("Concurrent memory audit", ""); err != nil {
			t.Fatal(err)
		}
		initial := &model.TaskData{ID: "shared", Status: model.TaskStatusCompleted, BaselineCommit: "unchanged-baseline", Aliases: []string{"original"}, Constraints: []string{"Preserve raw input"}, Approach: &model.TaskApproach{Version: 1, Summary: "Established", Rules: []string{"Keep original design"}}}
		if err := store.SaveTask(initial); err != nil {
			t.Fatal(err)
		}
		before, _ := json.MarshalIndent(initial, "", "  ")
		os.WriteFile(filepath.Join(out, mode+"-before.json"), before, 0644)
		const workers = 8
		commands := make([]*exec.Cmd, workers)
		outputs := make([]*strings.Builder, workers)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		for i := 0; i < workers; i++ {
			if mode == "public-alias-commands" {
				commands[i] = exec.CommandContext(ctx, bin, "identity", "alias", "shared", "--alias=worker-"+strconv.Itoa(i))
			} else {
				commands[i] = exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSharedMemoryProcessHelper$")
			}
			commands[i].Dir = root
			commands[i].Env = append(os.Environ(), "KNODEXA_AUDIT_ROOT="+root, "KNODEXA_AUDIT_WORKER="+strconv.Itoa(i), "KNODEXA_AUDIT_MODE="+mode)
			outputs[i] = &strings.Builder{}
			commands[i].Stdout = outputs[i]
			commands[i].Stderr = outputs[i]
			if err := commands[i].Start(); err != nil {
				cancel()
				t.Fatal(err)
			}
		}
		if mode != "public-alias-commands" {
			deadline := time.Now().Add(15 * time.Second)
			for {
				ready := 0
				for i := 0; i < workers; i++ {
					if _, err := os.Stat(filepath.Join(root, fmt.Sprintf("ready-%d", i))); err == nil {
						ready++
					}
				}
				if ready == workers {
					break
				}
				if time.Now().After(deadline) {
					cancel()
					t.Fatal("workers did not reach synchronized snapshot barrier")
				}
				time.Sleep(10 * time.Millisecond)
			}
			if err := os.WriteFile(filepath.Join(root, "release"), []byte("start"), 0644); err != nil {
				cancel()
				t.Fatal(err)
			}
		}
		success := 0
		staleRejections := 0
		errors := []string{}
		for i, cmd := range commands {
			if err := cmd.Wait(); err != nil {
				errors = append(errors, fmt.Sprintf("worker %d: %v", i, err))
				if strings.Contains(outputs[i].String(), "STALE_WRITE_REJECTED") {
					staleRejections++
				}
			} else {
				success++
			}
			os.WriteFile(filepath.Join(out, fmt.Sprintf("%s-worker-%d.log", mode, i)), []byte(outputs[i].String()), 0644)
		}
		cancel()
		final, err := store.ReadTask("shared")
		if err != nil {
			t.Fatal(err)
		}
		preserved := final.Status == initial.Status && final.BaselineCommit == initial.BaselineCommit && len(final.Constraints) == 1 && final.Approach != nil && final.Approach.Version == 1
		retained := 0
		if mode == "different-task-snapshots" {
			for i := 0; i < workers; i++ {
				task, err := store.ReadTask("worker-" + strconv.Itoa(i))
				if err == nil && len(task.Aliases) == 2 {
					retained++
				}
			}
		} else {
			for _, alias := range final.Aliases {
				if strings.HasPrefix(alias, "worker-") {
					retained++
				}
			}
		}
		data, _ := json.MarshalIndent(final, "", "  ")
		os.WriteFile(filepath.Join(out, mode+"-after.json"), data, 0644)
		safe := success == workers && retained == workers && preserved
		if mode == "same-task-snapshot" {
			safe = success+staleRejections == workers && retained == success && preserved
		}
		result := map[string]interface{}{"case": mode, "workers": workers, "successful_workers": success, "stale_rejections": staleRejections, "retained_updates": retained, "errors": errors, "baseline_and_agreement_preserved": preserved, "safe": safe}
		results = append(results, result)
		t.Logf("%s: completed=%d/%d retained=%d/%d safe=%v", mode, success, workers, retained, workers, result["safe"])
	}
	data, _ := json.MarshalIndent(map[string]interface{}{"scope": "Eight independent OS processes; same-task synchronized snapshots, disjoint writes and public CLI alias updates", "results": results}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "results.json"), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestSharedMemoryProcessHelper(t *testing.T) {
	root := os.Getenv("KNODEXA_AUDIT_ROOT")
	if root == "" {
		t.Skip("process helper")
	}
	worker := os.Getenv("KNODEXA_AUDIT_WORKER")
	store := storage.NewStorage(root)
	task, err := store.ReadTask("shared")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ready-"+worker), []byte("snapshot read"), 0644); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(20 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(root, "release")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("snapshot barrier timeout")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if os.Getenv("KNODEXA_AUDIT_MODE") == "different-task-snapshots" {
		task.ID = "worker-" + worker
	}
	if os.Getenv("KNODEXA_AUDIT_MODE") == "same-task-atomic-updates" {
		if err := store.UpdateTask("shared", func(current *model.TaskData) error {
			current.Aliases = append(current.Aliases, "worker-"+worker)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return
	}
	task.Aliases = append(task.Aliases, "worker-"+worker)
	if err := store.SaveTask(task); err != nil {
		if errors.Is(err, storage.ErrConflict) {
			t.Log("STALE_WRITE_REJECTED")
		}
		t.Fatal(err)
	}
}
