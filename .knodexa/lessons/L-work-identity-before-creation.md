---
summary: Restore a uniquely matched recorded work identity before task creation; preserve its completed state and baseline. Weak or conflicting evidence requires clarification.
id: L-work-identity-before-creation
title: Resolve incoming requests before creating work
status: VERIFIED
source_task_id: "1"
created_at: 2026-10-14T07:57:24Z
updated_at: 2026-10-14T07:58:52.814379Z
confidence: 0.8
applicability: When a new request may continue earlier work after unrelated tasks or a context change, retrieve identity before choosing an implementation approach or creating another task.
tags:
    - work-identity
    - continuity
related_files:
    - internal/identity/resolve.go
    - internal/continuity/request.go
    - internal/cli/identity.go
    - internal/mcp/tools.go
evidence_ids:
    - E-work-identity-lesson
---

Recorded aliases, titles, goals, and repository paths provide deterministic evidence. Do not rank by recency or infer families from task-name prefixes. Keep independent new work and intentional replacements distinct. Task creation should guard strong duplicate-work matches and existing IDs.

Evidence: audit/identity-checks.log and audit/identity-benchmark.txt. The indexed lookup measured about 3.1 ms across 503 tasks, excluding synchronization, setup, and Git calls. This verifies resolver behavior, not semantic understanding or superiority over another agent. Go is the only required runtime; obsolete Python runners were removed and historical source examples retained as text.
