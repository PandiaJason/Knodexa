---
summary: Publish measured results and limitations, not marketing superiority.
alternatives_considered:
    - Treating provider usage-limit interruptions as coding failures
    - Publishing only successful metrics or dropping the incomplete planned case
    - Adding raw local session traces to the public repository
id: D-cd5ab9e181bab291dc1a27bd65bb06c8
title: Publish current-release evaluation without superiority claims
status: VERIFIED
task_id: "1"
created_at: 2026-10-15T03:53:03.159999Z
updated_at: 2026-10-15T03:54:29.452957Z
confidence: 0.8
evidence_ids:
    - E-cc646fb1f3c5fa2bd7c1d6ed07145473
---

## Rationale
Publish a professional README, PNG/SVG charts, public plans/results/grading artifacts and CI. The completed paired trial tied on quality and Knodexa used 29% more agent-phase time and 64% more actions; the fresh-context pair is unscored due to account usage limits. Preserve failures and exclude partial timings from comparative charts. Keep local raw session traces out of Git. User explicitly requested a public GitHub repository.

## Rejected alternatives
- Treating provider usage-limit interruptions as coding failures
- Publishing only successful metrics or dropping the incomplete planned case
- Adding raw local session traces to the public repository
