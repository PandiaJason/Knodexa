---
summary: A passing handler and preserved old memory do not prove that new extension memory was successfully indexed.
id: L-grade-extension-persistence
title: Grade new extension persistence separately from behavior and source preservation
status: VERIFIED
source_task_id: "1"
created_at: 2026-10-15T05:09:35Z
updated_at: 2026-10-15T05:10:18.404839Z
confidence: 0
applicability: When evaluating continuity after refactors or claiming successful end-to-end memory workflows.
tags:
    - evaluation
    - refactor
    - continuity
    - benchmark-coverage
related_files:
    - internal/experiment/live_hard_test.go
    - internal/experiment/live_superiority_test.go
    - audit/superiority-evaluation-v1/REPORT.md
    - README.md
evidence_ids:
    - E-extension-persistence-review
---

The eight-pair audit yielded six scored quality ties and two unscored fixture failures. Both arms passed all six paired returns. The final Knodexa refactor return passed behavior, source preservation and old-memory preservation while its extension record was rejected for an inherited missing reference. Its final response reported an unindexed agreement. The safety rejection preserved history, but extension persistence was not established. Future evaluations should predeclare successful new-memory capture as a separate endpoint; do not retroactively alter the existing primary scores. A fixture that saves an agreement then installs it again also violates the replacement API: install initial agreements once and validate fixture setup offline before paid trials. Observed return time favored Knodexa in aggregate, but action counts increased; neither speed nor quality superiority is established generally.
