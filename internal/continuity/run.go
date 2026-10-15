package continuity

import (
	stdcontext "context"
	"encoding/json"
	"fmt"
	"knodexa/internal/approach"
	"knodexa/internal/codegraph"
	"knodexa/internal/identity"
	"os"
	"os/exec"
	"strings"
	"time"
)

type RunOptions struct {
	ResumeOptions
	Command []string
	Verify  []string
	Timeout time.Duration
}
type CommandOutcome struct {
	Command  []string `json:"command"`
	ExitCode int      `json:"exit_code"`
	Output   string   `json:"output"`
}
type RunResult struct {
	BehaviorChecks []BehaviorOutcome `json:"behavior_checks,omitempty"`
	Prepared       *PreparedWork     `json:"prepared"`
	Commands       []CommandOutcome  `json:"commands"`
	Captured       bool              `json:"captured"`
	Checks         *approach.Report  `json:"checks,omitempty"`
}

// Run is opt-in: it cannot intercept unrelated clients' edit tools. The invoked
// command receives the prepared package via KNODEXA_CONTEXT_FILE.
func (s *Service) Run(request identity.Request, budget int, options RunOptions) (*RunResult, error) {
	if len(options.Command) == 0 || len(options.Verify) == 0 {
		return nil, fmt.Errorf("run requires a command and explicit verification command")
	}
	prepared, err := s.Prepare(request, budget, options.ResumeOptions)
	if err != nil {
		return nil, err
	}
	result := &RunResult{Prepared: prepared}
	if prepared.Status != "resolved" || prepared.NeedsReview {
		return result, fmt.Errorf("work identity or established agreement needs review before execution")
	}
	if err := approach.CheckBehaviorInputs(s.store, prepared.Context.EstablishedApproach); err != nil {
		return result, err
	}
	originalTask, err := s.db.GetTaskData(prepared.SelectedTask)
	if err != nil {
		return result, err
	}
	paths := append([]string{}, originalTask.TouchesFiles...)
	file, err := os.CreateTemp("", "knodexa-context-*.json")
	if err != nil {
		return nil, err
	}
	defer os.Remove(file.Name())
	data, _ := json.Marshal(prepared)
	if _, err := file.Write(data); err != nil {
		file.Close()
		return nil, err
	}
	file.Close()
	if err := os.Chmod(file.Name(), 0400); err != nil {
		return nil, err
	}
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	execute := func(args []string) bool {
		ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), timeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = s.store.RootDir()
		cmd.Env = append(os.Environ(), "KNODEXA_CONTEXT_FILE="+file.Name())
		var output cappedOutput
		cmd.Stdout = &output
		cmd.Stderr = &output
		err := cmd.Run()
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
		result.Commands = append(result.Commands, CommandOutcome{args, code, text})
		return err == nil
	}
	if !execute(options.Command) {
		return result, fmt.Errorf("invoked command failed; memory not captured")
	}
	for _, verify := range options.Verify {
		if !execute([]string{"/bin/sh", "-c", verify}) {
			return result, fmt.Errorf("verification failed; memory not captured")
		}
	}
	result.BehaviorChecks, err = s.runBehaviorChecks(prepared.Context.EstablishedApproach, timeout)
	if err != nil {
		return result, err
	}
	if err := s.idx.Sync(); err != nil {
		return result, err
	}
	if err := s.assertBehaviorContracts(prepared.SelectedTask, prepared.Context.EstablishedApproach); err != nil {
		return result, err
	}
	// Evaluate the restored agreement, not an agreement the command could weaken.
	sources, err := codegraph.Scan(s.store.RootDir(), s.db)
	if err != nil {
		return result, err
	}
	for _, source := range sources {
		for _, contract := range prepared.Context.EstablishedApproach {
			for _, scope := range contract.Approach.Scopes {
				if approach.MatchesScope(scope, source.Path) {
					paths = append(paths, source.Path)
				}
			}
		}
	}
	result.Checks = approach.CheckContracts(s.store, prepared.SelectedTask, paths, prepared.Context.EstablishedApproach, "restored")
	if !result.Checks.StructuralChecksPassed {
		return result, fmt.Errorf("approach drift after execution; memory not captured")
	}
	if err := identity.Capture(s.store, s.db, s.idx, prepared.SelectedTask, paths); err != nil {
		return result, err
	}
	result.Captured = true
	return result, nil
}

type cappedOutput struct {
	text      strings.Builder
	truncated bool
}

func (output *cappedOutput) Write(data []byte) (int, error) {
	count := len(data)
	remaining := 8192 - output.text.Len()
	if count > remaining {
		data = data[:remaining]
		output.truncated = true
	}
	output.text.Write(data)
	return count, nil
}
