package continuity

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// executeCommand keeps all verification paths consistent. WaitDelay bounds pipe
// draining when a descendant retains stdout/stderr after the command exits.
// It does not terminate descendant processes.
func executeCommand(root string, args, env []string, timeout time.Duration) (CommandOutcome, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir, cmd.Env = root, env
	cmd.WaitDelay = 100 * time.Millisecond
	var output cappedOutput
	cmd.Stdout, cmd.Stderr = &output, &output
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
	return CommandOutcome{args, code, text}, err
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
