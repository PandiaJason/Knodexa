package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Renames compares a recorded checkpoint with the current working tree.
func (gi *Inspector) Renames(baseline string) (map[string]string, error) {
	result := map[string]string{}
	if baseline == "" || !gi.IsRepo() {
		return result, nil
	}
	for _, r := range baseline {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return nil, fmt.Errorf("baseline must be a Git object ID")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), gi.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-status", "-z", "--find-renames=50%", baseline, "--")
	cmd.Dir = gi.rootDir
	data, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	fields := strings.Split(string(data), "\x00")
	for i := 0; i < len(fields)-1; {
		kind := fields[i]
		i++
		if strings.HasPrefix(kind, "R") || strings.HasPrefix(kind, "C") {
			if i+1 >= len(fields) {
				return nil, fmt.Errorf("invalid rename output")
			}
			old, next := fields[i], fields[i+1]
			i += 2
			if strings.HasPrefix(kind, "R") {
				result[old] = next
			}
		} else {
			i++
		}
	}
	return result, nil
}
