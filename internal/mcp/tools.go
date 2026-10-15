package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"knodexa/internal/approach"
	"knodexa/internal/context"
	"knodexa/internal/continuity"
	"knodexa/internal/git"
	"knodexa/internal/identity"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"knodexa/internal/verify"
)

// ListTools returns the continuity, knowledge, and detail tools.
func ListTools() []ToolDefinition {
	definitions := []ToolDefinition{
		{Name: "knodexa_supersede_knowledge", Description: "Retire a decision or lesson using an existing VERIFIED replacement of the same kind and explicit reason. Preserve history and invalidation provenance; retired knowledge is excluded from active context.", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"node_id": map[string]interface{}{"type": "string"}, "replacement_id": map[string]interface{}{"type": "string"}, "reason": map[string]interface{}{"type": "string"}}, "required": []string{"node_id", "replacement_id", "reason"}}},
		{Name: "knodexa_record_work", Description: "Execute explicit verification commands, check proposed structural agreement, then save aliases, source identity and versioned approach in one canonical task update. Failures do not install partial memory.", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"task_id": map[string]interface{}{"type": "string"}, "agreement": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"aliases": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}, "files": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}, "approach": map[string]interface{}{"type": "object", "description": "summary, rules, scopes, references [{path}], checks, and change_reason for replacement"}}, "required": []string{"approach"}}, "verify": map[string]interface{}{"type": "array", "minItems": 1, "items": map[string]interface{}{"type": "string"}}}, "required": []string{"task_id", "agreement", "verify"}}},
		{Name: "knodexa_prepare_work", Description: "Restore work identity, agreements, Git baseline, drift checks and bounded reference excerpts in one operation. Review needs_review before coding.", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"request": map[string]interface{}{"type": "string"}, "select_task": map[string]interface{}{"type": "string"}, "files": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}, "previous_task_id": map[string]interface{}{"type": "string"}, "token_budget": map[string]interface{}{"type": "integer", "minimum": 1}}, "required": []string{"request"}}},
		{Name: "knodexa_capture_identity", Description: "Capture stable work ID, source hashes, Go symbols, and imports after successful work; preserve completion state and baseline.", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"task_id": map[string]interface{}{"type": "string"}, "files": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}}, "required": []string{"task_id"}}},
		{Name: "knodexa_resolve_request", Description: "Identify whether an incoming natural-language request continues recorded work before creating any task. Restore uniquely matched work with baseline and agreements; expose weak or overlapping matches for clarification. Does not create or reopen tasks.", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
			"request": map[string]interface{}{"type": "string", "minLength": 1}, "files": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}, "select_task": map[string]interface{}{"type": "string", "description": "Explicit existing-work selection after review"}, "intent": map[string]interface{}{"type": "string", "enum": []string{"auto", "continue", "new", "replace"}, "default": "auto"}, "previous_task_id": map[string]interface{}{"type": "string"}, "token_budget": map[string]interface{}{"type": "integer", "minimum": 1}, "detail_level": map[string]interface{}{"type": "string", "enum": []string{"compact", "full"}}}, "required": []string{"request"}}},
		{Name: "knodexa_add_work_aliases", Description: "Record phrases that refer to established work so future requests can recover its identity without task IDs. Completion status and agreements are preserved.", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"task_id": map[string]interface{}{"type": "string"}, "aliases": map[string]interface{}{"type": "array", "minItems": 1, "items": map[string]interface{}{"type": "string"}}}, "required": []string{"task_id", "aliases"}}},
		{Name: "knodexa_reuse_approach", Description: "Explicitly link an existing task's established approach to other work regardless of task names or tree position.", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"task_id": map[string]interface{}{"type": "string"}, "source_tasks": map[string]interface{}{"type": "array", "minItems": 1, "items": map[string]interface{}{"type": "string"}}}, "required": []string{"task_id", "source_tasks"}}},
		{Name: "knodexa_set_approach", Description: "Record an explicitly agreed implementation approach on a task; descendants inherit rules, reference hashes, skills, and literal checks. This records an agreement, not automatic proof of design quality.", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
			"scopes": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}, "task_id": map[string]interface{}{"type": "string"}, "summary": map[string]interface{}{"type": "string"}, "rules": map[string]interface{}{"type": "array", "minItems": 1, "items": map[string]interface{}{"type": "string"}}, "reference_files": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}, "skill_files": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}, "checks": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"path": map[string]interface{}{"type": "string"}, "must_contain": map[string]interface{}{"type": "string"}, "must_not_contain": map[string]interface{}{"type": "string"}}, "required": []string{"path"}}}}, "required": []string{"task_id", "summary", "rules"}}},
		{Name: "knodexa_check_approach", Description: "Check inherited reference hashes and explicit literal rules; returns failures and marks semantic/design review as still required.", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"task_id": map[string]interface{}{"type": "string"}}, "required": []string{"task_id"}}},
		{Name: "knodexa_get_detail", Description: "Expand one decision or lesson with full rationale, applicability, constraints, alternatives, and evidence IDs.", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"knowledge_id": map[string]interface{}{"type": "string"}}, "required": []string{"knowledge_id"}}},
		{
			Name:        "knodexa_get_context",
			Description: "Rehydrate the minimal, high-signal engineering context required to resume or execute a task.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id":      map[string]interface{}{"type": "string", "description": "The target task ID (e.g. 'A.2')"},
					"token_budget": map[string]interface{}{"type": "integer", "minimum": 1, "description": "Human-text token estimate budget (default: 1500)"},
					"detail_level": map[string]interface{}{"type": "string", "enum": []string{"compact", "full"}, "default": "compact"},
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
					"summary":                 map[string]interface{}{"type": "string", "description": "Concise actionable rule; detailed rationale is expanded on demand"},
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
					"summary":       map[string]interface{}{"type": "string", "description": "Concise actionable guidance"},
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
					"output_summary": map[string]interface{}{"type": "string", "description": "Recorded output or sign-off details; this tool does not execute commands"},
					"commit_sha":     map[string]interface{}{"type": "string", "description": "Git commit SHA"},
					"verified_by":    map[string]interface{}{"type": "string", "description": "Reporter identity; required for manual sign-off"},
				},
				"required": []string{"node_id", "title", "evidence_type", "output_summary"},
				"anyOf":    []interface{}{map[string]interface{}{"properties": map[string]interface{}{"evidence_type": map[string]interface{}{"const": "MANUAL_SIGN_OFF"}}, "required": []string{"verified_by"}}, map[string]interface{}{"required": []string{"command", "exit_code"}}},
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
			Description: "Restore task context and compare its Git baseline. Completed task records stay unchanged unless reopen or accept_baseline is explicitly requested.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id":          map[string]interface{}{"type": "string", "description": "Task ID to resume"},
					"token_budget":     map[string]interface{}{"type": "integer", "minimum": 1, "description": "Human-text token estimate budget"},
					"detail_level":     map[string]interface{}{"type": "string", "enum": []string{"compact", "full"}, "default": "compact"},
					"previous_task_id": map[string]interface{}{"type": "string", "description": "Previous focus whose unrelated conventions should be set aside"},
					"accept_baseline":  map[string]interface{}{"type": "boolean", "default": false, "description": "Explicitly acknowledge and replace a stored baseline; otherwise drift is preserved"},
					"reopen":           map[string]interface{}{"type": "boolean", "default": false, "description": "Explicitly reactivate completed work"},
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
	for i := range definitions {
		if definitions[i].Name == "knodexa_set_approach" {
			properties := definitions[i].InputSchema.(map[string]interface{})["properties"].(map[string]interface{})
			properties["change_reason"] = map[string]interface{}{"type": "string"}
			properties["reference_reconciliations"] = referenceReconciliationSchema()
			properties["behavior_checks"] = behaviorCheckSchema()
			checks := properties["checks"].(map[string]interface{})["items"].(map[string]interface{})["properties"].(map[string]interface{})
			checks["ast_kind"] = map[string]interface{}{"type": "string", "enum": []string{"require-call", "forbid-call", "forbid-import", "require-function"}}
			checks["symbol"] = map[string]interface{}{"type": "string"}
		}
	}
	for i := range definitions {
		if definitions[i].Name == "knodexa_record_work" {
			properties := definitions[i].InputSchema.(map[string]interface{})["properties"].(map[string]interface{})
			agreement := properties["agreement"].(map[string]interface{})["properties"].(map[string]interface{})
			agreement["approach"] = map[string]interface{}{"type": "object", "description": "summary, rules, scopes, references, checks, behavior_checks, change_reason; restored behavioral checks must pass before capture", "properties": map[string]interface{}{"reference_reconciliations": referenceReconciliationSchema(), "behavior_checks": behaviorCheckSchema()}}
			fields := map[string]interface{}{}
			for _, field := range []string{"source_task", "original_path", "current_path", "reason"} {
				fields[field] = map[string]interface{}{"type": "string", "minLength": 1}
			}
			agreement["reference_reviews"] = map[string]interface{}{"type": "array", "description": "Explicitly reviewed inherited refactors; derives source version and both hashes, never rewrites ancestors", "items": map[string]interface{}{"type": "object", "properties": fields, "required": []string{"source_task", "original_path", "current_path", "reason"}}}
		}
	}
	return definitions
}

