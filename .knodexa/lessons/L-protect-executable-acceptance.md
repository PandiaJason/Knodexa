---
summary: Run restored acceptance checks in addition to caller tests, pin test inputs, and retain omitted checks on routine capture.
id: L-protect-executable-acceptance
title: Preserve executable acceptance and its test inputs across extensions
status: VERIFIED
source_task_id: "1"
created_at: 2026-10-15T06:31:35Z
updated_at: 2026-10-15T06:32:27.324358Z
confidence: 0
applicability: When extending established behavior or implementing verified task capture after unrelated work and refactors.
tags:
    - coding-quality
    - acceptance
    - regression
    - refactor
related_files:
    - internal/approach/behavior.go
    - internal/continuity/behavior.go
    - internal/continuity/record.go
    - internal/continuity/run.go
    - internal/continuity/reference_review.go
    - internal/continuity/behavior_test.go
    - internal/integration/protocol.md
evidence_ids:
    - E-protected-acceptance-lesson
---

Structural checks and weak caller-selected tests can pass compilable behavior regressions. Execute named restored acceptance checks, check protected test/specification hashes before and after execution, reject changed acceptance definitions, and carry existing checks forward when routine capture omits them. Explicit acceptance changes require versioned review followed by actual verification. Refactor review can derive historical/current hashes from a named inherited source and inspected current path without rewriting the ancestor; a reason does not prove equivalence. Tests must cover the extension rather than only old reference code. A fresh agent successfully completed behavior and memory capture on the new path, but the earlier paired quality ties remain unchanged. These opt-in checks do not intercept direct file edits or establish general model superiority.
