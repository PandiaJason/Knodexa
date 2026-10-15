package cli

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"knodexa/internal/approach"
	"knodexa/internal/context"
	"knodexa/internal/continuity"
	"knodexa/internal/git"
	"knodexa/internal/identity"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"knodexa/internal/verify"
)

// App manages CLI execution.
type App struct {
	rootDir string
	store   *storage.Storage
	db      *index.DB
	indexer *index.Indexer
	git     *git.Inspector
	verify  *verify.Engine
	context *context.Builder
}

// NewApp initializes the Knodexa application environment.
func NewApp(rootDir string) (*App, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		absRoot = rootDir
	}

	store := storage.NewStorage(absRoot)
	gitInspector := git.NewInspector(absRoot)

	var db *index.DB
	var indexer *index.Indexer
	var verifier *verify.Engine
	var ctxBuilder *context.Builder

	if store.IsInitialized() {
		dbPath := filepath.Join(store.KnodexaDir(), "cache.db")
		d, err := index.Open(dbPath)
		if err != nil {
			return nil, fmt.Errorf("open knowledge index: %w", err)
		}
		db = d
		indexer = index.NewIndexer(db, store)
		verifier = verify.NewEngine(store, indexer, db)
		ctxBuilder = context.NewBuilder(db, store, gitInspector)
	}

	return &App{
		rootDir: absRoot,
		store:   store,
		db:      db,
		indexer: indexer,
		git:     gitInspector,
		verify:  verifier,
		context: ctxBuilder,
	}, nil
}

// Run parses and executes CLI arguments.
func (a *App) Run(args []string) (resultErr error) {
	defer func() {
		if errors.Is(resultErr, flag.ErrHelp) {
			resultErr = nil
		}
	}()
	if len(args) == 0 {
		a.printHelp()
		return nil
	}

	cmd := args[0]
	rest := args[1:]
	// Resume refreshes within the shared service; other operations refresh once.
	if a.indexer != nil && cmd != "agents" && cmd != "record" && cmd != "resume" && cmd != "resolve" && cmd != "prepare" && cmd != "run" && cmd != "help" && cmd != "--help" && cmd != "-h" {
		if err := a.indexer.Sync(); err != nil {
			return fmt.Errorf("sync knowledge index: %w", err)
		}
	}

	switch cmd {
	case "agents":
		return a.runAgents(rest)
	case "supersede":
		return a.runSupersede(rest)
	case "record":
		return a.runRecord(rest)
	case "prepare":
		return a.runPrepare(rest)
	case "run":
		return a.runContinuityCommand(rest)
	case "help", "-h", "--help":
		a.printHelp()
		return nil
	case "resolve":
		return a.runResolve(rest)
	case "identity":
		return a.runIdentity(rest)
	case "approach":
		return a.runApproach(rest)
	case "memory":
		return a.runMemory(rest)
	case "init":
		return a.runInit(rest)
	case "tree":
		return a.runTree(rest)
	case "task":
		return a.runTask(rest)
	case "context":
		return a.runContext(rest)
	case "resume":
		return a.runRehydrate(rest, true)
	case "show":
		return a.runShow(rest)
	case "related":
		return a.runRelated(rest)
	case "decisions":
		return a.runDecisions(rest)
	case "lessons":
		return a.runLessons(rest)
	case "verify":
		return a.runVerify(rest)
	case "reject":
		return a.runReject(rest)
	case "history":
		return a.runHistory(rest)
	case "doctor":
		return a.runDoctor(rest)
	default:
		return fmt.Errorf("unknown command: %s. Run 'knodexa help' for usage.", cmd)
	}
}

