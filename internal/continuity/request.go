package continuity

import (
	"fmt"
	"knodexa/internal/approach"
	"knodexa/internal/context"
	"knodexa/internal/identity"
)

type ResolvedRequest struct {
	*identity.Result
	Context *context.Package `json:"context,omitempty"`
}

// ResolveRequest is the read-only entry point before task creation. It neither
// reopens completed work nor creates a new task on a weak/no match.
func (s *Service) ResolveRequest(request identity.Request, budget int, options ResumeOptions) (*ResolvedRequest, error) {
	if budget <= 0 {
		return nil, fmt.Errorf("token budget must be positive")
	}
	if err := s.idx.Sync(); err != nil {
		return nil, err
	}
	result, err := identity.ResolveWithCode(s.store.RootDir(), s.db, request)
	if err != nil {
		return nil, err
	}
	resolved := &ResolvedRequest{Result: result}
	if result.Status != "resolved" {
		return resolved, nil
	}
	// Advisory navigation state only supplies an exclusion boundary after identity
	// has resolved. Missing/stale state never determines or blocks target selection.
	if options.PreviousTaskID == "" {
		if previous, err := s.store.ReadFocus(); err == nil && previous != "" {
			if _, err := s.db.GetTaskData(previous); err == nil {
				options.PreviousTaskID = previous
			}
		}
	}
	boundary, err := s.boundary(result.SelectedTask, options.PreviousTaskID)
	if err != nil {
		return nil, err
	}
	if request.Intent == "replace" {
		boundary.Instruction += " This request declares an intentional replacement: retain the original agreement as history and record the replacement rationale and evidence."
	}
	checkpoint, err := s.git.GetCheckpoint()
	if err != nil {
		return nil, err
	}
	report, err := approach.Check(s.store, s.db, result.SelectedTask)
	if err != nil {
		return nil, err
	}
	if len(report.Results) == 1 && report.Results[0].Kind == "missing-approach" {
		report = nil
	}
	resolved.Context, err = s.builder.BuildWithOptions(result.SelectedTask, budget, context.BuildOptions{Compact: !options.Full, Checkpoint: checkpoint, ApproachStatus: report, Boundary: boundary})
	return resolved, err
}

func (s *Service) boundary(taskID, previousID string) (*context.Boundary, error) {
	contracts, _, err := approach.Select(s.db, taskID)
	if err != nil {
		return nil, err
	}
	boundary := &context.Boundary{PreviousTask: previousID, TargetTask: taskID, Instruction: "Use this complete handoff to implement the requested extension with the selected goal, constraints, acceptance criteria, and established approach. Preserve completed source work and its agreement. Reopening completed work requires an explicit --reopen request. Set aside unrelated conventions in conversation history. Lessons are advisory, not overrides. Read supplied references before editing. Missing or ambiguous approach selection requires review before claiming consistency."}
	if previousID != "" {
		previous, _, err := approach.Select(s.db, previousID)
		if err != nil {
			return nil, fmt.Errorf("previous task: %w", err)
		}
		selected := map[string]bool{}
		for _, contract := range contracts {
			selected[contract.SourceTask] = true
		}
		for _, contract := range previous {
			if !selected[contract.SourceTask] {
				boundary.DiscardSources = append(boundary.DiscardSources, contract.SourceTask)
			}
		}
	}
	return boundary, nil
}
