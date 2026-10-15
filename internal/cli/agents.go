package cli

import (
	"flag"
	"fmt"
	"knodexa/internal/integration"
)

func (a *App) runAgents(args []string) error {
	if len(args) == 0 || (args[0] != "install" && args[0] != "print") {
		return fmt.Errorf("use agents install|print --format=agents|skill|rules [--path=relative/path]")
	}
	fs := flag.NewFlagSet("agents "+args[0], flag.ContinueOnError)
	format := fs.String("format", "agents", "Instruction format: agents, skill or rules")
	path := fs.String("path", "", "Repository-relative destination; default AGENTS.md, skills/knodexa-continuity/SKILL.md or RULES.md")
	if err := parseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("agents does not accept positional arguments")
	}
	if args[0] == "print" {
		if *path != "" {
			return fmt.Errorf("--path applies only to install")
		}
		text, err := integration.Render(*format)
		if err != nil {
			return err
		}
		fmt.Print(text)
		return nil
	}
	installed, err := integration.Install(a.rootDir, *format, *path)
	if err != nil {
		return err
	}
	fmt.Printf("Installed Knodexa instructions: %s\nConfigure your agent to load this file; installation does not establish host discovery.\n", installed)
	return nil
}
