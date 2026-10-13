package storage

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// SplitFrontmatter splits a markdown document into its YAML frontmatter byte slice and Markdown body string.
func SplitFrontmatter(data []byte) ([]byte, string, error) {
	content := string(data)
	if !strings.HasPrefix(strings.TrimSpace(content), "---") {
		return nil, content, fmt.Errorf("missing initial frontmatter delimiter '---'")
	}

	trimmedLeading := strings.TrimLeft(content, " \t\r\n")
	if !strings.HasPrefix(trimmedLeading, "---") {
		return nil, content, fmt.Errorf("malformed frontmatter delimiter")
	}

	rest := trimmedLeading[3:]
	// Look for next newline then '---'
	idx := strings.Index(rest, "\n---")
	if idx == -1 {
		idx = strings.Index(rest, "\r\n---")
		if idx == -1 {
			return nil, content, fmt.Errorf("missing closing frontmatter delimiter '---'")
		}
	}

	yamlBlock := rest[:idx]
	// Skip past closing delimiter
	afterClosing := rest[idx:]
	if strings.HasPrefix(afterClosing, "\n---") {
		afterClosing = afterClosing[4:]
	} else if strings.HasPrefix(afterClosing, "\r\n---") {
		afterClosing = afterClosing[5:]
	}

	// Consume optional trailing newline
	if strings.HasPrefix(afterClosing, "\r\n") {
		afterClosing = afterClosing[2:]
	} else if strings.HasPrefix(afterClosing, "\n") {
		afterClosing = afterClosing[1:]
	}

	return []byte(yamlBlock), afterClosing, nil
}

// RenderWithFrontmatter formats a struct as YAML frontmatter followed by a markdown body.
func RenderWithFrontmatter(meta interface{}, body string) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("---\n")
	yamlBytes, err := yaml.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal frontmatter YAML: %w", err)
	}
	buf.Write(yamlBytes)
	buf.WriteString("---\n\n")
	buf.WriteString(strings.TrimSpace(body))
	buf.WriteString("\n")
	return buf.Bytes(), nil
}
