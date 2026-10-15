package experiment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInterruptedReturnGradingRequiresActualArtifact(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "handlers"), 0755); err != nil {
		t.Fatal(err)
	}
	if canGradeLiveReturn(root, livePhase{ExitCode: 124}) {
		t.Fatal("empty timeout became a quality failure")
	}
	if !canGradeLiveReturn(root, livePhase{ExitCode: 0}) {
		t.Fatal("completed missing implementation escaped grading")
	}
	path := filepath.Join(root, "handlers", "webhook.go")
	if err := os.WriteFile(path, []byte("package handlers\n// func HandleWebhook() {}\nfunc Other() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if canGradeLiveReturn(root, livePhase{ExitCode: 124}) {
		t.Fatal("comment masqueraded as an artifact")
	}
	if err := os.WriteFile(path, []byte("package handlers\nfunc HandleWebhook() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if !canGradeLiveReturn(root, livePhase{ExitCode: 124}) {
		t.Fatal("produced function discarded after timeout")
	}
}
