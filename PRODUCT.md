# Knodexa: Product Specification

> **Engineering memory for AI coding agents across task boundaries.**

---

## 1. The Core Problem: Agent Amnesia Across Task Switches

AI coding agents are remarkably capable at focused, short-horizon tasks. However, when working on real-world software projects, workflows are rarely linear. Developers and agents constantly switch contexts:

```
Task A (e.g., Refactor Auth) → A.1 → A.2
       ↓ Context Switch
Task B (e.g., Performance Benchmarks) → B.1 → B.2 → B.3
       ↓ Context Switch
Return to Task A (A.3)
```

When an agent returns to Task A:
1. **Context Loss:** The original prompt context window is gone or truncated.
2. **Reconstruction Drift:** The agent re-interprets the task from scratch, often proposing architectural approaches that were previously evaluated and discarded.
3. **Constraint Violation:** Hard architectural constraints (e.g., "Must remain backwards-compatible with API v1", "Zero allocations in hot loop") are forgotten.
4. **Repeated Failures:** Failed approaches (e.g., "Using sync.Mutex here caused deadlock in integration tests") are retried because the failure evidence was not retained.
5. **Missed Cross-Pollination:** Discoveries made during Task B (e.g., "Always run pre-validation before retry loops to prevent duplicate DB writes") are invisible to Task A, even when directly applicable.

Humans maintain a continuous, evolving mental model of a project. Knodexa provides an **external, persistent, structured engineering memory** that replicates this continuity for AI coding agents.

---

## 2. What Knodexa Is NOT

To maintain focus and architectural purity, Knodexa explicitly rejects several common archetypes:

- **NOT a generic chat history store:** We do not store raw LLM conversations, transcript dumps, or verbose conversational back-and-forth.
- **NOT a hidden chain-of-thought store:** Knodexa never attempts to extract or store opaque model internal reasoning. Only explicit, verifiable engineering artifacts are retained.
- **NOT a generic RAG / vector dump:** We do not chunk files indiscriminately or rely on probabilistic fuzzy vector retrieval for core engineering constraints.
- **NOT a Jira/Linear clone:** Knodexa is agent-first. Human project managers have tools; agents need machine-digestible, token-budgeted, evidence-backed memory.
- **NOT an LLM wrapper:** The Knodexa engine is deterministic, local-first infrastructure. It does not require an LLM API key to store, index, or retrieve memory.

---

## 3. Core Philosophy

1. **Agent-First:** Tools, data structures, and outputs are optimized for machine readability, compact token usage, and unambiguous semantics.
2. **Human-Visible & Auditable:** All canonical knowledge is stored in clean Markdown and YAML. Humans can read, edit, git-diff, approve, and reject everything Knodexa stores.
3. **Local-First & Git-Friendly:** Zero cloud dependencies. Knowledge resides inside the project repository (`.knodexa/`). Checkpoints link directly to Git commits and tree state.
4. **Evidence-Driven:** An agent’s assertion is never accepted as fact without verification. Every claim must link to tests, benchmarks, or explicit human sign-off.
5. **Model-Independent:** Works with any agent, harness, or foundation model (Claude, GPT, Gemini, local models) via standard Model Context Protocol (MCP) or CLI.

---

## 4. The Knowledge Lifecycle States

A foundational flaw in naive agent memory is **memory pollution**: bad agent conclusions get persisted and poison future prompts.

Knodexa enforces strict lifecycle states:

```
[OBSERVATION] ───────────────► [PROPOSED] ───────────────► [VERIFIED]
 (Raw event or                 (Candidate decision,          (Backed by passing tests,
  agent hypothesis)             lesson, or constraint)        benchmarks, or human approval)
                                     │                              │
                                     ▼                              ▼
                                [REJECTED]                    [SUPERSEDED]
                             (Disproven by tests            (Replaced by newer
                              or human veto)                 verified architecture)
```

- **`OBSERVATION`**: Ephemeral or unverified notes (e.g., "Ran test suite; observed 3 flakiness failures in worker pool").
- **`PROPOSED`**: Candidate decisions, lessons, or constraints submitted by the agent. Not yet active in baseline task context.
- **`VERIFIED`**: Validated knowledge backed by concrete evidence (test runs, benchmark outputs, human approval). Included in context rehydration.
- **`REJECTED`**: Explicitly refuted approaches. Crucial for negative knowledge—prevents agents from repeating known mistakes.
- **`SUPERSEDED`**: Historical decisions or lessons that were once valid but have been intentionally replaced by newer verified items.

> **Principle:** Agent statements never automatically become trusted project knowledge.

---

## 5. Core Workflows

### 5.1 Context Rehydration (The Primary Feature)
When an agent is assigned or resumes task `A.2`, Knodexa produces a minimal, dense, high-signal context package:
- Current task goal & status
- Active parent & project constraints
- Accepted architectural decisions
- Known previous failures (negative constraints)
- Relevant verified lessons (including cross-task discoveries)
- Baseline Git commit SHA and status
- Required verification criteria

### 5.2 Progressive Disclosure
Instead of filling the context window with the entire project graph:
1. `knodexa context <task_id>` returns the immediate, highly-scored context (within a token budget).
2. Deep links (IDs) allow the agent to query specific decisions (`D-001`), evidence (`E-003`), or full lessons (`L-017`) on demand via `knodexa_find_related` or CLI.

### 5.3 Cross-Task Experience Transfer
- When Task `B.2` discovers an insight (e.g., "Buffer pool reuse requires explicit reset to avoid data races"), it records a proposed lesson `L-018`.
- When verified with benchmark evidence, `L-018` is indexed with tags and affected file patterns.
- When Task `A.2` later works on buffer pools or related components, Knodexa surfaces `L-018` as a *candidate suggestion* with its confidence and evidence.
- The agent or human chooses whether to adopt the lesson.

---

## 6. Success Metrics & Verification

Knodexa's success is measured not by feature count, but by measurable improvement in agent engineering continuity:
- **Regression Reduction:** Fewer re-introductions of previously fixed bugs.
- **Constraint Compliance:** 100% adherence to active project and parent task constraints.
- **Mistake De-duplication:** Zero retries of explicitly rejected approaches.
- **Context Efficiency:** Under 1,500 tokens for full task rehydration.
- **Deterministic Replay:** Rebuilding SQLite index from files reproduces identical graph state.