func behaviorCheckSchema() map[string]interface{} {
	return map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "object", "properties": map[string]interface{}{
		"name": map[string]interface{}{"type": "string", "minLength": 1}, "command": map[string]interface{}{"type": "string", "minLength": 1},
		"protected_files": map[string]interface{}{"type": "array", "minItems": 1, "items": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"path": map[string]interface{}{"type": "string"}, "sha256": map[string]interface{}{"type": "string"}}, "required": []string{"path"}}},
	}, "required": []string{"name", "command", "protected_files"}}}
}

func referenceReconciliationSchema() map[string]interface{} {
	fields := []string{"source_task", "original_path", "original_sha256", "current_path", "current_sha256", "reason"}
	properties := map[string]interface{}{}
	for _, field := range fields {
		properties[field] = map[string]interface{}{"type": "string", "minLength": 1}
	}
	properties["source_version"] = map[string]interface{}{"type": "integer", "minimum": 1}
	return map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "object", "properties": properties, "required": append(fields, "source_version")}}
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
	case "knodexa_record_work":
		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return nil, fmt.Errorf("task_id required")
		}
		data, err := json.Marshal(args["agreement"])
		if err != nil {
			return nil, err
		}
		var input continuity.RecordInput
		if err := json.Unmarshal(data, &input); err != nil {
			return nil, err
		}
		data, err = json.Marshal(args["verify"])
		if err != nil {
			return nil, err
		}
		var verify []string
		if err := json.Unmarshal(data, &verify); err != nil {
			return nil, err
		}
		return continuity.NewService(h.store, h.db, h.indexer, h.context, h.git).Record(taskID, input, verify, 5*time.Minute)
	case "knodexa_capture_identity":
		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return nil, fmt.Errorf("task_id required")
		}
		var files []string
		if raw, ok := args["files"]; ok {
			data, err := json.Marshal(raw)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(data, &files); err != nil {
				return nil, err
			}
		}
		if err := identity.Capture(h.store, h.db, h.indexer, taskID, files); err != nil {
			return nil, err
		}
		return map[string]interface{}{"task_id": taskID, "captured": true}, nil
	case "knodexa_resolve_request", "knodexa_prepare_work":
		data, err := json.Marshal(args)
		if err != nil {
			return nil, err
		}
		var request identity.Request
		if err := json.Unmarshal(data, &request); err != nil {
			return nil, err
		}
		budget, full, err := contextOptions(args)
		if err != nil {
			return nil, err
		}
		previous := ""
		if raw, exists := args["previous_task_id"]; exists {
			var ok bool
			previous, ok = raw.(string)
			if !ok {
				return nil, fmt.Errorf("previous_task_id must be a string")
			}
		}
		service := continuity.NewService(h.store, h.db, h.indexer, h.context, h.git)
		if name == "knodexa_prepare_work" {
			return service.Prepare(request, budget, continuity.ResumeOptions{Full: full, PreviousTaskID: previous})
		}
		return service.ResolveRequest(request, budget, continuity.ResumeOptions{Full: full, PreviousTaskID: previous})
	case "knodexa_add_work_aliases":
		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return nil, fmt.Errorf("task_id is required")
		}
		data, err := json.Marshal(args["aliases"])
		if err != nil {
			return nil, err
		}
		var aliases []string
		if err := json.Unmarshal(data, &aliases); err != nil {
			return nil, err
		}
		if err := identity.AddAliases(h.store, h.db, h.indexer, taskID, aliases); err != nil {
			return nil, err
		}
		return map[string]interface{}{"task_id": taskID, "aliases_added": aliases}, nil

	case "knodexa_reuse_approach":
		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return nil, fmt.Errorf("task_id is required")
		}
		data, err := json.Marshal(args["source_tasks"])
		if err != nil {
			return nil, err
		}
		var sources []string
		if err := json.Unmarshal(data, &sources); err != nil {
			return nil, err
		}
		if err := approach.Reuse(h.store, h.db, h.indexer, taskID, sources); err != nil {
			return nil, err
		}
		return map[string]interface{}{"task_id": taskID, "source_tasks": sources, "linked": true}, nil

	case "knodexa_set_approach":
		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return nil, fmt.Errorf("task_id is required")
		}
		data, err := json.Marshal(args)
		if err != nil {
			return nil, err
		}
		var value model.TaskApproach
		if err := json.Unmarshal(data, &value); err != nil {
			return nil, err
		}
		if raw, exists := args["reference_files"]; exists {
			data, err := json.Marshal(raw)
			if err != nil {
				return nil, err
			}
			var paths []string
			if err := json.Unmarshal(data, &paths); err != nil {
				return nil, err
			}
			value.References = nil
			for _, path := range paths {
				value.References = append(value.References, model.ApproachReference{Path: path})
			}
		}
		if err := approach.Set(h.store, h.db, h.indexer, taskID, value); err != nil {
			return nil, err
		}
		return map[string]interface{}{"task_id": taskID, "recorded": true, "agreement_not_quality_proof": true}, nil
	case "knodexa_check_approach":
		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return nil, fmt.Errorf("task_id is required")
		}
		if err := h.indexer.Sync(); err != nil {
			return nil, err
		}
		return approach.Check(h.store, h.db, taskID)

	case "knodexa_get_detail":
		id, ok := args["knowledge_id"].(string)
		if !ok || id == "" {
			return nil, fmt.Errorf("knowledge_id is required")
		}
		if err := h.indexer.Sync(); err != nil {
			return nil, err
		}
		return h.context.GetDetail(id)
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
	case "knodexa_supersede_knowledge":
		id, _ := args["node_id"].(string)
		replacement, _ := args["replacement_id"].(string)
		reason, _ := args["reason"].(string)
		if err := h.verify.SupersedeNode(id, replacement, reason, "agent"); err != nil {
			return nil, err
		}
		return map[string]interface{}{"id": id, "replacement_id": replacement, "status": "SUPERSEDED"}, nil
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
	budget, full, err := contextOptions(args)
	if err != nil {
		return nil, err
	}
	if err := h.indexer.Sync(); err != nil {
		return nil, err
	}
	return h.context.BuildWithOptions(taskID, budget, context.BuildOptions{Compact: !full})
}