func (a *App) printHelp() {
	fmt.Println(`Knodexa - Continuity and Consistency Guard for Coding Agents

Usage:
  knodexa <command> [arguments]

Commands:
  agents install/print  Integrate the continuity protocol as AGENTS.md, SKILL.md or rules
  init                  Initialize a new .knodexa directory in the project
  tree                  Show the tree-oriented view of project knowledge
  task create           Create a new task or subtask
  task show <id>        Display full details for a task
  task list             List all tasks
  context <id>          Rehydrate compact context (--full expands rationale)
  resolve <request>     Identify existing work before creating a task
  identity alias <id>   Record phrases that refer to established work
  identity capture <id> Capture stable identity, Go symbols and source hashes
  prepare <request>     Restore identity, baseline, checks and reference excerpts
  record <id>           Verify then capture aliases, rules and source identity
  run <request>         Restore, execute with context, verify, then capture identity
  resume <id>           Resume, compare Git baseline, and retrieve context
  approach set/check    Record and check inherited implementation approaches
  approach history <id> Inspect current and previous agreement versions
  memory export         Render the memory tree and modular AGENTS.md/SKILL.md
  show <D-id|L-id>       Expand a decision or lesson on demand
  related <id>          Find related decisions, lessons, and failures
  decisions             List all architectural decisions
  lessons               List all reusable engineering lessons
  verify <id>           Promote proposed knowledge to VERIFIED state
  reject <id>           Reject proposed knowledge
  supersede <id>        Retire knowledge with a verified replacement and reason
  history <id>          View event audit history for a task or knowledge node
  doctor                Run diagnostics and optionally rebuild index (--rebuild)
  mcp                   Launch Model Context Protocol (MCP) server over stdio

Flags for commands:
  Use '<command> --help' for details on specific command arguments.`)
}

func (a *App) runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	name := fs.String("name", filepath.Base(a.rootDir), "Project name")
	desc := fs.String("desc", "Engineering project managed with Knodexa", "Project description")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if a.store.IsInitialized() {
		fmt.Printf("Knodexa is already initialized in %s\n", a.store.KnodexaDir())
		return nil
	}

	cfg, err := a.store.Init(*name, *desc)
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	// Initialize SQLite
	dbPath := filepath.Join(a.store.KnodexaDir(), "cache.db")
	db, err := index.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize sqlite: %w", err)
	}
	defer db.Close()

	idx := index.NewIndexer(db, a.store)
	if err := idx.Sync(); err != nil {
		return fmt.Errorf("failed to sync index: %w", err)
	}

	fmt.Printf("Initialized Knodexa project %q in %s\n", cfg.Name, a.store.KnodexaDir())
	return nil
}

