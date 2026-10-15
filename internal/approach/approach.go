package approach

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Contract records an inherited approach with its explicit source task.
type Contract struct {
	SourceTask string             `json:"source_task"`
	Approach   model.TaskApproach `json:"approach"`
}

// Resolve follows explicit hierarchy and reuse links, never task-name prefixes.
func Resolve(db *index.DB, taskID string) ([]Contract, error) { return resolve(db, taskID, nil) }
func ResolveProposed(db *index.DB, task *model.TaskData) ([]Contract, error) {
	return resolve(db, task.ID, task)
}
func resolve(db *index.DB, taskID string, override *model.TaskData) ([]Contract, error) {
	result := []Contract{}
	state := map[string]int{}
	var visit func(string, bool) error
	visit = func(id string, required bool) error {
		if id == "" {
			return nil
		}
		if state[id] == 1 {
			return fmt.Errorf("approach dependency cycle at %s", id)
		}
		if state[id] == 2 {
			return nil
		}
		var task *model.TaskData
		var err error
		if override != nil && id == override.ID {
			task = override
		} else {
			task, err = db.GetTaskData(id)
		}
		if err != nil {
			if !required && errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		state[id] = 1
		if err := visit(task.ParentID, false); err != nil {
			return err
		}
		for _, source := range task.ApproachRefs {
			if err := visit(source, true); err != nil {
				return err
			}
		}
		if task.Approach != nil {
			result, err = Reconcile(result, task.ID, task.Approach.ReferenceReconciliations)
			if err != nil {
				return err
			}
			result = append(result, Contract{SourceTask: task.ID, Approach: *task.Approach})
		}
		state[id] = 2
		return nil
	}
	if err := visit(taskID, true); err != nil {
		return nil, err
	}
	return result, nil
}

// ValidateTask validates explicit reuse links before saving a new/edited task.
func ValidateTask(db *index.DB, task *model.TaskData) error {
	if _, err := resolve(db, task.ID, task); err != nil {
		return err
	}
	for _, source := range task.ApproachRefs {
		contracts, err := Resolve(db, source)
		if err != nil {
			return err
		}
		if len(contracts) == 0 {
			return fmt.Errorf("source %s has no established approach", source)
		}
	}
	return nil
}

func Reuse(store *storage.Storage, db *index.DB, idx *index.Indexer, taskID string, sources []string) error {
	if len(sources) == 0 {
		return fmt.Errorf("at least one approach source is required")
	}
	if err := idx.Sync(); err != nil {
		return err
	}
	task, err := db.GetTaskData(taskID)
	if err != nil {
		return err
	}
	for _, source := range sources {
		exists := false
		for _, old := range task.ApproachRefs {
			if old == source {
				exists = true
				break
			}
		}
		if !exists {
			task.ApproachRefs = append(task.ApproachRefs, source)
		}
	}
	if err := ValidateTask(db, task); err != nil {
		return err
	}
	task.UpdatedAt = time.Now().UTC()
	if err := store.SaveTask(task); err != nil {
		return err
	}
	return idx.Sync()
}

// Set records an explicitly selected approach and captures reference hashes.
func ValidateValue(store *storage.Storage, value *model.TaskApproach) error {
	if err := validateBehaviorChecks(store, value.BehaviorChecks); err != nil {
		return err
	}
	for _, binding := range value.ReferenceReconciliations {
		data, err := readWithin(store.RootDir(), binding.CurrentPath)
		if err != nil {
			return err
		}
		hash := sha256.Sum256(data)
		if hex.EncodeToString(hash[:]) != binding.CurrentSHA256 {
			return fmt.Errorf("reviewed reconciliation hash changed: %s", binding.CurrentPath)
		}
	}
	for _, scope := range value.Scopes {
		if err := validateScope(scope); err != nil {
			return fmt.Errorf("invalid scope %q: %w", scope, err)
		}
	}
	if strings.TrimSpace(value.Summary) == "" || len(value.Rules) == 0 {
		return fmt.Errorf("an approach needs a summary and at least one rule")
	}
	for _, rule := range value.Rules {
		if strings.TrimSpace(rule) == "" {
			return fmt.Errorf("approach rules cannot be blank")
		}
	}
	for i := range value.References {
		data, err := readWithin(store.RootDir(), value.References[i].Path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		value.References[i].SHA256 = hex.EncodeToString(sum[:])
	}
	for _, path := range value.SkillFiles {
		if _, err := readWithin(store.RootDir(), path); err != nil {
			return err
		}
	}
	for _, check := range value.Checks {
		if check.Path == "" || (check.MustContain == "" && check.MustNotContain == "" && check.ASTKind == "") {
			return fmt.Errorf("checks need a path and a required or forbidden literal")
		}
		if check.ASTKind != "" {
			if err := validateASTCheck(check); err != nil {
				return err
			}
		}
		if check.Path != "@touched" {
			if _, err := safePath(store.RootDir(), check.Path); err != nil {
				return err
			}
		}
	}
	return nil
}

func Set(store *storage.Storage, db *index.DB, idx *index.Indexer, taskID string, value model.TaskApproach) error {
	if err := ValidateValue(store, &value); err != nil {
		return err
	}
	if err := idx.Sync(); err != nil {
		return err
	}
	task, err := db.GetTaskData(taskID)
	if err != nil {
		return err
	}
	if task.Approach != nil && strings.TrimSpace(value.ChangeReason) == "" {
		return fmt.Errorf("replacing an agreement requires change_reason")
	}
	InstallVersion(task, &value)
	if _, err := resolve(db, task.ID, task); err != nil {
		return err
	}
	if err := store.SaveTask(task); err != nil {
		return err
	}
	return idx.Sync()
}

// InstallVersion retains the previous agreement without performing I/O.
func InstallVersion(task *model.TaskData, value *model.TaskApproach) {
	value.RecordedAt = time.Now().UTC()
	value.Version = 1
	if task.Approach != nil {
		old := *task.Approach
		if old.Version == 0 {
			old.Version = 1
		}
		task.ApproachHistory = append(task.ApproachHistory, old)
		value.Version = old.Version + 1
	}
	task.Approach = value
	task.UpdatedAt = value.RecordedAt
}

func safePath(root, rel string) (string, error) {
	clean, err := storage.SanitizeRelPath(rel)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, clean)
	// Resolve the closest existing ancestor, preserving future target suffixes.
	probe := path
	var suffix []string
	var resolved string
	for {
		var resolveErr error
		resolved, resolveErr = filepath.EvalSymlinks(probe)
		if resolveErr == nil {
			break
		}
		if !os.IsNotExist(resolveErr) {
			return "", resolveErr
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return "", resolveErr
		}
		suffix = append(suffix, filepath.Base(probe))
		probe = parent
	}
	for i := len(suffix) - 1; i >= 0; i-- {
		resolved = filepath.Join(resolved, suffix[i])
	}
	absRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(absRoot, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes repository: %s", rel)
	}
	return resolved, nil
}
func readWithin(root, rel string) ([]byte, error) {
	path, err := safePath(root, rel)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", rel, err)
	}
	return data, nil
}

