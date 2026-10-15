package approach

import "sort"

// Structural conflicts are explicit opposing obligations. Free-form prose still
// requires human/agent review; it is not interpreted as executable policy.
func structuralConflicts(contracts []Contract) []Result {
	required := map[string]string{}
	forbidden := map[string]string{}
	for _, contract := range contracts {
		for _, check := range contract.Approach.Checks {
			if check.MustContain != "" {
				required[check.Path+"\x00literal\x00"+check.MustContain] = contract.SourceTask
			}
			if check.MustNotContain != "" {
				forbidden[check.Path+"\x00literal\x00"+check.MustNotContain] = contract.SourceTask
			}
			if check.ASTKind == "require-call" {
				required[check.Path+"\x00call\x00"+check.Symbol] = contract.SourceTask
			}
			if check.ASTKind == "forbid-call" {
				forbidden[check.Path+"\x00call\x00"+check.Symbol] = contract.SourceTask
			}
		}
	}
	var results []Result
	keys := make([]string, 0, len(required))
	for key := range required {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		source := required[key]
		if other, exists := forbidden[key]; exists {
			results = append(results, Result{SourceTask: source, Kind: "agreement-conflict", Passed: false, Detail: "Opposing structural obligations from " + source + " and " + other + "; explicitly replace or unlink the conflicting agreement"})
		}
	}
	return results
}
