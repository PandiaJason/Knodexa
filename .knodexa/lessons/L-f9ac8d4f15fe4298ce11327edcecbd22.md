---
id: L-f9ac8d4f15fe4298ce11327edcecbd22
title: Check restored contracts and new scoped files before capture
status: VERIFIED
source_task_id: "1"
created_at: 2026-10-14T17:29:11.883129Z
updated_at: 2026-10-14T17:29:38.330203Z
confidence: 0.8
applicability: When validating an agent's extension against an established implementation agreement.
tags:
    - consistency-guard
    - drift
    - continuity
related_files:
    - internal/continuity/run.go
    - internal/continuity/run_test.go
    - internal/approach/approach.go
evidence_ids:
    - E-4f07463842daaf725b28c1dcb9bf0a2a
---

Pin checks before execution, then discover new files in declared scopes. A compiling extension can violate an established pattern even when an invoked command removes its saved checks. Require explicit reasons for replacements; AST rules remain syntactic and files are not rolled back.
