package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"knodexa/internal/approach"
	"knodexa/internal/model"
	"os"
	"strings"
)

type repeatedStrings []string

func (s *repeatedStrings) String() string         { return strings.Join(*s, ", ") }
func (s *repeatedStrings) Set(value string) error { *s = append(*s, value); return nil }

func (a *App) runApproach(args []string) error {
	if a.db == nil {
		return fmt.Errorf("knodexa is not initialized")
	}
	if len(args) == 0 {
		return fmt.Errorf("use approach set <task-id> or approach check <task-id>")
	}
	switch args[0] {
	case "history":
		if len(args) != 2 {
			return fmt.Errorf("approach history needs one task ID")
		}
		task, err := a.db.GetTaskData(args[1])
		if err != nil {
			return err
		}
		data, err := json.Marshal(map[string]interface{}{"current": task.Approach, "history": task.ApproachHistory})
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	case "reuse":
		fs := flag.NewFlagSet("approach reuse", flag.ContinueOnError)
		var sources repeatedStrings
		fs.Var(&sources, "from", "Source task whose approach should be reused (repeatable)")
		if err := parseInterspersed(fs, args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("approach reuse needs one target task ID")
		}
		if err := approach.Reuse(a.store, a.db, a.indexer, fs.Arg(0), sources); err != nil {
			return err
		}
		fmt.Println("Approach reuse linked explicitly; task names do not determine inheritance.")
		return nil

	case "set":
		fs := flag.NewFlagSet("approach set", flag.ContinueOnError)
		file := fs.String("file", "", "Complete approach JSON, including behavior_checks; do not combine with inline agreement flags")
		summary := fs.String("summary", "", "Established implementation pattern")
		reason := fs.String("reason", "", "Reason for intentionally replacing an agreement")
		var astChecks repeatedStrings
		fs.Var(&astChecks, "ast", "Go syntax check path::kind::symbol (repeatable)")
		var rules, refs, skills, required, forbidden, scopes repeatedStrings
		fs.Var(&scopes, "scope", "File glob or directory prefix ending in / (repeatable)")
		fs.Var(&rules, "rule", "Rule to preserve (repeatable)")
		fs.Var(&refs, "reference", "Reference implementation path (repeatable)")
		fs.Var(&skills, "skill", "Reusable skill file path (repeatable)")
		fs.Var(&required, "require", "Literal check path::text (repeatable)")
		fs.Var(&forbidden, "forbid", "Forbidden literal path::text (repeatable)")
		if err := parseInterspersed(fs, args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("approach set needs exactly one task ID")
		}
		value := model.TaskApproach{Summary: *summary, Rules: rules, SkillFiles: skills, Scopes: scopes, ChangeReason: *reason}
		if *file != "" {
			mixed := false
			fs.Visit(func(f *flag.Flag) {
				if f.Name != "file" {
					mixed = true
				}
			})
			if mixed {
				return fmt.Errorf("--file cannot be combined with inline agreement flags; include change_reason in JSON")
			}
			data, err := os.ReadFile(*file)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(data, &value); err != nil {
				return err
			}
		}
		for _, spec := range astChecks {
			parts := strings.SplitN(spec, "::", 3)
			if len(parts) != 3 {
				return fmt.Errorf("AST check must be path::kind::symbol")
			}
			value.Checks = append(value.Checks, model.ApproachCheck{Path: parts[0], ASTKind: parts[1], Symbol: parts[2]})
		}
		for _, path := range refs {
			value.References = append(value.References, model.ApproachReference{Path: path})
		}
		for _, spec := range []struct {
			values    []string
			forbidden bool
		}{{required, false}, {forbidden, true}} {
			for _, text := range spec.values {
				parts := strings.SplitN(text, "::", 2)
				if len(parts) != 2 || parts[1] == "" {
					return fmt.Errorf("check must be path::text")
				}
				check := model.ApproachCheck{Path: parts[0]}
				if spec.forbidden {
					check.MustNotContain = parts[1]
				} else {
					check.MustContain = parts[1]
				}
				value.Checks = append(value.Checks, check)
			}
		}
		if err := approach.Set(a.store, a.db, a.indexer, fs.Arg(0), value); err != nil {
			return err
		}
		fmt.Printf("Established approach recorded for %s. Descendants inherit its rules.\n", fs.Arg(0))
		return nil
	case "check":
		fs := flag.NewFlagSet("approach check", flag.ContinueOnError)
		asJSON := fs.Bool("json", false, "Structured consistency report")
		if err := parseInterspersed(fs, args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("approach check needs exactly one task ID")
		}
		report, err := approach.Check(a.store, a.db, fs.Arg(0))
		if err != nil {
			return err
		}
		if *asJSON {
			data, err := json.Marshal(report)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
		} else {
			for _, result := range report.Results {
				status := "PASS"
				if !result.Passed {
					status = "FAIL"
				}
				fmt.Printf("%s [%s] %s: %s\n", status, result.SourceTask, result.Path, result.Detail)
			}
			fmt.Printf("Structural checks passed: %t; semantic/design review required: %t\n", report.StructuralChecksPassed, report.SemanticReviewRequired)
		}
		if !report.StructuralChecksPassed {
			return fmt.Errorf("established approach checks failed")
		}
		return nil
	default:
		return fmt.Errorf("unknown approach subcommand: %s", args[0])
	}
}

func (a *App) runMemory(args []string) error {
	if a.db == nil {
		return fmt.Errorf("knodexa is not initialized")
	}
	if len(args) != 1 || args[0] != "export" {
		return fmt.Errorf("use memory export to render the task tree and task modules")
	}
	path, err := approach.Export(a.store, a.db)
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}
