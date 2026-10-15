package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"knodexa/internal/continuity"
	"knodexa/internal/identity"
	"strings"
)

func (a *App) runIdentity(args []string) error {
	if a.db == nil {
		return fmt.Errorf("knodexa is not initialized")
	}
	if len(args) > 0 && args[0] == "capture" {
		fs := flag.NewFlagSet("identity capture", flag.ContinueOnError)
		files := fs.String("files", "", "Comma-separated code files; defaults to declared touched files")
		if err := parseInterspersed(fs, args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("identity capture requires one task ID")
		}
		var paths []string
		for _, path := range strings.Split(*files, ",") {
			if strings.TrimSpace(path) != "" {
				paths = append(paths, strings.TrimSpace(path))
			}
		}
		if err := identity.Capture(a.store, a.db, a.indexer, fs.Arg(0), paths); err != nil {
			return err
		}
		fmt.Println("Stable work ID and source anchors captured.")
		return nil
	}
	if len(args) == 0 || args[0] != "alias" {
		return fmt.Errorf("use identity alias <task-id> --alias=<phrase>")
	}
	fs := flag.NewFlagSet("identity alias", flag.ContinueOnError)
	var aliases repeatedStrings
	fs.Var(&aliases, "alias", "Work phrase (repeatable)")
	if err := parseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("identity alias needs exactly one existing task ID")
	}
	if err := identity.AddAliases(a.store, a.db, a.indexer, fs.Arg(0), aliases); err != nil {
		return err
	}
	fmt.Printf("Identity phrases recorded for existing work %s.\n", fs.Arg(0))
	return nil
}
func (a *App) runResolve(args []string) error {
	if a.context == nil {
		return fmt.Errorf("knodexa is not initialized; run knodexa init")
	}
	fs := flag.NewFlagSet("resolve", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Structured resolution and restored context")
	files := fs.String("files", "", "Optional comma-separated file hints")
	selected := fs.String("select", "", "Explicit existing-work selection")
	intent := fs.String("intent", "auto", "auto, continue, new, or replace")
	previous := fs.String("from-task", "", "Previous focus; does not affect matching")
	budget := fs.Int("budget", 1500, "Context human-text token estimate budget")
	full := fs.Bool("full", false, "Expand rationale")
	if err := parseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("resolve needs one quoted natural-language request")
	}
	request := identity.Request{Text: fs.Arg(0), Intent: *intent, SelectTask: *selected}
	for _, file := range strings.Split(*files, ",") {
		if strings.TrimSpace(file) != "" {
			request.Files = append(request.Files, strings.TrimSpace(file))
		}
	}
	result, err := continuity.NewService(a.store, a.db, a.indexer, a.context, a.git).ResolveRequest(request, *budget, continuity.ResumeOptions{Full: *full, PreviousTaskID: *previous})
	if err != nil {
		return err
	}
	if *jsonOutput {
		data, err := json.Marshal(result)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("work_resolution: %s\nnext_action: %s\n%s\n", result.Status, result.NextAction, result.Explanation)
	for _, candidate := range result.Candidates {
		fmt.Printf("* %s (work %s): %s; score %d; evidence %v\n", candidate.TaskID, candidate.WorkID, candidate.Title, candidate.Score, candidate.Signals)
	}
	if result.CandidatesTruncated {
		fmt.Printf("Showing %d of %d candidates; all were considered for ambiguity.\n", len(result.Candidates), result.CandidateCount)
	}
	if result.Context != nil {
		fmt.Println()
		fmt.Println(result.Context.FormatHuman())
	}
	return nil
}
