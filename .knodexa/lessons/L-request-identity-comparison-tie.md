---
id: L-request-identity-comparison-tie
source_task_id: "1"
title: A completed request-identity comparison tied against scoped file memory
summary: In a fresh real-agent pair without timeouts, both Knodexa and modular native file memory preserved behavior and completed records after an unrelated task. Knodexa used 20 command/file items versus 13, with no quality advantage observed.
applicability: When evaluating continuity claims, compare the same independent behavior and preservation requirements; keep response completion distinct and grade produced artifacts after timeout when possible. A successful Knodexa run is not by itself evidence of superiority.
status: PROPOSED
confidence: 0.8
related_files:
  - internal/experiment/live_identity_comparison_test.go
  - internal/experiment/live_identity_test.go
  - internal/experiment/live_return_grading_test.go
tags:
  - evaluation
  - work-identity
created_at: 2026-10-14T09:27:00Z
updated_at: 2026-10-14T09:27:00Z
---
Evidence: audit/identity-comparison/results.json and REPORT.md. Both independently passed the returned Go implementation and preserved reference/memory. Workflow seconds were native 174.27 and Knodexa 167.82; one unrandomized pair cannot establish a reliable speed difference. Total input tokens were 376498 and 717521, respectively; these cumulative values include cached tokens and do not measure active window occupancy or direct cost. Prior inconclusive traces are retained under audit/identity-comparison-before-timeout-grading.
