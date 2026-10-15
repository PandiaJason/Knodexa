package identity

import (
	"fmt"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"strings"
	"time"
)

func ValidateAliases(aliases []string) error {
	for _, alias := range aliases {
		if strings.TrimSpace(alias) == "" || len(tokens(alias)) == 0 {
			return fmt.Errorf("alias must name meaningful work: %q", alias)
		}
	}
	return nil
}

// AddAliases records vocabulary for existing work without reopening it.
func AddAliases(store *storage.Storage, db *index.DB, idx *index.Indexer, taskID string, aliases []string) error {
	if len(aliases) == 0 {
		return fmt.Errorf("at least one alias is required")
	}
	if err := ValidateAliases(aliases); err != nil {
		return err
	}
	if err := idx.Sync(); err != nil {
		return err
	}
	if err := store.UpdateTask(taskID, func(task *model.TaskData) error {
		changed := false
		for _, alias := range aliases {
			alias = strings.TrimSpace(alias)
			exists := false
			for _, existing := range task.Aliases {
				if phrase(existing) == phrase(alias) {
					exists = true
					break
				}
			}
			if !exists {
				task.Aliases = append(task.Aliases, alias)
				changed = true
			}
		}
		if changed {
			task.UpdatedAt = time.Now().UTC()
		}
		return nil
	}); err != nil {
		return err
	}
	return idx.Sync()
}
