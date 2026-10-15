package continuity

import (
	"fmt"
	"knodexa/internal/approach"
	"knodexa/internal/context"
	"knodexa/internal/git"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"time"
)

// Service combines resume, checkpoint comparison, and context retrieval.
type Service struct {
	db      *index.DB
	store   *storage.Storage
	idx     *index.Indexer
	builder *context.Builder
	git     *git.Inspector
}

func NewService(store *storage.Storage, db *index.DB, idx *index.Indexer, builder *context.Builder, inspector *git.Inspector) *Service {
	return &Service{db: db, store: store, idx: idx, builder: builder, git: inspector}
}

type ResumeOptions struct {
	PreviousTaskID string
	Full           bool
	AcceptBaseline bool
	Reopen         bool
}

// Resume preserves an existing baseline unless the caller explicitly accepts a
// new one. Comparing after silently overwriting it would hide branch drift.
func (s *Service) Resume(taskID string, budget int, options ResumeOptions) (*context.Package, error) {
	if budget <= 0 {
		return nil, fmt.Errorf("token budget must be positive")
	}
	if err := s.idx.Sync(); err != nil {
		return nil, err
	}
	task, err := s.db.GetTaskData(taskID)
	if err != nil {
		return nil, err
	}
	boundary, err := s.boundary(taskID, options.PreviousTaskID)
	if err != nil {
		return nil, err
	}
	checkpoint, err := s.git.GetCheckpoint()
	if err != nil {
		return nil, err
	}
	changed := false
	if task.Status != model.TaskStatusCompleted || options.Reopen {
		changed = task.Status != model.TaskStatusActive
		task.Status = model.TaskStatusActive
	}
	if (task.Status != model.TaskStatusCompleted || options.AcceptBaseline) && checkpoint.CommitSHA != "" && checkpoint.CommitSHA != "none" && (task.BaselineCommit == "" || options.AcceptBaseline) {
		if task.BaselineCommit != checkpoint.CommitSHA {
			task.BaselineCommit = checkpoint.CommitSHA
			changed = true
		}
	}
	if changed {
		task.UpdatedAt = time.Now().UTC()
		if err := s.store.SaveTask(task); err != nil {
			return nil, err
		}
		if err := s.idx.Sync(); err != nil {
			return nil, err
		}
	}
	report, err := approach.Check(s.store, s.db, taskID)
	if err != nil {
		return nil, err
	}
	if len(report.Results) == 1 && report.Results[0].Kind == "missing-approach" {
		report = nil
	}
	pkg, err := s.builder.BuildWithOptions(taskID, budget, context.BuildOptions{Compact: !options.Full, Checkpoint: checkpoint, ApproachStatus: report, Boundary: boundary})
	if err != nil {
		return nil, err
	}
	if err := s.store.SaveFocus(taskID); err != nil {
		return nil, err
	}
	return pkg, nil
}
