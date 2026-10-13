package cli

import (
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
