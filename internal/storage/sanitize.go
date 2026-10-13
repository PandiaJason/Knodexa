package storage

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var idRegex = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ValidateID ensures that an identifier is safe for filesystem naming and contains no path traversal.
func ValidateID(id string) error {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return fmt.Errorf("id cannot be empty")
	}
	if len(trimmed) > 128 {
		return fmt.Errorf("id exceeds maximum length of 128 characters: %q", trimmed)
	}
	if trimmed == "." || trimmed == ".." {
		return fmt.Errorf("invalid id: %q", trimmed)
	}
	if !idRegex.MatchString(trimmed) {
		return fmt.Errorf("id contains invalid characters (allowed: alphanumeric, '.', '_', '-'): %q", trimmed)
	}
	return nil
}

// SanitizeRelPath ensures a relative path cannot traverse outside the repository or Knodexa directory.
func SanitizeRelPath(relPath string) (string, error) {
	cleaned := filepath.Clean(relPath)
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("absolute paths are not permitted: %q", relPath)
	}
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, string(filepath.Separator)+"..") {
		return "", fmt.Errorf("path traversal attempt detected: %q", relPath)
	}
	return cleaned, nil
}
