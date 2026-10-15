---
description: Restore the recorded approach when resuming task 1.5 or its descendants.
name: knodexa-task-1-5
---

# Task 1.5: CLI & MCP Server

Goal: Machine and human interfaces via stdio JSON-RPC and CLI

Recorded identity phrases: []. Resolve new requests against existing work before creating another task.

Resume with `knodexa resume 1.5 --json`. Restore this branch's established approach before coding. Read reference implementations and referenced skill documents. Record rationale and evidence for deliberate deviations. These generated modules are snapshots; consult canonical context for current decisions, failures, and lessons.

Selection: explicit. Candidates: []. Excluded approach sources: [].

File scopes: []

- Source constraint: Deterministic core
- Source constraint: No cloud dependencies
- Source constraint: Rebuildable SQLite index
## Approach inherited from 1

Implement continuity behavior in shared deterministic Go packages, with CLI and MCP adapters using those packages.

- Keep canonical Markdown/YAML authoritative and the SQLite cache rebuildable
- Preserve the deterministic zero-LLM core
- Choose approach inheritance through explicit parent and reuse relationships, never task-name prefixes
- Preserve established rules and acceptance criteria when compacting context
- Treat reference/literal checks as structural evidence; review semantic design consistency separately
- Prevent task-switch contamination through explicit relationships or unambiguous file scopes and return a target context boundary; never select authority by recency.
- Resolve incoming requests against recorded work identity before creating tasks; preserve existing completion state and distinguish continuation, independent work, and intentional replacement.
- Use Go for the application and required validation; retain one check wrapper and historical evidence rather than obsolete experiment scripts.
- Reference: `internal/approach/approach.go`, recorded SHA256 `0e3b4957429ec9bc74508e29feedef79d8780a146273fa2f44b4bee8f0f771d2`.
- Reference: `internal/continuity/resume.go`, recorded SHA256 `1634577f68815e4c488ad148220ff57cae555de4b86c74a212504a1a5ef3b608`.
- Reference: `internal/identity/resolve.go`, recorded SHA256 `f12904a2d68f1e9f5701e591ab3c6146ec763b0cd689519d8b2a11e0cc679253`.
- Required behavior check: continuity and adapter acceptance; command `go test -race ./internal/continuity ./internal/cli ./internal/mcp`.
  - Protected test/specification: `internal/continuity/behavior_test.go`, SHA256 `14363a1a77426fb9657ff3b98ded892d9889a7bb0b03c4c4d48c4faf6822bec0`.
  - Protected test/specification: `internal/cli/record_test.go`, SHA256 `387c14918486fd2e7eda8efa4f5c4ccc35c9de9629725e485f4b3fb09d27be9c`.
  - Protected test/specification: `internal/mcp/behavior_test.go`, SHA256 `c27d90b12f2acb00a77800bec371d26dd5efb5f5c6aad0a521d72ed2f2916978`.

