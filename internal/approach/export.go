package approach

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"html"
	"knodexa/internal/index"
	"knodexa/internal/storage"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Export renders human-readable task modules and a navigable memory tree.
// The generated files are snapshots; canonical task YAML remains authoritative.
func Export(store *storage.Storage, db *index.DB) (string, error) {
	tasks, err := store.ListTasks()
	if err != nil {
		return "", err
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	root, err := exportPath(store.RootDir(), filepath.Join(".knodexa", "exports"))
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return "", err
	}
	var view strings.Builder
	view.WriteString("# Project memory tree\n\nGenerated snapshot of recorded project understanding: task goals, agreed approaches, inherited rules, references, and skills. This does not expose hidden model reasoning. Refresh after changing canonical memory.\n\n```mermaid\nflowchart TD\n")
	ids := map[string]int{}
	for i, task := range tasks {
		if err := storage.ValidateID(task.ID); err != nil {
			return "", err
		}
		ids[task.ID] = i
		label := html.EscapeString(task.ID + ": " + strings.ReplaceAll(task.Title, "\n", " "))
		fmt.Fprintf(&view, "n%d[\"%s\"]\n", i, label)
	}
	for i, task := range tasks {
		if parent, ok := ids[task.ParentID]; ok {
			fmt.Fprintf(&view, "n%d --> n%d\n", parent, i)
		}
	}
	for i, task := range tasks {
		for _, source := range task.ApproachRefs {
			if sourceID, ok := ids[source]; ok {
				fmt.Fprintf(&view, "n%d -. reuse approach .-> n%d\n", sourceID, i)
			}
		}
	}
	view.WriteString("```\n\n| Task | Parent | Established approach sources | Modules |\n|---|---|---|---|\n")
	for _, task := range tasks {
		contracts, selection, err := Select(db, task.ID)
		if err != nil {
			return "", err
		}
		var body strings.Builder
		fmt.Fprintf(&body, "# Task %s: %s\n\nGoal: %s\n\n", task.ID, task.Title, task.Goal)
		fmt.Fprintf(&body, "Recorded identity phrases: %v. Resolve new requests against existing work before creating another task.\n\n", task.Aliases)
		body.WriteString("Resume with `knodexa resume " + task.ID + " --json`. Restore this branch's established approach before coding. Read reference implementations and referenced skill documents. Record rationale and evidence for deliberate deviations. These generated modules are snapshots; consult canonical context for current decisions, failures, and lessons.\n\n")
		fmt.Fprintf(&body, "Selection: %s. Candidates: %v. Excluded approach sources: %v.\n\n", selection.Mode, selection.Candidates, selection.ExcludedSources)
		var sources []string
		for _, contract := range contracts {
			sources = append(sources, contract.SourceTask)
			fmt.Fprintf(&body, "File scopes: %v\n\n", contract.Approach.Scopes)
			for _, binding := range contract.Approach.ReferenceReconciliations {
				fmt.Fprintf(&body, "- Reviewed inherited reference from %s version %d: `%s` (%s) → `%s` (%s). Reason: %s\n", binding.SourceTask, binding.SourceVersion, binding.OriginalPath, binding.OriginalSHA256, binding.CurrentPath, binding.CurrentSHA256, binding.Reason)
			}
			source, err := db.GetTaskData(contract.SourceTask)
			if err != nil {
				return "", err
			}
			for _, constraint := range source.Constraints {
				fmt.Fprintf(&body, "- Source constraint: %s\n", constraint)
			}
			for _, criterion := range source.RequiredVerification {
				fmt.Fprintf(&body, "- Source acceptance: %s\n", criterion)
			}
			fmt.Fprintf(&body, "## Approach inherited from %s\n\n%s\n\n", contract.SourceTask, contract.Approach.Summary)
			for _, rule := range contract.Approach.Rules {
				body.WriteString("- " + rule + "\n")
			}
			for _, ref := range contract.Approach.References {
				fmt.Fprintf(&body, "- Reference: `%s`, recorded SHA256 `%s`.\n", ref.Path, ref.SHA256)
			}
			for _, skill := range contract.Approach.SkillFiles {
				fmt.Fprintf(&body, "- Read skill: `%s`.\n", skill)
			}
			for _, check := range contract.Approach.Checks {
				if check.ASTKind != "" {
					fmt.Fprintf(&body, "- Go AST check: `%s`, `%s`, `%s`.\n", check.Path, check.ASTKind, check.Symbol)
					continue
				}
				fmt.Fprintf(&body, "- Structural check: `%s`, required literal `%s`, forbidden literal `%s`.\n", check.Path, check.MustContain, check.MustNotContain)
			}
			body.WriteString("\n")
		}
		if len(contracts) == 0 {
			body.WriteString("No established approach has been recorded for this branch.\n\n")
		}
		if len(task.Constraints) > 0 {
			body.WriteString("## Task constraints\n\n")
			for _, rule := range task.Constraints {
				body.WriteString("- " + rule + "\n")
			}
		}
		if len(task.RequiredVerification) > 0 {
			body.WriteString("\n## Acceptance checks\n\n")
			for _, check := range task.RequiredVerification {
				body.WriteString("- " + check + "\n")
			}
		}
		dir, err := exportPath(store.RootDir(), filepath.Join(".knodexa", "exports", "tasks", task.ID))
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
		// Reject symlink destinations rather than following them while regenerating.
		write := func(name string, data []byte) error {
			path, err := exportPath(store.RootDir(), filepath.Join(".knodexa", "exports", "tasks", task.ID, name))
			if err != nil {
				return err
			}
			return os.WriteFile(path, data, 0644)
		}
		if err := write("AGENTS.md", []byte(body.String())); err != nil {
			return "", err
		}
		skillName := "knodexa-task-" + strings.ToLower(strings.NewReplacer(".", "-", "_", "-").Replace(task.ID))
		fm, err := yaml.Marshal(map[string]string{"name": skillName, "description": "Restore the recorded approach when resuming task " + task.ID + " or its descendants."})
		if err != nil {
			return "", err
		}
		if err := write("SKILL.md", []byte("---\n"+string(fm)+"---\n\n"+body.String())); err != nil {
			return "", err
		}
		sourceText := strings.Join(sources, ", ")
		if sourceText == "" {
			sourceText = "not recorded"
		}
		fmt.Fprintf(&view, "| %s | %s | %s | [AGENTS.md](tasks/%s/AGENTS.md) · [SKILL.md](tasks/%s/SKILL.md) |\n", task.ID, task.ParentID, sourceText, task.ID, task.ID)
	}
	path, err := exportPath(store.RootDir(), filepath.Join(".knodexa", "exports", "memory-tree.md"))
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(view.String()), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func exportPath(root, rel string) (string, error) {
	resolved, err := safePath(root, rel)
	if err != nil {
		return "", err
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	if resolved != filepath.Join(realRoot, rel) {
		return "", fmt.Errorf("export destination cannot follow symlinks: %s", rel)
	}
	return resolved, nil
}
