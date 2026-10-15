package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"knodexa/internal/continuity"
	"knodexa/internal/identity"
	"time"
)

func (a *App) runPrepare(args []string) error {
	if a.context == nil {
		return fmt.Errorf("knodexa not initialized")
	}
	fs := flag.NewFlagSet("prepare", flag.ContinueOnError)
	selected := fs.String("select", "", "Explicit work choice")
	previous := fs.String("from-task", "", "Optional known previous task ID; otherwise resumed focus is automatic")
	budget := fs.Int("budget", 3000, "Bundle budget")
	text := fs.Bool("text", false, "Compact readable handoff; retains constraints and acceptance criteria")
	fs.Bool("json", true, "JSON bundle")
	if err := parseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("prepare requires one quoted request")
	}
	result, err := continuity.NewService(a.store, a.db, a.indexer, a.context, a.git).Prepare(identity.Request{Text: fs.Arg(0), SelectTask: *selected}, *budget, continuity.ResumeOptions{PreviousTaskID: *previous})
	if err != nil {
		return err
	}
	if *text {
		fmt.Println(result.FormatHuman())
		return nil
	}
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
func (a *App) runContinuityCommand(args []string) error {
	if a.context == nil {
		return fmt.Errorf("knodexa not initialized")
	}
	separator := -1
	for i, arg := range args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator < 0 || separator == len(args)-1 {
		return fmt.Errorf("run requires: run <request> --verify=<command> -- <command> [args]")
	}
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	selected := fs.String("select", "", "Explicit work choice")
	previous := fs.String("from-task", "", "Previous focus")
	budget := fs.Int("budget", 3000, "Bundle budget")
	timeout := fs.Duration("timeout", 5*time.Minute, "Per command timeout")
	var verify repeatedStrings
	fs.Var(&verify, "verify", "Verification shell command (repeatable)")
	if err := parseInterspersed(fs, args[:separator]); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("run requires one quoted request")
	}
	result, err := continuity.NewService(a.store, a.db, a.indexer, a.context, a.git).Run(identity.Request{Text: fs.Arg(0), SelectTask: *selected}, *budget, continuity.RunOptions{ResumeOptions: continuity.ResumeOptions{PreviousTaskID: *previous}, Command: args[separator+1:], Verify: verify, Timeout: *timeout})
	if result != nil {
		data, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Println(string(data))
	}
	return err
}
