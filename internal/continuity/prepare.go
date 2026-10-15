package continuity

import (
	"fmt"
	"knodexa/internal/codegraph"
	"knodexa/internal/identity"
	"os"
	"path/filepath"
	"strings"
)

type ReferenceExcerpt struct {
	SourceTask string `json:"source_task"`
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
	Content    string `json:"content"`
	Truncated  bool   `json:"truncated"`
}

// FormatHuman removes duplicated JSON metadata while preserving the complete
// selected context and every reference's path/hash/truncation marker.
func (prepared *PreparedWork) FormatHuman() string {
	var text strings.Builder
	fmt.Fprintf(&text, "work_resolution: %s\nrequest: %q\nselected_task: %s\nwork_id: %s\nneeds_review: %t\n", prepared.Status, prepared.Request, prepared.SelectedTask, prepared.WorkID, prepared.NeedsReview)
	for _, reason := range prepared.ReviewReasons {
		fmt.Fprintf(&text, "review: %s\n", reason)
	}
	if prepared.Context == nil {
		fmt.Fprintf(&text, "next_action: %s\n%s\n", prepared.NextAction, prepared.Explanation)
		for _, candidate := range prepared.Candidates {
			fmt.Fprintf(&text, "candidate: %s signals=%v\n", candidate.TaskID, candidate.Signals)
		}
		return text.String()
	}
	text.WriteString(prepared.Context.FormatHuman())
	for _, reference := range prepared.References {
		fmt.Fprintf(&text, "\nreference_source: %s task=%s sha256=%s truncated=%t\n", reference.Path, reference.SourceTask, reference.SHA256, reference.Truncated)
		text.WriteString(reference.Content)
		text.WriteByte('\n')
	}
	return text.String()
}

type PreparedWork struct {
	*ResolvedRequest
	References    []ReferenceExcerpt `json:"references,omitempty"`
	NeedsReview   bool               `json:"needs_review"`
	ReviewReasons []string           `json:"review_reasons,omitempty"`
}

// Prepare restores identity, agreements, checkpoint, checks and bounded reference
// excerpts together; unresolved identity never injects competing references.
func (s *Service) Prepare(request identity.Request, budget int, options ResumeOptions) (*PreparedWork, error) {
	resolved, err := s.ResolveRequest(request, budget, options)
	if err != nil {
		return nil, err
	}
	prepared := &PreparedWork{ResolvedRequest: resolved, NeedsReview: resolved.Status != "resolved"}
	if resolved.Context == nil {
		return prepared, nil
	}
	remaining := budget*4 - len([]rune(resolved.Context.FormatHuman()))
	if remaining < 0 {
		remaining = 0
	}
	seen := map[string]bool{}
	for _, contract := range resolved.Context.EstablishedApproach {
		for _, ref := range contract.Approach.References {
			if seen[ref.Path] {
				continue
			}
			seen[ref.Path] = true
			path := ref.Path
			for _, move := range resolved.RelocatedFiles {
				if move.Original == path {
					path = move.Current
				}
			}
			anchor, err := codegraph.Read(s.store.RootDir(), path)
			if err != nil {
				prepared.NeedsReview = true
				prepared.ReviewReasons = append(prepared.ReviewReasons, fmt.Sprintf("reference %s unavailable", path))
				continue
			}
			data, err := os.ReadFile(filepath.Join(s.store.RootDir(), path))
			if err != nil {
				return nil, err
			}
			characters := []rune(string(data))
			limit := remaining
			if limit > 6000 {
				limit = 6000
			}
			truncated := len(characters) > limit
			if truncated {
				characters = characters[:limit]
			}
			remaining -= len(characters)
			prepared.References = append(prepared.References, ReferenceExcerpt{contract.SourceTask, path, anchor.SHA256, string(characters), truncated})
			if anchor.SHA256 != ref.SHA256 {
				prepared.NeedsReview = true
				prepared.ReviewReasons = append(prepared.ReviewReasons, "reference changed: "+path)
			}
		}
	}
	if resolved.Context.ApproachSelection != nil && resolved.Context.ApproachSelection.ReviewRequired {
		prepared.NeedsReview = true
		prepared.ReviewReasons = append(prepared.ReviewReasons, "approach selection requires review")
	}
	if report := resolved.Context.ApproachStatus; report != nil && !report.StructuralChecksPassed {
		prepared.NeedsReview = true
		prepared.ReviewReasons = append(prepared.ReviewReasons, "established approach checks failed")
	}
	return prepared, nil
}
