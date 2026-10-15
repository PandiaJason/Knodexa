# Audit remediation

The original observations remain in REPORT.md and results.json. fixed-results.json
records the same probes against the repaired binary; every problem flag is false.

- Evidence verification rejects unrelated, unsuccessful, or incomplete execution records. Canonical omitted/null exit codes are distinguished from zero. MCP requires an explicit integer exit code, command, and output. Manual sign-off requires recorded details and an actor. Records remain supplied assertions, not independently executed or cryptographically authenticated proof.
- Index synchronization replaces the derived snapshot inside one SQLite transaction. Deleted nodes and removed file/tag/constraint associations disappear. Malformed canonical records abort synchronization and roll back instead of silently dropping data. This remains a full scan; incremental indexing and large-project performance are not established.
- Context loads full canonical tasks, retaining acceptance criteria and stored baseline commits. Stored/current baseline mismatch is exposed.
- Decisions and failures traverse all ancestors with cycle-safe recursive SQL, retaining the immediate-sibling policy for compatibility.
- Lessons require explicit relevance links or real file/tag overlap, use stable ordering, and include their guidance and applicability. Decision rationale is also included.
- The decision MCP handler saves supplied constraints and rejected alternatives. Generated record IDs now use 128-bit random identifiers rather than a short clock remainder. Mutation handlers report indexing failures.
- Budgets explicitly report budget_exceeded when essential context cannot fit and context_pruned when optional context is removed. estimated_tokens remains a documented human-text approximation rather than total model usage.
- The uncomputed comparison scorecard was removed. Fixture tests and logs are labeled as representation/integration tests. They no longer establish a Codex performance advantage.

Verification: scripts/check.sh runs static analysis, all race-enabled Go tests,
fresh-binary audit probes, and independent CLI smoke checks. fixed-checks.log
contains the run output. Live Codex pilot results are reported separately under
live/ and do not determine whether these correctness regressions pass.
