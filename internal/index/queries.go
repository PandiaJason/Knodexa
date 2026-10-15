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
	node, err := scanNode(row)
	if err != nil {
		return nil, err
	}

	// Fetch touches files from node_files
	fRows, fErr := d.sqlDB.Query(`SELECT file_path FROM node_files WHERE node_id = ?;`, id)
	if fErr == nil {
		for fRows.Next() {
			var fp string
			if fRows.Scan(&fp) == nil && fp != "" {
				node.TouchesFiles = append(node.TouchesFiles, fp)
			}
		}
		fRows.Close()
	}

	return node, nil
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

// GetDecisionsForTask returns all verified decisions associated with taskID, its ancestors, or sibling subtasks.
func (d *DB) GetDecisionsForTask(taskID string) ([]model.Node, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	taskIDs, err := d.relatedTaskIDs(taskID)
	if err != nil {
		return nil, err
	}

	placeholders := strings.Repeat("?,", len(taskIDs)-1) + "?"
	args := make([]interface{}, len(taskIDs)*2)
	for i, id := range taskIDs {
		args[i] = id
		args[len(taskIDs)+i] = id
	}

	query := fmt.Sprintf(`
	SELECT id, type, parent_id, title, description, status,
	       confidence, verification_status, source, created_at, updated_at,
	       file_path, content_hash
	FROM nodes
	WHERE type = 'DECISION' AND status = 'VERIFIED'
	  AND (parent_id IN (%s) OR id IN (SELECT to_id FROM edges WHERE from_id IN (%s) AND type = 'PRODUCED'))
	ORDER BY created_at DESC, id ASC;
	`, placeholders, placeholders)

	rows, err := d.sqlDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []model.Node
	seen := make(map[string]bool)
	for rows.Next() {
		n, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		if !seen[n.ID] {
			seen[n.ID] = true
			decisions = append(decisions, *n)
		}
	}
	return decisions, rows.Err()
}

// GetFailuresForTask returns recorded failures associated with taskID, its ancestors, or sibling subtasks.
func (d *DB) GetFailuresForTask(taskID string) ([]model.FailureData, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	taskIDs, err := d.relatedTaskIDs(taskID)
	if err != nil {
		return nil, err
	}

	placeholders := strings.Repeat("?,", len(taskIDs)-1) + "?"
	args := make([]interface{}, len(taskIDs))
	for i, id := range taskIDs {
		args[i] = id
	}

	query := fmt.Sprintf(`
	SELECT id, task_id, summary, causal_details, attempted_at
	FROM node_failures
	WHERE task_id IN (%s)
	ORDER BY attempted_at DESC, id ASC;
	`, placeholders)

	rows, err := d.sqlDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var failures []model.FailureData
	seen := make(map[string]bool)
	for rows.Next() {
		var f model.FailureData
		var dt string
		if err := rows.Scan(&f.ID, &f.TaskID, &f.Summary, &f.CausalDetails, &dt); err != nil {
			return nil, err
		}
		if !seen[f.ID] {
			seen[f.ID] = true
			t, _ := time.Parse(time.RFC3339, dt)
			f.AttemptedAt = t
			failures = append(failures, f)
		}
	}
	return failures, rows.Err()
}

