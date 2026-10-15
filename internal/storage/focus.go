package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Focus is advisory local navigation state, never work-selection authority.
// A concurrent client may change it; target identity still comes from evidence.
func (s *Storage) SaveFocus(taskID string) error {
	data, err := json.Marshal(struct {
		TaskID string `json:"task_id"`
	}{taskID})
	if err != nil {
		return err
	}
	return s.atomicWriteFile(filepath.Join(s.KnodexaDir(), "focus.json"), data, 0644)
}
func (s *Storage) ReadFocus() (string, error) {
	data, err := os.ReadFile(filepath.Join(s.KnodexaDir(), "focus.json"))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var focus struct {
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal(data, &focus); err != nil {
		return "", fmt.Errorf("read advisory focus: %w", err)
	}
	return focus.TaskID, nil
}
