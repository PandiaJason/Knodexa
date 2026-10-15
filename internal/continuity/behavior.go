package continuity

import (
	"encoding/json"
	"fmt"
	"knodexa/internal/approach"
	"knodexa/internal/model"
	"reflect"
	"time"
)

type BehaviorOutcome struct {
	SourceTask string `json:"source_task"`
	Name       string `json:"name"`
	CommandOutcome
}

// Routine re-recording carries acceptance forward. Editing/removing existing
// checks requires the separate explicit versioned approach-set operation.
func retainBehaviorChecks(old *model.TaskApproach, proposed *model.TaskApproach) error {
	if old == nil {
		return nil
	}
	for _, existing := range old.BehaviorChecks {
		found := false
		for _, next := range proposed.BehaviorChecks {
			if next.Name != existing.Name {
				continue
			}
			if !reflect.DeepEqual(existing, next) {
				return fmt.Errorf("behavior check %s changed; review and install the acceptance change with approach set before recording", existing.Name)
			}
			found = true
		}
		if !found {
			proposed.BehaviorChecks = append(proposed.BehaviorChecks, existing)
		}
	}
	return nil
}

func (s *Service) assertBehaviorContracts(taskID string, restored []approach.Contract) error {
	current, err := approach.Resolve(s.db, taskID)
	if err != nil {
		return err
	}
	checks := func(contracts []approach.Contract) map[string]interface{} {
		out := map[string]interface{}{}
		for _, contract := range contracts {
			if len(contract.Approach.BehaviorChecks) > 0 {
				out[contract.SourceTask] = contract.Approach.BehaviorChecks
			}
		}
		return out
	}
	if !reflect.DeepEqual(checks(restored), checks(current)) {
		return fmt.Errorf("behavior agreement changed during execution; memory not captured")
	}
	return nil
}

// Restored and proposed contracts both apply: omitting an old acceptance check
// from a new agreement cannot skip it during this recording operation.
func (s *Service) runBehaviorChecks(contracts []approach.Contract, timeout time.Duration) ([]BehaviorOutcome, error) {
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	results := []BehaviorOutcome{}
	seen := map[string]bool{}
	for _, contract := range contracts {
		for _, check := range contract.Approach.BehaviorChecks {
			if err := approach.CheckBehaviorInputs(s.store, contracts); err != nil {
				return results, err
			}
			key, _ := json.Marshal([]interface{}{contract.SourceTask, check})
			if seen[string(key)] {
				continue
			}
			seen[string(key)] = true
			outcome, err := executeCommand(s.store.RootDir(), []string{"/bin/sh", "-c", check.Command}, nil, timeout)
			results = append(results, BehaviorOutcome{contract.SourceTask, check.Name, outcome})
			if err != nil {
				return results, fmt.Errorf("behavior check %s from %s failed; memory unchanged", check.Name, contract.SourceTask)
			}
		}
	}
	return results, approach.CheckBehaviorInputs(s.store, contracts)
}