func (a *App) runTask(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task requires subcommand: 'create', 'show', or 'list'")
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "create":
		fs := flag.NewFlagSet("task create", flag.ContinueOnError)
		id := fs.String("id", "", "Stable task ID (any valid name; numbering has no inheritance meaning)")
		parent := fs.String("parent", "", "Parent task ID (optional)")
		refs := fs.String("approach", "", "Comma-separated source task IDs for explicitly reused approaches")
		newWork := fs.Bool("new", false, "Explicitly create independent work despite matching existing work")
		title := fs.String("title", "", "Task title")
		goal := fs.String("goal", "", "Task goal")
		constraints := fs.String("constraints", "", "Comma-separated list of constraints")
		files := fs.String("files", "", "Comma-separated list of affected files")
		var aliases repeatedStrings
		fs.Var(&aliases, "alias", "Phrase referring to this work (repeatable)")

		if err := fs.Parse(rest); err != nil {
			return err
		}
		if *id == "" || *title == "" {
			return fmt.Errorf("both --id and --title are required")
		}
		if a.db != nil {
			if _, err := a.db.GetTaskData(*id); err == nil {
				return fmt.Errorf("task %s already exists; resolve or resume it instead of overwriting it", *id)
			} else if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			if !*newWork && *parent == "" && *refs == "" {
				match, err := identity.Resolve(a.db, identity.Request{Text: *title + " " + *goal})
				if err != nil {
					return err
				}
				for _, candidate := range match.Candidates {
					if candidate.Strong {
						return fmt.Errorf("description matches existing work %s (%s); run knodexa resolve first, then continue with --parent or --approach, or explicitly request independent work with --new", candidate.TaskID, candidate.Title)
					}
				}
			}
		}

		var cList []string
		if *constraints != "" {
			for _, c := range strings.Split(*constraints, ",") {
				if t := strings.TrimSpace(c); t != "" {
					cList = append(cList, t)
				}
			}
		}

		var fList []string
		if *files != "" {
			for _, f := range strings.Split(*files, ",") {
				if t := strings.TrimSpace(f); t != "" {
					fList = append(fList, t)
				}
			}
		}

		if err := identity.ValidateAliases(aliases); err != nil {
			return err
		}
		now := time.Now().UTC()
		task := &model.TaskData{
			Aliases:      aliases,
			ID:           *id,
			ParentID:     *parent,
			Title:        *title,
			Goal:         *goal,
			Status:       model.TaskStatusActive,
			CreatedAt:    now,
			UpdatedAt:    now,
			Constraints:  cList,
			TouchesFiles: fList,
		}

		if *refs != "" {
			for _, source := range strings.Split(*refs, ",") {
				source = strings.TrimSpace(source)
				if source != "" {
					task.ApproachRefs = append(task.ApproachRefs, source)
				}
			}
		}
		if a.db != nil {
			if err := approach.ValidateTask(a.db, task); err != nil {
				return err
			}
		}

		if err := a.store.SaveTask(task); err != nil {
			return err
		}

		// Append creation event
		_ = a.store.AppendEvent(&model.Event{
			EventID:   fmt.Sprintf("evt-%d", time.Now().UnixNano()),
			Timestamp: now,
			Actor:     "user",
			Action:    "CREATE_TASK",
			TargetID:  *id,
			Payload: map[string]interface{}{
				"title": *title,
			},
		})

		if a.indexer != nil {
			if err := a.indexer.Sync(); err != nil {
				return err
			}
		}

		fmt.Printf("Task %s created successfully.\n", *id)
		return nil

	case "show":
		if len(rest) == 0 {
			return fmt.Errorf("task show requires a task ID")
		}
		taskID := rest[0]
		task, err := a.store.ReadTask(taskID)
		if err != nil {
			return err
		}
		fmt.Printf("Task ID:    %s\n", task.ID)
		if task.ParentID != "" {
			fmt.Printf("Parent ID:  %s\n", task.ParentID)
		}
		fmt.Printf("Title:      %s\n", task.Title)
		fmt.Printf("Status:     %s\n", task.Status)
		fmt.Printf("Goal:       %s\n", task.Goal)
		if len(task.Constraints) > 0 {
			fmt.Println("Constraints:")
			for _, c := range task.Constraints {
				fmt.Printf("  * %s\n", c)
			}
		}
		if len(task.TouchesFiles) > 0 {
			fmt.Println("Files:")
			for _, f := range task.TouchesFiles {
				fmt.Printf("  * %s\n", f)
			}
		}
		return nil

	case "list":
		tasks, err := a.store.ListTasks()
		if err != nil {
			return err
		}
		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}
		fmt.Println("TASKS:")
		for _, t := range tasks {
			statusMark := "●"
			if t.Status == model.TaskStatusCompleted {
				statusMark = "✓"
			}
			parentInfo := ""
			if t.ParentID != "" {
				parentInfo = fmt.Sprintf(" (parent: %s)", t.ParentID)
			}
			fmt.Printf("  [%s] %s %s - %s%s\n", statusMark, t.ID, t.Status, t.Title, parentInfo)
		}
		return nil

	default:
		return fmt.Errorf("unknown task subcommand: %s", sub)
	}
}

