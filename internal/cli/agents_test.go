package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAgentsCLIInstallsWithoutKnowledgeInitialization(t *testing.T) {
	root := t.TempDir()
	app, err := NewApp(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"agents", "skill", "rules"} {
		if err := app.Run([]string{"agents", "install", "--format=" + format}); err != nil {
			t.Fatal(err)
		}
	}
	for _, rel := range []string{"AGENTS.md", "RULES.md", "skills/knodexa-continuity/SKILL.md"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.Run([]string{"agents", "install", "--format=unknown"}); err == nil {
		t.Fatal("invalid format accepted")
	}
	if err := app.Run([]string{"agents", "install", "--format=rules", "--path=config/agent-rules.md"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "config/agent-rules.md")); err != nil {
		t.Fatal(err)
	}
}
