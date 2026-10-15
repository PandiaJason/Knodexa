package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestInstallPreservesInstructionsAndUpdatesIdempotently(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "AGENTS.md")
	old := "# Existing rules\nKeep public interfaces stable.\n"
	if err := os.WriteFile(path, []byte(old), 0640); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(root, "agents", ""); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(first), old) {
		t.Fatal("existing rules overwritten")
	}
	if _, err := Install(root, "agents", ""); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(path)
	if string(first) != string(second) {
		t.Fatal("installation not idempotent")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0640 {
		t.Fatal("existing permissions changed")
	}
	stale := old + begin + "\nOld protocol\n" + end + "\nAdditional user rule.\n"
	os.WriteFile(path, []byte(stale), 0640)
	if _, err := Install(root, "agents", ""); err != nil {
		t.Fatal(err)
	}
	updated, _ := os.ReadFile(path)
	if strings.Contains(string(updated), "Old protocol") || !strings.HasPrefix(string(updated), old) || !strings.HasSuffix(string(updated), "\nAdditional user rule.\n") {
		t.Fatal("managed update lost unrelated instructions")
	}
}

func TestInstallRejectsUnsafeOrConflictingDestinations(t *testing.T) {
	root := t.TempDir()
	for _, text := range []string{begin, end, end + begin, begin + end + begin + end} {
		path := filepath.Join(root, "AGENTS.md")
		os.WriteFile(path, []byte(text), 0644)
		if _, err := Install(root, "agents", ""); err == nil {
			t.Fatal("malformed block accepted")
		}
		after, _ := os.ReadFile(path)
		if string(after) != text {
			t.Fatal("rejected install changed file")
		}
	}
	if _, err := Install(root, "agents", "../outside.md"); err == nil {
		t.Fatal("traversal accepted")
	}
	out := t.TempDir()
	if err := os.Symlink(out, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(root, "rules", "linked/RULES.md"); err == nil {
		t.Fatal("symlink accepted")
	}
	if _, err := os.Stat(filepath.Join(out, "RULES.md")); !os.IsNotExist(err) {
		t.Fatal("outside file written")
	}
	path := filepath.Join(root, "SKILL.md")
	old := "---\nname: existing-skill\ndescription: Existing skill\n---\n"
	os.WriteFile(path, []byte(old), 0644)
	if _, err := Install(root, "skill", "SKILL.md"); err == nil {
		t.Fatal("unrelated skill overwritten")
	}
	after, _ := os.ReadFile(path)
	if string(after) != old {
		t.Fatal("skill collision changed file")
	}
}

func TestPortableSkillHasValidMetadataAndSharedProtocol(t *testing.T) {
	root := t.TempDir()
	path, err := Install(root, "skill", "")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	parts := strings.SplitN(string(data), "---\n", 3)
	if len(parts) != 3 {
		t.Fatal("missing skill frontmatter")
	}
	var metadata map[string]string
	if err := yaml.Unmarshal([]byte(parts[1]), &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata["name"] != "knodexa-continuity" || metadata["description"] == "" {
		t.Fatal("invalid skill metadata")
	}
	if !strings.Contains(parts[2], protocol) {
		t.Fatal("skill diverged from shared protocol")
	}
	if _, err := Install(root, "skill", ""); err != nil {
		t.Fatal(err)
	}
}