func (a *App) runContext(args []string) error { return a.runRehydrate(args, false) }
func (a *App) runRehydrate(args []string, resume bool) error {
	command := "context"
	if resume {
		command = "resume"
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	full := fs.Bool("full", false, "Include full rationale and lesson descriptions")
	var accept *bool
	var reopen *bool
	previous := ""
	if resume {
		fs.StringVar(&previous, "from-task", "", "Previous focus; report its conventions to set aside")
		accept = fs.Bool("accept-baseline", false, "Explicitly replace the stored baseline with current HEAD")
		reopen = fs.Bool("reopen", false, "Explicitly reactivate completed work; otherwise restore it without modifying its task")
	}
	jsonOutput := fs.Bool("json", false, "Output structured JSON instead of formatted text")
	budget := fs.Int("budget", 1500, "Maximum token budget")

	if err := parseInterspersed(fs, args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return fmt.Errorf("task ID is required: knodexa context <task_id>")
	}
	if len(remaining) != 1 {
		return fmt.Errorf("%s requires exactly one task ID", command)
	}
	if *budget <= 0 {
		return fmt.Errorf("token budget must be positive")
	}
	taskID := remaining[0]

	if a.context == nil {
		return fmt.Errorf("knodexa is not initialized in this directory; run 'knodexa init' first")
	}

	var pkg *context.Package
	var err error
	if resume {
		pkg, err = continuity.NewService(a.store, a.db, a.indexer, a.context, a.git).Resume(taskID, *budget, continuity.ResumeOptions{Full: *full, AcceptBaseline: *accept, Reopen: *reopen, PreviousTaskID: previous})
	} else {
		pkg, err = a.context.BuildWithOptions(taskID, *budget, context.BuildOptions{Compact: !*full})
	}
	if err != nil {
		return fmt.Errorf("failed to rehydrate context: %w", err)
	}

	if *jsonOutput {
		out, err := json.Marshal(pkg)
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	if pkg.BudgetExceeded {
		fmt.Fprintf(os.Stderr, "Warning: essential context exceeds token budget (%d estimated human-text tokens, budget %d)\n", pkg.EstimatedTokens, pkg.TokenBudget)
	}
	fmt.Println(pkg.FormatHuman())
	return nil
}

// parseInterspersed accepts flags on either side of positional arguments.
// FlagSet.Parse otherwise silently stops at the first positional argument.
func parseInterspersed(fs *flag.FlagSet, args []string) error {
	var options, positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-h" || arg == "--help" {
			return fs.Parse([]string{"-h"})
		}
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positional = append(positional, arg)
			continue
		}
		options = append(options, arg)
		name := strings.TrimLeft(arg, "-")
		if strings.Contains(name, "=") {
			continue
		}
		f := fs.Lookup(name)
		if f == nil {
			return fmt.Errorf("unknown flag: %s", arg)
		}
		if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
			continue
		}
		if i+1 >= len(args) {
			return fmt.Errorf("flag needs an argument: %s", arg)
		}
		i++
		options = append(options, args[i])
	}
	options = append(options, "--")
	return fs.Parse(append(options, positional...))
}

func (a *App) runTree(args []string) error {
	if a.db == nil {
		return fmt.Errorf("knodexa is not initialized in this directory")
	}

	proj, err := a.store.ReadProject()
	projectName := "Project Brain"
	if err == nil && proj != nil {
		projectName = proj.Name
	}

	fmt.Printf("KNODEXA\n\n%s\n", projectName)
	fmt.Println("├── Project Rules")
	fmt.Println("│   └── AGENTS.md")

	tasks, _ := a.store.ListTasks()
	// Group top-level tasks
	topTasks := make([]model.TaskData, 0)
	childrenMap := make(map[string][]model.TaskData)
	for _, t := range tasks {
		if t.ParentID == "" {
			topTasks = append(topTasks, t)
		} else {
			childrenMap[t.ParentID] = append(childrenMap[t.ParentID], t)
		}
	}

	for i, top := range topTasks {
		isLastTop := i == len(topTasks)-1 && len(childrenMap) == 0
		prefix := "├──"
		if isLastTop {
			prefix = "└──"
		}
		mark := "●"
		if top.Status == model.TaskStatusCompleted {
			mark = "✓"
		}
		fmt.Printf("%s Task %s (%s) %s\n", prefix, top.ID, top.Title, mark)

		children := childrenMap[top.ID]
		for j, child := range children {
			isLastChild := j == len(children)-1
			childPrefix := "│   ├──"
			if isLastChild {
				childPrefix = "│   └──"
			}
			cMark := "●"
			if child.Status == model.TaskStatusCompleted {
				cMark = "✓"
			}
			fmt.Printf("%s %s (%s) %s\n", childPrefix, child.ID, child.Title, cMark)
		}
	}

	fmt.Println("├── Skills")
	fmt.Println("│   └── (standard skills directory)")
	fmt.Println("└── Shared Knowledge")

	decisions, _ := a.store.ListDecisions()
	lessons, _ := a.store.ListLessons()
	fmt.Printf("    ├── Decisions (%d)\n", len(decisions))
	fmt.Printf("    ├── Lessons (%d)\n", len(lessons))
	fmt.Println("    └── Cross-task relationships")

	return nil
}

