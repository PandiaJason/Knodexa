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

## Compact continuity and expansion

`knodexa_get_context` and `knodexa_resume_task` accept `detail_level` (`compact`
by default, or `full`) and a positive integer `token_budget`. Compact packages
retain inherited task/project/decision constraints, acceptance criteria, failure
causes, and actionable rules. Authors can supply `summary` on decision/lesson
proposals; the deterministic fallback uses a complete first paragraph.

`knodexa_get_detail` accepts `knowledge_id` (a D- or L- ID) and returns the full
canonical document, including rationale or guidance and evidence references.
This is an additional tool alongside the original nine.

Resume activates the task and returns context with stored/current Git baseline
comparison. It preserves the previous checkpoint unless `accept_baseline: true`
is explicitly supplied. A missing baseline is initialized from a valid HEAD.
Repeated resume of an unchanged active task does not rewrite its record.

## Established approach tools

- `knodexa_set_approach`: task_id, summary, rules, optional reference_files,
  skill_files, and checks (path, must_contain, must_not_contain). Records an
  explicit agreement and captures reference hashes; it is not a quality claim.
- `knodexa_reuse_approach`: task_id and source_tasks. Links established approaches
  independently of hierarchy or task names, rejecting missing sources/cycles.
- `knodexa_check_approach`: task_id. Checks reference drift, skill availability,
  and declared literal checks. `semantic_review_required` remains explicit.

Reusable checks may use `@touched` to bind to the target task's touched files.
Resume includes established approach provenance and structural status; mandatory
approach rules are not pruned to fit the context budget. There are now 13 tools.

### Scoped task-switch context

Call `knodexa_resolve_request` first when the request supplies no task ID. Required:
`request`. Optional: `files`, `select_task`, `intent` (`auto`, `continue`, `new`,
`replace`), `previous_task_id`, `token_budget`, and `detail_level`. It returns
matching evidence and restores selected existing work in one read-only operation.
Weak, overlapping, and unmatched results contain no invented task context.
Capture phrases with `knodexa_add_work_aliases` (`task_id`, `aliases`). Neither
operation reopens completed work. The server now exposes 15 tools.

`knodexa_set_approach` accepts `scopes`: file globs or directory prefixes ending
in `/`. `knodexa_resume_task` accepts `previous_task_id` and returns
`approach_selection` plus `context_boundary`. Selection uses explicit
inheritance/reuse, then a unique scope covering every touched file. Ambiguous
matches are exposed without merging rules. Selected source constraints and
acceptance criteria survive compaction. The context boundary is an instruction
to set aside unrelated conventions, not a mechanism that erases model history.

# Source continuity tools

`knodexa_supersede_knowledge` takes `node_id`, `replacement_id` and a nonblank
`reason`. Both nodes must be decisions or both lessons; the replacement must
already be VERIFIED and differ from the original. Retirement preserves historic
content and records replacement, reason and invalidation time in canonical
frontmatter. Superseded items are excluded from active context. The server now
exposes 19 tools.

Context summaries include source/evidence provenance and lesson `match_reason`.
Recall prioritizes selected task/agreement sources, explicit relevance links,
file overlap, tags, then confidence. Budget pruning reports `deferred_count` and
up to eight `deferred_memory` pointers for retrieved but omitted detail; constraints,
agreements, acceptance criteria and failures remain retained.

`knodexa_prepare_work` combines request resolution, checkpoint comparison,
agreements, drift checks and bounded reference excerpts. Inspect `needs_review`
before editing. `knodexa_capture_identity` records stable work IDs and Go source
anchors after successful verification. `knodexa_set_approach` supports
`change_reason` and checks with `ast_kind` / `symbol`; supported kinds are
`require-call`, `forbid-call`, `forbid-import` and `require-function`.

Agreement replacement requires a nonblank `change_reason` and retains previous versions. AST evidence is syntactic;
free-form design rules still require review. These tools do not intercept client
commands automatically.

`knodexa_record_work` accepts `task_id`, an `agreement` object containing aliases,
files and approach, and a nonempty `verify` array of explicit shell commands.
It executes verification and proposed/inherited structural checks before saving
one canonical task update. A replacement requires `approach.change_reason`.
It preserves completed state and baseline and retains the prior agreement version.

`knodexa_resume_task` now preserves completed task records by default.
Set `reopen: true` only to intentionally reactivate completed work; baseline
replacement remains separately explicit through `accept_baseline`.

For reviewed refactors, `approach.reference_reconciliations` can bind an inherited
reference to its current location and hash for the child only. Each entry requires
`source_task`, `source_version`, `original_path`, `original_sha256`, `current_path`,
`current_sha256`, and `reason`. Completed parent records stay unchanged; source
version/hash mismatches and subsequent file drift still fail. See
[validation and replay](audit/reference-reconciliation/REPORT.md).
