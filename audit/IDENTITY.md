# Work identity refinement

The entry point is now `knodexa resolve "<request>" --json` or MCP
`knodexa_resolve_request`. It accepts requests without a task ID, searches
recorded titles, aliases, goals, and files, exposes matching evidence, and
restores selected existing work. Completed status and saved baseline remain
unchanged; resolution does not generate a task.

Weak, overlapping, and unmatched requests remain unresolved. Explicit new work
and intentional replacement are separate intents. CLI creation guards strong
matches to existing work and duplicate IDs; extensions use an explicit parent
or approach link. This is deterministic lexical retrieval, not a general
semantic classifier or proof of model obedience.

Validation: the full race-enabled Go suite and standalone CLI checks passed
(`identity-checks.log`). Dedicated tests exercise no-ID requests, unrelated task
switches, ambiguity, explicit intent, state preservation, inherited agreements,
creation guards, and MCP integration. The 503-task lookup benchmark measured
3.1033 ms, 3.133625 ms, and 3.276633 ms per query (`identity-benchmark.txt`),
excluding setup, synchronization, and Git subprocesses.

The obsolete Python runners were removed. Captured generated implementations
were retained as `.py.txt` evidence. Active application code and validation
checks use Go; `scripts/check.sh` is the single shell wrapper. Previous live
comparisons supplied task identity upfront and cannot validate the new entry
point or establish superiority over Codex.
