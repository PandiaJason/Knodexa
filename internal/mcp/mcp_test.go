package mcp

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"knodexa/internal/model"
	"knodexa/internal/storage"
)

func TestMCPServerEndToEnd(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewStorage(tmpDir)
	_, err := store.Init("MCP Test Project", "Testing MCP protocol")
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	// Create task A.1
	_ = store.SaveTask(&model.TaskData{
		ID:          "A.1",
		Title:       "Test Task",
		Goal:        "Validate MCP tooling",
		Status:      model.TaskStatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		Constraints: []string{"Must pass MCP suite"},
	})

	srv, err := NewServer(tmpDir)
	if err != nil {
		t.Fatalf("failed to create MCP server: %v", err)
	}

	// Helper to send a request and read response
	sendRPC := func(req Request) *Response {
		reqBytes, _ := json.Marshal(req)
		in := bytes.NewReader(append(reqBytes, '\n'))
		var out bytes.Buffer

		if err := srv.Serve(in, &out); err != nil {
			t.Fatalf("serve failed: %v", err)
		}

		var resp Response
		if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response %q: %v", out.String(), err)
		}
		return &resp
	}

	// 1. Initialize
	initResp := sendRPC(Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	})
	if initResp.Error != nil {
		t.Fatalf("initialize returned error: %+v", initResp.Error)
	}

	// 2. Tools list
	toolsResp := sendRPC(Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	})
	if toolsResp.Error != nil {
		t.Fatalf("tools/list error: %+v", toolsResp.Error)
	}
	toolsMap, ok := toolsResp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("invalid tools result: %+v", toolsResp.Result)
	}
	toolsList, _ := toolsMap["tools"].([]interface{})
	if len(toolsList) != 9 {
		t.Errorf("expected 9 tools, got %d", len(toolsList))
	}

	// 3. knodexa_get_context
	ctxParams, _ := json.Marshal(CallToolParams{
		Name: "knodexa_get_context",
		Arguments: map[string]interface{}{
			"task_id":      "A.1",
			"token_budget": 1500,
		},
	})
	ctxResp := sendRPC(Request{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params:  ctxParams,
	})
	if ctxResp.Error != nil {
		t.Fatalf("get_context tool error: %+v", ctxResp.Error)
	}

	// 4. knodexa_propose_decision
	decParams, _ := json.Marshal(CallToolParams{
		Name: "knodexa_propose_decision",
		Arguments: map[string]interface{}{
			"task_id":   "A.1",
			"title":     "Use stdio JSON-RPC",
			"rationale": "Direct process pipes are fast and portable",
		},
	})
	decResp := sendRPC(Request{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params:  decParams,
	})
	if decResp.Error != nil {
		t.Fatalf("propose_decision tool error: %+v", decResp.Error)
	}

	// 5. knodexa_record_observation
	obsParams, _ := json.Marshal(CallToolParams{
		Name: "knodexa_record_observation",
		Arguments: map[string]interface{}{
			"task_id": "A.1",
			"content": "Observed 0ms latency in local pipe transport",
		},
	})
	obsResp := sendRPC(Request{
		JSONRPC: "2.0",
		ID:      5,
		Method:  "tools/call",
		Params:  obsParams,
	})
	if obsResp.Error != nil {
		t.Fatalf("record_observation error: %+v", obsResp.Error)
	}

	// 6. knodexa_task_status
	statusParams, _ := json.Marshal(CallToolParams{
		Name: "knodexa_task_status",
		Arguments: map[string]interface{}{
			"task_id": "A.1",
		},
	})
	statusResp := sendRPC(Request{
		JSONRPC: "2.0",
		ID:      6,
		Method:  "tools/call",
		Params:  statusParams,
	})
	if statusResp.Error != nil {
		t.Fatalf("task_status error: %+v", statusResp.Error)
	}

	// Verify decisions were saved in filesystem
	decisions, _ := store.ListDecisions()
	if len(decisions) != 1 {
		t.Errorf("expected 1 decision in filesystem, got %d", len(decisions))
	}
}

func TestMCPUnknownMethod(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewStorage(tmpDir)
	_, _ = store.Init("Test", "Desc")

	srv, err := NewServer(tmpDir)
	if err != nil {
		t.Fatalf("failed to init server: %v", err)
	}

	req := Request{
		JSONRPC: "2.0",
		ID:      99,
		Method:  "unknown/method",
	}
	reqBytes, _ := json.Marshal(req)
	in := bytes.NewReader(append(reqBytes, '\n'))
	var out bytes.Buffer

	_ = srv.Serve(in, &out)

	var resp Response
	_ = json.Unmarshal(out.Bytes(), &resp)
	if resp.Error == nil || resp.Error.Code != CodeMethodNotFound {
		t.Errorf("expected MethodNotFound error, got %+v", resp.Error)
	}
}