// relatedTaskIDs runs under the caller's lock and includes every ancestor plus
// immediate siblings for compatibility with the existing task-neighborhood policy.
func (d *DB) relatedTaskIDs(taskID string) ([]string, error) {
	rows, err := d.sqlDB.Query(`WITH RECURSIVE ancestors(id, parent_id) AS (
	 SELECT id, parent_id FROM nodes WHERE id = ?
	 UNION
	 SELECT n.id, n.parent_id FROM nodes n JOIN ancestors a ON n.id = a.parent_id
	)
	SELECT id FROM ancestors
	UNION SELECT id FROM nodes WHERE type='TASK' AND COALESCE(parent_id,'')<>'' AND parent_id = (SELECT parent_id FROM nodes WHERE id = ?) AND parent_id NOT IN (SELECT id FROM nodes WHERE type='PROJECT')
	ORDER BY id`, taskID, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{taskID}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if id != taskID {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

// GetRelatedLessons returns verified lessons with explicit links or actual file/tag
// overlap. Unrelated high-confidence lessons are not task context.
func (d *DB) GetRelatedLessons(taskID string, touchedFiles []string, tags []string, limit int) ([]model.Node, error) {
	return d.GetRelatedLessonsForSources(taskID, nil, touchedFiles, tags, limit)
}

// GetRelatedLessonsForSources includes lessons belonging to selected agreement
// sources. Relationship relevance ranks before confidence; recency is not authority.
func (d *DB) GetRelatedLessonsForSources(taskID string, sources, touchedFiles, tags []string, limit int) ([]model.Node, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if limit <= 0 {
		limit = 5
	}
	ids := []string{taskID}
	seen := map[string]bool{taskID: true}
	for _, source := range sources {
		if !seen[source] {
			ids = append(ids, source)
			seen[source] = true
		}
	}
	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	matches := []string{"n.parent_id IN (" + placeholders + ")", `EXISTS (SELECT 1 FROM edges e WHERE e.from_id = n.id AND e.to_id IN (` + placeholders + `) AND e.type = 'RELEVANT_TO')`}
	args := []interface{}{}
	for repeat := 0; repeat < 2; repeat++ {
		for _, id := range ids {
			args = append(args, id)
		}
	}
	ranks := []int{4, 3}
	for _, spec := range []struct {
		values        []string
		table, column string
	}{{touchedFiles, "node_files", "file_path"}, {tags, "node_tags", "tag"}} {
		if len(spec.values) == 0 {
			continue
		}
		placeholders := strings.Repeat("?,", len(spec.values)-1) + "?"
		matches = append(matches, fmt.Sprintf("EXISTS (SELECT 1 FROM %s m WHERE m.node_id = n.id AND m.%s IN (%s))", spec.table, spec.column, placeholders))
		if spec.table == "node_files" {
			ranks = append(ranks, 2)
		} else {
			ranks = append(ranks, 1)
		}
		for _, value := range spec.values {
			args = append(args, value)
		}
	}
	var ranking strings.Builder
	ranking.WriteString("CASE ")
	for i, match := range matches {
		fmt.Fprintf(&ranking, "WHEN %s THEN %d ", match, ranks[i])
	}
	ranking.WriteString("ELSE 0 END")
	// SELECT rank parameters precede WHERE parameters.
	allArgs := append(append([]interface{}{}, args...), args...)
	allArgs = append(allArgs, limit)
	query := `SELECT n.id, n.type, n.parent_id, n.title, n.description, n.status,
	 n.confidence, n.verification_status, n.source, n.created_at, n.updated_at, n.file_path, n.content_hash, ` + ranking.String() + ` AS match_rank
	 FROM nodes n WHERE n.type = 'LESSON' AND n.status = 'VERIFIED' AND (` + strings.Join(matches, " OR ") + `)
	 ORDER BY match_rank DESC, n.confidence DESC, n.id ASC LIMIT ?`
	rows, err := d.sqlDB.Query(query, allArgs...)
	if err != nil {
		return nil, err
	}
	results := make([]model.Node, 0)
	for rows.Next() {
		var rank int
		n, err := scanNode(rankedScanner{rows, &rank})
		if err != nil {
			rows.Close()
			return nil, err
		}
		n.MatchReason = map[int]string{4: "selected_source", 3: "explicit_link", 2: "file_overlap", 1: "tag_overlap"}[rank]
		results = append(results, *n)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	// Close the first query before hydrating associations: SQLite uses one connection.
	for i := range results {
		files, err := d.sqlDB.Query(`SELECT file_path FROM node_files WHERE node_id = ? ORDER BY file_path`, results[i].ID)
		if err != nil {
			return nil, err
		}
		for files.Next() {
			var path string
			if err := files.Scan(&path); err != nil {
				files.Close()
				return nil, err
			}
			results[i].TouchesFiles = append(results[i].TouchesFiles, path)
		}
		err = files.Err()
		files.Close()
		if err != nil {
			return nil, err
		}
	}
	return results, nil
}

type rankedScanner struct {
	scannable
	rank *int
}

func (s rankedScanner) Scan(dest ...interface{}) error {
	return s.scannable.Scan(append(dest, s.rank)...)
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
