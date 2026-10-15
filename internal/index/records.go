package index

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"knodexa/internal/model"
	"knodexa/internal/storage"
)

// recordData reads the canonical bytes captured by the last successful sync.
// This avoids recursive task-file scans and keeps metadata in the same snapshot.
func (d *DB) recordData(id, kind string) ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var data []byte
	err := d.sqlDB.QueryRow(`SELECT data FROM indexed_records WHERE node_id=? AND kind=?`, id, kind).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("%s record %s: %w", kind, id, err)
	}
	return data, nil
}
func (d *DB) GetTaskData(id string) (*model.TaskData, error) {
	data, err := d.recordData(id, "tasks")
	if err != nil {
		return nil, err
	}
	var task model.TaskData
	if err := yaml.Unmarshal(data, &task); err != nil {
		return nil, err
	}
	task.SnapshotHash = ComputeHash(data)
	return &task, nil
}
func (d *DB) GetDecisionDocument(id string) (*model.DecisionDocument, error) {
	data, err := d.recordData(id, "decisions")
	if err != nil {
		return nil, err
	}
	meta, body, err := storage.SplitFrontmatter(data)
	if err != nil {
		return nil, err
	}
	var fm model.DecisionFrontmatter
	if err := yaml.Unmarshal(meta, &fm); err != nil {
		return nil, err
	}
	return &model.DecisionDocument{Frontmatter: fm, Body: body}, nil
}
func (d *DB) GetLessonDocument(id string) (*model.LessonDocument, error) {
	data, err := d.recordData(id, "lessons")
	if err != nil {
		return nil, err
	}
	meta, body, err := storage.SplitFrontmatter(data)
	if err != nil {
		return nil, err
	}
	var fm model.LessonFrontmatter
	if err := yaml.Unmarshal(meta, &fm); err != nil {
		return nil, err
	}
	return &model.LessonDocument{Frontmatter: fm, Body: body}, nil
}

// ListApproachTasks returns approach metadata from the synchronized snapshot.
func (d *DB) ListApproachTasks() ([]model.TaskData, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.sqlDB.Query(`SELECT r.data FROM indexed_records r JOIN approach_tasks a ON a.node_id=r.node_id WHERE r.kind='tasks' ORDER BY r.node_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.TaskData
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var task model.TaskData
		if err := yaml.Unmarshal(data, &task); err != nil {
			return nil, err
		}
		result = append(result, task)
	}
	return result, rows.Err()
}

// ListTaskIdentities reads compact JSON identity metadata from the synchronized snapshot.
// Unlike resume, resolving an unknown request searches the recorded task corpus.
func (d *DB) ListTaskIdentities() ([]model.TaskData, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.sqlDB.Query(`SELECT data FROM work_identities ORDER BY node_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.TaskData
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var task model.TaskData
		if err := json.Unmarshal(data, &task); err != nil {
			return nil, err
		}
		result = append(result, task)
	}
	return result, rows.Err()
}
