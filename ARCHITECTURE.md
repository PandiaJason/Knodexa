# Knodexa: System Architecture

---

## 1. Architectural Philosophy

Knodexa is built on four architectural pillars:

1. **Separation of Canon and Index:**
   - **Canonical Truth:** Clean, human-readable YAML and Markdown files committed to Git.
   - **Derived Query Index:** SQLite database (`.knodexa/cache.db`) used solely for sub-millisecond graph queries, full-text search, and relationship traversal. SQLite is 100% ephemeral and rebuildable from the canonical files at any time.
2. **Deterministic Core:**
   - Zero LLM calls in the core engine. Knodexa is deterministic infrastructure. Relevance, graph scoring, and context packing use transparent, verifiable algorithms.
3. **Strict State Transitions & Evidence-Gating:**
   - Unverified observations or proposed decisions cannot enter the verified context pool without explicit evidence (tests, benchmarks, or human verification).
4. **Token-Budgeted Progressive Disclosure:**
   - Rehydration outputs are compact, structured, and bounded by token budgets. Deeper context is retrieved via explicit ID drill-downs.

---

## 2. High-Level Architecture Diagram

```
                     ┌──────────────────────────────────────────────────┐
                     │                  Clients                         │
                     │  - CLI (`knodexa ...`)                           │
                     │  - Coding Agents via MCP (stdio JSON-RPC)        │
                     └─────────────────────────┬────────────────────────┘
                                               │
                        ┌──────────────────────▼──────────────────────┐
                        │              Application Service            │
                        │  - Context Rehydration                      │
                        │  - Experience Transfer & Search             │
                        │  - Knowledge Lifecycle & Verification       │
                        │  - Git Checkpoint Association               │
                        └──────┬───────────────┬───────────────┬──────┘
                               │               │               │
            ┌──────────────────▼──┐   ┌────────▼─────────┐   ┌─▼──────────────────┐
            │   Canonical Storage │   │   SQLite Index   │   │   Git Inspector    │
            │   (File System)     │   │   (Graph/Cache)  │   │   (Subprocess/CLI) │
            ├─────────────────────┤   ├──────────────────┤   ├────────────────────┤
            │ .knodexa/           │   │ .knodexa/        │   │ - Commit SHA       │
            │  ├── project.yaml   │   │   cache.db       │   │ - Branch name      │
            │  ├── tasks/         │   │                  │   │ - Changed files    │
            │  ├── decisions/     │   │  [Nodes Table]   │   │ - Dirty state      │
            │  ├── lessons/       │   │  [Edges Table]   │   └────────────────────┘
            │  ├── evidence/      │   │  [Events Table]  │
            │  ├── events.jsonl   │   │                  │
            │  └── links.jsonl    │   │ (Rebuildable on  │
            └─────────────────────┘   │  demand/startup) │
                                      └──────────────────┘
```

---

## 3. Core Subsystems

### 3.1 Domain Model (`internal/model`)
Defines the typed graph entities:
- **Node Types:**
  - `PROJECT`: Root project metadata, global rules, and invariants.
  - `TASK`: Unit of engineering work (hierarchical: Task → Subtask → Sub-subtask).
  - `DECISION`: Architectural Decision Record (ADR) detailing context, alternatives, rationale, and chosen path.
  - `LESSON`: Reusable cross-task discovery with applicability criteria and evidence.
  - `OBSERVATION`: Raw, unverified agent note or test observation.
  - `FAILURE`: Recorded failure mode or anti-pattern with causal explanation.
  - `EVIDENCE`: Concrete verification proof (test output, benchmark report, Git diff).
  - `RULE`: Formal project requirement or constraint (e.g. from `AGENTS.md`).
  - `SKILL`: Agent skill reference (e.g. from `SKILL.md`).
  - `FILE`: Repository file tracked in connection with tasks or lessons.
  - `COMMIT`: Git commit reference anchoring knowledge to exact tree states.
