package cli

import (
	"flag"
	"fmt"
)

func (a *App) runSupersede(args []string) error {
	if a.verify == nil {
		return fmt.Errorf("knodexa is not initialized")
	}
	fs := flag.NewFlagSet("supersede", flag.ContinueOnError)
	by := fs.String("by", "", "Verified replacement decision or lesson ID")
	reason := fs.String("reason", "", "Why the previous knowledge no longer applies")
	if err := parseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("use supersede <D-id|L-id> --by=<verified-id> --reason=<reason>")
	}
	if err := a.verify.SupersedeNode(fs.Arg(0), *by, *reason, "user"); err != nil {
		return err
	}
	fmt.Printf("%s superseded by %s; historical content retained.\n", fs.Arg(0), *by)
	return nil
}
