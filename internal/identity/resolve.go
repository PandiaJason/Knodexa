// Package identity resolves requests against recorded work before task creation.
// Retrieval is deterministic and explainable; it is not semantic understanding.
package identity

import (
	"fmt"
	"knodexa/internal/approach"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"sort"
	"strings"
	"unicode"
)

type Request struct {
	Text       string   `json:"request"`
	Files      []string `json:"files,omitempty"`
	SelectTask string   `json:"select_task,omitempty"`
	Intent     string   `json:"intent,omitempty"` // auto | continue | new | replace
}
type Signal struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}
type Candidate struct {
	TaskID  string           `json:"task_id"`
	WorkID  string           `json:"work_id"`
	Title   string           `json:"title"`
	Goal    string           `json:"goal"`
	Status  model.TaskStatus `json:"status"`
	Score   int              `json:"score"`
	Signals []Signal         `json:"signals"`
	Strong  bool             `json:"strong_match"`
}
type Result struct {
	Request              string       `json:"request"`
	Intent               string       `json:"intent"`
	Status               string       `json:"status"` // resolved | needs_clarification | unmatched | new_work_requested
	SelectedTask         string       `json:"selected_task,omitempty"`
	WorkID               string       `json:"work_id,omitempty"`
	Candidates           []Candidate  `json:"candidates"`
	CandidateCount       int          `json:"candidate_count"`
	CandidatesTruncated  bool         `json:"candidates_truncated"`
	NextAction           string       `json:"next_action"`
	CreateNewWorkAllowed bool         `json:"create_new_work_allowed"`
	Explanation          string       `json:"explanation"`
	RelocatedFiles       []Relocation `json:"relocated_files,omitempty"`
}

var stopWords = func() map[string]bool {
	result := map[string]bool{}
	for _, word := range strings.Fields("a an the this that it its of to for in on and or with without please now again another new add make do task work code implement implementation same previous format change update improve fix continue existing use should must our we i you then") {
		result[word] = true
	}
	return result
}()

func tokens(text string) []string {
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	result := []string{}
	seen := map[string]bool{}
	for _, word := range words {
		if len(word) > 4 && strings.HasSuffix(word, "s") {
			word = strings.TrimSuffix(word, "s")
		}
		if !stopWords[word] && len([]rune(word)) > 1 && !seen[word] {
			result = append(result, word)
			seen[word] = true
		}
	}
	return result
}

// phrase deliberately does not remove stop words: "request a new handler" is
// not an exact title match for "new handler" after arbitrary word removal.
func phrase(text string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }), " ")
}
func containsPhrase(text, value string) bool {
	value = phrase(value)
	return value != "" && strings.Contains(" "+phrase(text)+" ", " "+value+" ")
}
func fileHints(request Request) ([]string, error) {
	result := []string{}
	seen := map[string]bool{}
	add := func(value string) error {
		clean, err := storage.SanitizeRelPath(value)
		if err != nil {
			return err
		}
		if !seen[clean] {
			result = append(result, clean)
			seen[clean] = true
		}
		return nil
	}
	for _, file := range request.Files {
		if err := add(file); err != nil {
			return nil, fmt.Errorf("file hint %q: %w", file, err)
		}
	}
	// Only repository-relative path-looking words are inferred, not URLs or IDs.
	for _, word := range strings.Fields(request.Text) {
		word = strings.Trim(word, "`\"'(),:;.!?")
		if strings.Contains(word, "/") && !strings.Contains(word, ":") && !strings.HasPrefix(word, "/") {
			if _, err := storage.SanitizeRelPath(word); err == nil {
				if err := add(word); err != nil {
					return nil, err
				}
			}
		}
	}
	return result, nil
}

