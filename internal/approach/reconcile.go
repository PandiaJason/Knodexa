package approach

import (
	"encoding/hex"
	"fmt"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"strings"
)

func validDigest(value string) bool {
	data, err := hex.DecodeString(value)
	return err == nil && len(data) == 32 && value == strings.ToLower(value)
}

// Reconcile changes only the returned child-local contracts. Canonical ancestors
// stay immutable; all rules, checks, skills and acceptance criteria remain active.
func Reconcile(contracts []Contract, owner string, bindings []model.ReferenceReconciliation) ([]Contract, error) {
	result := append([]Contract{}, contracts...)
	seen := map[string]bool{}
	for _, binding := range bindings {
		if binding.SourceTask == "" || binding.SourceTask == owner || binding.SourceVersion < 1 || strings.TrimSpace(binding.Reason) == "" || !validDigest(binding.OriginalSHA256) || !validDigest(binding.CurrentSHA256) {
			return nil, fmt.Errorf("reference reconciliation requires an inherited source/version, two SHA-256 hashes and a review reason")
		}
		for _, path := range []string{binding.OriginalPath, binding.CurrentPath} {
			clean, err := storage.SanitizeRelPath(path)
			if err != nil || clean != path {
				return nil, fmt.Errorf("invalid reconciliation path %q", path)
			}
		}
		key := binding.SourceTask + "\x00" + binding.OriginalPath
		if seen[key] {
			return nil, fmt.Errorf("duplicate reconciliation for %s", binding.OriginalPath)
		}
		seen[key] = true
		matched := false
		for i, contract := range result {
			if contract.SourceTask != binding.SourceTask {
				continue
			}
			version := contract.Approach.Version
			if version == 0 {
				version = 1
			}
			if version != binding.SourceVersion {
				return nil, fmt.Errorf("reconciliation source version changed: %s", binding.SourceTask)
			}
			refs := append([]model.ApproachReference{}, contract.Approach.References...)
			for j, ref := range refs {
				if ref.Path == binding.OriginalPath {
					if matched || ref.SHA256 != binding.OriginalSHA256 {
						return nil, fmt.Errorf("reconciliation historical reference mismatch: %s", binding.OriginalPath)
					}
					refs[j] = model.ApproachReference{Path: binding.CurrentPath, SHA256: binding.CurrentSHA256}
					matched = true
				}
			}
			if matched {
				contract.Approach.References = refs
				checks := append([]model.ApproachCheck{}, contract.Approach.Checks...)
				for j := range checks {
					if checks[j].Path == binding.OriginalPath {
						checks[j].Path = binding.CurrentPath
					}
				}
				contract.Approach.Checks = checks
				result[i] = contract
			}
		}
		if !matched {
			return nil, fmt.Errorf("reconciliation does not identify an inherited reference: %s", binding.OriginalPath)
		}
	}
	return result, nil
}