func (a *App) runDecisions(args []string) error {
	decisions, err := a.store.ListDecisions()
	if err != nil {
		return err
	}
	if len(decisions) == 0 {
		fmt.Println("No decisions recorded.")
		return nil
	}

	fmt.Println("ARCHITECTURAL DECISIONS:")
	for _, d := range decisions {
		fm := d.Frontmatter
		taskInfo := ""
		if fm.TaskID != "" {
			taskInfo = fmt.Sprintf(" [Task: %s]", fm.TaskID)
		}
		fmt.Printf("  * %s (%s)%s: %s (Confidence: %.2f)\n", fm.ID, fm.Status, taskInfo, fm.Title, fm.Confidence)
	}
	return nil
}

func (a *App) runLessons(args []string) error {
	lessons, err := a.store.ListLessons()
	if err != nil {
		return err
	}
	if len(lessons) == 0 {
		fmt.Println("No lessons recorded.")
		return nil
	}

	fmt.Println("REUSABLE LESSONS:")
	for _, l := range lessons {
		fm := l.Frontmatter
		taskInfo := ""
		if fm.SourceTaskID != "" {
			taskInfo = fmt.Sprintf(" [Source: %s]", fm.SourceTaskID)
		}
		fmt.Printf("  * %s (%s)%s: %s\n      Applicability: %s\n", fm.ID, fm.Status, taskInfo, fm.Title, fm.Applicability)
	}
	return nil
}

func (a *App) runVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	evidence := fs.String("evidence", "", "Comma-separated list of supporting evidence IDs")
	notes := fs.String("notes", "", "Verification notes")
	if err := parseInterspersed(fs, args); err != nil {
		return err
	}

	rem := fs.Args()
	if len(rem) == 0 {
		return fmt.Errorf("node ID is required: knodexa verify <id> --evidence=<E-001>")
	}
	nodeID := rem[0]

	if *evidence == "" {
		return fmt.Errorf("verification requires at least one evidence ID via --evidence=<id>")
	}

	var evList []string
	for _, e := range strings.Split(*evidence, ",") {
		if t := strings.TrimSpace(e); t != "" {
			evList = append(evList, t)
		}
	}

	if a.verify == nil {
		return fmt.Errorf("knodexa engine not initialized")
	}

	if err := a.verify.VerifyNode(nodeID, evList, *notes, "user"); err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	fmt.Printf("Node %s successfully promoted to VERIFIED state.\n", nodeID)
	return nil
}

func (a *App) runReject(args []string) error {
	fs := flag.NewFlagSet("reject", flag.ContinueOnError)
	reason := fs.String("reason", "", "Reason for rejection")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rem := fs.Args()
	if len(rem) == 0 {
		return fmt.Errorf("node ID is required: knodexa reject <id> --reason=...")
	}
	nodeID := rem[0]

	if a.verify == nil {
		return fmt.Errorf("knodexa engine not initialized")
	}

	if err := a.verify.RejectNode(nodeID, *reason, "user"); err != nil {
		return fmt.Errorf("rejection failed: %w", err)
	}

	fmt.Printf("Node %s marked as REJECTED.\n", nodeID)
	return nil
}

func (a *App) runRelated(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required: knodexa related <task_id>")
	}
	taskID := args[0]

	if a.db == nil {
		return fmt.Errorf("knodexa engine not initialized")
	}

	task, _ := a.store.ReadTask(taskID)
	var touchedFiles []string
	if task != nil {
		touchedFiles = task.TouchesFiles
	}

	lessons, _ := a.db.GetRelatedLessons(taskID, touchedFiles, nil, 5)
	decisions, _ := a.db.GetDecisionsForTask(taskID)
	failures, _ := a.db.GetFailuresForTask(taskID)

	fmt.Printf("RELATED KNOWLEDGE FOR %s:\n\n", taskID)

	if len(decisions) > 0 {
		fmt.Println("Decisions:")
		for _, d := range decisions {
			fmt.Printf("  * %s: %s (%s)\n", d.ID, d.Title, d.Status)
		}
		fmt.Println()
	}

	if len(failures) > 0 {
		fmt.Println("Failures to Avoid:")
		for _, f := range failures {
			fmt.Printf("  * %s: %s (Cause: %s)\n", f.ID, f.Summary, f.CausalDetails)
		}
		fmt.Println()
	}

	if len(lessons) > 0 {
		fmt.Println("Relevant Lessons:")
		for _, l := range lessons {
			fmt.Printf("  * %s: %s (From %s, Confidence: %.2f)\n", l.ID, l.Title, l.ParentID, l.Confidence)
		}
		fmt.Println()
	}

	if len(decisions) == 0 && len(failures) == 0 && len(lessons) == 0 {
		fmt.Println("No related knowledge items found.")
	}

	return nil
}

