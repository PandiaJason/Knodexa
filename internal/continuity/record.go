package continuity

import (
	"context"
	"fmt"
	"knodexa/internal/approach"
	"knodexa/internal/codegraph"
	"knodexa/internal/identity"
	"knodexa/internal/model"
	"os/exec"
	"strings"
	"time"
)

type RecordInput struct {
	Aliases  []string           `json:"aliases"`
	Files    []string           `json:"files"`
	Approach model.TaskApproach `json:"approach"`
}
type RecordResult struct {
	TaskID       string           `json:"task_id"`
	WorkID       string           `json:"work_id"`
	Version      int              `json:"version"`
	Recorded     bool             `json:"recorded"`
	Verification []CommandOutcome `json:"verification"`
	Checks       *approach.Report `json:"checks,omitempty"`
}

// Record verifies current code, then installs identity and agreement together.
// Invalid input or failed tests/checks never write partial canonical memory.
func (s *Service) Record(taskID string, input RecordInput, verify []string, timeout time.Duration) (*RecordResult, error) {
	if len(verify) == 0 {
		return nil, fmt.Errorf("record requires explicit --verify commands")
	}
	for _, command := range verify {
		if strings.TrimSpace(command) == "" {
			return nil, fmt.Errorf("verification commands cannot be blank")
		}
	}
	if err := identity.ValidateAliases(input.Aliases); err != nil {
		return nil, err
	}
	if err := s.idx.Sync(); err != nil {
		return nil, err
	}
	task, err := s.db.GetTaskData(taskID)
	if err != nil {
		return nil, err
	}
	if len(input.Files) == 0 {
		input.Files = task.TouchesFiles
	}
	if len(input.Files) == 0 {
		return nil, fmt.Errorf("record requires source files")
	}
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	result := &RecordResult{TaskID: taskID}
	for _, command := range verify {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		cmd := exec.CommandContext(ctx, "/bin/sh", "-c", command)
		cmd.Dir = s.store.RootDir()
		var output cappedOutput
		cmd.Stdout = &output
		cmd.Stderr = &output
		err := cmd.Run()
		cancel()
		code := 0
		if err != nil {
			code = 1
			if exit, ok := err.(*exec.ExitError); ok {
				code = exit.ExitCode()
			}
		}
		text := output.text.String()
		if output.truncated {
			text += "\n[output truncated]"
		}
		result.Verification = append(result.Verification, CommandOutcome{Command: []string{"/bin/sh", "-c", command}, ExitCode: code, Output: text})
		if err != nil {
			return result, fmt.Errorf("verification failed; memory unchanged")
		}
	}
	// Reload after verification so existing metadata is not overwritten by a stale
	// snapshot when an explicit verification command edits canonical task data.
	if err := s.idx.Sync(); err != nil {
		return result, err
	}
	task, err = s.db.GetTaskData(taskID)
	if err != nil {
		return result, err
	}
	if task.Approach != nil && strings.TrimSpace(input.Approach.ChangeReason) == "" {
		return result, fmt.Errorf("replacing an agreement requires approach.change_reason")
	}
	if err := approach.ValidateValue(s.store, &input.Approach); err != nil {
		return result, err
	}
	proposed := *task
	proposed.Approach = &input.Approach
	contracts, err := approach.ResolveProposed(s.db, &proposed)
	if err != nil {
		return result, err
	}
	result.Checks = approach.CheckContracts(s.store, taskID, input.Files, contracts, "explicit")
	if !result.Checks.StructuralChecksPassed {
		return result, fmt.Errorf("structural checks failed; memory unchanged")
	}
	anchors := []model.CodeAnchor{}
	seen := map[string]bool{}
	for _, file := range input.Files {
		if seen[file] {
			continue
		}
		seen[file] = true
		anchor, err := codegraph.Read(s.store.RootDir(), file)
		if err != nil {
			return result, err
		}
		anchors = append(anchors, anchor)
	}
	if task.WorkID == "" {
		selection, err := identity.Resolve(s.db, identity.Request{Text: "Record verified work", SelectTask: taskID})
		if err != nil {
			return result, err
		}
		task.WorkID = selection.WorkID
	}
	for _, alias := range input.Aliases {
		alias = strings.TrimSpace(alias)
		exists := false
		for _, old := range task.Aliases {
			if strings.EqualFold(old, alias) {
				exists = true
				break
			}
		}
		if !exists {
			task.Aliases = append(task.Aliases, alias)
		}
	}
	task.CodeAnchors = anchors
	task.TouchesFiles = make([]string, 0, len(anchors))
	for _, anchor := range anchors {
		task.TouchesFiles = append(task.TouchesFiles, anchor.Path)
	}
	approach.InstallVersion(task, &input.Approach)
	if err := s.store.SaveTask(task); err != nil {
		return result, err
	}
	if err := s.idx.Sync(); err != nil {
		return result, err
	}
	result.WorkID = task.WorkID
	result.Version = task.Approach.Version
	result.Recorded = true
	return result, nil
}
