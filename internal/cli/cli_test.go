package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIWorkflows(t *testing.T) {
	tmpDir := t.TempDir()

	app, err := NewApp(tmpDir)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// 1. init
	if err := app.Run([]string{"init", "--name=CLI Project", "--desc=Testing CLI"}); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Recreate app to connect to newly initialized store and DB
	app, err = NewApp(tmpDir)
	if err != nil {
		t.Fatalf("failed to reload app: %v", err)
	}

	// 2. task create
	if err := app.Run([]string{"task", "create", "--id=T1", "--title=First Task", "--goal=Build engine"}); err != nil {
		t.Fatalf("task create failed: %v", err)
	}

	// 3. subtask create
	if err := app.Run([]string{"task", "create", "--id=T1.1", "--parent=T1", "--title=Subtask 1", "--goal=Write unit tests"}); err != nil {
		t.Fatalf("subtask create failed: %v", err)
	}

	// 4. task show
	if err := app.Run([]string{"task", "show", "T1.1"}); err != nil {
		t.Fatalf("task show failed: %v", err)
	}

	// 5. task list
	if err := app.Run([]string{"task", "list"}); err != nil {
		t.Fatalf("task list failed: %v", err)
	}

	// 6. context
	if err := app.Run([]string{"context", "T1.1"}); err != nil {
		t.Fatalf("context failed: %v", err)
	}

	// 7. context --json
	if err := app.Run([]string{"context", "T1.1", "--json"}); err != nil {
		t.Fatalf("context --json failed: %v", err)
	}

	// 8. tree
	if err := app.Run([]string{"tree"}); err != nil {
		t.Fatalf("tree failed: %v", err)
	}

	// 9. decisions & lessons (empty)
	if err := app.Run([]string{"decisions"}); err != nil {
		t.Fatalf("decisions failed: %v", err)
	}
	if err := app.Run([]string{"lessons"}); err != nil {
		t.Fatalf("lessons failed: %v", err)
	}

	// 10. doctor
	if err := app.Run([]string{"doctor"}); err != nil {
		t.Fatalf("doctor failed: %v", err)
	}

	// 11. doctor --rebuild
	if err := app.Run([]string{"doctor", "--rebuild"}); err != nil {
		t.Fatalf("doctor --rebuild failed: %v", err)
	}
}

func TestContextFlagOutput(t *testing.T) {
	app, err := NewApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"init", "--name=Flags"}); err != nil {
		t.Fatal(err)
	}
	app, err = NewApp(app.rootDir)
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	if err := app.Run([]string{"task", "create", "--id=T1", "--title=Flags", "--goal=Check output"}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"context", "T1", "--json"},
		{"context", "--json", "T1"},
		{"context", "T1", "--budget", "500", "--json=true"},
		{"context", "--budget=500", "T1", "--json"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			old := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			os.Stdout = w
			runErr := app.Run(args)
			os.Stdout = old
			w.Close()
			output, readErr := io.ReadAll(r)
			r.Close()
			if runErr != nil {
				t.Fatal(runErr)
			}
			if readErr != nil {
				t.Fatal(readErr)
			}
			var result struct {
				Task struct {
					ID string `json:"id"`
				} `json:"task"`
			}
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatalf("invalid JSON: %v; output: %s", err, output)
			}
			if result.Task.ID != "T1" {
				t.Fatalf("wrong task: %s", result.Task.ID)
			}
		})
	}
	for _, args := range [][]string{
		{"context", "T1", "--unknown"},
		{"context", "T1", "extra"},
		{"context", "T1", "--budget"},
		{"context", "T1", "--budget=0"},
		{"context", "T1", "--budget=-1"},
		{"context", "T1", "--json=invalid"},
	} {
		if err := app.Run(args); err == nil {
			t.Errorf("expected rejection for %v", args)
		}
	}
}

func TestNewAppReportsIndexOpenFailure(t *testing.T) {
	app, err := NewApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"init", "--name=Broken index"}); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(app.store.KnodexaDir(), "cache.db")
	if err := os.Remove(cache); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.Mkdir(cache, 0755); err != nil {
		t.Fatal(err)
	}
	_, err = NewApp(app.rootDir)
	if err == nil || !strings.Contains(err.Error(), "open knowledge index") {
		t.Fatalf("expected actionable index error, got %v", err)
	}
}
