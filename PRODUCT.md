# Knodexa: Product Specification

> **Work identity and continuity across new requests and context changes.**

The primary agent integration is a portable project protocol: `AGENTS.md`, a
`knodexa-continuity/SKILL.md` bundle, or generic rules, generated from one source
through `agents install`. Hosts must load the selected format. This makes
restoration and agreement checking an explicit repository workflow; it does not
claim universal adoption or automatic interception of all coding tools.

---

## 1. The Core Problem: Losing Work Identity

An agent establishes correct work, handles unrelated requests, then treats a later request of the original kind as a completely new problem. The original work's identity, state, and implementation approach are lost. Context contamination is a consequence; identifying the work comes first. Task labels are incidental.

The primary product loop is:

0. Resolve the incoming request against recorded work before creating a task. Return the existing identity and evidence when uniquely matched; surface ambiguity and unknown intent. Restore completed work without automatically reopening it. Explicit independent work and intentional replacement remain separate actions.

1. Capture explicit implementation agreements, file scopes, references, acceptance criteria, and checks after correct work.
2. On each switch or return, select the target work's agreement by explicit relationships or unambiguous file coverage.
3. Return a context boundary: target rules are authoritative; prior unrelated conventions should be set aside; cross-domain lessons remain advisory.
4. Preserve source constraints and acceptance criteria under budget pressure. Surface ambiguity rather than mixing conflicting agreements.
5. Compare resulting code against references and behavioral checks. Record intentional changes with rationale and evidence.

Current routing is deterministic file-scope routing, not natural-language intent classification. It requires declared touched files. The boundary instructs an agent; it does not delete attention history or enforce semantic obedience. Human exports show recorded agreements and provenance, not hidden model thoughts.

Request identity retrieval now accepts natural-language text without a task ID,
using recorded aliases, titles, goal terms, and optional file hints. It searches
compact indexed metadata and does not call a model. Exact recorded phrases or
complete file coverage can select a unique match; weaker or conflicting matches
remain unresolved. This is conservative lexical retrieval, not general semantic
understanding. The CLI guards strong duplicate-work descriptions and duplicate
IDs; external agents still need to call the entry point before coding.

The implementation and required validation use Go only. Historical model
evaluation scripts have been retired; their evidence remains in audit reports.

The primary evaluation must keep agents in shared conversations, introduce conflicting conventions in unrelated work, and return to the original kind of work. Measure approach violations, lost acceptance criteria, unrelated-rule carryover, tool overhead, and repeated-trial outcomes. Fresh-session retrieval demos are secondary evidence. Existing pilots do not establish superiority over competent native file memory.

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

## Established approach continuity

The unit of continuity is an implementation agreement, independent of task
labels: rules, reference implementations, reusable skills, and concrete checks.
A task inherits its ancestors' agreements or explicitly links another approach.
Unrelated work does not become mandatory guidance through matching names.

On return, restore the approach before coding, inspect referenced artifacts,
compare against its recorded baseline, and check both functionality and declared
structural consistency. Record the rationale and evidence for a deliberate
change. The visible memory graph exposes recorded understanding and provenance,
not hidden reasoning. Generated task modules are inspectable snapshots.

The first implementation supports these explicit contracts and structural checks.
It does not infer semantic task families automatically, generate a trustworthy
agreement from arbitrary code without agent involvement, or guarantee future
models obey the agreement. Those require separate harness integration and
behavioral evaluation against the agreed approach, beyond functional test success.