func (a *App) runHistory(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("target ID is required: knodexa history <id>")
	}
	targetID := args[0]

	events, err := a.store.ReadEvents()
	if err != nil {
		return err
	}

	var matched []model.Event
	for _, ev := range events {
		if ev.TargetID == targetID {
			matched = append(matched, ev)
		}
	}

	if len(matched) == 0 {
		fmt.Printf("No event history found for %s.\n", targetID)
		return nil
	}

	fmt.Printf("EVENT HISTORY FOR %s:\n", targetID)
	for _, ev := range matched {
		fmt.Printf("  [%s] %s by %s (Event ID: %s)\n",
			ev.Timestamp.Format("2006-01-02 15:04:05"), ev.Action, ev.Actor, ev.EventID)
	}
	return nil
}

func (a *App) runDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	rebuild := fs.Bool("rebuild", false, "Rebuild SQLite cache from canonical files")
	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Println("KNODEXA DIAGNOSTICS:")

	if !a.store.IsInitialized() {
		fmt.Println("  [FAIL] .knodexa directory not found. Run 'knodexa init'.")
		return nil
	}
	fmt.Printf("  [PASS] Storage initialized at: %s\n", a.store.KnodexaDir())

	proj, err := a.store.ReadProject()
	if err != nil {
		fmt.Printf("  [FAIL] project.yaml invalid: %v\n", err)
	} else {
		fmt.Printf("  [PASS] Project manifest: %s (ID: %s)\n", proj.Name, proj.ID)
	}

	if a.git.IsRepo() {
		head, _ := a.git.HeadCommit()
		branch, _ := a.git.CurrentBranch()
		dirty, _ := a.git.IsDirty()
		fmt.Printf("  [PASS] Git repository: branch=%s, commit=%s, dirty=%t\n", branch, head, dirty)
	} else {
		fmt.Println("  [WARN] Not inside a Git repository.")
	}

	tasks, _ := a.store.ListTasks()
	decisions, _ := a.store.ListDecisions()
	lessons, _ := a.store.ListLessons()
	evidence, _ := a.store.ListEvidence()
	fmt.Printf("  [INFO] Canonical count: %d tasks, %d decisions, %d lessons, %d evidence\n",
		len(tasks), len(decisions), len(lessons), len(evidence))

	if a.db != nil {
		fmt.Println("  [PASS] SQLite index connected.")
		if *rebuild {
			fmt.Println("  [ACTION] Rebuilding SQLite index from scratch...")
			if err := a.indexer.Rebuild(); err != nil {
				return fmt.Errorf("rebuild failed: %w", err)
			}
			fmt.Println("  [PASS] Rebuild complete.")
		}
	} else {
		fmt.Println("  [FAIL] SQLite index not connected.")
	}

	return nil
}

func (a *App) runShow(args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output full structured document")
	if err := parseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("show requires exactly one decision or lesson ID")
	}
	if a.context == nil {
		return fmt.Errorf("knodexa is not initialized; run 'knodexa init' first")
	}
	detail, err := a.context.GetDetail(fs.Arg(0))
	if err != nil {
		return err
	}
	if *jsonOutput {
		data, err := json.Marshal(detail)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}
	switch doc := detail.(type) {
	case *model.DecisionDocument:
		fmt.Printf("# %s (%s)\n%s\n", doc.Frontmatter.Title, doc.Frontmatter.Status, doc.Body)
		fmt.Printf("evidence: %s\n", strings.Join(doc.Frontmatter.EvidenceIDs, ", "))
	case *model.LessonDocument:
		fmt.Printf("# %s (%s)\nWhen: %s\n%s\n", doc.Frontmatter.Title, doc.Frontmatter.Status, doc.Frontmatter.Applicability, doc.Body)
		fmt.Printf("evidence: %s\n", strings.Join(doc.Frontmatter.EvidenceIDs, ", "))
	}
	return nil
}
