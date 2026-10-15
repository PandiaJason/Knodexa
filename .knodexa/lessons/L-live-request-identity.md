---
summary: A real agent recovered completed handler work from a no-ID request after unrelated work when its recorded alias matched the user's phrase, then reused the original implementation instead of reconstructing it.
id: L-live-request-identity
title: Capture the user's work phrase before switching focus
status: VERIFIED
source_task_id: "1"
created_at: 2026-10-14T08:14:38Z
updated_at: 2026-10-14T08:14:38.073755Z
confidence: 0.8
applicability: When testing work identity continuity, have the producer record aliases and reference code, switch work in the same conversation, and return without task IDs or file paths. Observe actual resolver use and independently check original behavior and state preservation.
tags:
    - work-identity
    - live-evaluation
related_files:
    - internal/experiment/live_identity_test.go
    - internal/identity/resolve.go
    - internal/continuity/request.go
evidence_ids:
    - E-live-request-identity
---

Evidence: audit/identity-live/results.json and REPORT.md. One controlled real-agent trial passed in 193 seconds. The agent recorded its own aliases and approach, resolved work-17 on return, reused Handle, preserved completed memory and reference code, and created no unrelated task. This supports one end-to-end case, not general semantic matching or superiority over native memory.
