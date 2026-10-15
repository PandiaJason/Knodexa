package approach

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"strings"
)

func validateBehaviorChecks(store *storage.Storage, checks []model.BehaviorCheck) error {
	seen := map[string]bool{}
	for i := range checks {
		check := &checks[i]
		if strings.TrimSpace(check.Name) == "" || strings.TrimSpace(check.Command) == "" || len(check.ProtectedFiles) == 0 {
			return fmt.Errorf("behavior checks require a name, command and protected test/specification files")
		}
		if seen[check.Name] {
			return fmt.Errorf("duplicate behavior check: %s", check.Name)
		}
		seen[check.Name] = true
		for j := range check.ProtectedFiles {
			ref := &check.ProtectedFiles[j]
			data, err := readWithin(store.RootDir(), ref.Path)
			if err != nil {
				return fmt.Errorf("behavior check %s: %w", check.Name, err)
			}
			sum := sha256.Sum256(data)
			digest := hex.EncodeToString(sum[:])
			if ref.SHA256 != "" && ref.SHA256 != digest {
				return fmt.Errorf("protected behavior input changed: %s", ref.Path)
			}
			ref.SHA256 = digest
		}
	}
	return nil
}

// CheckBehaviorInputs validates recorded hashes without refreshing them.
// Commands run only through opt-in executable record/run workflows.
func CheckBehaviorInputs(store *storage.Storage, contracts []Contract) error {
	for _, contract := range contracts {
		for _, check := range contract.Approach.BehaviorChecks {
			if strings.TrimSpace(check.Name) == "" || strings.TrimSpace(check.Command) == "" || len(check.ProtectedFiles) == 0 {
				return fmt.Errorf("invalid behavior check in %s", contract.SourceTask)
			}
			for _, ref := range check.ProtectedFiles {
				data, err := readWithin(store.RootDir(), ref.Path)
				if err != nil {
					return fmt.Errorf("protected behavior input %s: %w", ref.Path, err)
				}
				sum := sha256.Sum256(data)
				if !validDigest(ref.SHA256) || hex.EncodeToString(sum[:]) != ref.SHA256 {
					return fmt.Errorf("protected behavior input changed: %s (source %s, check %s)", ref.Path, contract.SourceTask, check.Name)
				}
			}
		}
	}
	return nil
}
