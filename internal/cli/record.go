package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"knodexa/internal/continuity"
	"os"
	"time"
)

func (a *App) runRecord(args []string) error {
	fs := flag.NewFlagSet("record", flag.ContinueOnError)
	file := fs.String("file", "", "Agreement JSON: aliases, files, approach (summary, rules, scopes, references, checks, behavior_checks [{name,command,protected_files:[{path}]}], change_reason). Optional reference_reviews [{source_task,original_path,current_path,reason}] derives reviewed inherited bindings without manual hashes; completed sources stay unchanged.")
	timeout := fs.Duration("timeout", 5*time.Minute, "Per verification command timeout")
	var verify repeatedStrings
	fs.Var(&verify, "verify", "Verification shell command (repeatable)")
	if err := parseInterspersed(fs, args); err != nil {
		return err
	}
	if a.context == nil {
		return fmt.Errorf("knodexa not initialized")
	}
	if fs.NArg() != 1 || *file == "" {
		return fmt.Errorf("use record <task-id> --file=agreement.json --verify='go test ./...'")
	}
	data, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	var input continuity.RecordInput
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}
	result, err := continuity.NewService(a.store, a.db, a.indexer, a.context, a.git).Record(fs.Arg(0), input, verify, *timeout)
	if result != nil {
		data, e := json.Marshal(result)
		if e != nil {
			return e
		}
		fmt.Println(string(data))
	}
	return err
}