type Result struct {
	SourceTask string `json:"source_task"`
	Path       string `json:"path"`
	Kind       string `json:"kind"`
	Passed     bool   `json:"passed"`
	Detail     string `json:"detail"`
}
type Report struct {
	TaskID                 string   `json:"task_id"`
	StructuralChecksPassed bool     `json:"structural_checks_passed"`
	SemanticReviewRequired bool     `json:"semantic_review_required"`
	Results                []Result `json:"results"`
}

// Check evaluates explicit structural checks and reference drift. Free-form
// design/quality rules still require review; this never claims semantic proof.
func Check(store *storage.Storage, db *index.DB, taskID string) (*Report, error) {
	contracts, selection, err := Select(db, taskID)
	if err != nil {
		return nil, err
	}
	task, err := db.GetTaskData(taskID)
	if err != nil {
		return nil, err
	}
	return CheckContracts(store, taskID, task.TouchesFiles, contracts, selection.Mode), nil
}

// CheckContracts evaluates a proposed package before any canonical write.
func CheckContracts(store *storage.Storage, taskID string, touched []string, contracts []Contract, selectionMode string) *Report {
	report := &Report{TaskID: taskID, StructuralChecksPassed: true, Results: []Result{}}
	if len(contracts) == 0 {
		report.SemanticReviewRequired = true
		report.StructuralChecksPassed = false
		report.Results = append(report.Results, Result{SourceTask: taskID, Kind: "missing-approach", Passed: false, Detail: "No authoritative approach selected: " + selectionMode})
	}
	add := func(result Result) {
		report.Results = append(report.Results, result)
		if !result.Passed {
			report.StructuralChecksPassed = false
		}
	}
	for _, conflict := range structuralConflicts(contracts) {
		add(conflict)
	}
	for _, contract := range contracts {
		if len(contract.Approach.Rules) > 0 {
			report.SemanticReviewRequired = true
		}
		for _, ref := range contract.Approach.References {
			data, err := readWithin(store.RootDir(), ref.Path)
			passed := false
			detail := "reference missing or unreadable"
			if err == nil {
				hash := sha256.Sum256(data)
				passed = hex.EncodeToString(hash[:]) == ref.SHA256
				detail = "reference unchanged"
				if !passed {
					detail = "reference changed: review or explicitly update the approach"
				}
			}
			add(Result{SourceTask: contract.SourceTask, Path: ref.Path, Kind: "reference", Passed: passed, Detail: detail})
		}
		for _, path := range contract.Approach.SkillFiles {
			_, err := readWithin(store.RootDir(), path)
			add(Result{SourceTask: contract.SourceTask, Path: path, Kind: "skill", Passed: err == nil, Detail: "required skill document must be available"})
		}
		for _, check := range contract.Approach.Checks {
			paths := []string{check.Path}
			if check.Path == "@touched" {
				paths = touched
				if len(paths) == 0 {
					add(Result{SourceTask: contract.SourceTask, Kind: "missing-check-target", Passed: false, Detail: "Declare touched files for reusable @touched checks"})
				}
			}
			for _, path := range paths {
				data, err := readWithin(store.RootDir(), path)
				if check.ASTKind != "" {
					passed := false
					detail := "source unreadable"
					if err == nil {
						passed, detail = checkAST(path, data, check)
					}
					add(Result{SourceTask: contract.SourceTask, Path: path, Kind: "ast-" + check.ASTKind, Passed: passed, Detail: detail})
					continue
				}
				passed := err == nil
				if passed && check.MustContain != "" {
					passed = strings.Contains(string(data), check.MustContain)
				}
				if passed && check.MustNotContain != "" {
					passed = !strings.Contains(string(data), check.MustNotContain)
				}
				add(Result{SourceTask: contract.SourceTask, Path: path, Kind: "literal-check", Passed: passed, Detail: fmt.Sprintf("required=%q forbidden=%q", check.MustContain, check.MustNotContain)})
			}
		}
	}
	return report
}
