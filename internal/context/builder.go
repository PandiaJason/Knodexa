package context

import (
	"fmt"

	"knodexa/internal/git"
	"knodexa/internal/index"
	"knodexa/internal/model"
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

// Build compiles the minimal, high-signal engineering context package for taskID.
func (b *Builder) Build(taskID string, tokenBudget int) (*Package, error) {
	if tokenBudget <= 0 {
		tokenBudget = 1500
	}

	// 1. Retrieve task (check SQLite index first for sub-millisecond retrieval)
	var taskData *model.TaskData
	node, nErr := b.db.GetNode(taskID)
	if nErr == nil && node != nil {
		taskData = &model.TaskData{
			ID:           node.ID,
			ParentID:     node.ParentID,
			Title:        node.Title,
			Goal:         node.Description,
			Status:       model.TaskStatus(node.Status),
			CreatedAt:    node.CreatedAt,
			UpdatedAt:    node.UpdatedAt,
			TouchesFiles: node.TouchesFiles,
		}
	} else {
		// Fallback to disk scan
		var err error
		taskData, err = b.store.ReadTask(taskID)
		if err != nil {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
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

	// 2. Constraints (Upstream & Task)
	constraints, err := b.db.GetUpstreamConstraints(taskID)
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

	// 3. Previous Decisions (VERIFIED only)
	decisions, err := b.db.GetDecisionsForTask(taskID)
	if err == nil {
		for _, d := range decisions {
			pkg.PreviousDecisions = append(pkg.PreviousDecisions, DecisionSummary{
				ID:         d.ID,
				Title:      d.Title,
				Status:     d.Status,
				Confidence: d.Confidence,
			})
		}
	}

	// 4. Previous Failures
	failures, err := b.db.GetFailuresForTask(taskID)
	if err == nil {
		for _, f := range failures {
			pkg.PreviousFailures = append(pkg.PreviousFailures, FailureSummary{
				ID:            f.ID,
				Summary:       f.Summary,
				CausalDetails: f.CausalDetails,
			})
		}
	}

	// 5. Relevant Lessons (Cross-task experience transfer)
	lessons, err := b.db.GetRelatedLessons(taskID, taskData.TouchesFiles, nil, 5)
	if err == nil {
		for _, l := range lessons {
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
				ID:                 l.ID,
				Title:              l.Title,
				SourceTask:         l.ParentID,
				Confidence:         l.Confidence,
				Relevance:          relevance,
				VerificationStatus: string(l.VerificationStatus),
			})
		}
	}

	// 6. Baseline Git Checkpoint
	chk, err := b.git.GetCheckpoint()
	if err == nil && chk != nil {
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

	// 7. Required Verification Criteria
	if len(taskData.RequiredVerification) > 0 {
		pkg.RequiredVerification = taskData.RequiredVerification
	} else {
		pkg.RequiredVerification = []string{
			"Correctness test suite must pass",
			"No regressions in existing functionality",
		}
	}

	// 8. Apply Budget Pruning
	b.applyTokenBudget(pkg, tokenBudget)

	return pkg, nil
}

// applyTokenBudget ensures the serialized human representation stays within tokenBudget.
func (b *Builder) applyTokenBudget(pkg *Package, maxTokens int) {
	rawText := pkg.FormatHuman()
	tokens := EstimateTokens(rawText)

	// If within budget, calculate and return
	if tokens <= maxTokens {
		pkg.EstimatedTokens = tokens
		return
	}

	// Step 1: Prune lowest-priority relevant lessons if needed
	for len(pkg.RelevantLessons) > 1 && EstimateTokens(pkg.FormatHuman()) > maxTokens {
		pkg.RelevantLessons = pkg.RelevantLessons[:len(pkg.RelevantLessons)-1]
	}

	// Step 2: Prune decisions if still over budget
	for len(pkg.PreviousDecisions) > 1 && EstimateTokens(pkg.FormatHuman()) > maxTokens {
		pkg.PreviousDecisions = pkg.PreviousDecisions[:len(pkg.PreviousDecisions)-1]
	}

	pkg.EstimatedTokens = EstimateTokens(pkg.FormatHuman())
}
