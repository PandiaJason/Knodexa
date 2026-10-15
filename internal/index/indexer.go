package index

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"knodexa/internal/model"
	"knodexa/internal/storage"
)

// Indexer synchronizes a rebuildable index with canonical records.
type Indexer struct {
	db    *DB
	store *storage.Storage
}

// SyncStats describes work performed by a successful synchronization.
type SyncStats struct {
	ScannedRecords int   `json:"scanned_records"`
	UpdatedRecords int   `json:"updated_records"`
	DeletedRecords int   `json:"deleted_records"`
	BytesRead      int64 `json:"bytes_read"`
}

func NewIndexer(db *DB, store *storage.Storage) *Indexer { return &Indexer{db: db, store: store} }
func ComputeHash(data []byte) string                     { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

// Rebuild replaces all derived records in one transaction, preserving the old
// snapshot if any canonical file is malformed.
func (idx *Indexer) Rebuild() error {
	idx.db.mu.Lock()
	defer idx.db.mu.Unlock()
	return idx.db.withWriteLock(func() error { _, err := idx.syncInternal(true); return err })
}
func (idx *Indexer) Sync() error { _, err := idx.SyncWithStats(); return err }

// SyncWithStats hashes canonical content, parses only changed files, and writes
// only changed/deleted records. Hashing deliberately detects same-size edits
// even when filesystem timestamps are preserved.
func (idx *Indexer) SyncWithStats() (SyncStats, error) {
	idx.db.mu.Lock()
	defer idx.db.mu.Unlock()
	var stats SyncStats
	err := idx.db.withWriteLock(func() error { var err error; stats, err = idx.syncInternal(false); return err })
	return stats, err
}

type cachedRecord struct{ hash, id string }
type sourceRecord struct {
	path, kind, hash string
	data             []byte
}
type parsedRecord struct {
	hasApproach  bool
	identityData []byte
	sourceRecord
	node        *model.Node
	failure     *model.FailureData
	constraints []string
	edges       []model.Edge
}

func (idx *Indexer) inventory() ([]sourceRecord, error) {
	root := idx.store.KnodexaDir()
	paths := map[string]string{"project.yaml": "project"}
	if _, err := os.Stat(filepath.Join(root, "project.yaml")); err != nil {
		return nil, fmt.Errorf("read project: %w", err)
	}
	if _, err := os.Stat(filepath.Join(root, "links.jsonl")); err == nil {
		paths["links.jsonl"] = "links"
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	for _, kind := range []string{"tasks", "decisions", "lessons", "evidence", "failures", "observations"} {
		dir := filepath.Join(root, kind)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			extension := ".yaml"
			if kind == "decisions" || kind == "lessons" {
				extension = ".md"
			}
			if !strings.HasSuffix(entry.Name(), extension) || strings.HasPrefix(entry.Name(), ".tmp_") {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("canonical record cannot be a symlink: %s", path)
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			paths[filepath.ToSlash(rel)] = kind
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	keys := make([]string, 0, len(paths))
	for path := range paths {
		keys = append(keys, path)
	}
	sort.Strings(keys)
	records := make([]sourceRecord, 0, len(keys))
	for _, path := range keys {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		records = append(records, sourceRecord{path: path, kind: paths[path], hash: ComputeHash(data), data: data})
	}
	return records, nil
}

func (idx *Indexer) syncInternal(force bool) (SyncStats, error) {
	stats := SyncStats{}
	sources, err := idx.inventory()
	if err != nil {
		return stats, err
	}
	stats.ScannedRecords = len(sources)
	for _, r := range sources {
		stats.BytesRead += int64(len(r.data))
	}
	tx, err := idx.db.sqlDB.Begin()
	if err != nil {
		return stats, err
	}
	defer tx.Rollback()
	cached := map[string]cachedRecord{}
	rows, err := tx.Query("SELECT path, content_hash, COALESCE(node_id,'') FROM indexed_records")
	if err != nil {
		return stats, err
	}
	for rows.Next() {
		var path string
		var r cachedRecord
		if err := rows.Scan(&path, &r.hash, &r.id); err != nil {
			rows.Close()
			return stats, err
		}
		cached[path] = r
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return stats, err
	}
	// Build the approach lookup once for older caches, transactionally.
	var featureCount int
	if err := tx.QueryRow("SELECT COUNT(*) FROM index_features WHERE name='approach-routing-v1'").Scan(&featureCount); err != nil {
		return stats, err
	}
	force = force || featureCount == 0
	var identityFeature int
	if err := tx.QueryRow("SELECT COUNT(*) FROM index_features WHERE name='work-identity-v2'").Scan(&identityFeature); err != nil {
		return stats, err
	}
	force = force || identityFeature == 0
	// A pre-incremental cache has no ownership records; migrate it once.
	if force || len(cached) == 0 {
		for _, table := range []string{"work_identities", "approach_tasks", "indexed_records", "record_edges", "node_failures", "node_evidence", "node_constraints", "node_tags", "node_files", "edges", "nodes"} {
			if _, err := tx.Exec("DELETE FROM " + table); err != nil {
				return stats, err
			}
		}
		cached = map[string]cachedRecord{}
	}
	pending := []parsedRecord{}
	changedPaths := map[string]bool{}
	seen := map[string]bool{}
	for _, source := range sources {
		seen[source.path] = true
		if old, ok := cached[source.path]; ok && old.hash == source.hash {
			continue
		}
		record, err := parseRecord(source)
		if err != nil {
			return stats, fmt.Errorf("parse %s: %w", source.path, err)
		}
		pending = append(pending, record)
		changedPaths[source.path] = true
	}
	// Remove old ownership before applying changed records, so moves and ID edits
	// can be reconciled without retaining obsolete associations.
	for path, old := range cached {
		if !seen[path] {
			stats.DeletedRecords++
		}
		changed := !seen[path] || changedPaths[path]
		if !changed {
			continue
		}
		if old.id != "" {
			for _, table := range []string{"work_identities", "approach_tasks", "nodes", "node_files", "node_tags", "node_constraints", "node_evidence"} {
				column := "node_id"
				if table == "nodes" {
					column = "id"
				}
				if _, err := tx.Exec("DELETE FROM "+table+" WHERE "+column+" = ?", old.id); err != nil {
					return stats, err
				}
			}
			if _, err := tx.Exec("DELETE FROM node_failures WHERE id = ?", old.id); err != nil {
				return stats, err
			}
		}
		if _, err := tx.Exec("DELETE FROM record_edges WHERE source_path = ?", path); err != nil {
			return stats, err
		}
		if _, err := tx.Exec("DELETE FROM indexed_records WHERE path = ?", path); err != nil {
			return stats, err
		}
	}
	for _, record := range pending {
		var id interface{}
		if record.node != nil {
			n := record.node
			id = n.ID
			if record.identityData != nil {
				if _, err := tx.Exec("INSERT INTO work_identities(node_id,data) VALUES(?,?)", n.ID, record.identityData); err != nil {
					return stats, err
				}
			}
			if record.hasApproach {
				if _, err := tx.Exec("INSERT INTO approach_tasks(node_id) VALUES(?)", n.ID); err != nil {
					return stats, err
				}
			}
			if err := upsertNode(tx, n); err != nil {
				return stats, err
			}
			for _, f := range n.TouchesFiles {
				if err := insertFile(tx, n.ID, f); err != nil {
					return stats, err
				}
			}
			for _, tag := range n.Tags {
				if err := insertTag(tx, n.ID, tag); err != nil {
					return stats, err
				}
			}
			for _, c := range record.constraints {
				if err := insertConstraint(tx, n.ID, c); err != nil {
					return stats, err
				}
			}
			for _, e := range n.EvidenceIDs {
				if err := insertEvidenceLink(tx, n.ID, e); err != nil {
					return stats, err
				}
			}
		}
		if f := record.failure; f != nil {
			id = f.ID
			if _, err := tx.Exec(`INSERT INTO node_failures(id,task_id,summary,causal_details,attempted_at) VALUES(?,?,?,?,?)`, f.ID, f.TaskID, f.Summary, f.CausalDetails, f.AttemptedAt.Format(time.RFC3339)); err != nil {
				return stats, err
			}
		}
		// UNIQUE(node_id) rejects conflicting canonical files instead of silently
		// overwriting one record with another. A failure rolls back the whole sync.
		if _, err := tx.Exec(`INSERT INTO indexed_records(path,kind,content_hash,node_id,data) VALUES(?,?,?,?,?)`, record.path, record.kind, record.hash, id, record.data); err != nil {
			return stats, fmt.Errorf("index %s (duplicate ID or invalid record): %w", record.path, err)
		}
		for _, edge := range record.edges {
			if edge.FromID == "" || edge.ToID == "" {
				continue
			}
			if _, err := tx.Exec(`INSERT OR IGNORE INTO record_edges(source_path,from_id,to_id,type,created_at) VALUES(?,?,?,?,?)`, record.path, edge.FromID, edge.ToID, string(edge.Type), edge.CreatedAt.Format(time.RFC3339)); err != nil {
				return stats, err
			}
		}
		stats.UpdatedRecords++
	}
	if stats.UpdatedRecords > 0 || stats.DeletedRecords > 0 {
		// Preserve edges owned by other records when one source is edited/deleted.
		if _, err := tx.Exec(`DELETE FROM edges WHERE NOT EXISTS (SELECT 1 FROM record_edges r WHERE r.from_id=edges.from_id AND r.to_id=edges.to_id AND r.type=edges.type)`); err != nil {
			return stats, err
		}
		if _, err := tx.Exec(`INSERT INTO edges(from_id,to_id,type,created_at)
   SELECT from_id,to_id,type,MIN(created_at) FROM record_edges WHERE true GROUP BY from_id,to_id,type
   ON CONFLICT(from_id,to_id,type) DO UPDATE SET created_at=excluded.created_at WHERE edges.created_at<>excluded.created_at`); err != nil {
			return stats, err
		}
	}
	if featureCount == 0 {
		if _, err := tx.Exec("INSERT INTO index_features(name) VALUES('approach-routing-v1')"); err != nil {
			return stats, err
		}
	}
	if identityFeature == 0 {
		if _, err := tx.Exec("INSERT INTO index_features(name) VALUES('work-identity-v2')"); err != nil {
			return stats, err
		}
	}
	if err := tx.Commit(); err != nil {
		return stats, err
	}
	return stats, nil
}

func parseRecord(source sourceRecord) (parsedRecord, error) {
	r := parsedRecord{sourceRecord: source}
	edge := func(from, to string, kind model.EdgeType, at time.Time) {
		r.edges = append(r.edges, model.Edge{FromID: from, ToID: to, Type: kind, CreatedAt: at})
	}
	switch source.kind {
	case "links":
		scanner := bufio.NewScanner(bytes.NewReader(source.data))
		scanner.Buffer(make([]byte, 4096), 10*1024*1024)
		for scanner.Scan() {
			line := bytes.TrimSpace(scanner.Bytes())
			if len(line) == 0 {
				continue
			}
			var e model.Edge
			if err := json.Unmarshal(line, &e); err != nil {
				return r, err
			}
			if e.FromID == "" || e.ToID == "" {
				return r, fmt.Errorf("edge endpoints are required")
			}
			r.edges = append(r.edges, e)
		}
		return r, scanner.Err()
	case "project":
		var v model.ProjectConfig
		if err := yaml.Unmarshal(source.data, &v); err != nil {
			return r, err
		}
		r.node = &model.Node{ID: v.ID, Type: model.NodeTypeProject, Title: v.Name, Description: v.Description, Status: v.Status, VerificationStatus: model.StatusVerified, Confidence: 1, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
		r.constraints = v.GlobalConstraints
	case "tasks":
		var v model.TaskData
		if err := yaml.Unmarshal(source.data, &v); err != nil {
			return r, err
		}
		r.node = &model.Node{ID: v.ID, Type: model.NodeTypeTask, ParentID: v.ParentID, Title: v.Title, Description: v.Goal, Status: string(v.Status), VerificationStatus: model.StatusVerified, Confidence: 1, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt, TouchesFiles: v.TouchesFiles}
		r.constraints = v.Constraints
		r.hasApproach = v.Approach != nil
		identity := model.TaskData{ID: v.ID, WorkID: v.WorkID, CodeAnchors: v.CodeAnchors, ParentID: v.ParentID, Title: v.Title, Goal: v.Goal, Status: v.Status, Aliases: v.Aliases, TouchesFiles: v.TouchesFiles}
		if v.Approach != nil {
			identity.Approach = &model.TaskApproach{Scopes: v.Approach.Scopes}
		}
		var err error
		r.identityData, err = json.Marshal(identity)
		if err != nil {
			return r, err
		}
		edge(v.ParentID, v.ID, model.EdgeParentOf, v.CreatedAt)
	case "decisions":
		meta, body, err := storage.SplitFrontmatter(source.data)
		if err != nil {
			return r, err
		}
		var v model.DecisionFrontmatter
		if err := yaml.Unmarshal(meta, &v); err != nil {
			return r, err
		}
		r.node = &model.Node{ID: v.ID, Type: model.NodeTypeDecision, ParentID: v.TaskID, Title: v.Title, Description: body, Status: string(v.Status), VerificationStatus: v.Status, Confidence: v.Confidence, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt, EvidenceIDs: v.EvidenceIDs}
		edge(v.TaskID, v.ID, model.EdgeProduced, v.CreatedAt)
		edge(v.ID, v.Supersedes, model.EdgeSupersedes, v.CreatedAt)
		for _, id := range v.EvidenceIDs {
			edge(v.ID, id, model.EdgeVerifiedBy, v.CreatedAt)
		}
	case "lessons":
		meta, body, err := storage.SplitFrontmatter(source.data)
		if err != nil {
			return r, err
		}
		var v model.LessonFrontmatter
		if err := yaml.Unmarshal(meta, &v); err != nil {
			return r, err
		}
		r.node = &model.Node{ID: v.ID, Type: model.NodeTypeLesson, ParentID: v.SourceTaskID, Title: v.Title, Description: body, Status: string(v.Status), VerificationStatus: v.Status, Confidence: v.Confidence, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt, EvidenceIDs: v.EvidenceIDs, Tags: v.Tags, TouchesFiles: v.RelatedFiles}
		edge(v.SourceTaskID, v.ID, model.EdgeLearnedFrom, v.CreatedAt)
		for _, id := range v.EvidenceIDs {
			edge(v.ID, id, model.EdgeVerifiedBy, v.CreatedAt)
		}
	case "evidence":
		var v model.EvidenceData
		if err := yaml.Unmarshal(source.data, &v); err != nil {
			return r, err
		}
		r.node = &model.Node{ID: v.ID, Type: model.NodeTypeEvidence, ParentID: v.NodeID, Title: v.Title, Description: v.OutputSummary, Status: string(model.StatusVerified), VerificationStatus: model.StatusVerified, Confidence: 1, Source: v.Command, CreatedAt: v.CreatedAt, UpdatedAt: v.CreatedAt}
		edge(v.NodeID, v.ID, model.EdgeVerifiedBy, v.CreatedAt)
	case "failures":
		var v model.FailureData
		if err := yaml.Unmarshal(source.data, &v); err != nil {
			return r, err
		}
		r.failure = &v
		edge(v.TaskID, v.ID, model.EdgeFailedBecause, v.AttemptedAt)
	case "observations":
		var v model.ObservationData
		if err := yaml.Unmarshal(source.data, &v); err != nil {
			return r, err
		}
		r.node = &model.Node{ID: v.ID, Type: model.NodeTypeObservation, ParentID: v.TaskID, Title: v.ID, Description: v.Content, Status: string(model.StatusObservation), VerificationStatus: model.StatusObservation, TouchesFiles: v.Files, CreatedAt: v.CreatedAt, UpdatedAt: v.CreatedAt}
	default:
		return r, fmt.Errorf("unknown record kind %s", source.kind)
	}
	id := ""
	if r.node != nil {
		id = r.node.ID
		r.node.FilePath = source.path
		r.node.ContentHash = source.hash
	}
	if r.failure != nil {
		id = r.failure.ID
	}
	if err := storage.ValidateID(id); err != nil {
		return r, err
	}
	return r, nil
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
