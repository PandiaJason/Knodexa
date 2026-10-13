# Knodexa

> **Local-first engineering memory system for AI coding agents across task boundaries.**

Knodexa solves **agent task amnesia**: when AI coding agents switch between tasks and return later, they often suffer from context degradation, violate previously agreed architectural constraints, repeat failed approaches, or fail to apply lessons learned in other branches of work.

Knodexa provides an external, persistent, structured engineering memory that preserves continuity without storing opaque chain-of-thought, polluting contexts, or requiring cloud dependencies.

---

## Key Principles

- **Agent-First, Human-Visible:** Optimized for machine consumption while storing all canonical knowledge in transparent Markdown and YAML files.
- **Evidence-Driven:** An agent claim never becomes project knowledge automatically. Lifecycle states (`OBSERVATION` → `PROPOSED` → `VERIFIED` / `REJECTED` / `SUPERSEDED`) require explicit verification evidence.
- **Context Rehydration:** Returns the smallest high-signal engineering context within a configurable token budget.
- **Cross-Task Experience Transfer:** Surfaces verified lessons learned in one task when an agent works on a related task.
- **Local-First & Git-Friendly:** Pure-Go, single-binary architecture with embedded SQLite index that is 100% rebuildable from disk files. Anchors knowledge directly to Git commits.

---

## Architectural & Design Documents

1. [PRODUCT.md](file:///Users/admin/Jas%20Apps/Knodexa/PRODUCT.md) — Problem statement, product philosophy, knowledge lifecycle, and workflows.
2. [ARCHITECTURE.md](file:///Users/admin/Jas%20Apps/Knodexa/ARCHITECTURE.md) — System architecture, subsystems, data flow, and critical design review.
3. [STORAGE.md](file:///Users/admin/Jas%20Apps/Knodexa/STORAGE.md) — Canonical file schemas, SQLite schema, index rebuild algorithm, and security.
4. [MCP.md](file:///Users/admin/Jas%20Apps/Knodexa/MCP.md) — Model Context Protocol (MCP) tool specifications and response schemas.
5. [TESTING.md](file:///Users/admin/Jas%20Apps/Knodexa/TESTING.md) — Unit testing, integration testing, and the multi-task continuity experiment protocol.
6. [TDR.md](file:///Users/admin/Jas%20Apps/Knodexa/TDR.md) — Technology Decision Record evaluating Go, Rust, C++, and Python.

---

## First Implementation Milestone (MVP)

1. `knodexa init` — Initialize `.knodexa/` repository structure and `project.yaml`.
2. Task hierarchy — Create tasks and subtasks (`knodexa task create`).
3. Knowledge storage — Decisions (`.knodexa/decisions/`), lessons (`.knodexa/lessons/`), and evidence (`.knodexa/evidence/`).
4. SQLite index & rebuild — Rebuildable query index in `.knodexa/cache.db`.
5. Context retrieval — `knodexa context <task_id>` with progressive disclosure and token budgeting.
6. Git checkpointing — Anchor tasks and evidence to Git commit SHAs and working tree state.
7. Visual tree — `knodexa tree` visual hierarchy representation.
8. MCP Server — Stdio JSON-RPC server with the 9 core tools for agent harnesses.
9. Verification & testing — Comprehensive unit, integration, and continuity test suite.