func Resolve(db *index.DB, request Request) (*Result, error) {
	return resolve(db, request, nil, nil)
}
func resolve(db *index.DB, request Request, current []model.CodeAnchor, renames map[string]string) (*Result, error) {
	if strings.TrimSpace(request.Text) == "" {
		return nil, fmt.Errorf("request must not be blank")
	}
	if request.Intent == "" {
		request.Intent = "auto"
	}
	switch request.Intent {
	case "auto", "continue", "new", "replace":
	default:
		return nil, fmt.Errorf("intent must be auto, continue, new, or replace")
	}
	if request.Intent == "new" && request.SelectTask != "" {
		return nil, fmt.Errorf("new work cannot also select an existing task")
	}
	files, err := fileHints(request)
	if err != nil {
		return nil, err
	}
	tasks, err := db.ListTaskIdentities()
	if err != nil {
		return nil, err
	}
	byID := map[string]model.TaskData{}
	for _, task := range tasks {
		byID[task.ID] = task
	}
	root := func(task model.TaskData) (string, error) {
		if task.WorkID != "" {
			return task.WorkID, nil
		}
		seen := map[string]bool{}
		id := task.ID
		for {
			if seen[id] {
				return "", fmt.Errorf("work hierarchy cycle at %s", id)
			}
			seen[id] = true
			current, ok := byID[id]
			if !ok {
				return task.ID, nil
			}
			if _, ok := byID[current.ParentID]; !ok {
				return id, nil
			}
			id = current.ParentID
		}
	}
	result := &Result{Request: request.Text, Intent: request.Intent, Status: "unmatched", Candidates: []Candidate{}, NextAction: "clarify_work_identity", Explanation: "No recorded work matched. Confirm whether this is new work; do not silently create a task."}
	requestTokens := map[string]bool{}
	sourceByPath := map[string]model.CodeAnchor{}
	sourceByHash := map[string][]model.CodeAnchor{}
	for _, source := range current {
		sourceByPath[source.Path] = source
		sourceByHash[source.SHA256] = append(sourceByHash[source.SHA256], source)
	}
	for _, word := range tokens(request.Text) {
		requestTokens[word] = true
	}
	for _, task := range tasks {
		workID, err := root(task)
		if err != nil {
			return nil, err
		}
		candidate := Candidate{TaskID: task.ID, WorkID: workID, Title: task.Title, Goal: task.Goal, Status: task.Status, Signals: []Signal{}}
		add := func(kind, value string, score int) {
			candidate.Signals = append(candidate.Signals, Signal{Kind: kind, Value: value})
			candidate.Score += score
		}
		if len(tokens(task.Title)) >= 2 && containsPhrase(request.Text, task.Title) {
			add("title_phrase", task.Title, 12)
			candidate.Strong = true
		}
		for _, alias := range task.Aliases {
			if len(tokens(alias)) > 0 && containsPhrase(request.Text, alias) {
				add("recorded_alias", alias, 12)
				candidate.Strong = true
			}
		}
		shared := []string{}
		for _, word := range tokens(task.Title + " " + task.Goal + " " + strings.Join(task.Aliases, " ")) {
			if requestTokens[word] {
				shared = append(shared, word)
			}
		}
		for _, word := range shared {
			add("shared_term", word, 2)
		}
		covered := 0
		locations := map[string]model.CodeAnchor{}
		for _, anchor := range task.CodeAnchors {
			matching := append([]model.CodeAnchor{}, sourceByHash[anchor.SHA256]...)
			if source, ok := sourceByPath[anchor.Path]; ok {
				matching = append(matching, source)
			}
			if source, ok := sourceByPath[renames[anchor.Path]]; ok {
				matching = append(matching, source)
			}
			sort.Slice(matching, func(i, j int) bool { return matching[i].Path < matching[j].Path })
			seenSources := map[string]bool{}
			for _, source := range matching {
				if seenSources[source.Path] {
					continue
				}
				seenSources[source.Path] = true
				evidence := ""
				if source.Path == anchor.Path {
					evidence = "recorded_source"
				} else if renames[anchor.Path] == source.Path {
					evidence = "git_rename"
				} else if source.SHA256 == anchor.SHA256 {
					evidence = "content_identity"
				}
				if evidence == "" {
					continue
				}
				locations[source.Path] = source
				if source.Path != anchor.Path {
					add(evidence, anchor.Path+" -> "+source.Path, 1)
				}
				for _, symbol := range source.Symbols {
					if len(symbol) >= 4 && (containsPhrase(request.Text, symbol) || containsPhrase(request.Text, symbolPhrase(symbol))) {
						add("code_symbol", source.Path+":"+symbol, 20)
						candidate.Strong = true
					}
				}
				for _, dependency := range source.Imports {
					if containsPhrase(request.Text, dependency) {
						add("dependency", source.Path+" imports "+dependency, 4)
					}
				}
			}
		}
		for _, file := range files {
			hit := false
			for _, existing := range task.TouchesFiles {
				if existing == file {
					add("recorded_file", file, 20)
					hit = true
					break
				}
			}
			if !hit && task.Approach != nil {
				for _, scope := range task.Approach.Scopes {
					if err := approach.ValidateScope(scope); err != nil {
						return nil, fmt.Errorf("scope on %s: %w", task.ID, err)
					}
					if approach.MatchesScope(scope, file) {
						add("recorded_scope", scope+" -> "+file, 20)
						hit = true
						break
					}
				}
			}
			if !hit {
				if _, ok := locations[file]; ok {
					add("anchored_current_file", file, 20)
					hit = true
				}
			}
			if hit {
				covered++
			}
		}
		// Text cannot override contradictory or only partially covered file hints.
		if len(files) > 0 {
			candidate.Strong = covered == len(files)
		}
		if candidate.Score > 0 || request.SelectTask == task.ID {
			result.Candidates = append(result.Candidates, candidate)
		}
	}
	sort.Slice(result.Candidates, func(i, j int) bool {
		if result.Candidates[i].Score == result.Candidates[j].Score {
			return result.Candidates[i].TaskID < result.Candidates[j].TaskID
		}
		return result.Candidates[i].Score > result.Candidates[j].Score
	})
	result.CandidateCount = len(result.Candidates)
	if request.Intent == "new" {
		result.Status = "new_work_requested"
		result.NextAction = "propose_new_work"
		result.CreateNewWorkAllowed = true
		result.Explanation = "New work explicitly requested. Related candidates are informational; no task has been created."
	} else if request.SelectTask != "" {
		chosen, ok := byID[request.SelectTask]
		if !ok {
			return nil, fmt.Errorf("selected task %s does not exist", request.SelectTask)
		}
		result.SelectedTask = chosen.ID
		result.WorkID, err = root(chosen)
		if err != nil {
			return nil, err
		}
		result.Status = "resolved"
		result.Explanation = "Existing work selected explicitly; previous task status and records are preserved."
	} else if len(result.Candidates) > 0 {
		result.Status = "needs_clarification"
		result.Explanation = "Matching evidence is weak or overlaps existing work. Select the intended work before coding or creating a task."
		top := result.Candidates[0]
		// A second strong match is always ambiguous, regardless of scoring margin.
		strong := 0
		for _, candidate := range result.Candidates {
			if candidate.Strong {
				strong++
			}
		}
		margin := top.Score
		if len(result.Candidates) > 1 {
			margin -= result.Candidates[1].Score
		}
		if top.Strong && strong == 1 && margin >= 6 {
			result.Status = "resolved"
			result.SelectedTask = top.TaskID
			result.WorkID = top.WorkID
			result.Explanation = "A unique recorded phrase or complete file-scope match identifies existing work. Recency and task-name prefixes were not used."
		}
	}
	if result.Status == "resolved" {
		for _, task := range tasks {
			if task.ID != result.SelectedTask {
				continue
			}
			for _, anchor := range task.CodeAnchors {
				for _, source := range current {
					if source.Path == anchor.Path {
						continue
					}
					if renames[anchor.Path] == source.Path {
						result.RelocatedFiles = append(result.RelocatedFiles, Relocation{anchor.Path, source.Path, "git_rename"})
					} else if source.SHA256 == anchor.SHA256 {
						result.RelocatedFiles = append(result.RelocatedFiles, Relocation{anchor.Path, source.Path, "content_identity"})
					}
				}
			}
		}
		result.NextAction = "continue_existing_work"
		if request.Intent == "replace" {
			result.NextAction = "review_replacement"
			result.Explanation += " Preserve the prior agreement as history and record the intentional replacement with rationale and evidence."
		}
	}
	if len(result.Candidates) > 8 {
		result.CandidatesTruncated = true
		result.Candidates = result.Candidates[:8]
	}
	return result, nil
}
