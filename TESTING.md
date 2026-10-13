# Knodexa: Testing & Verification Strategy

---

## 1. Test Architecture

Knodexa follows a multi-tiered testing hierarchy designed for high confidence, fast feedback loops, and deterministic reproducibility:

```
┌────────────────────────────────────────────────────────┐
│   Tier 4: Task Continuity Experiment (Section 14)      │
│   Multi-task context switch evaluation (Control/Exp)   │
├────────────────────────────────────────────────────────┤
│   Tier 3: End-to-End MCP & CLI Integration Tests       │
│   Full process execution, stdio JSON-RPC, Git repos    │
├────────────────────────────────────────────────────────┤
│   Tier 2: Subsystem Integration Tests                  │
│   SQLite indexer, Rebuild fidelity, Context packager   │
├────────────────────────────────────────────────────────┤
│   Tier 1: Unit Tests                                   │
│   Parsers, Sanitizers, State Transitions, Scorers      │
└────────────────────────────────────────────────────────┘
```

---

## 2. Testing Tiers

### 2.1 Tier 1: Unit Tests
- **Path Sanitization & Security (`internal/storage`):**
  - Verify rejection of `..`, `../..`, absolute paths (`/etc/passwd`), null bytes, Windows drive letters (`C:\`), and special characters.
- **YAML & Markdown Frontmatter Parsers (`internal/storage`):**
  - Verify round-trip parsing of tasks, decisions, lessons, and evidence.
  - Verify preservation of body Markdown while extracting frontmatter.
- **Lifecycle State Machine (`internal/verify`):**
  - Valid: `OBSERVATION` → `PROPOSED` → `VERIFIED`.
  - Valid: `PROPOSED` → `REJECTED`.
  - Valid: `VERIFIED` → `SUPERSEDED`.
  - Invalid: `OBSERVATION` → `VERIFIED` directly (must have evidence).
  - Invalid: `REJECTED` → `VERIFIED` without new proposal.
- **Context Scoring & Budgeting (`internal/context`):**
  - Test ranking algorithm: $\text{Relevance} \times \text{Importance} \times \text{Recency} \times \text{Confidence}$.
  - Test strict adherence to maximum token budgets without cutting off critical constraints.

### 2.2 Tier 2: Subsystem Integration Tests
- **SQLite Indexer & Rebuild Fidelity (`internal/index`):**
  - Create graph with 20 tasks, 15 decisions, 10 lessons, 10 evidence items, and 30 edges.
  - Query graph via SQLite.
  - Delete `.knodexa/cache.db`.
  - Trigger `Rebuild()`.
  - Assert that all nodes, edges, tags, and file links match the pre-deletion state with 100% fidelity.
- **Git Inspector (`internal/git`):**
  - Initialize a temporary Git repository inside `t.TempDir()`.
  - Commit files, check branch resolution, dirty status detection, and commit SHA retrieval.

### 2.3 Tier 3: CLI & MCP Integration Tests
- **CLI Workflows (`cmd/knodexa`):**
  - Execute `knodexa init` in a fresh directory.
  - Execute `knodexa task create --id=A.1 --title="Task A.1"`.
  - Execute `knodexa context A.1`.
  - Execute `knodexa tree`.
  - Execute `knodexa doctor`.
- **MCP Server Stdio Harness (`internal/mcp`):**
  - Spin up MCP server over pipes (`stdin`/`stdout`).
  - Send JSON-RPC `initialize` request; verify capabilities.
  - Send `tools/call` for `knodexa_get_context`, `knodexa_propose_decision`, `knodexa_verify`.
  - Verify error handling on malformed JSON or illegal params.

---

## 3. Tier 4: The Task Continuity Experiment (Section 14)

The primary goal of Knodexa is to prevent degradation of engineering consistency when an agent switches contexts.

### 3.1 Scenario Design
1. **Phase 1: Task A (A.1, A.2)**
   - Goal: Build a high-throughput priority queue.
   - Constraint: Must guarantee O(log N) insertion and zero background goroutine leakage.
   - Decision D-01: Use a 4-ary min-heap instead of binary heap.
   - Failure F-01: Linked-list heap failed allocation benchmarks.
2. **Phase 2: Context Switch to Task B (B.1, B.2, B.3)**
   - Goal: Implement network protocol parser.
   - Lesson Learned L-17: "Zero-copy slice slicing must retain backing buffer pointer until frame decoding completes."
3. **Phase 3: Return to Task A (Task A.3)**
   - Goal: Add bulk insertion and buffer recycling to the priority queue.

### 3.2 Control vs. Experiment
- **Control Group (Standard Agent):**
  - Provided only the latest prompt and code repository.
  - Evaluated on whether it:
    - Retries the failed linked-list heap or breaks the 4-ary structure.
    - Violates the goroutine leakage constraint.
    - Re-invents buffer pooling without the lesson learned in Task B.
- **Experiment Group (Knodexa Agent):**
  - Calls `knodexa_get_context A.3`.
  - Receives constraints, Decision D-01, Failure F-01, and suggested Lesson L-17.
  - Evaluated on consistency, avoidance of known failures, and constraint adherence.

### 3.3 Evaluation Metrics
| Metric | Target | Formula |
| :--- | :---: | :--- |
| **Constraint Violation Rate** | 0% | $\frac{\text{Violated Constraints}}{\text{Total Active Constraints}}$ |
| **Mistake Repetition Rate** | 0% | $\frac{\text{Repeated Known Failures}}{\text{Total Known Failures}}$ |
| **Cross-Task Lesson Adoption** | > 80% | $\frac{\text{Applicable Lessons Applied}}{\text{Total Relevant Lessons}}$ |
| **Rehydration Token Budget** | < 1,500 tokens | Direct token count of context package |
| **Test Suite Pass Rate** | 100% | Passing unit and integration tests |