func (h *ToolHandler) handleResumeTask(args map[string]interface{}) (interface{}, error) {
	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	budget, full, err := contextOptions(args)
	if err != nil {
		return nil, err
	}
	accept := false
	reopen := false
	if value, exists := args["reopen"]; exists {
		var ok bool
		reopen, ok = value.(bool)
		if !ok {
			return nil, fmt.Errorf("reopen must be boolean")
		}
	}
	if value, exists := args["accept_baseline"]; exists {
		var ok bool
		accept, ok = value.(bool)
		if !ok {
			return nil, fmt.Errorf("accept_baseline must be boolean")
		}
	}
	previous := ""
	if raw, exists := args["previous_task_id"]; exists {
		var ok bool
		previous, ok = raw.(string)
		if !ok {
			return nil, fmt.Errorf("previous_task_id must be a string")
		}
	}
	return continuity.NewService(h.store, h.db, h.indexer, h.context, h.git).Resume(taskID, budget, continuity.ResumeOptions{Full: full, AcceptBaseline: accept, Reopen: reopen, PreviousTaskID: previous})
}

func contextOptions(args map[string]interface{}) (int, bool, error) {
	budget := 1500
	if value, exists := args["token_budget"]; exists {
		number, ok := value.(float64)
		if integer, isInt := value.(int); isInt {
			number = float64(integer)
			ok = true
		}
		if !ok || math.IsNaN(number) || math.IsInf(number, 0) || math.Trunc(number) != number || number <= 0 || number > 2147483647 {
			return 0, false, fmt.Errorf("token_budget must be a positive integer")
		}
		budget = int(number)
	}
	level := "compact"
	if value, exists := args["detail_level"]; exists {
		var ok bool
		level, ok = value.(string)
		if !ok {
			return 0, false, fmt.Errorf("detail_level must be compact or full")
		}
	}
	if level != "compact" && level != "full" {
		return 0, false, fmt.Errorf("detail_level must be compact or full")
	}
	return budget, level == "full", nil
}

