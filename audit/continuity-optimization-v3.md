# Continuity overhead changes

The prior three-pair live comparison tied on quality and used 71 Knodexa
command/file actions versus 32 with scoped native memory. Traces included
separate approach/alias/source capture calls, CLI discovery and invalid flags.

Changes:

- `record <id> --file=agreement.json --verify='go test ./...'` executes explicit
  verification, evaluates proposed/inherited structural checks, and saves aliases,
  source anchors and a versioned agreement in one canonical task update. It
  preserves completion state and baseline. Failed tests/checks do not install a
  partial record. Replacement requires a reason.
- `prepare '<request>' --text` gives a readable handoff without duplicated JSON
  metadata. JSON remains available. Mandatory context, source hashes and excerpt
  truncation markers remain visible. In the focused fixture, text was 1,583 bytes
  versus 2,735 JSON bytes (42% smaller); this is not a billed-token measurement.
- Source matching builds path/hash lookups instead of comparing every current
  file against every recorded anchor. Ambiguity remains conservative.

Agreement input:

```json
{
  "aliases": ["webhook handler"],
  "files": ["handlers/reference.go"],
  "approach": {
    "summary": "Delegate raw values through the injected service",
    "rules": ["Preserve result and error identity"],
    "scopes": ["handlers/"],
    "references": [{"path": "handlers/reference.go"}]
  }
}
```

Live verification uses a fresh agent context on return for both arms. Same model,
same behavior requirements and independent grader, competent scoped native file
memory baseline. It tests cross-context retrieval rather than forced compaction.
Results are retained under `continuity-optimized-fresh-v3`, including failures or
timeouts. A single paired trial cannot establish general superiority.

That trial tied: native 155.13 seconds / 16 actions, Knodexa 159.77 seconds /
28 actions. The Knodexa return trace copied a `prior-focus` documentation
placeholder as a literal task ID, causing a failed preparation and multiple
discovery commands. The result remains scored and retained.

Correction: resume stores advisory local focus; prepare uses it only after target
identity is already resolved, to exclude unrelated conventions. It never selects
work by recency. Stale/missing advisory focus cannot block retrieval; explicit
invalid previous-task inputs still fail. Corrected trial evidence is retained
separately under `continuity-optimized-fresh-v4`.

The v4 trial favored native memory on the scored quality criteria. Both handlers
passed independent behavior checks, but the Knodexa agent called `resume` after
preparation, which implicitly reactivated completed source work. Knodexa was
faster (125.30 vs 197.56 seconds) with equal action counts (18), but lost source
memory preservation. This is a real failure, not a successful comparison.

Fix: `resume` preserves completed task bytes and baseline by default. Reactivation
requires `--reopen` / MCP `reopen: true`. A Git-backed regression test verifies
retrieval cannot initialize a missing completed baseline or change task state;
explicit reopening still works. The post-fix pair is retained under
`continuity-optimized-fresh-v5`.

Post-fix result: quality tied; both preserved behavior and completed source memory.
The Knodexa agent again called resume after preparation, exercising the fixed API.

| Metric | Native scoped memory | Knodexa |
|---|---:|---:|
| Workflow seconds | 180.84 | 173.08 |
| Command/file actions | 18 | 22 |
| Input tokens including cached input | 306,471 | 294,566 |

This is about 4% lower observed time and cumulative input, with more actions.
A single pair cannot demonstrate reliable superiority. All prior tied/failed
comparisons remain visible. The unit handoff fixture after these changes is
1,730 text bytes versus 2,882 JSON bytes (40% smaller).

Full report: [post-fix trial](continuity-optimized-fresh-v5/REPORT.md).
