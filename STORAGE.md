# Knodexa: Storage Specification

---

## 1. Storage Architecture Overview

Knodexa adheres to a strict dual-layer storage model:

```
+-------------------------------------------------------------------------+
|                        CANONICAL KNOWLEDGE BASE                         |
|  - Human-readable, Git-versioned Markdown & YAML files                  |
|  - Append-only JSONL ledgers for events and cross-node links            |
|  - Sole source of truth                                                 |
+-------------------------------------------------------------------------+
                                    │
                                    │ (Automatic sync / Rebuild)
                                    ▼
+-------------------------------------------------------------------------+
|                        DERIVED QUERY INDEX                              |
|  - SQLite database (.knodexa/cache.db)                                  |
|  - Sub-millisecond graph queries, full-text search, and caching        |
|  - Ephemeral: can be deleted and regenerated at any time                |
+-------------------------------------------------------------------------+
```

---

## 2. Directory Layout

The `.knodexa` directory is stored in the root of the repository:

```
<project-root>/
├── .knodexa/
│   ├── project.yaml         # Project metadata, root goals, and global constraints
│   ├── tasks/               # Hierarchical task tree
│   │   ├── A/
│   │   │   ├── task.yaml    # Task A metadata
│   │   │   ├── A.1.yaml     # Subtask A.1
│   │   │   └── A.2.yaml     # Subtask A.2
│   │   └── B/
│   │       ├── task.yaml    # Task B metadata
│   │       ├── B.1.yaml     # Subtask B.1
│   │       └── B.2.yaml     # Subtask B.2
│   ├── decisions/           # Architectural Decision Records (Markdown + Frontmatter)
│   │   ├── D-001.md
│   │   └── D-002.md
│   ├── lessons/             # Reusable engineering lessons (Markdown + Frontmatter)
│   │   ├── L-001.md
│   │   └── L-002.md
│   ├── evidence/            # Concrete test/benchmark/git verification artifacts
│   │   ├── E-001.yaml
│   │   └── E-002.yaml
│   ├── events.jsonl         # Append-only ledger of state transitions & actions
│   ├── links.jsonl          # Explicit cross-node graph edges
│   └── cache.db             # Derived SQLite index (ignored by Git or committed optionally)
├── AGENTS.md                # Standard agent rules (canonical file in root)
└── Skills/                  # Standard Agent Skills directories
```

---

## 3. Canonical File Schemas

### 3.1 Project File (`.knodexa/project.yaml`)
```yaml
id: "proj_root"
name: "Knodexa"
description: "Engineering memory system for AI coding agents"
status: "ACTIVE"
created_at: "2026-10-13T12:00:00Z"
updated_at: "2026-10-13T12:00:00Z"
global_constraints:
  - "Preserve public API compatibility"
  - "Zero external cloud service dependencies"
  - "Core engine must be deterministic with no LLM calls"
rules_files:
  - "AGENTS.md"
```

### 3.2 Task File (`.knodexa/tasks/A/task.yaml` or `A.1.yaml`)
```yaml
id: "A.1"
parent_id: "A"
title: "Benchmark Baseline Sorting Implementation"
goal: "Measure sorting latency and allocations across 10k, 100k, and 1M integer slices."
status: "COMPLETED"             # PROPOSED | ACTIVE | BLOCKED | COMPLETED | ABANDONED
created_at: "2026-10-13T12:05:00Z"
updated_at: "2026-10-13T12:20:00Z"
constraints:
  - "Correctness suite must pass before running benchmarks"
touches_files:
  - "internal/sort/sort.go"
  - "internal/sort/sort_test.go"
baseline_commit: "a1b2c3d4"
accepted_commit: "e5f6a7b8"
```

### 3.3 Decision File (`.knodexa/decisions/D-001.md`)
```markdown
---
id: "D-001"
title: "Use Pure-Go SQLite Implementation"
status: "VERIFIED"              # PROPOSED | VERIFIED | REJECTED | SUPERSEDED
task_id: "A.1"
created_at: "2026-10-13T12:10:00Z"
updated_at: "2026-10-13T12:25:00Z"
confidence: 0.95
supersedes: ""
superseded_by: ""
evidence_ids:
  - "E-001"
---

## Context & Problem
We require an embedded database for graph indexing that compiles statically across macOS, Linux, and Windows without requiring CGO toolchains.

## Decision
Adopt `modernc.org/sqlite` instead of `mattn/go-sqlite3`.

## Consequences
- **Positive:** Single binary cross-compilation without gcc/clang.
- **Positive:** Eliminates CGO runtime overhead for cross-platform releases.
- **Negative:** Slightly slower insert throughput compared to native C SQLite, which is negligible for our index size (< 50,000 nodes).
```

### 3.4 Lesson File (`.knodexa/lessons/L-001.md`)
```markdown
---
id: "L-001"
title: "Always Reset Buffer Before Returning to sync.Pool"
status: "VERIFIED"              # PROPOSED | VERIFIED | REJECTED | SUPERSEDED
source_task_id: "B.2"
created_at: "2026-10-13T12:15:00Z"
updated_at: "2026-10-13T12:30:00Z"
confidence: 0.90
applicability: "Go buffer pooling and memory reuse patterns"
tags:
  - "memory"
  - "concurrency"
  - "pooling"
related_files:
  - "internal/buffer/pool.go"
evidence_ids:
  - "E-002"
---

## Observation & Lesson
When recycling byte buffers into `sync.Pool`, failing to call `b.Reset()` causes dirty data leakages across concurrent goroutine invocations.

## Recommended Pattern
Always wrap pool retrievals with deferral or explicit reset before `Put()`:
```go
buf := pool.Get().(*bytes.Buffer)
buf.Reset()
defer pool.Put(buf)
```
```

