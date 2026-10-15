// Package integration distributes one portable project continuity protocol.
package integration

import (
	_ "embed"
	"fmt"
	"knodexa/internal/storage"
	"os"
	"path/filepath"
	"strings"
)

//go:embed protocol.md
var protocol string

const begin = "<!-- knodexa:begin -->"
const end = "<!-- knodexa:end -->"
const skillHeader = "---\nname: knodexa-continuity\ndescription: Restore and check established implementation agreements when coding, switching work, or returning to related work in a Knodexa-enabled repository.\n---\n\n"

func Render(format string) (string, error) {
	block := begin + "\n" + protocol + "\n" + end + "\n"
	switch format {
	case "agents", "rules":
		return block, nil
	case "skill":
		return skillHeader + block, nil
	default:
		return "", fmt.Errorf("format must be agents, skill or rules")
	}
}

func DefaultPath(format string) string {
	switch format {
	case "skill":
		return "skills/knodexa-continuity/SKILL.md"
	case "rules":
		return "RULES.md"
	default:
		return "AGENTS.md"
	}
}

// Install replaces only its managed block, preserving unrelated instructions.
func Install(root, format, rel string) (string, error) {
	generated, err := Render(format)
	if err != nil {
		return "", err
	}
	if rel == "" {
		rel = DefaultPath(format)
	}
	clean, err := storage.SanitizeRelPath(rel)
	if err != nil {
		return "", err
	}
	if clean == "." {
		return "", fmt.Errorf("destination must be a file")
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	probe := root
	for _, component := range strings.Split(clean, string(filepath.Separator)) {
		probe = filepath.Join(probe, component)
		info, err := os.Lstat(probe)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("instruction destination cannot follow symlinks: %s", probe)
		}
	}
	path := filepath.Join(root, clean)
	old, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	text := string(old)
	starts, ends := strings.Count(text, begin), strings.Count(text, end)
	if starts != ends || starts > 1 || (starts == 1 && strings.Index(text, begin) > strings.Index(text, end)) {
		return "", fmt.Errorf("malformed Knodexa instruction block; existing file preserved")
	}
	if format == "skill" && text != "" && !strings.HasPrefix(text, skillHeader) {
		return "", fmt.Errorf("destination is an existing different skill; choose another path")
	}
	if starts == 1 {
		block := strings.TrimPrefix(generated, skillHeader)
		block = strings.TrimSuffix(block, "\n")
		text = text[:strings.Index(text, begin)] + block + text[strings.Index(text, end)+len(end):]
	} else if text != "" {
		text += "\n\n" + generated
	} else {
		text = generated
	}
	if string(old) == text {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	mode := os.FileMode(0644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".knodexa-instructions-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(file.Name())
	if _, err := file.WriteString(text); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return "", err
	}
	return path, nil
}
