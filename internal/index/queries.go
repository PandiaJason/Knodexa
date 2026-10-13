package index

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"knodexa/internal/model"
)

// GetNode retrieves a node by its unique ID.
func (d *DB) GetNode(id string) (*model.Node, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
	SELECT id, type, parent_id, title, description, status,
	       confidence, verification_status, source, created_at, updated_at,
	       file_path, content_hash
	FROM nodes WHERE id = ? LIMIT 1;
	`
	row := d.sqlDB.QueryRow(query, id)
	return scanNode(row)
}

// GetNodesByType returns all nodes of a specific NodeType.
func (d *DB) GetNodesByType(nodeType model.NodeType) ([]model.Node, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
	SELECT id, type, parent_id, title, description, status,
	       confidence, verification_status, source, created_at, updated_at,
	       file_path, content_hash
	FROM nodes WHERE type = ? ORDER BY id ASC;
	`
	rows, err := d.sqlDB.Query(query, string(nodeType))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []model.Node
	for rows.Next() {
		n, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, *n)
	}
	return nodes, rows.Err()
}

// GetEdgesFrom retrieves all outgoing edges originating from fromID.
func (d *DB) GetEdgesFrom(fromID string) ([]model.Edge, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `SELECT from_id, to_id, type, created_at FROM edges WHERE from_id = ? ORDER BY created_at ASC;`
	rows, err := d.sqlDB.Query(query, fromID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEdges(rows)
}

// GetEdgesTo retrieves all incoming edges terminating at toID.
func (d *DB) GetEdgesTo(toID string) ([]model.Edge, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `SELECT from_id, to_id, type, created_at FROM edges WHERE to_id = ? ORDER BY created_at ASC;`
	rows, err := d.sqlDB.Query(query, toID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEdges(rows)
}

// GetUpstreamConstraints resolves all constraints attached to taskID and all of its ancestor tasks up to proj_root.
func (d *DB) GetUpstreamConstraints(taskID string) ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var constraints []string
	visited := make(map[string]bool)
	currID := taskID

	for currID != "" && !visited[currID] {
		visited[currID] = true

		cRows, err := d.sqlDB.Query(`SELECT constraint_text FROM node_constraints WHERE node_id = ?;`, currID)
		if err == nil {
			for cRows.Next() {
				var text string
				if err := cRows.Scan(&text); err == nil && text != "" {
					constraints = append(constraints, text)
				}
			}
			cRows.Close()
		}

		// Find parent
		var parentID sql.NullString
		err = d.sqlDB.QueryRow(`SELECT parent_id FROM nodes WHERE id = ?;`, currID).Scan(&parentID)
		if err != nil || !parentID.Valid || parentID.String == "" {
			break
		}
		currID = parentID.String
	}

	// Also include global project constraints from proj_root
	if !visited["proj_root"] {
		cRows, err := d.sqlDB.Query(`SELECT constraint_text FROM node_constraints WHERE node_id = 'proj_root';`)
		if err == nil {
			for cRows.Next() {
				var text string
				if err := cRows.Scan(&text); err == nil && text != "" {
					constraints = append(constraints, text)
				}
			}
			cRows.Close()
		}
	}

	return constraints, nil
}

// GetDecisionsForTask returns all verified decisions associated with taskID or its ancestors.
func (d *DB) GetDecisionsForTask(taskID string) ([]model.Node, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
	SELECT id, type, parent_id, title, description, status,
	       confidence, verification_status, source, created_at, updated_at,
	       file_path, content_hash
	FROM nodes
	WHERE type = 'DECISION' AND status = 'VERIFIED'
	  AND (parent_id = ? OR id IN (SELECT to_id FROM edges WHERE from_id = ? AND type = 'PRODUCED'))
	ORDER BY created_at DESC;
	`
	rows, err := d.sqlDB.Query(query, taskID, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []model.Node
	for rows.Next() {
		n, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, *n)
	}
	return decisions, rows.Err()
}

// GetFailuresForTask returns recorded failures associated with taskID or its subtasks.
func (d *DB) GetFailuresForTask(taskID string) ([]model.FailureData, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
	SELECT id, task_id, summary, causal_details, attempted_at
	FROM node_failures
	WHERE task_id = ?
	ORDER BY attempted_at DESC;
	`
	rows, err := d.sqlDB.Query(query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var failures []model.FailureData
	for rows.Next() {
		var f model.FailureData
		var dt string
		if err := rows.Scan(&f.ID, &f.TaskID, &f.Summary, &f.CausalDetails, &dt); err != nil {
			return nil, err
		}
		t, _ := time.Parse(time.RFC3339, dt)
		f.AttemptedAt = t
		failures = append(failures, f)
	}
	return failures, rows.Err()
}

// GetRelatedLessons finds verified lessons related to the given task, touched files, or tags.
func (d *DB) GetRelatedLessons(taskID string, touchedFiles []string, tags []string, limit int) ([]model.Node, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 {
		limit = 5
	}

	// 1. Direct explicit links: edges with type RELEVANT_TO
	queryDirect := `
	SELECT n.id, n.type, n.parent_id, n.title, n.description, n.status,
	       n.confidence, n.verification_status, n.source, n.created_at, n.updated_at,
	       n.file_path, n.content_hash
	FROM nodes n
	JOIN edges e ON n.id = e.from_id
	WHERE e.to_id = ? AND e.type = 'RELEVANT_TO' AND n.type = 'LESSON' AND n.status = 'VERIFIED'
	ORDER BY n.confidence DESC
	LIMIT ?;
	`
	rowsDirect, err := d.sqlDB.Query(queryDirect, taskID, limit)
	if err != nil {
		return nil, err
	}
	defer rowsDirect.Close()

	results := make([]model.Node, 0)
	seen := make(map[string]bool)

	for rowsDirect.Next() {
		n, err := scanNode(rowsDirect)
		if err == nil && !seen[n.ID] {
			seen[n.ID] = true
			results = append(results, *n)
		}
	}

	if len(results) >= limit {
		return results, nil
	}

	// 2. Overlapping files
	if len(touchedFiles) > 0 {
		placeholders := strings.Repeat("?,", len(touchedFiles)-1) + "?"
		args := make([]interface{}, len(touchedFiles))
		for i, f := range touchedFiles {
			args[i] = f
		}
		fileQuery := fmt.Sprintf(`
		SELECT n.id, n.type, n.parent_id, n.title, n.description, n.status,
		       n.confidence, n.verification_status, n.source, n.created_at, n.updated_at,
		       n.file_path, n.content_hash
		FROM nodes n
		JOIN node_files nf ON n.id = nf.node_id
		WHERE nf.file_path IN (%s) AND n.type = 'LESSON' AND n.status = 'VERIFIED'
		ORDER BY n.confidence DESC
		LIMIT ?;
		`, placeholders)
		args = append(args, limit-len(results))

		fRows, err := d.sqlDB.Query(fileQuery, args...)
		if err == nil {
			for fRows.Next() {
				n, err := scanNode(fRows)
				if err == nil && !seen[n.ID] && n.ParentID != taskID {
					seen[n.ID] = true
					results = append(results, *n)
				}
			}
			fRows.Close()
		}
	}

	// 3. Fallback: highest confidence verified lessons not from current task
	if len(results) < limit {
		rem := limit - len(results)
		fallbackQuery := `
		SELECT id, type, parent_id, title, description, status,
		       confidence, verification_status, source, created_at, updated_at,
		       file_path, content_hash
		FROM nodes
		WHERE type = 'LESSON' AND status = 'VERIFIED' AND parent_id != ?
		ORDER BY confidence DESC, created_at DESC
		LIMIT ?;
		`
		fbRows, err := d.sqlDB.Query(fallbackQuery, taskID, rem)
		if err == nil {
			for fbRows.Next() {
				n, err := scanNode(fbRows)
				if err == nil && !seen[n.ID] {
					seen[n.ID] = true
					results = append(results, *n)
				}
			}
			fbRows.Close()
		}
	}

	return results, nil
}

// SearchNodes performs keyword search on titles and descriptions.
func (d *DB) SearchNodes(query string, nodeTypes []model.NodeType, limit int) ([]model.Node, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}

	filterTypeClause := ""
	args := []interface{}{"%" + query + "%", "%" + query + "%"}

	if len(nodeTypes) > 0 {
		placeholders := strings.Repeat("?,", len(nodeTypes)-1) + "?"
		filterTypeClause = fmt.Sprintf(" AND type IN (%s)", placeholders)
		for _, nt := range nodeTypes {
			args = append(args, string(nt))
		}
	}
	args = append(args, limit)

	sqlQuery := fmt.Sprintf(`
	SELECT id, type, parent_id, title, description, status,
	       confidence, verification_status, source, created_at, updated_at,
	       file_path, content_hash
	FROM nodes
	WHERE (title LIKE ? OR description LIKE ?)%s
	ORDER BY created_at DESC
	LIMIT ?;
	`, filterTypeClause)

	rows, err := d.sqlDB.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []model.Node
	for rows.Next() {
		n, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		matches = append(matches, *n)
	}
	return matches, rows.Err()
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanNode(s scannable) (*model.Node, error) {
	var n model.Node
	var parentID, desc, src, filePath, contentHash sql.NullString
	var ca, ua string
	var nt, status, vStatus string

	err := s.Scan(
		&n.ID, &nt, &parentID, &n.Title, &desc, &status,
		&n.Confidence, &vStatus, &src, &ca, &ua,
		&filePath, &contentHash,
	)
	if err != nil {
		return nil, err
	}

	n.Type = model.NodeType(nt)
	n.Status = status
	n.VerificationStatus = model.KnowledgeStatus(vStatus)
	if parentID.Valid {
		n.ParentID = parentID.String
	}
	if desc.Valid {
		n.Description = desc.String
	}
	if src.Valid {
		n.Source = src.String
	}
	if filePath.Valid {
		n.FilePath = filePath.String
	}
	if contentHash.Valid {
		n.ContentHash = contentHash.String
	}
	t1, _ := time.Parse(time.RFC3339, ca)
	t2, _ := time.Parse(time.RFC3339, ua)
	n.CreatedAt = t1
	n.UpdatedAt = t2

	return &n, nil
}

func scanEdges(rows *sql.Rows) ([]model.Edge, error) {
	var edges []model.Edge
	for rows.Next() {
		var e model.Edge
		var et, ca string
		if err := rows.Scan(&e.FromID, &e.ToID, &et, &ca); err != nil {
			return nil, err
		}
		e.Type = model.EdgeType(et)
		t, _ := time.Parse(time.RFC3339, ca)
		e.CreatedAt = t
		edges = append(edges, e)
	}
	return edges, rows.Err()
}
