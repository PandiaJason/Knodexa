package identity

import (
	"fmt"
	"knodexa/internal/codegraph"
	"knodexa/internal/git"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"strings"
	"time"
	"unicode"
)

type Relocation struct {
	Original string `json:"original"`
	Current  string `json:"current"`
	Evidence string `json:"evidence"`
}

func symbolPhrase(value string) string {
	var b strings.Builder
	runes := []rune(value)
	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 && (unicode.IsLower(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return phrase(b.String())
}
func ResolveWithCode(root string, db *index.DB, request Request) (*Result, error) {
	tasks, err := db.ListTaskIdentities()
	if err != nil {
		return nil, err
	}
	anchored := false
	for _, task := range tasks {
		anchored = anchored || len(task.CodeAnchors) > 0
	}
	if !anchored {
		return Resolve(db, request)
	}
	current, err := codegraph.Scan(root, db)
	if err != nil {
		return nil, err
	}
	relocations := map[string]string{}
	baselineSeen := map[string]bool{}
	for _, task := range tasks {
		if len(task.CodeAnchors) == 0 {
			continue
		}
		full, err := db.GetTaskData(task.ID)
		if err != nil {
			return nil, err
		}
		if !baselineSeen[full.BaselineCommit] {
			baselineSeen[full.BaselineCommit] = true
			renames, err := git.NewInspector(root).Renames(full.BaselineCommit)
			if err == nil {
				for before, after := range renames {
					relocations[before] = after
				}
			}
		}
	}
	return resolve(db, request, current, relocations)
}

// Capture stores source identities without changing completed status or baseline.
func Capture(store *storage.Storage, db *index.DB, idx *index.Indexer, taskID string, files []string) error {
	if err := idx.Sync(); err != nil {
		return err
	}
	task, err := db.GetTaskData(taskID)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		files = task.TouchesFiles
	}
	if len(files) == 0 {
		return fmt.Errorf("capture requires declared files")
	}
	anchors := []model.CodeAnchor{}
	seen := map[string]bool{}
	for _, file := range files {
		if seen[file] {
			continue
		}
		seen[file] = true
		anchor, err := codegraph.Read(store.RootDir(), file)
		if err != nil {
			return err
		}
		anchors = append(anchors, anchor)
	}
	if task.WorkID == "" {
		result, err := Resolve(db, Request{Text: "Capture recorded work", SelectTask: taskID})
		if err != nil {
			return err
		}
		task.WorkID = result.WorkID
	}
	task.CodeAnchors = anchors
	task.UpdatedAt = time.Now().UTC()
	if err := store.SaveTask(task); err != nil {
		return err
	}
	return idx.Sync()
}
