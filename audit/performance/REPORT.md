# Indexing and context performance

Measured on 2026-10-13 against the pre-optimization implementation from this
conversation, with correctness refinements already present. Same host, Go
benchmark fixtures, 500 tasks and 500 lessons plus manifest and empty link ledger.
Three runs of five operations per case. Medians are shown; timings exclude fixture
creation and do not represent whole-agent latency.

| Operation | Before | After | Speedup |
|---|---:|---:|---:|
| Unchanged synchronization | 130.38 ms | 36.82 ms | 3.54x |
| Save and synchronize one edited task | 134.35 ms | 46.53 ms | 2.89x |

Unchanged synchronization went from 6,008 SQLite row changes per operation to
zero. Allocation volume fell from roughly 16.36 MB to 3.00 MB per operation.
The changed-task benchmark includes the canonical file save.

The index now persists content hashes and canonical bytes. It parses and writes
only changed records, removes deleted record ownership, and preserves shared
edges owned by other records. Updates and rebuilds roll back on malformed files
or duplicate IDs. Unchanged fingerprints survive reopening the cache.
CLI resume refreshes once through the shared continuity service.

Every synchronization still inventories and reads canonical content. This is
intentional: timestamp-preserving edits must not escape detection. The remaining
cost scales with corpus size; there is no claim of constant-time startup, a
filesystem watcher, or a measured whole-agent 3.54x speedup.

For this repository's current context, compact JSON was 1368
bytes versus 1706 bytes in full mode. Both retain constraints and
acceptance criteria. This is one corpus sample, not a model token measurement.
Explicit author summaries and on-demand rationale reduce verbose history;
otherwise a complete first paragraph is retained without cutting clauses.

Evidence: before.txt, after.txt, summary.json, context JSON files, and checks.log.
Run `go test ./internal/index -run '^$' -bench '^BenchmarkSync' -benchtime=5x -count=3`
to measure the current implementation. The final standalone checks include
static analysis, race-enabled package tests, and all ten original audit probes.