func (h *ToolHandler) handleTaskStatus(args map[string]interface{}) (interface{}, error) {
	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	if err := h.indexer.Sync(); err != nil {
		return nil, err
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

	obsID, err := newKnowledgeID("OBS")
	if err != nil {
		return nil, err
	}
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
	if err := h.indexer.Sync(); err != nil {
		return nil, err
	}

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

	decID, err := newKnowledgeID("D")
	if err != nil {
		return nil, err
	}
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

	if summary, ok := args["summary"].(string); ok {
		doc.Frontmatter.Summary = summary
	}
	for _, field := range []struct{ key, heading string }{{"constraints", "Constraints"}, {"alternatives_considered", "Rejected alternatives"}} {
		if raw, exists := args[field.key]; exists {
			values, ok := raw.([]interface{})
			if !ok {
				return nil, fmt.Errorf("%s must be an array of strings", field.key)
			}
			doc.Body += "\n## " + field.heading + "\n"
			for _, value := range values {
				text, ok := value.(string)
				if !ok {
					return nil, fmt.Errorf("%s must contain strings", field.key)
				}
				doc.Body += "- " + text + "\n"
				if field.key == "constraints" {
					doc.Frontmatter.Constraints = append(doc.Frontmatter.Constraints, text)
				} else {
					doc.Frontmatter.AlternativesConsidered = append(doc.Frontmatter.AlternativesConsidered, text)
				}
			}
		}
	}

	if err := h.store.SaveDecision(doc); err != nil {
		return nil, err
	}
	if err := h.indexer.Sync(); err != nil {
		return nil, err
	}

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

	lesID, err := newKnowledgeID("L")
	if err != nil {
		return nil, err
	}
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

	if summary, ok := args["summary"].(string); ok {
		doc.Frontmatter.Summary = summary
	}

	if err := h.store.SaveLesson(doc); err != nil {
		return nil, err
	}
	if err := h.indexer.Sync(); err != nil {
		return nil, err
	}

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

	evID, err := newKnowledgeID("E")
	if err != nil {
		return nil, err
	}
	ev := &model.EvidenceData{
		ID:           evID,
		Title:        title,
		NodeID:       nodeID,
		EvidenceType: model.EvidenceType(evTypeStr),
		CreatedAt:    time.Now().UTC(),
		VerifiedBy:   "agent",
	}

	if reporter, ok := args["verified_by"].(string); ok {
		ev.VerifiedBy = strings.TrimSpace(reporter)
	}
	if ev.EvidenceType == model.EvidenceManualSignOff {
		reporter, ok := args["verified_by"].(string)
		if !ok || strings.TrimSpace(reporter) == "" {
			return nil, fmt.Errorf("verified_by is required for manual sign-off")
		}
	}

	if cmd, ok := args["command"].(string); ok {
		ev.Command = cmd
	}
	if ev.EvidenceType != model.EvidenceManualSignOff {
		ec, ok := args["exit_code"].(float64)
		if !ok || math.IsNaN(ec) || math.IsInf(ec, 0) || math.Trunc(ec) != ec || ec < -2147483648 || ec > 2147483647 {
			return nil, fmt.Errorf("explicit integer exit_code is required for execution evidence")
		}
		ev.ExitCode = int(ec)
		if strings.TrimSpace(ev.Command) == "" {
			return nil, fmt.Errorf("command is required for execution evidence")
		}
	}
	if summary, ok := args["output_summary"].(string); ok {
		ev.OutputSummary = summary
	}
	if commit, ok := args["commit_sha"].(string); ok {
		ev.CommitSHA = commit
	}

	if strings.TrimSpace(ev.OutputSummary) == "" {
		return nil, fmt.Errorf("output_summary is required")
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

// newKnowledgeID avoids collisions between independent processes and sessions.
func newKnowledgeID(prefix string) (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate knowledge ID: %w", err)
	}
	return prefix + "-" + hex.EncodeToString(bytes[:]), nil
}
