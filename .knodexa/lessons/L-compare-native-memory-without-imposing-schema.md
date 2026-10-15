---
id: L-compare-native-memory-without-imposing-schema
source_task_id: "1"
title: Accept native memory formats and retain code on interrupted turns
summary: A file-memory baseline should not need Knodexa's typed schema. Preserve produced code before timeout cleanup, and keep incomplete turns unscored rather than claiming superiority.
applicability: When evaluating request-to-work identity against native file tools, grade the same behavioral outcomes, allow native prose, record harness interruptions, and distinguish incomplete execution from demonstrated design drift.
status: PROPOSED
confidence: 0.8
related_files:
  - internal/experiment/live_identity_comparison_test.go
  - internal/experiment/live_identity_test.go
tags:
  - evaluation
  - work-identity
created_at: 2026-10-14T09:09:18Z
updated_at: 2026-10-14T09:09:18Z
---
Evidence: audit/identity-comparison/results.json and REPORT.md. Knodexa completed independent checks in the fresh trial. Native code passed agent-authored tests but its return timed out before independent grading; the comparison is inconclusive. An earlier native producer passed independent checks but the harness incorrectly rejected its prose approach representation. Its traces were retained and not counted as a quality failure. Future timeout paths retain source snapshots.
