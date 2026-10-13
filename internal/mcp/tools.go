package mcp

import (
	"fmt"
	"time"

	"knodexa/internal/context"
	"knodexa/internal/git"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"knodexa/internal/verify"
)

// ListTools returns definitions for all 9 required Knodexa tools.
func ListTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "knodexa_get_context",
			Description: "Rehydrate the minimal, high-signal engineering context required to resume or execute a task.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id":      map[string]interface{}{"type": "string", "description": "The target task ID (e.g. 'A.2')"},
					"token_budget": map[string]interface{}{"type": "integer", "description": "Maximum token budget (default: 1500)"},
				},
				"required": []string{"task_id"},
			},
		},
		{
			Name:        "knodexa_find_related",
			Description: "Discover cross-task decisions, lessons, failures, and related tasks.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{"type": "string", "description": "Optional source task ID"},
					"query":   map[string]interface{}{"type": "string", "description": "Search keyword or phrase"},
					"types":   map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
					"limit":   map[string]interface{}{"type": "integer", "description": "Max results to return"},
				},
			},
		},
		{
			Name:        "knodexa_record_observation",
			Description: "Record an explicit, unverified observation from agent execution or tests.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{"type": "string", "description": "Active task ID"},
					"content": map[string]interface{}{"type": "string", "description": "Observation details"},
					"files":   map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				},
				"required": []string{"task_id", "content"},
			},
		},
		{
			Name:        "knodexa_propose_decision",
			Description: "Propose an architectural or implementation decision for a task.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id":                 map[string]interface{}{"type": "string", "description": "Task ID"},
					"title":                   map[string]interface{}{"type": "string", "description": "Decision title"},
					"rationale":               map[string]interface{}{"type": "string", "description": "Why this decision is made"},
					"alternatives_considered": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
					"constraints":             map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				},
				"required": []string{"task_id", "title", "rationale"},
			},
		},
		{
			Name:        "knodexa_propose_lesson",
			Description: "Record a candidate reusable lesson learned from completed task work.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id":       map[string]interface{}{"type": "string", "description": "Source task ID"},
					"title":         map[string]interface{}{"type": "string", "description": "Lesson title"},
					"description":   map[string]interface{}{"type": "string", "description": "Explanation and guidance"},
					"applicability": map[string]interface{}{"type": "string", "description": "When does this lesson apply?"},
					"tags":          map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
					"related_files": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				},
				"required": []string{"task_id", "title", "description", "applicability"},
			},
		},
		{
			Name:        "knodexa_add_evidence",
			Description: "Attach concrete test, benchmark, or Git evidence to a decision, lesson, or task.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"node_id":        map[string]interface{}{"type": "string", "description": "Target node ID (e.g. D-001, L-017)"},
					"title":          map[string]interface{}{"type": "string", "description": "Evidence title"},
					"evidence_type":  map[string]interface{}{"type": "string", "enum": []string{"TEST_RUN", "BENCHMARK", "BUILD_TEST", "GIT_DIFF", "MANUAL_SIGN_OFF"}},
					"command":        map[string]interface{}{"type": "string", "description": "Command executed"},
					"exit_code":      map[string]interface{}{"type": "integer", "description": "Process exit code"},
					"output_summary": map[string]interface{}{"type": "string", "description": "Output snippet"},
					"commit_sha":     map[string]interface{}{"type": "string", "description": "Git commit SHA"},
				},
				"required": []string{"node_id", "title", "evidence_type"},
			},
		},
		{
			Name:        "knodexa_verify",
			Description: "Change knowledge verification state to VERIFIED backed by evidence.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"node_id":            map[string]interface{}{"type": "string", "description": "Target node ID (e.g. D-001, L-017)"},
					"evidence_ids":       map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
					"verification_notes": map[string]interface{}{"type": "string", "description": "Explanation of verification"},
				},
				"required": []string{"node_id", "evidence_ids"},
			},
		},
		{
			Name:        "knodexa_resume_task",
			Description: "Transition a task to ACTIVE, capture Git baseline, and rehydrate context.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id":      map[string]interface{}{"type": "string", "description": "Task ID to resume"},
					"token_budget": map[string]interface{}{"type": "integer", "description": "Token budget"},
				},
				"required": []string{"task_id"},
			},
		},
		{
			Name:        "knodexa_task_status",
			Description: "Return current task state, metrics, and relationships.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{"type": "string", "description": "Task ID to inspect"},
				},
				"required": []string{"task_id"},
			},
		},
	}
}

// ToolHandler routes and executes MCP tools.
type ToolHandler struct {
	store   *storage.Storage
	db      *index.DB
	indexer *index.Indexer
	verify  *verify.Engine
	context *context.Builder
	git     *git.Inspector
}

func NewToolHandler(store *storage.Storage, db *index.DB, idx *index.Indexer, ver *verify.Engine, ctx *context.Builder, gi *git.Inspector) *ToolHandler {
	return &ToolHandler{
		store:   store,
		db:      db,
		indexer: idx,
		verify:  ver,
		context: ctx,
		git:     gi,
	}
}

