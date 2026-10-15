package continuity

import (
	"errors"
	"os/exec"
	"testing"
	"time"
)

func TestCommandBoundsInheritedOutputPipes(t *testing.T) {
	start := time.Now()
	outcome, err := executeCommand(t.TempDir(), []string{"/bin/sh", "-c", "sleep 3 & exit 0"}, nil, time.Second)
	if !errors.Is(err, exec.ErrWaitDelay) || outcome.ExitCode == 0 {
		t.Fatalf("unclosed pipes must fail verification: %+v, %v", outcome, err)
	}
	if time.Since(start) >= 2*time.Second {
		t.Fatal("waited for background descendant instead of bounding pipe drain")
	}
}
