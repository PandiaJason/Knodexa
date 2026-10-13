# Knodexa: Model Context Protocol (MCP) Specification

---

## 1. Overview

Knodexa exposes its engineering memory engine to AI coding agents via the Model Context Protocol (MCP, 2024-11-05 spec) over standard input/output (`stdio`).

The server operates locally alongside the agent inside the project root:
```bash
knodexa mcp
```

### Core Design Rules for Agent Responses
- **Token Efficiency:** No decorative Markdown formatting or repetitive boilerplate.
- **Structured Density:** JSON objects optimized for direct machine consumption.
- **Explicit Uncertainty:** All proposed knowledge includes confidence levels and unverified status tags.
- **Strict Input Validation:** IDs and file paths are sanitized; invalid queries fail fast with typed error codes.

---

## 2. Tool Definitions

### 2.1 `knodexa_get_context`
Rehydrates the minimal, high-signal engineering context required to execute or resume a task.

- **Parameters:**
  ```json
  {
    "type": "object",
    "properties": {
      "task_id": { "type": "string", "description": "The target task ID (e.g. 'A.2')" },
      "token_budget": { "type": "integer", "description": "Maximum approximate tokens for response (default: 1500)", "default": 1500 }
    },
    "required": ["task_id"]
  }
  ```
- **Response Format:**
  ```json
  {
    "task": {
      "id": "A.2",
      "parent_id": "A",
      "title": "Optimize integer slice sort",
      "goal": "Improve sorting performance by 2x for small arrays.",
      "status": "ACTIVE"
    },
    "accepted_constraints": [
      "Preserve public API",
      "Preserve deterministic output",
      "Existing correctness suite must pass"
    ],
    "previous_decisions": [
      { "id": "D-001", "title": "Benchmark before optimization", "status": "VERIFIED" },
      { "id": "D-004", "title": "Keep current fallback algorithm", "status": "VERIFIED" }
    ],
    "previous_failures": [
      { "id": "F-002", "summary": "Parallel implementation regressed small inputs", "cause": "Goroutine spawn overhead > partition savings" }
    ],
    "relevant_lessons": [
      {
        "id": "L-017",
        "title": "Short-circuit arrays under 32 elements to insertion sort",
        "source_task": "B.2",
        "confidence": 0.95,
        "verification_status": "VERIFIED"
      }
    ],
    "baseline": {
      "commit": "abc1234",
      "dirty": false
    },
    "required_verification": [
      "go test -v ./...",
      "go test -bench=BenchmarkSort ./..."
    ]
  }
  ```

---

### 2.2 `knodexa_resume_task`
Convenience tool that transitions a task to `ACTIVE`, verifies the current Git baseline, and returns rehydrated context.

- **Parameters:**
  ```json
  {
    "type": "object",
    "properties": {
      "task_id": { "type": "string", "description": "Task ID to resume" },
      "token_budget": { "type": "integer", "default": 1500 }
    },
    "required": ["task_id"]
  }
  ```
- **Response:** Same payload as `knodexa_get_context`, plus updated task status.

---

### 2.3 `knodexa_task_status`
Returns the status, metadata, and progress metrics for a task.

- **Parameters:**
  ```json
  {
    "type": "object",
    "properties": {
      "task_id": { "type": "string", "description": "Task ID to inspect" }
    },
    "required": ["task_id"]
  }
  ```
- **Response:**
  ```json
  {
    "id": "A.2",
    "title": "Optimize integer slice sort",
    "status": "ACTIVE",
    "parent_id": "A",
    "subtasks_count": 0,
    "decisions_count": 2,
    "evidence_count": 1,
    "last_updated": "2026-10-13T12:20:00Z"
  }
  ```

---

### 2.4 `knodexa_record_observation`
Records an unverified observation or test outcome during agent execution.

- **Parameters:**
  ```json
  {
    "type": "object",
    "properties": {
      "task_id": { "type": "string", "description": "Task ID under which observation occurred" },
      "content": { "type": "string", "description": "Observation details" },
      "files": { "type": "array", "items": { "type": "string" }, "description": "Associated file paths" }
    },
    "required": ["task_id", "content"]
  }
  ```
- **Response:**
  ```json
  { "id": "OBS-104", "status": "OBSERVATION", "recorded": true }
  ```

---

### 2.5 `knodexa_propose_decision`
Proposes an architectural or implementation decision for a task.