- **Edge Types:**
  - `PARENT_OF`, `DEPENDS_ON`, `PRODUCED`, `LEARNED_FROM`, `RELEVANT_TO`, `SUPERSEDES`, `VERIFIED_BY`, `FAILED_BECAUSE`, `IMPLEMENTED_BY`, `TOUCHES_FILE`, `USES_SKILL`, `CONSTRAINED_BY`.

### 3.2 Canonical Storage (`internal/storage`)
- Manages reading and writing `.knodexa/` disk files.
- Thread-safe file operations with atomic write-and-rename semantics (`tempfile` + `os.Rename`) to prevent corrupt states.
- Strict path sanitization to prevent path-traversal attacks.
- Appends mutations to `.knodexa/events.jsonl` (audit ledger) and `.knodexa/links.jsonl` (graph edges).

### 3.3 SQLite Graph Indexer (`internal/index`)
- Indexes all canonical files into an embedded SQLite database (`cache.db`).
- Tracks file modification times (`mtime`) and content hashes to provide instantaneous incremental indexing.
- Provides a full `Rebuild()` method that drops tables and scans the filesystem from scratch.
- Runs graph queries: upstream constraint collection, downstream failure lookups, cross-task lesson similarity.

### 3.4 Context Engine (`internal/context`)
- Implements the **Context Rehydration Algorithm**:
  $$\text{Score} = \text{Relevance} \times \text{Importance} \times \text{Recency} \times \text{Confidence}$$
- Collects:
  1. Task Goal & Current State.
  2. Active Constraints (accumulated up the task hierarchy to root).
  3. Previous Architectural Decisions (status: `VERIFIED`).
  4. Previous Failures (negative constraints to avoid repeating mistakes).
  5. Cross-Task Lessons (scored by matching touched files, tags, and dependencies).
  6. Git Baseline (commit SHA, dirty status).
  7. Required Verification Criteria.
- Enforces strict token budgets (e.g., 1,500 tokens / ~6,000 characters) by truncating lower-priority items while retaining full high-priority constraints.

### 3.5 Git Engine (`internal/git`)
- Interacts with the local Git repository via deterministic subprocess execution.
- Captures: current commit SHA, branch name, dirty file status, commit message, and diff stats.
- Never mutates Git history.

### 3.6 MCP Server (`internal/mcp`)
- Implements standard Model Context Protocol (JSON-RPC 2.0 over `stdin`/`stdout`).
- Exposes typed tools for agents to query context, record observations, propose decisions, propose lessons, attach evidence, and verify knowledge.
- Formats tool outputs for maximum density and minimum token waste.

---

## 4. Repository Directory Structure

```
Knodexa/
├── cmd/
│   └── knodexa/
│       └── main.go              # CLI & MCP server entry point
├── internal/
│   ├── model/                  # Domain types: Node, Edge, Status, Event
│   │   ├── node.go
│   │   ├── edge.go
│   │   ├── status.go
│   │   └── event.go
│   ├── storage/                # Canonical file I/O & YAML/Markdown parsers
│   │   ├── fs.go
│   │   ├── yaml.go
│   │   ├── sanitize.go
│   │   └── storage_test.go
│   ├── index/                  # SQLite schema, indexing & graph queries
│   │   ├── db.go
│   │   ├── schema.go
│   │   ├── indexer.go
│   │   ├── queries.go
│   │   └── index_test.go
│   ├── context/                # Context rehydration & token budgeting
│   │   ├── builder.go
│   │   ├── budget.go
│   │   └── context_test.go
│   ├── git/                    # Git integration & provenance inspection
│   │   ├── git.go
│   │   └── git_test.go
│   ├── verify/                 # Verification engine & lifecycle transitions
│   │   ├── verify.go
│   │   └── verify_test.go
│   ├── mcp/                    # MCP server implementation (stdio JSON-RPC)
│   │   ├── server.go
│   │   ├── protocol.go
│   │   ├── tools.go
│   │   └── mcp_test.go
│   └── cli/                    # CLI commands & formatting
│       ├── root.go
│       ├── init.go
│       ├── task.go
│       ├── context.go
│       ├── decision.go
│       ├── lesson.go
│       ├── tree.go
│       ├── verify.go
│       └── doctor.go
├── docs/                       # Technical specifications & ADRs
│   ├── PRODUCT.md
│   ├── ARCHITECTURE.md
│   ├── STORAGE.md
│   ├── MCP.md
│   ├── TESTING.md
│   └── TDR.md
├── go.mod
├── go.sum
└── README.md
```