func (h *ToolHandler) HandleTool(name string, args map[string]interface{}) (interface{}, error) {
	switch name {
	case "knodexa_get_context":
		return h.handleGetContext(args)
	case "knodexa_find_related":
		return h.handleFindRelated(args)
	case "knodexa_record_observation":
		return h.handleRecordObservation(args)
	case "knodexa_propose_decision":
		return h.handleProposeDecision(args)
	case "knodexa_propose_lesson":
		return h.handleProposeLesson(args)
	case "knodexa_add_evidence":
		return h.handleAddEvidence(args)
	case "knodexa_verify":
		return h.handleVerify(args)
	case "knodexa_resume_task":
		return h.handleResumeTask(args)
	case "knodexa_task_status":
		return h.handleTaskStatus(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (h *ToolHandler) handleGetContext(args map[string]interface{}) (interface{}, error) {
	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	budget := 1500
	if b, ok := args["token_budget"].(float64); ok && b > 0 {
		budget = int(b)
	}
	return h.context.Build(taskID, budget)
}

func (h *ToolHandler) handleResumeTask(args map[string]interface{}) (interface{}, error) {
	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	task, err := h.store.ReadTask(taskID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	task.Status = model.TaskStatusActive
	task.UpdatedAt = now

	// Checkpoint baseline commit
	if chk, err := h.git.GetCheckpoint(); err == nil && chk != nil && chk.CommitSHA != "" {
		task.BaselineCommit = chk.CommitSHA
	}
	_ = h.store.SaveTask(task)
	_ = h.indexer.Sync()

	budget := 1500
	if b, ok := args["token_budget"].(float64); ok && b > 0 {
		budget = int(b)
	}
	return h.context.Build(taskID, budget)
}

func (h *ToolHandler) handleTaskStatus(args map[string]interface{}) (interface{}, error) {
	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	task, err := h.store.ReadTask(taskID)
	if err != nil {
		return nil, err
	}

	decisions, _ := h.db.GetDecisionsForTask(taskID)
	failures, _ := h.db.GetFailuresForTask(taskID)

	return map[string]interface{}{
		"id":              task.ID,
		"parent_id":       task.ParentID,
		"title":           task.Title,
		"status":          task.Status,
		"goal":            task.Goal,
		"decisions_count": len(decisions),
		"failures_count":  len(failures),
		"last_updated":    task.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (h *ToolHandler) handleRecordObservation(args map[string]interface{}) (interface{}, error) {
	taskID, _ := args["task_id"].(string)
	content, _ := args["content"].(string)
	if taskID == "" || content == "" {
		return nil, fmt.Errorf("task_id and content are required")
	}

	var files []string
	if rawFiles, ok := args["files"].([]interface{}); ok {
		for _, f := range rawFiles {
			if s, ok := f.(string); ok {
				files = append(files, s)
			}
		}
	}

	obsID := fmt.Sprintf("OBS-%d", time.Now().UnixNano()%1000000)
	obs := &model.ObservationData{
		ID:        obsID,
		TaskID:    taskID,
		Content:   content,
		Files:     files,
		CreatedAt: time.Now().UTC(),
	}
	if err := h.store.SaveObservation(obs); err != nil {
		return nil, err
	}
	_ = h.indexer.Sync()

	return map[string]interface{}{
		"id":       obsID,
		"status":   "OBSERVATION",
		"recorded": true,
	}, nil
}

func (h *ToolHandler) handleProposeDecision(args map[string]interface{}) (interface{}, error) {
	taskID, _ := args["task_id"].(string)
	title, _ := args["title"].(string)
	rationale, _ := args["rationale"].(string)
	if taskID == "" || title == "" || rationale == "" {
		return nil, fmt.Errorf("task_id, title, and rationale are required")
	}

	decID := fmt.Sprintf("D-%d", time.Now().UnixNano()%1000000)
	now := time.Now().UTC()
	doc := &model.DecisionDocument{
		Frontmatter: model.DecisionFrontmatter{
			ID:         decID,
			Title:      title,
			Status:     model.StatusProposed,
			TaskID:     taskID,
			CreatedAt:  now,
			UpdatedAt:  now,
			Confidence: 0.8,
		},
		Body: fmt.Sprintf("## Rationale\n%s\n", rationale),
	}

	if err := h.store.SaveDecision(doc); err != nil {
		return nil, err
	}
	_ = h.indexer.Sync()

	return map[string]interface{}{
		"id":     decID,
		"status": "PROPOSED",
		"file":   fmt.Sprintf(".knodexa/decisions/%s.md", decID),
	}, nil
}

func (h *ToolHandler) handleProposeLesson(args map[string]interface{}) (interface{}, error) {
	taskID, _ := args["task_id"].(string)
	title, _ := args["title"].(string)
	desc, _ := args["description"].(string)
	app, _ := args["applicability"].(string)

	if taskID == "" || title == "" || desc == "" || app == "" {
		return nil, fmt.Errorf("task_id, title, description, and applicability are required")
	}

	var tags []string
	if rawTags, ok := args["tags"].([]interface{}); ok {
		for _, t := range rawTags {
			if s, ok := t.(string); ok {
				tags = append(tags, s)
			}
		}
	}

	var files []string
	if rawFiles, ok := args["related_files"].([]interface{}); ok {
		for _, f := range rawFiles {
			if s, ok := f.(string); ok {
				files = append(files, s)
			}
		}
	}

	lesID := fmt.Sprintf("L-%d", time.Now().UnixNano()%1000000)
	now := time.Now().UTC()
	doc := &model.LessonDocument{
		Frontmatter: model.LessonFrontmatter{
			ID:            lesID,
			Title:         title,
			Status:        model.StatusProposed,
			SourceTaskID:  taskID,
			CreatedAt:     now,
			UpdatedAt:     now,
			Confidence:    0.8,
			Applicability: app,
			Tags:          tags,
			RelatedFiles:  files,
		},
		Body: desc,
	}

	if err := h.store.SaveLesson(doc); err != nil {
		return nil, err
	}
	_ = h.indexer.Sync()

	return map[string]interface{}{
		"id":     lesID,
		"status": "PROPOSED",
		"file":   fmt.Sprintf(".knodexa/lessons/%s.md", lesID),
	}, nil
}

func (h *ToolHandler) handleAddEvidence(args map[string]interface{}) (interface{}, error) {
	nodeID, _ := args["node_id"].(string)
	title, _ := args["title"].(string)
	evTypeStr, _ := args["evidence_type"].(string)

	if nodeID == "" || title == "" || evTypeStr == "" {
		return nil, fmt.Errorf("node_id, title, and evidence_type are required")
	}

	evID := fmt.Sprintf("E-%d", time.Now().UnixNano()%1000000)
	ev := &model.EvidenceData{
		ID:           evID,
		Title:        title,
		NodeID:       nodeID,
		EvidenceType: model.EvidenceType(evTypeStr),
		CreatedAt:    time.Now().UTC(),
		VerifiedBy:   "agent",
	}

	if cmd, ok := args["command"].(string); ok {
		ev.Command = cmd
	}
	if ec, ok := args["exit_code"].(float64); ok {
		ev.ExitCode = int(ec)
	}
	if summary, ok := args["output_summary"].(string); ok {
		ev.OutputSummary = summary
	}
	if commit, ok := args["commit_sha"].(string); ok {
		ev.CommitSHA = commit
	}

	if err := h.verify.AddEvidence(ev, "agent"); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":       evID,
		"node_id":  nodeID,
		"recorded": true,
	}, nil
}

func (h *ToolHandler) handleVerify(args map[string]interface{}) (interface{}, error) {
	nodeID, _ := args["node_id"].(string)
	if nodeID == "" {
		return nil, fmt.Errorf("node_id is required")
	}

	var evIDs []string
	if rawEv, ok := args["evidence_ids"].([]interface{}); ok {
		for _, e := range rawEv {
			if s, ok := e.(string); ok {
				evIDs = append(evIDs, s)
			}
		}
	}
	notes, _ := args["verification_notes"].(string)

	if err := h.verify.VerifyNode(nodeID, evIDs, notes, "agent"); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":             nodeID,
		"current_status": "VERIFIED",
		"evidence_count": len(evIDs),
	}, nil
}

func (h *ToolHandler) handleFindRelated(args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	taskID, _ := args["task_id"].(string)
	limit := 5
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	type Match struct {
		ID             string  `json:"id"`
		Type           string  `json:"type"`
		Title          string  `json:"title"`
		Status         string  `json:"status"`
		RelevanceScore float64 `json:"relevance_score"`
	}

	matches := make([]Match, 0)
	if query != "" {
		nodes, err := h.db.SearchNodes(query, nil, limit)
		if err == nil {
			for _, n := range nodes {
				matches = append(matches, Match{
					ID:             n.ID,
					Type:           string(n.Type),
					Title:          n.Title,
					Status:         n.Status,
					RelevanceScore: 0.85,
				})
			}
		}
	}

	if taskID != "" && len(matches) < limit {
		task, _ := h.store.ReadTask(taskID)
		var files []string
		if task != nil {
			files = task.TouchesFiles
		}
		lessons, err := h.db.GetRelatedLessons(taskID, files, nil, limit-len(matches))
		if err == nil {
			for _, l := range lessons {
				matches = append(matches, Match{
					ID:             l.ID,
					Type:           "LESSON",
					Title:          l.Title,
					Status:         l.Status,
					RelevanceScore: l.Confidence,
				})
			}
		}
	}

	return map[string]interface{}{
		"matches": matches,
	}, nil
}
