package cli

import (
	"encoding/json"
	"io"
	"knodexa/internal/continuity"
	"os"
	"testing"
)

func TestResolveCLIAndNewTaskGuard(t *testing.T) {
	app, err := NewApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"init", "--name=Identity"}); err != nil {
		t.Fatal(err)
	}
	app, err = NewApp(app.rootDir)
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	if err := app.Run([]string{"task", "create", "--id=recorded-work", "--title=Inbound event delivery", "--alias=webhook callbacks"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"identity", "alias", "recorded-work", "--alias=event notification"}); err != nil {
		t.Fatal(err)
	}
	// CLI accepts the request itself and flags after it, without a task ID.
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := app.Run([]string{"resolve", "Extend webhook callbacks", "--json"})
	os.Stdout = old
	w.Close()
	data, readErr := io.ReadAll(r)
	r.Close()
	if runErr != nil {
		t.Fatal(runErr)
	}
	if readErr != nil {
		t.Fatal(readErr)
	}
	var result continuity.ResolvedRequest
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if result.SelectedTask != "recorded-work" || result.Context == nil {
		t.Fatalf("request failed to restore work: %s", data)
	}
	if err := app.Run([]string{"task", "create", "--id=accidental-new", "--title=Extend webhook callbacks"}); err == nil {
		t.Fatal("silent duplicate work created")
	}
	tasks, err := app.store.ListTasks()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatal("blocked creation wrote a task")
	}
	if err := app.Run([]string{"task", "create", "--id=continuation", "--parent=recorded-work", "--title=Extend webhook callbacks"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"task", "create", "--id=independent", "--new", "--title=Extend webhook callbacks"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"task", "create", "--id=recorded-work", "--new", "--title=Overwrite existing record"}); err == nil {
		t.Fatal("existing task overwritten")
	}
	for _, args := range [][]string{{"resolve", "--json"}, {"resolve", "something", "--budget=0"}, {"resolve", "something", "--unknown"}, {"identity", "alias", "recorded-work", "--alias=another task"}} {
		if err := app.Run(args); err == nil {
			t.Fatalf("invalid arguments accepted: %v", args)
		}
	}
}