### 3.5 Evidence File (`.knodexa/evidence/E-001.yaml`)
```yaml
id: "E-001"
title: "CGO-Free Cross-Compilation Test"
node_id: "D-001"
evidence_type: "BUILD_TEST"     # TEST_RUN | BENCHMARK | BUILD_TEST | GIT_DIFF | MANUAL_SIGN_OFF
created_at: "2026-10-13T12:20:00Z"
command: "CGO_ENABLED=0 go build -o bin/test ./cmd/knodexa"
exit_code: 0
output_summary: "Build succeeded in 1.4s with zero external shared library dependencies."
commit_sha: "e5f6a7b8"
verified_by: "agent-01"
```

### 3.6 Links Ledger (`.knodexa/links.jsonl`)
Lines represent typed edges between nodes:
```json
{"from_id":"A.1","to_id":"D-001","type":"PRODUCED","created_at":"2026-10-13T12:10:00Z"}
{"from_id":"D-001","to_id":"E-001","type":"VERIFIED_BY","created_at":"2026-10-13T12:20:00Z"}
{"from_id":"L-001","to_id":"A.2","type":"RELEVANT_TO","created_at":"2026-10-13T12:35:00Z"}
```

### 3.7 Events Ledger (`.knodexa/events.jsonl`)
Append-only log of every mutation:
```json
{"event_id":"evt-001","timestamp":"2026-10-13T12:10:00Z","actor":"agent","action":"PROPOSE_DECISION","target_id":"D-001","payload":{"title":"Use Pure-Go SQLite"}}
{"event_id":"evt-002","timestamp":"2026-10-13T12:20:00Z","actor":"agent","action":"VERIFY_NODE","target_id":"D-001","payload":{"evidence_id":"E-001"}}
```

---

## 4. SQLite Schema (`.knodexa/cache.db`)

The SQLite database is initialized with strict relational schemas and indexes:

```sql
CREATE TABLE IF NOT EXISTS nodes (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    parent_id TEXT,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    confidence REAL DEFAULT 1.0,
    verification_status TEXT NOT NULL,
    source TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    file_path TEXT NOT NULL,
    content_hash TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS edges (
    from_id TEXT NOT NULL,
    to_id TEXT NOT NULL,
    type TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (from_id, to_id, type),
    FOREIGN KEY(from_id) REFERENCES nodes(id) ON DELETE CASCADE,
    FOREIGN KEY(to_id) REFERENCES nodes(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS node_files (
    node_id TEXT NOT NULL,
    file_path TEXT NOT NULL,
    PRIMARY KEY (node_id, file_path),
    FOREIGN KEY(node_id) REFERENCES nodes(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS node_tags (
    node_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    PRIMARY KEY (node_id, tag),
    FOREIGN KEY(node_id) REFERENCES nodes(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_nodes_type ON nodes(type);
CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);
CREATE INDEX IF NOT EXISTS idx_nodes_parent ON nodes(parent_id);
CREATE INDEX IF NOT EXISTS idx_edges_from ON edges(from_id);
CREATE INDEX IF NOT EXISTS idx_edges_to ON edges(to_id);
CREATE INDEX IF NOT EXISTS idx_edges_type ON edges(type);
CREATE INDEX IF NOT EXISTS idx_node_files_path ON node_files(file_path);
```

---

## 5. Rebuild and Synchronization Algorithm

1. **Incremental Sync:**
   - On startup or command invocation, Knodexa compares file modification timestamps (`mtime`) and SHA-256 hashes against `nodes.content_hash`.
   - Only modified or added files are parsed and updated in SQLite.
2. **Full Rebuild (`knodexa doctor --rebuild`):**
   - Closes current SQLite connection.
   - Deletes `.knodexa/cache.db`.
   - Re-applies the DDL schema.
   - Scans `.knodexa/project.yaml`, `.knodexa/tasks/**`, `.knodexa/decisions/**`, `.knodexa/lessons/**`, `.knodexa/evidence/**`.
   - Inserts all nodes.
   - Replays `.knodexa/links.jsonl` to populate all edges.
   - Guarantees 100% deterministic reconstruction of the graph state.

---

## 6. Security & Safety Principles

1. **Atomic Writes:**
   All canonical files are written to temporary files first (`.knodexa/.tmp_<uuid>`) and atomically renamed via `os.Rename`. This guarantees no partial or corrupted files if a process is killed.
2. **Path Sanitization:**
   Task IDs, decision IDs, and file paths are strictly validated:
   - Must match `^[A-Za-z0-9._-]+$`.
   - Path separators, `..`, and absolute paths are rejected.
   - Paths are strictly contained within `.knodexa/`.
3. **Protected Files:**
   Project-level rules (`AGENTS.md`) and project configuration cannot be modified by unverified agent tool calls.
