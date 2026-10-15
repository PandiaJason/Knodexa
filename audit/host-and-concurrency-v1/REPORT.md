# Host discovery and concurrent-memory audit

Three predeclared fresh ephemeral Codex CLI contexts ran the same plain coding
request using gpt-6.1-sol with low reasoning effort and background memories
disabled. The request did not name Knodexa or invoke a skill. Each isolated Go
repository had the same completed production agreement and reference code.
No failed trial was selectively rerun. All raw traces and snapshots are retained.

| Instruction setup | Restored before editing | Agreement checked | Independent behavior | Parent memory unchanged | Seconds | Commands |
|---|---|---|---|---|---:|---:|
| Installed AGENTS.md | Pass | Pass | Pass | Pass | 76.29 | 12 |
| Skill in .agents/skills/knodexa-continuity | Pass | Pass | Pass | Pass | 70.60 | 13 |
| Skill in distribution folder skills/knodexa-continuity | Pass | Pass | Pass | Pass | 78.53 | 11 |

The distribution-folder control crossed over: the agent searched repository
files, read `skills/knodexa-continuity/SKILL.md`, then invoked Knodexa. This is
successful manual discovery, not proof that the host automatically loads that
location. In the configured-path trial the agent read the hidden skill before
its hidden-file inventory; the trace is consistent with host skill discovery.
[Official AGENTS.md guidance](https://learn.chatgpt.com/docs/agent-configuration/agents-md)
documents project instruction loading, and
[official skill guidance](https://learn.chatgpt.com/docs/build-skills) identifies
`.agents/skills` as the repository discovery path. Implicit skill selection remains
a model decision; one successful trial does not establish guaranteed invocation.

Independent tests checked raw whitespace, pointer identity, integers, nil input,
single service delegation, original result identity and original error identity.
Before/after parent YAML was compared byte-for-byte. All trials completed without
timeouts. Shell-command counts do not include every file-edit tool item. Reported
input token usage is cumulative usage reported by the host, includes cached input,
and is not a unique-context-size or billing claim. No baseline-quality comparison
against other memory tools was performed.

## Concurrent processes

Eight independent OS processes ran each case. Same-task snapshot workers all
read the canonical record before a barrier released their writes. The public CLI
case used actual `identity alias` commands against the shared repository.

| Case | Workers exiting successfully | Updates retained | Safety result |
|---|---:|---:|---|
| Same-task synchronized read-modify-write | 8/8 | 1/8 | Fail: seven updates lost |
| Distinct-task canonical snapshots | 8/8 | 8/8 | Pass for this case |
| Public CLI aliases on the same task | 4/8 | 3/8 | Fail: lost successful update plus four initialization failures |

All four failed public commands reported SQLITE_BUSY during schema initialization.
Task completion state, baseline and agreement stayed intact in these cases; that
does not compensate for lost updates. Atomic rename prevents torn individual
files, but separate processes can overwrite stale snapshots. The distinct-task
case tested canonical record writes, not concurrent SQLite/index mutations.

The concurrency audit harness exits zero when it successfully collects findings;
its `safe=false` results are safety failures, not passing concurrency checks.
Go's race detector does not detect this cross-process lost-update problem.

## Reproduce and inspect

```sh
GOCACHE=/private/tmp/knodexa-go-cache KNODEXA_SHARED_MEMORY_AUDIT=1 \
  go test ./internal/experiment -run '^TestSharedMemoryAudit$' -count=1 -v
GOCACHE=/private/tmp/knodexa-go-cache KNODEXA_LIVE_HOST=1 \
  go test ./internal/experiment -run '^TestLiveHostIntegration$' -count=1 -timeout=11m -v
```

Existing audit output is intentionally protected against overwrite; preserve it
under a distinct directory before a new run. Live trials consume authenticated
agent usage. Their sessions were ephemeral; retained traces are local artifacts.

- [Host results](host/results.json), [declared plan](host/plan.json), raw JSONL,
  final messages, independent grading logs, source copies and parent snapshots.
- [Concurrency results](concurrency/results.json), individual worker logs and
  before/after canonical snapshots.
- Full validation: `GOCACHE=/private/tmp/knodexa-go-cache ./scripts/check.sh`
  passed Go vet, race tests and standalone CLI smoke. Normal checks skip live
  trials and the opt-in unsafe-concurrency audit.

Conclusion: the installed protocol was followed in three real Codex trials,
including manual repository discovery in the control. Shared-task updates are
not safe. The next implementation should serialize read-modify-write operations
across processes or reject stale writes with compare-and-swap/retry, and handle
SQLite contention with bounded waiting. This audit does not implement those fixes
or establish superiority over all memory tools or adoption by other hosts.
