# Knodexa implementation and comparison audit

Date: 2026-10-13. Baseline: `133bf2310370a679e592291a82e5087ed87d7747`, with the uncommitted refinements from this conversation. Audit reproductions use a freshly compiled binary and disposable projects; application source was not changed during the audit.

## Verdict

Knodexa has a working storage, CLI, MCP, and retrieval pipeline. It is not yet reliable enough to treat its rehydrated context or VERIFIED labels as authoritative. The current evaluation suite does not establish better coding outcomes or lower total token usage than Codex's built-in continuity.

Existing passing tests confirm selected behaviors. Ten additional behavioral probes reproduced the problems below; `results.json` contains their observations. A true field value denotes a reproduced problem, not a passing safety test.

## Findings

### P1: Failed and unrelated evidence can verify knowledge

Location: `internal/verify/verify.go:78`.

Verification only checks that an evidence file exists. The reproduction attached a TEST_RUN with exit code 1 to a decision and successfully promoted it to VERIFIED. The same evidence then verified an unrelated lesson. Neither evidence ownership nor successful execution is checked. Empty evidence metadata can also default to a successful exit code through the MCP handler; it records assertions and does not execute the command.

Impact: downstream agents receive unsupported assertions as verified knowledge. Require matching node ownership, type-specific evidence requirements, explicit successful execution for test/build evidence, and clear provenance. Manual sign-off needs its own explicit policy; merely executing a supplied command would introduce a separate trust problem.

### P1: Synchronization retains removed canonical knowledge

Locations: `internal/index/indexer.go:50`, `internal/index/indexer.go:295`, `internal/index/indexer.go:357`.

Sync upserts nodes and inserts relationships without reconciling removals. Replacing OLD constraint with NEW constraint left OLD in the next context package. Deleting the task YAML still allowed a subsequent CLI process to retrieve that task through SQLite.

Impact: obsolete constraints remain active and deleted tasks survive as phantom context. Make synchronization reconcile node and relationship deletions transactionally. A rebuild can repair the cache, but routine retrieval should not require users to know it is stale.

### P1: Task-specific acceptance criteria disappear on the index path

Location: `internal/context/builder.go:38`.

The indexed node is reconstructed as TaskData without RequiredVerification or BaselineCommit. A task with the explicit criterion "Run the custom acceptance benchmark" returned only generic defaults. The reproduction used the normal CLI path, where the index is present.

Impact: agents can declare completion without the task's actual acceptance check. Preserve the full task metadata or read the canonical task when building context. Baseline comparison also needs explicit stored-versus-current handling; this audit directly reproduced acceptance-criteria loss.

### P1: Upstream decisions stop at the immediate parent

Location: `internal/index/queries.go:156`.

The query collects the task, its immediate parent, and siblings. It does not traverse all ancestors despite the function's documented promise. A verified ROOT decision was absent from a ROOT -> child -> leaf context package. Failure lookup uses the same one-parent pattern, though the reproduction here exercised decisions.

Impact: deeply nested tasks lose inherited architectural decisions. Traverse the ancestor chain with cycle protection and make sibling relevance explicit.

### P2: Cross-task transfer includes unrelated lessons and omits their guidance

Locations: `internal/index/queries.go:343`, `internal/context/builder.go:136`, `internal/context/types.go:25`.

A leaf task touching auth.go received a lesson tagged images and touching images.png through the highest-confidence fallback. It was labeled medium relevance. The package included the title and status but omitted the lesson description and applicability. The tags argument to GetRelatedLessons is not used to filter or rank results.

Impact: transfer can add noise while dropping the instructions the next agent needs. Return actionable guidance or an explicit expansion reference; use actual file/tag matching and label fallback results honestly. File matches also lack stable tie-breaking and scanNode does not hydrate lesson file associations, weakening relevance labeling.

### P2: Decision tool silently discards advertised inputs

Location: `internal/mcp/tools.go:297`.

The schema accepts constraints and alternatives_considered. The handler stores only the title, task, and rationale. Both a supplied constraint and rejected alternative were absent from the saved decision document.

Impact: architectural rationale becomes incomplete even when the agent follows the documented protocol. Persist these inputs or remove unsupported fields from the schema.

### P2: Token budget is a soft approximation without an overflow signal

Location: `internal/context/builder.go:186`.

A request for budget 1 returned a package estimated at 122 tokens. Pruning retains at least one lesson/decision and cannot reduce mandatory sections. The estimate covers human text rather than serialized JSON or actual model tokenization.

Impact: callers cannot rely on the declared maximum. Preserve essential constraints, but explicitly report the minimum required budget or return an overflow status. Document estimation and transport overhead.

## Why the existing comparison is not evidence against Codex

`internal/experiment/eval_test.go:20` names its comparator as an Antigravity native stack, not an executed Codex configuration. Fixtures are manually constructed. No model or coding-agent session runs in these tests.

- Token tests compare an entire AGENTS.md plus rules and a skill against a much smaller context summary. They omit the Knodexa arm's continuing instructions, tool definitions, calls, record-writing costs, and follow-up reads. Smaller output also loses semantic content, as reproduced above. Token counts use character heuristics, not measured model usage.
- Failure-recall scores reward Knodexa's data structure with custom extra points. They do not measure whether an agent avoids a mistake.
- Decision tests check existence and status, not whether a resumed agent receives or applies rationale correctly.
- The final scorecard at `internal/experiment/eval_test.go:581` prints literal metrics and conclusions; it is not computed from results.
- Merge tests create separate decision files on each branch. They do not exercise normal operations that also append shared events.jsonl and links.jsonl. They cannot establish general conflict-free multi-agent operation.

These are useful fixture and integration tests when named accordingly. They cannot support claims of superior completion rate, real token savings, or absence of merge conflicts.

## Verification and reproduction

Run from the repository:

```sh
GOCACHE=/tmp/knodexa-go-cache go build -o /tmp/knodexa-audit-bin ./cmd/knodexa
python3 audit/reproduce.py /tmp/knodexa-audit-bin
GOCACHE=/tmp/knodexa-go-cache ./scripts/check.sh
GOCACHE=/tmp/knodexa-go-cache go test ./internal/experiment -run '^TestEval' -v -count=1
```

Artifacts: `results.json`, `checks.log`, and `evaluation.log`. The reproduction mutates only its temporary project and removes it afterward.

## Limits and next decision

This is an implementation and evaluation-methodology audit, not a live Codex-versus-Knodexa trial. It does not cover every storage, security, concurrency, or performance edge case. The evidence gate, cache reconciliation, and context completeness should be fixed before measuring benefit. Then run matched coding tasks with fresh sessions across AGENTS.md alone, enabled Codex memories, and Knodexa. Keep model, task information, acceptance tests, and resource budgets equal; record total usage, errors, task-switch recovery, and completion rather than awarding points for product features.
