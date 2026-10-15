package continuity

import (
	"fmt"
	"knodexa/internal/approach"
	"knodexa/internal/codegraph"
	"knodexa/internal/model"
	"strings"
)

// ReferenceReview is explicit caller review, never inferred semantic equivalence.
// The service fills hashes/version from the inherited agreement and current code.
type ReferenceReview struct {
	SourceTask   string `json:"source_task"`
	OriginalPath string `json:"original_path"`
	CurrentPath  string `json:"current_path"`
	Reason       string `json:"reason"`
}

func (s *Service) applyReferenceReviews(task *model.TaskData, input *RecordInput) error {
	if len(input.ReferenceReviews) == 0 {
		return nil
	}
	proposed := *task
	value := input.Approach
	value.ReferenceReconciliations = nil
	proposed.Approach = &value
	contracts, err := approach.ResolveProposed(s.db, &proposed)
	if err != nil {
		return err
	}
	for _, review := range input.ReferenceReviews {
		if review.SourceTask == task.ID || strings.TrimSpace(review.Reason) == "" {
			return fmt.Errorf("reference review requires an inherited source and explicit review reason")
		}
		matched := false
		for _, contract := range contracts {
			if contract.SourceTask != review.SourceTask {
				continue
			}
			for _, ref := range contract.Approach.References {
				if ref.Path != review.OriginalPath {
					continue
				}
				anchor, err := codegraph.Read(s.store.RootDir(), review.CurrentPath)
				if err != nil {
					return err
				}
				version := contract.Approach.Version
				if version == 0 {
					version = 1
				}
				input.Approach.ReferenceReconciliations = append(input.Approach.ReferenceReconciliations, model.ReferenceReconciliation{SourceTask: review.SourceTask, SourceVersion: version, OriginalPath: ref.Path, OriginalSHA256: ref.SHA256, CurrentPath: review.CurrentPath, CurrentSHA256: anchor.SHA256, Reason: review.Reason})
				matched = true
			}
		}
		if !matched {
			return fmt.Errorf("reference review does not identify an inherited reference: %s:%s", review.SourceTask, review.OriginalPath)
		}
	}
	return nil
}