---

## 5. Critical Design Review (Addressing Section 19 Questions)

### Q1: Are we building something existing tools already solve?
**No.**
- Issue trackers (Jira, Linear, GitHub Issues) manage human scheduling, not agent engineering memory.
- Generic RAG chunks random text files into high-dimensional vector spaces, losing explicit hierarchy, causal links (`FAILED_BECAUSE`), and verification state.
- Chat history stores keep unstructured conversational transcripts with huge token bloat and hallucinated conclusions.
- Knodexa is specifically engineered to provide structured, verified, progressive context rehydration across task boundaries.

### Q2: Is every component necessary?
**Yes.**
- The canonical files provide human inspection and Git portability.
- The SQLite database provides fast relational and graph queries.
- The Git inspector anchors knowledge to physical code checkpoints.
- The MCP server enables native agent integration.
- Every component directly serves the single goal of engineering continuity.

### Q3: Is the source of truth clear?
**Yes.**
- The filesystem (`.knodexa/*.yaml`, `*.md`, `*.jsonl`) is the sole canonical source of truth.
- SQLite is an index. It can be deleted and fully regenerated in milliseconds.

### Q4: Can SQLite be rebuilt?
**Yes.**
- `knodexa doctor --rebuild` wipes `cache.db`, re-reads all canonical files and JSONL links, and rebuilds the SQLite tables with 100% fidelity.

### Q5: Can humans inspect everything?
**Yes.**
- All decisions and lessons are standard Markdown with YAML frontmatter.
- Tasks are clean YAML files.
- All files can be reviewed in standard IDEs and git diffs.

### Q6: Can agents retrieve context cheaply?
**Yes.**
- `knodexa context <task_id>` compiles constraints, verified decisions, previous failures, and relevant lessons into a compact, token-budgeted payload (typically ~500–1,500 tokens).

### Q7: Can bad agent knowledge contaminate future work?
**No.**
- Agent submissions enter as `OBSERVATION` or `PROPOSED`.
- Context rehydration includes only `VERIFIED` decisions and explicitly marked constraints.
- `REJECTED` knowledge acts as negative constraints (failures to avoid).

### Q8: Can we objectively demonstrate improved task continuity?
**Yes.**
- Section 14 defines the Control vs Experiment benchmark: measuring regression rates, constraint violations, and token consumption across a multi-task switch scenario.

---

## 6. MVP Milestone Plan

- **Milestone 1: Project Setup, Domain Models & Storage Layer**
  - Go module initialization.
  - Core domain models (`Node`, `Edge`, `Status`, `Event`).
  - Storage engine for `.knodexa` directory structure, YAML/Markdown serialization, and atomic writes.
- **Milestone 2: SQLite Index & Graph Query Engine**
  - Embedded SQLite setup (pure-Go `modernc.org/sqlite`).
  - Schema creation (`nodes`, `edges`, `events`, `file_associations`).
  - Full indexer with automatic sync and rebuild capability.
  - Graph traversal queries (hierarchical constraint resolution, cross-task lesson discovery).
- **Milestone 3: Git Integration & Verification Lifecycle**
  - Git repository inspection (commit SHA, branch, status, diff stats).
  - Verification engine enforcing state transitions with evidence attachment.
- **Milestone 4: Context Rehydration & Token Budgeting**
  - Retrieval engine computing the rehydration package.
  - Token budget limiter and compact text/JSON formatter.
- **Milestone 5: CLI Implementation**
  - Human CLI: `init`, `tree`, `task`, `context`, `decisions`, `lessons`, `verify`, `reject`, `doctor`.
- **Milestone 6: MCP Server & Integration Testing**
  - Stdio JSON-RPC 2.0 MCP server exposing the 9 core tools.
  - End-to-end integration test suite and validation experiment harness.
