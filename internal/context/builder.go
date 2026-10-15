package context

import (
	"fmt"
	"strings"

	"knodexa/internal/approach"
	"knodexa/internal/git"
	"knodexa/internal/index"
	"knodexa/internal/storage"
)

// Builder orchestrates context rehydration for coding agents.
type Builder struct {
	db    *index.DB
	store *storage.Storage
	git   *git.Inspector
}

// NewBuilder creates a new Context Builder.
func NewBuilder(db *index.DB, store *storage.Storage, git *git.Inspector) *Builder {
	return &Builder{
		db:    db,
		store: store,
		git:   git,
	}
}

// BuildOptions selects compact or full context and optionally reuses a checkpoint.
type BuildOptions struct {
	Boundary       *Boundary
	ApproachStatus *approach.Report
	Compact        bool
	Checkpoint     *git.Checkpoint
}

func (b *Builder) Build(taskID string, tokenBudget int) (*Package, error) {
	return b.BuildWithOptions(taskID, tokenBudget, BuildOptions{})
}
func (b *Builder) BuildCompact(taskID string, tokenBudget int) (*Package, error) {
	return b.BuildWithOptions(taskID, tokenBudget, BuildOptions{Compact: true})
}

// BuildWithOptions compiles the minimal, high-signal engineering context package for taskID.
func (b *Builder) BuildWithOptions(taskID string, tokenBudget int, options BuildOptions) (*Package, error) {
	if tokenBudget <= 0 {
		tokenBudget = 1500
	}

	// Cached canonical bytes preserve full task metadata without directory scans.
	taskData, err := b.db.GetTaskData(taskID)
	if err != nil {
		return nil, fmt.Errorf("read task %s: %w", taskID, err)
	}

	pkg := &Package{
		Task: TaskSummary{
			ID:       taskData.ID,
			ParentID: taskData.ParentID,
			Title:    taskData.Title,
			Goal:     taskData.Goal,
			Status:   string(taskData.Status),
		},
		AcceptedConstraints:  make([]string, 0),
		PreviousDecisions:    make([]DecisionSummary, 0),
		PreviousFailures:     make([]FailureSummary, 0),
		RelevantLessons:      make([]LessonSummary, 0),
		RequiredVerification: make([]string, 0),
	}

	pkg.EstablishedApproach, pkg.ApproachSelection, err = approach.Select(b.db, taskID)
	if err != nil {
		return nil, fmt.Errorf("restore established approach: %w", err)
	}
	pkg.ApproachStatus = options.ApproachStatus
	pkg.ContextBoundary = options.Boundary
	pkg.DetailLevel = "full"
	if options.Compact {
		pkg.DetailLevel = "compact"
		pkg.DetailHint = "Full rationale and evidence IDs: knodexa show <id> or knodexa_get_detail"
	}

	// 2. Constraints (Upstream & Task)
	constraints, err := b.db.GetUpstreamConstraints(taskID)
	if err != nil {
		return nil, fmt.Errorf("retrieve constraints: %w", err)
	}
	if err == nil {
		seenC := make(map[string]bool)
		for _, c := range constraints {
			if !seenC[c] {
				seenC[c] = true
				pkg.AcceptedConstraints = append(pkg.AcceptedConstraints, c)
			}
		}
	}
	for _, c := range taskData.Constraints {
		found := false
		for _, existing := range pkg.AcceptedConstraints {
			if existing == c {
				found = true
				break
			}
		}
		if !found {
			pkg.AcceptedConstraints = append(pkg.AcceptedConstraints, c)
		}
	}

	// Selected agreements retain their original acceptance criteria even when
	// the new task is outside the source task's hierarchy.
	for _, contract := range pkg.EstablishedApproach {
		source, err := b.db.GetTaskData(contract.SourceTask)
		if err != nil {
			return nil, err
		}
		sourceConstraints, err := b.db.GetUpstreamConstraints(source.ID)
		if err != nil {
			return nil, err
		}
		pkg.AcceptedConstraints = appendUnique(pkg.AcceptedConstraints, sourceConstraints...)
		pkg.AcceptedConstraints = appendUnique(pkg.AcceptedConstraints, source.Constraints...)
		pkg.RequiredVerification = appendUnique(pkg.RequiredVerification, source.RequiredVerification...)
	}
	// 3. Previous Decisions (VERIFIED only)
	decisions, err := b.db.GetDecisionsForTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("retrieve decisions: %w", err)
	}
	for _, contract := range pkg.EstablishedApproach {
		sourceDecisions, err := b.db.GetDecisionsForTask(contract.SourceTask)
		if err != nil {
			return nil, err
		}
		for _, decision := range sourceDecisions {
			if decision.ParentID != contract.SourceTask {
				continue
			}
			found := false
			for _, existing := range decisions {
				if existing.ID == decision.ID {
					found = true
					break
				}
			}
			if !found {
				decisions = append(decisions, decision)
			}
		}
	}
	if err == nil {
		for _, d := range decisions {
			doc, err := b.db.GetDecisionDocument(d.ID)
			if err != nil {
				return nil, err
			}
			for _, constraint := range decisionConstraints(doc) {
				found := false
				for _, existing := range pkg.AcceptedConstraints {
					if existing == constraint {
						found = true
						break
					}
				}
				if !found {
					pkg.AcceptedConstraints = append(pkg.AcceptedConstraints, constraint)
				}
			}
			summary := doc.Frontmatter.Summary
			if summary == "" {
				summary = firstParagraph(doc.Body)
			}
			rationale := strings.TrimSpace(doc.Body)
			if options.Compact {
				rationale = ""
			} else {
				summary = ""
			}
			pkg.PreviousDecisions = append(pkg.PreviousDecisions, DecisionSummary{ID: d.ID, Title: d.Title, Status: d.Status, Confidence: d.Confidence, Summary: summary, Rationale: rationale, SourceTask: doc.Frontmatter.TaskID, EvidenceIDs: doc.Frontmatter.EvidenceIDs})
		}
	}

	// 4. Previous Failures
	failures, err := b.db.GetFailuresForTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("retrieve failures: %w", err)
	}
	for _, contract := range pkg.EstablishedApproach {
		sourceFailures, err := b.db.GetFailuresForTask(contract.SourceTask)
		if err != nil {
			return nil, err
		}
		for _, failure := range sourceFailures {
			if failure.TaskID != contract.SourceTask {
				continue
			}
			found := false
			for _, existing := range failures {
				if existing.ID == failure.ID {
					found = true
					break
				}
			}
			if !found {
				failures = append(failures, failure)
			}
		}
	}
	if err == nil {
		for _, f := range failures {
			pkg.PreviousFailures = append(pkg.PreviousFailures, FailureSummary{
				SourceTask:    f.TaskID,
				ID:            f.ID,
				Summary:       f.Summary,
				CausalDetails: f.CausalDetails,
			})
		}
	}

	// 5. Relevant Lessons (Cross-task experience transfer)
	sources := []string{}
	for _, contract := range pkg.EstablishedApproach {
		sources = append(sources, contract.SourceTask)
	}
	lessons, err := b.db.GetRelatedLessonsForSources(taskID, sources, taskData.TouchesFiles, nil, 5)
	if err != nil {
		return nil, fmt.Errorf("retrieve lessons: %w", err)
	}
	if err == nil {
		for _, l := range lessons {
			doc, err := b.db.GetLessonDocument(l.ID)
			if err != nil {
				return nil, fmt.Errorf("read lesson %s: %w", l.ID, err)
			}
			description := strings.TrimSpace(l.Description)
			if options.Compact {
				description = doc.Frontmatter.Summary
				if description == "" {
					description = firstParagraph(doc.Body)
				}
			}
			relevance := "medium"
			if len(taskData.TouchesFiles) > 0 {
				for _, tf := range taskData.TouchesFiles {
					for _, lf := range l.TouchesFiles {
						if tf == lf {
							relevance = "high"
							break
						}
					}
				}
			}

			pkg.RelevantLessons = append(pkg.RelevantLessons, LessonSummary{
				MatchReason:        l.MatchReason,
				EvidenceIDs:        doc.Frontmatter.EvidenceIDs,
				ID:                 l.ID,
				Title:              l.Title,
				Description:        description,
				Applicability:      doc.Frontmatter.Applicability,
				SourceTask:         l.ParentID,
				Confidence:         l.Confidence,
				Relevance:          relevance,
				VerificationStatus: string(l.VerificationStatus),
			})
		}
	}

	// 6. Baseline Git Checkpoint
	chk := options.Checkpoint
	if chk == nil {
		chk, err = b.git.GetCheckpoint()
	}
	if err == nil && chk != nil && chk.CommitSHA != "none" && chk.CommitSHA != "" {
		commit := chk.CommitSHA
		if len(commit) > 7 {
			commit = commit[:7]
		}
		pkg.Baseline = BaselineInfo{
			Commit: commit,
			Branch: chk.Branch,
			Dirty:  chk.IsDirty,
		}
	} else if taskData.BaselineCommit != "" {
		pkg.Baseline = BaselineInfo{
			Commit: taskData.BaselineCommit,
		}
	} else {
		pkg.Baseline = BaselineInfo{
			Commit: "unknown",
		}
	}

	pkg.Baseline.StoredCommit = taskData.BaselineCommit
	if taskData.BaselineCommit != "" && pkg.Baseline.Commit != "unknown" && pkg.Baseline.Commit != "none" {
		pkg.Baseline.Mismatch = !strings.HasPrefix(taskData.BaselineCommit, pkg.Baseline.Commit) && !strings.HasPrefix(pkg.Baseline.Commit, taskData.BaselineCommit)
	}

	// 7. Required Verification Criteria
	if len(taskData.RequiredVerification) > 0 {
		pkg.RequiredVerification = appendUnique(pkg.RequiredVerification, taskData.RequiredVerification...)
	} else if len(pkg.RequiredVerification) == 0 {
		pkg.RequiredVerification = []string{
			"Correctness test suite must pass",
			"No regressions in existing functionality",
		}
	}

	// 8. Apply Budget Pruning
	b.applyTokenBudget(pkg, tokenBudget)

	return pkg, nil
}

