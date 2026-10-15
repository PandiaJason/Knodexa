package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"knodexa/internal/context"
	"knodexa/internal/git"
	"knodexa/internal/index"
	"knodexa/internal/storage"
	"knodexa/internal/verify"
)

// Server coordinates the JSON-RPC 2.0 stdio loop for MCP.
type Server struct {
	handler *ToolHandler
	mu      sync.Mutex
}

// NewServer initializes an MCP Server instance.
func NewServer(rootDir string) (*Server, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		absRoot = rootDir
	}

	store := storage.NewStorage(absRoot)
	if !store.IsInitialized() {
		return nil, fmt.Errorf("knodexa repository is not initialized in %s; run 'knodexa init' first", absRoot)
	}

	dbPath := filepath.Join(store.KnodexaDir(), "cache.db")
	db, err := index.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index: %w", err)
	}

	idx := index.NewIndexer(db, store)
	if err := idx.Sync(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to sync index: %w", err)
	}

	gitInspector := git.NewInspector(absRoot)
	verifier := verify.NewEngine(store, idx, db)
	ctxBuilder := context.NewBuilder(db, store, gitInspector)

	handler := NewToolHandler(store, db, idx, verifier, ctxBuilder, gitInspector)
	return &Server{
		handler: handler,
	}, nil
}

// Serve reads JSON-RPC requests from in and writes responses to out.
func (s *Server) Serve(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	// Buffer up to 10MB per line for large evidence payloads
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(out, nil, CodeParseError, "Parse error")
			continue
		}

		s.handleRequest(out, &req)
	}

	return scanner.Err()
}

func (s *Server) handleRequest(out io.Writer, req *Request) {
	switch req.Method {
	case "initialize":
		result := map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{
					"listChanged": false,
				},
			},
			"serverInfo": map[string]interface{}{
				"name":    "knodexa",
				"version": "0.1.0",
			},
		}
		s.sendResult(out, req.ID, result)

	case "notifications/initialized":
		// No-op for initialized notification

	case "ping":
		s.sendResult(out, req.ID, map[string]interface{}{})

	case "tools/list":
		s.sendResult(out, req.ID, map[string]interface{}{
			"tools": ListTools(),
		})

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendError(out, req.ID, CodeInvalidParams, "Invalid params for tools/call")
			return
		}

		res, err := s.handler.HandleTool(params.Name, params.Arguments)
		if err != nil {
			s.sendResult(out, req.ID, CallToolResult{
				Content: []ToolContent{
					{Type: "text", Text: fmt.Sprintf("Error: %v", err)},
				},
				IsError: true,
			})
			return
		}

		textBytes, _ := json.MarshalIndent(res, "", "  ")
		s.sendResult(out, req.ID, CallToolResult{
			Content: []ToolContent{
				{Type: "text", Text: string(textBytes)},
			},
		})

	default:
		s.sendError(out, req.ID, CodeMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (s *Server) sendResult(out io.Writer, id interface{}, result interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	bytes, err := json.Marshal(resp)
	if err == nil {
		_, _ = out.Write(append(bytes, '\n'))
	}
}

func (s *Server) sendError(out io.Writer, id interface{}, code int, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
	bytes, err := json.Marshal(resp)
	if err == nil {
		_, _ = out.Write(append(bytes, '\n'))
	}
}

// RunServer is the entry point for 'knodexa mcp' command.
func RunServer(rootDir string) error {
	srv, err := NewServer(rootDir)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Knodexa MCP server listening on stdio (PID: %d)...\n", os.Getpid())
	return srv.Serve(os.Stdin, os.Stdout)
}