- **Parameters:**
  ```json
  {
    "type": "object",
    "properties": {
      "task_id": { "type": "string", "description": "Task ID" },
      "title": { "type": "string", "description": "Concise decision title" },
      "rationale": { "type": "string", "description": "Why this decision is being made" },
      "alternatives_considered": { "type": "array", "items": { "type": "string" } },
      "constraints": { "type": "array", "items": { "type": "string" } }
    },
    "required": ["task_id", "title", "rationale"]
  }
  ```
- **Response:**
  ```json
  { "id": "D-012", "status": "PROPOSED", "file": ".knodexa/decisions/D-012.md" }
  ```

---

### 2.6 `knodexa_propose_lesson`
Proposes a reusable engineering lesson derived from task execution.

- **Parameters:**
  ```json
  {
    "type": "object",
    "properties": {
      "task_id": { "type": "string", "description": "Task ID where lesson was learned" },
      "title": { "type": "string", "description": "Lesson title" },
      "description": { "type": "string", "description": "Detailed explanation and guidance" },
      "applicability": { "type": "string", "description": "When does this lesson apply?" },
      "tags": { "type": "array", "items": { "type": "string" } },
      "related_files": { "type": "array", "items": { "type": "string" } }
    },
    "required": ["task_id", "title", "description", "applicability"]
  }
  ```
- **Response:**
  ```json
  { "id": "L-008", "status": "PROPOSED", "file": ".knodexa/lessons/L-008.md" }
  ```

---

### 2.7 `knodexa_add_evidence`
Attaches test, benchmark, or Git evidence to a decision, lesson, or task.

- **Parameters:**
  ```json
  {
    "type": "object",
    "properties": {
      "node_id": { "type": "string", "description": "Target node ID (decision, lesson, task, failure)" },
      "title": { "type": "string", "description": "Short description of evidence" },
      "evidence_type": { "type": "string", "enum": ["TEST_RUN", "BENCHMARK", "BUILD_TEST", "GIT_DIFF", "MANUAL_SIGN_OFF"] },
      "command": { "type": "string", "description": "Command executed" },
      "exit_code": { "type": "integer", "description": "Process exit code" },
      "output_summary": { "type": "string", "description": "Key stdout/stderr summary" },
      "commit_sha": { "type": "string", "description": "Git commit SHA" }
    },
    "required": ["node_id", "title", "evidence_type"]
  }
  ```
- **Response:**
  ```json
  { "id": "E-019", "node_id": "D-012", "recorded": true }
  ```

---

### 2.8 `knodexa_verify`
Promotes a proposed decision, lesson, or constraint to `VERIFIED` status upon provision of supporting evidence.

- **Parameters:**
  ```json
  {
    "type": "object",
    "properties": {
      "node_id": { "type": "string", "description": "Node to verify" },
      "evidence_ids": { "type": "array", "items": { "type": "string" }, "description": "Supporting evidence IDs" },
      "verification_notes": { "type": "string", "description": "Explanation of verification" }
    },
    "required": ["node_id", "evidence_ids"]
  }
  ```
- **Response:**
  ```json
  { "id": "D-012", "previous_status": "PROPOSED", "current_status": "VERIFIED", "evidence_count": 1 }
  ```

---

### 2.9 `knodexa_find_related`
Discovers cross-task decisions, lessons, failures, and skills related to a query or task context.

- **Parameters:**
  ```json
  {
    "type": "object",
    "properties": {
      "task_id": { "type": "string", "description": "Optional source task to compare against" },
      "query": { "type": "string", "description": "Keywords or search term" },
      "types": { "type": "array", "items": { "type": "string" }, "description": "Filter by node types (DECISION, LESSON, FAILURE, SKILL)" },
      "limit": { "type": "integer", "default": 5 }
    }
  }
  ```
- **Response:**
  ```json
  {
    "matches": [
      {
        "id": "L-017",
        "type": "LESSON",
        "title": "Sort short arrays with insertion sort",
        "status": "VERIFIED",
        "relevance_score": 0.88,
        "shared_files": ["internal/sort/sort.go"]
      }
    ]
  }
  ```

---

## 3. Protocol Transport & Error Handling

- **Transport:** Standard MCP JSON-RPC 2.0 messages over standard I/O streams (`stdin` / `stdout`). Diagnostic logs are sent strictly to `stderr` to prevent JSON-RPC stream pollution.
- **Error Codes:**
  - `-32602`: Invalid params (e.g. malformed ID, path traversal attempt).
  - `-32001`: Node not found.
  - `-32002`: Invalid state transition (e.g. verifying without evidence).
  - `-32003`: Path security violation.
