package approach

import (
	"fmt"
	"knodexa/internal/index"
	"knodexa/internal/storage"
	"path"
	"strings"
)

// Selection makes routing inspectable without injecting unrelated rules.
// Ambiguous scopes are never merged or selected by recency.
type Selection struct {
	Mode            string   `json:"mode"`
	Sources         []string `json:"sources"`
	Candidates      []string `json:"candidates,omitempty"`
	ExcludedSources []string `json:"excluded_sources,omitempty"`
	ReviewRequired  bool     `json:"review_required"`
}

func validateScope(scope string) error {
	if strings.TrimSpace(scope) != scope || scope == "" {
		return fmt.Errorf("scope cannot be blank or padded")
	}
	if _, err := storage.SanitizeRelPath(strings.TrimSuffix(scope, "/")); err != nil {
		return err
	}
	_, err := path.Match(scope, "probe")
	return err
}
func matches(scope, file string) bool {
	if _, err := storage.SanitizeRelPath(file); err != nil {
		return false
	}
	if strings.HasSuffix(scope, "/") {
		return strings.HasPrefix(file, scope)
	}
	ok, _ := path.Match(scope, file)
	return ok
}

// Select uses explicit inheritance first, then an unambiguous scope covering all
// target files. Names, recent tasks, and fuzzy text never determine authority.
func Select(db *index.DB, taskID string) ([]Contract, *Selection, error) {
	contracts, err := Resolve(db, taskID)
	if err != nil {
		return nil, nil, err
	}
	task, err := db.GetTaskData(taskID)
	if err != nil {
		return nil, nil, err
	}
	tasks, err := db.ListApproachTasks()
	if err != nil {
		return nil, nil, err
	}
	selection := &Selection{Mode: "missing", Sources: []string{}, ReviewRequired: true}
	for _, source := range tasks {
		if source.Approach == nil {
			continue
		}
		for _, scope := range source.Approach.Scopes {
			if err := validateScope(scope); err != nil {
				return nil, nil, fmt.Errorf("scope on %s: %w", source.ID, err)
			}
		}
		covers := len(task.TouchesFiles) > 0 && len(source.Approach.Scopes) > 0
		for _, file := range task.TouchesFiles {
			found := false
			for _, scope := range source.Approach.Scopes {
				if matches(scope, file) {
					found = true
					break
				}
			}
			covers = covers && found
		}
		if covers {
			selection.Candidates = append(selection.Candidates, source.ID)
		}
	}
	if len(contracts) > 0 {
		selection.Mode = "explicit"
		selection.ReviewRequired = false
	} else if len(selection.Candidates) == 1 {
		contracts, err = Resolve(db, selection.Candidates[0])
		if err != nil {
			return nil, nil, err
		}
		selection.Mode = "file_scope"
		selection.ReviewRequired = false
	} else if len(selection.Candidates) > 1 {
		selection.Mode = "ambiguous"
	}
	included := map[string]bool{}
	for _, c := range contracts {
		included[c.SourceTask] = true
		selection.Sources = append(selection.Sources, c.SourceTask)
	}
	for _, source := range tasks {
		if source.Approach != nil && !included[source.ID] {
			selection.ExcludedSources = append(selection.ExcludedSources, source.ID)
		}
	}
	return contracts, selection, nil
}

// ValidateScope and MatchesScope expose the same deterministic scope semantics
// to request resolution; routing and identity must not interpret paths differently.
func ValidateScope(scope string) error     { return validateScope(scope) }
func MatchesScope(scope, file string) bool { return matches(scope, file) }
