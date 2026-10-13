package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitInspector(t *testing.T) {
	tmpDir := t.TempDir()

	inspector := NewInspector(tmpDir)
	if inspector.IsRepo() {
		t.Errorf("empty dir should not be a git repo")
	}

	// Initialize git repo in tmpDir
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git init: %v", err)
	}

	// Config user for commits in test
	_ = exec.Command("git", "-C", tmpDir, "config", "user.name", "Tester").Run()
	_ = exec.Command("git", "-C", tmpDir, "config", "user.email", "test@example.com").Run()

	if !inspector.IsRepo() {
		t.Errorf("expected isRepo to be true after git init")
	}

	branch, err := inspector.CurrentBranch()
	if err != nil {
		t.Fatalf("current branch error: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected branch 'main', got %q", branch)
	}

	// Create a file and commit
	testFile := filepath.Join(tmpDir, "test.txt")
	_ = os.WriteFile(testFile, []byte("content"), 0644)

	dirty, _ := inspector.IsDirty()
	if !dirty {
		t.Errorf("expected isDirty to be true with untracked file")
	}

	_ = exec.Command("git", "-C", tmpDir, "add", "test.txt").Run()
	_ = exec.Command("git", "-C", tmpDir, "commit", "-m", "initial commit").Run()

	dirtyAfterCommit, _ := inspector.IsDirty()
	if dirtyAfterCommit {
		t.Errorf("expected clean working tree after commit")
	}

	head, err := inspector.HeadCommit()
	if err != nil {
		t.Fatalf("head commit error: %v", err)
	}
	if len(head) < 7 {
		t.Errorf("expected valid commit hash, got %q", head)
	}

	chk, err := inspector.GetCheckpoint()
	if err != nil {
		t.Fatalf("get checkpoint error: %v", err)
	}
	if chk.CommitSHA != head || chk.Branch != "main" || chk.IsDirty {
		t.Errorf("checkpoint mismatch: %+v", chk)
	}
}
