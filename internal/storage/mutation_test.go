package storage

import (
	"errors"
	"knodexa/internal/model"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestTaskWritesRejectStaleSnapshotsAndAtomicChangesPreserveMetadata(t *testing.T) {
	root := t.TempDir()
	store := NewStorage(root)
	if _, err := store.Init("mutations", ""); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTask(&model.TaskData{ID: "task", Status: model.TaskStatusCompleted, BaselineCommit: "historical", Aliases: []string{"original"}}); err != nil {
		t.Fatal(err)
	}
	a, err := store.ReadTask("task")
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewStorage(root).ReadTask("task")
	if err != nil {
		t.Fatal(err)
	}
	a.Aliases = append(a.Aliases, "first")
	if err := store.SaveTask(a); err != nil {
		t.Fatal(err)
	}
	b.Aliases = append(b.Aliases, "stale")
	if err := NewStorage(root).SaveTask(b); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale write not rejected: %v", err)
	}
	if err := store.SaveTask(&model.TaskData{ID: "task", Status: model.TaskStatusActive}); !errors.Is(err, ErrConflict) {
		t.Fatal("blind replacement accepted")
	}
	var wg sync.WaitGroup
	for _, alias := range []string{"second", "third", "fourth"} {
		wg.Add(1)
		go func(alias string) {
			defer wg.Done()
			if err := NewStorage(root).UpdateTask("task", func(task *model.TaskData) error { task.Aliases = append(task.Aliases, alias); return nil }); err != nil {
				t.Error(err)
			}
		}(alias)
	}
	wg.Wait()
	final, err := store.ReadTask("task")
	if err != nil {
		t.Fatal(err)
	}
	if len(final.Aliases) != 5 || final.Status != model.TaskStatusCompleted || final.BaselineCommit != "historical" {
		t.Fatalf("metadata lost: %+v", final)
	}
	before, err := os.ReadFile(final.SnapshotPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateTask("task", func(task *model.TaskData) error { return nil }); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(final.SnapshotPath)
	if string(before) != string(after) {
		t.Fatal("no-op rewrote memory")
	}
	if err := store.UpdateTask("task", func(task *model.TaskData) error { task.Aliases = nil; return errors.New("reject proposed change") }); err == nil {
		t.Fatal("failed callback accepted")
	}
	after, _ = os.ReadFile(final.SnapshotPath)
	if string(before) != string(after) {
		t.Fatal("failed callback wrote partial data")
	}
}

func TestProcessLockIsReleasedAfterHolderCrashes(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "memory.lock")
	ready := filepath.Join(root, "ready")
	cmd := exec.Command(os.Args[0], "-test.run=^TestProcessLockCrashHelper$")
	cmd.Env = append(os.Environ(), "KNODEXA_CRASH_LOCK="+path, "KNODEXA_CRASH_READY="+ready)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("holder did not acquire lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	if err := withProcessLock(path, time.Second, func() error { return nil }); err != nil {
		t.Fatal("crash left lock held", err)
	}
}

func TestProcessLockCrashHelper(t *testing.T) {
	path := os.Getenv("KNODEXA_CRASH_LOCK")
	if path == "" {
		t.Skip("process helper")
	}
	if err := WithProcessLock(path, func() error {
		if err := os.WriteFile(os.Getenv("KNODEXA_CRASH_READY"), []byte("locked"), 0644); err != nil {
			return err
		}
		for {
			time.Sleep(100 * time.Millisecond)
		}
	}); err != nil {
		t.Fatal(err)
	}
}

func TestProcessLockWaitIsBoundedAndReleasedAfterFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.lock")
	if err := WithProcessLock(path, func() error {
		started := time.Now()
		if err := withProcessLock(path, 30*time.Millisecond, func() error { t.Fatal("conflicting lock acquired"); return nil }); err == nil {
			t.Fatal("contended lock did not time out")
		}
		if time.Since(started) > time.Second {
			t.Fatal("lock wait was unbounded")
		}
		return errors.New("callback failed")
	}); err == nil {
		t.Fatal("callback error hidden")
	}
	if err := WithProcessLock(path, func() error { return nil }); err != nil {
		t.Fatal("failure left lock held", err)
	}
}