// applyTokenBudget prunes optional context using a human-text token estimate.
// Essential context is preserved, with BudgetExceeded reporting an impossible budget.
func (b *Builder) applyTokenBudget(pkg *Package, maxTokens int) {
	pkg.TokenBudget = maxTokens
	rawText := pkg.FormatHuman()
	tokens := EstimateTokens(rawText)

	// If within budget, calculate and return
	if tokens <= maxTokens {
		pkg.EstimatedTokens = tokens
		return
	}

	// Step 1: Prune lowest-priority relevant lessons if needed
	for len(pkg.RelevantLessons) > 0 && EstimateTokens(pkg.FormatHuman()) > maxTokens {
		item := pkg.RelevantLessons[len(pkg.RelevantLessons)-1]
		deferMemory(pkg, item.ID, "lesson", item.SourceTask)
		pkg.RelevantLessons = pkg.RelevantLessons[:len(pkg.RelevantLessons)-1]
		pkg.ContextPruned = true
	}

	// Step 2: Prune decisions if still over budget
	for len(pkg.PreviousDecisions) > 0 && EstimateTokens(pkg.FormatHuman()) > maxTokens {
		item := pkg.PreviousDecisions[len(pkg.PreviousDecisions)-1]
		deferMemory(pkg, item.ID, "decision", item.SourceTask)
		pkg.PreviousDecisions = pkg.PreviousDecisions[:len(pkg.PreviousDecisions)-1]
		pkg.ContextPruned = true
	}

	pkg.EstimatedTokens = EstimateTokens(pkg.FormatHuman())
	pkg.BudgetExceeded = pkg.EstimatedTokens > maxTokens
}

func deferMemory(pkg *Package, id, kind, source string) {
	pkg.DeferredCount++
	if len(pkg.DeferredMemory) < 8 {
		pkg.DeferredMemory = append(pkg.DeferredMemory, DeferredMemory{ID: id, Kind: kind, SourceTask: source, Reason: "token_budget"})
	}
}

func appendUnique(values []string, additions ...string) []string {
	for _, value := range additions {
		found := false
		for _, existing := range values {
			if existing == value {
				found = true
				break
			}
		}
		if !found {
			values = append(values, value)
		}
	}
	return values
}
