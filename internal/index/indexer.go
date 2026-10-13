package index

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"knodexa/internal/model"
	"knodexa/internal/storage"
)

// Indexer coordinates the synchronization between canonical files and the SQLite index.
type Indexer struct {
	db    *DB
	store *storage.Storage
}

// NewIndexer creates a new Indexer instance.
func NewIndexer(db *DB, store *storage.Storage) *Indexer {
	return &Indexer{
		db:    db,
		store: store,
	}
}

// ComputeHash returns a sha256 hex string for the provided bytes.
func ComputeHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// Rebuild wipes all index tables and re-populates the entire database from canonical files.
func (idx *Indexer) Rebuild() error {
	idx.db.mu.Lock()
	defer idx.db.mu.Unlock()

	if err := idx.db.dropSchema(); err != nil {
		return fmt.Errorf("failed to drop existing schema: %w", err)
	}
	if err := idx.db.initSchema(); err != nil {
		return fmt.Errorf("failed to re-initialize schema: %w", err)
	}

	return idx.syncInternal()
}

// Sync updates the SQLite index from canonical files.
func (idx *Indexer) Sync() error {
	idx.db.mu.Lock()
	defer idx.db.mu.Unlock()

	return idx.syncInternal()
}

func (idx *Indexer) syncInternal() error {
	tx, err := idx.db.sqlDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Index Project
	if idx.store.IsInitialized() {
		proj, err := idx.store.ReadProject()
		if err == nil && proj != nil {
			err = upsertNode(tx, &model.Node{
				ID:                 proj.ID,
				Type:               model.NodeTypeProject,
				Title:              proj.Name,
				Description:        proj.Description,
				Status:             proj.Status,
				Confidence:         1.0,
				VerificationStatus: model.StatusVerified,
				CreatedAt:          proj.CreatedAt,
				UpdatedAt:          proj.UpdatedAt,
				FilePath:           "project.yaml",
				ContentHash:        ComputeHash([]byte(proj.Name + proj.Description)),
			})
			if err != nil {
				return fmt.Errorf("failed to index project: %w", err)
			}
			for _, c := range proj.GlobalConstraints {
				if err := insertConstraint(tx, proj.ID, c); err != nil {
					return err
				}
			}
		}
	}

	// 2. Index Tasks
	tasks, err := idx.store.ListTasks()
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}
	for _, task := range tasks {
		n := &model.Node{
			ID:                 task.ID,
			Type:               model.NodeTypeTask,
			ParentID:           task.ParentID,
			Title:              task.Title,
			Description:        task.Goal,
			Status:             string(task.Status),
			Confidence:         1.0,
			VerificationStatus: model.StatusVerified,
			CreatedAt:          task.CreatedAt,
			UpdatedAt:          task.UpdatedAt,
			FilePath:           fmt.Sprintf("tasks/%s/%s.yaml", task.ParentID, task.ID),
			TouchesFiles:       task.TouchesFiles,
		}
		if err := upsertNode(tx, n); err != nil {
			return fmt.Errorf("failed to index task %s: %w", task.ID, err)
		}
		for _, f := range task.TouchesFiles {
			if err := insertFile(tx, task.ID, f); err != nil {
				return err
			}
		}
		for _, c := range task.Constraints {
			if err := insertConstraint(tx, task.ID, c); err != nil {
				return err
			}
		}
		if task.ParentID != "" {
			if err := insertEdge(tx, task.ParentID, task.ID, model.EdgeParentOf, task.CreatedAt); err != nil {
				return err
			}
		}
	}

	// 3. Index Decisions
	decisions, err := idx.store.ListDecisions()
	if err != nil {
		return fmt.Errorf("failed to list decisions: %w", err)
	}
	for _, dec := range decisions {
		fm := dec.Frontmatter
		n := &model.Node{
			ID:                 fm.ID,
			Type:               model.NodeTypeDecision,
			ParentID:           fm.TaskID,
			Title:              fm.Title,
			Description:        dec.Body,
			Status:             string(fm.Status),
			Confidence:         fm.Confidence,
			VerificationStatus: fm.Status,
			CreatedAt:          fm.CreatedAt,
			UpdatedAt:          fm.UpdatedAt,
			FilePath:           fmt.Sprintf("decisions/%s.md", fm.ID),
			EvidenceIDs:        fm.EvidenceIDs,
		}
		if err := upsertNode(tx, n); err != nil {
			return fmt.Errorf("failed to index decision %s: %w", fm.ID, err)
		}
		if fm.TaskID != "" {
			if err := insertEdge(tx, fm.TaskID, fm.ID, model.EdgeProduced, fm.CreatedAt); err != nil {
				return err
			}
		}
		for _, evID := range fm.EvidenceIDs {
			if err := insertEvidenceLink(tx, fm.ID, evID); err != nil {
				return err
			}
			if err := insertEdge(tx, fm.ID, evID, model.EdgeVerifiedBy, fm.CreatedAt); err != nil {
				return err
			}
		}
		if fm.Supersedes != "" {
			if err := insertEdge(tx, fm.ID, fm.Supersedes, model.EdgeSupersedes, fm.CreatedAt); err != nil {
				return err
			}
		}
	}

	// 4. Index Lessons
	lessons, err := idx.store.ListLessons()
	if err != nil {
		return fmt.Errorf("failed to list lessons: %w", err)
	}
	for _, les := range lessons {
		fm := les.Frontmatter
		n := &model.Node{
			ID:                 fm.ID,
			Type:               model.NodeTypeLesson,
			ParentID:           fm.SourceTaskID,
			Title:              fm.Title,
			Description:        les.Body,
			Status:             string(fm.Status),
			Confidence:         fm.Confidence,
			VerificationStatus: fm.Status,
			CreatedAt:          fm.CreatedAt,
			UpdatedAt:          fm.UpdatedAt,
			FilePath:           fmt.Sprintf("lessons/%s.md", fm.ID),
			Tags:               fm.Tags,
			TouchesFiles:       fm.RelatedFiles,
			EvidenceIDs:        fm.EvidenceIDs,
		}
		if err := upsertNode(tx, n); err != nil {
			return fmt.Errorf("failed to index lesson %s: %w", fm.ID, err)
		}
		if fm.SourceTaskID != "" {
			if err := insertEdge(tx, fm.SourceTaskID, fm.ID, model.EdgeLearnedFrom, fm.CreatedAt); err != nil {
				return err
			}
		}
		for _, tag := range fm.Tags {
			if err := insertTag(tx, fm.ID, tag); err != nil {
				return err
			}
		}
		for _, rf := range fm.RelatedFiles {
			if err := insertFile(tx, fm.ID, rf); err != nil {
				return err
			}
		}
		for _, evID := range fm.EvidenceIDs {
			if err := insertEvidenceLink(tx, fm.ID, evID); err != nil {
				return err
			}
			if err := insertEdge(tx, fm.ID, evID, model.EdgeVerifiedBy, fm.CreatedAt); err != nil {
				return err
			}
		}
	}

	// 5. Index Evidence
	evidenceList, err := idx.store.ListEvidence()
	if err != nil {
		return fmt.Errorf("failed to list evidence: %w", err)
	}
	for _, ev := range evidenceList {
		n := &model.Node{
			ID:                 ev.ID,
			Type:               model.NodeTypeEvidence,
			ParentID:           ev.NodeID,
			Title:              ev.Title,
			Description:        ev.OutputSummary,
			Status:             string(model.StatusVerified),
			Confidence:         1.0,
			VerificationStatus: model.StatusVerified,
			Source:             ev.Command,
			CreatedAt:          ev.CreatedAt,
			UpdatedAt:          ev.CreatedAt,
			FilePath:           fmt.Sprintf("evidence/%s.yaml", ev.ID),
		}
		if err := upsertNode(tx, n); err != nil {
			return fmt.Errorf("failed to index evidence %s: %w", ev.ID, err)
		}
		if ev.NodeID != "" {
			if err := insertEdge(tx, ev.NodeID, ev.ID, model.EdgeVerifiedBy, ev.CreatedAt); err != nil {
				return err
			}
		}
	}

	// 6. Index Failures
	failures, err := idx.store.ListFailures()
	if err != nil {
		return fmt.Errorf("failed to list failures: %w", err)
	}
	for _, fail := range failures {
		_, err := tx.Exec(`
			INSERT INTO node_failures (id, task_id, summary, causal_details, attempted_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				task_id = excluded.task_id,
				summary = excluded.summary,
				causal_details = excluded.causal_details,
				attempted_at = excluded.attempted_at;
		`, fail.ID, fail.TaskID, fail.Summary, fail.CausalDetails, fail.AttemptedAt.Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("failed to index failure %s: %w", fail.ID, err)
		}
		if fail.TaskID != "" {
			if err := insertEdge(tx, fail.TaskID, fail.ID, model.EdgeFailedBecause, fail.AttemptedAt); err != nil {
				return err
			}
		}
	}

	// 7. Index Explicit Links
	links, err := idx.store.ReadLinks()
	if err == nil {
		for _, l := range links {
			if err := insertEdge(tx, l.FromID, l.ToID, l.Type, l.CreatedAt); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func upsertNode(tx *sql.Tx, n *model.Node) error {
	query := `
	INSERT INTO nodes (
		id, type, parent_id, title, description, status,
		confidence, verification_status, source, created_at, updated_at,
		file_path, content_hash
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		type = excluded.type,
		parent_id = excluded.parent_id,
		title = excluded.title,
		description = excluded.description,
		status = excluded.status,
		confidence = excluded.confidence,
		verification_status = excluded.verification_status,
		source = excluded.source,
		created_at = excluded.created_at,
		updated_at = excluded.updated_at,
		file_path = excluded.file_path,
		content_hash = excluded.content_hash;
	`
	_, err := tx.Exec(
		query,
		n.ID,
		string(n.Type),
		n.ParentID,
		n.Title,
		n.Description,
		n.Status,
		n.Confidence,
		string(n.VerificationStatus),
		n.Source,
		n.CreatedAt.Format(time.RFC3339),
		n.UpdatedAt.Format(time.RFC3339),
		n.FilePath,
		n.ContentHash,
	)
	return err
}

func insertEdge(tx *sql.Tx, fromID, toID string, edgeType model.EdgeType, createdAt time.Time) error {
	if fromID == "" || toID == "" {
		return nil
	}
	query := `
	INSERT OR IGNORE INTO edges (from_id, to_id, type, created_at)
	VALUES (?, ?, ?, ?);
	`
	_, err := tx.Exec(query, fromID, toID, string(edgeType), createdAt.Format(time.RFC3339))
	return err
}

func insertFile(tx *sql.Tx, nodeID, filePath string) error {
	if filePath == "" {
		return nil
	}
	_, err := tx.Exec(`INSERT OR IGNORE INTO node_files (node_id, file_path) VALUES (?, ?);`, nodeID, filePath)
	return err
}

func insertTag(tx *sql.Tx, nodeID, tag string) error {
	if tag == "" {
		return nil
	}
	_, err := tx.Exec(`INSERT OR IGNORE INTO node_tags (node_id, tag) VALUES (?, ?);`, nodeID, tag)
	return err
}

func insertConstraint(tx *sql.Tx, nodeID, constraintText string) error {
	if constraintText == "" {
		return nil
	}
	_, err := tx.Exec(`INSERT OR IGNORE INTO node_constraints (node_id, constraint_text) VALUES (?, ?);`, nodeID, constraintText)
	return err
}

func insertEvidenceLink(tx *sql.Tx, nodeID, evidenceID string) error {
	if evidenceID == "" {
		return nil
	}
	_, err := tx.Exec(`INSERT OR IGNORE INTO node_evidence (node_id, evidence_id) VALUES (?, ?);`, nodeID, evidenceID)
	return err
}
