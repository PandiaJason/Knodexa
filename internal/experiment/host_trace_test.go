package experiment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestHostTraceRequiresSuccessfulRestorationBeforeEdits(t *testing.T) {
	for _, test := range []struct {
		name   string
		events []map[string]interface{}
		want   bool
	}{
		{"valid", []map[string]interface{}{
			{"type": "item.completed", "item": map[string]interface{}{"type": "command_execution", "command": "knodexa prepare 'production webhook delivery' --text", "exit_code": 0, "aggregated_output": "selected_task: production\n"}},
			{"type": "item.completed", "item": map[string]interface{}{"type": "file_change", "changes": []map[string]interface{}{{"path": "handlers/webhook.go"}}}},
		}, true},
		{"too-late", []map[string]interface{}{
			{"type": "item.completed", "item": map[string]interface{}{"type": "file_change", "changes": []map[string]interface{}{{"path": "handlers/webhook.go"}}}},
			{"type": "item.completed", "item": map[string]interface{}{"type": "command_execution", "command": "knodexa prepare 'production webhook delivery' --text", "exit_code": 0, "aggregated_output": "selected_task: production\n"}},
		}, false},
		{"failed", []map[string]interface{}{{"type": "item.completed", "item": map[string]interface{}{"type": "command_execution", "command": "knodexa prepare request", "exit_code": 1, "aggregated_output": "selected_task: production"}}}, false},
		{"echo-only", []map[string]interface{}{{"type": "item.completed", "item": map[string]interface{}{"type": "command_execution", "command": "echo knodexa prepare request", "exit_code": 0, "aggregated_output": "knodexa prepare request"}}}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "trace.jsonl")
			file, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			encoder := json.NewEncoder(file)
			for _, event := range test.events {
				if err := encoder.Encode(event); err != nil {
					t.Fatal(err)
				}
			}
			file.Close()
			trace := readHostTrace(t, path)
			if trace.RestoredBeforeEdit != test.want {
				t.Fatalf("incorrect restoration timing: %+v", trace)
			}
		})
	}
}
