package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Inspector provides read-only Git operations to anchor knowledge items to commit states.
type Inspector struct {
	rootDir string
	timeout time.Duration
}

// Checkpoint represents repository state at a specific snapshot in time.
type Checkpoint struct {
	CommitSHA   string   `json:"commit_sha" yaml:"commit_sha"`
	Branch      string   `json:"branch" yaml:"branch"`
	IsDirty     bool     `json:"is_dirty" yaml:"is_dirty"`
	ChangedFiles []string `json:"changed_files,omitempty" yaml:"changed_files,omitempty"`
}

// NewInspector constructs a Git Inspector for the specified repository root.
func NewInspector(rootDir string) *Inspector {
	return &Inspector{
		rootDir: rootDir,
		timeout: 5 * time.Second,
	}
}

// IsRepo checks if rootDir is inside a valid Git repository.
func (gi *Inspector) IsRepo() bool {
	ctx, cancel := context.WithTimeout(context.Background(), gi.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = gi.rootDir
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// HeadCommit returns the current HEAD commit hash (or empty string if repository has no commits yet).
func (gi *Inspector) HeadCommit() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gi.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = gi.rootDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "unknown revision") || strings.Contains(string(out), "ambiguous argument 'HEAD'") {
			return "", nil // Empty repo
		}
		return "", fmt.Errorf("git rev-parse HEAD failed: %s", string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

// CurrentBranch returns the active git branch name (handling unborn branches as well).
func (gi *Inspector) CurrentBranch() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gi.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	cmd.Dir = gi.rootDir
	out, err := cmd.CombinedOutput()
	if err == nil && len(bytes.TrimSpace(out)) > 0 {
		return strings.TrimSpace(string(out)), nil
	}

	cmd2 := exec.CommandContext(ctx, "git", "symbolic-ref", "--short", "HEAD")
	cmd2.Dir = gi.rootDir
	out2, err2 := cmd2.CombinedOutput()
	if err2 == nil && len(bytes.TrimSpace(out2)) > 0 {
		return strings.TrimSpace(string(out2)), nil
	}

	return "unknown", nil
}

// IsDirty returns true if there are uncommitted working tree changes.
func (gi *Inspector) IsDirty() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gi.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = gi.rootDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("git status failed: %s", string(out))
	}
	return len(bytes.TrimSpace(out)) > 0, nil
}

// GetCheckpoint returns the complete snapshot of the current Git state.
func (gi *Inspector) GetCheckpoint() (*Checkpoint, error) {
	if !gi.IsRepo() {
		return &Checkpoint{
			CommitSHA: "none",
			Branch:    "none",
			IsDirty:   false,
		}, nil
	}

	commit, err := gi.HeadCommit()
	if err != nil {
		return nil, err
	}
	branch, _ := gi.CurrentBranch()
	dirty, _ := gi.IsDirty()

	return &Checkpoint{
		CommitSHA: commit,
		Branch:    branch,
		IsDirty:   dirty,
	}, nil
}
