# Knodexa validation

Go is the only required programming runtime. Run:

```sh
GOCACHE=/tmp/knodexa-go-cache ./scripts/check.sh
```

This runs `go vet ./...`, the full race-enabled Go suite, builds the CLI, and
checks an isolated project with `init`, `task create`, `context`, `resolve`, and
`doctor`. CLI Go tests decode JSON and verify identity restoration and task
creation guards. No agent account, Python installation, or network is required.

## Work identity

`internal/identity` tests requests without task IDs, recorded vocabulary,
file-scope matches, weak and ambiguous matches, explicit new/replacement intent,
parent identity, unrelated task switches, alias edits, and invalid inputs.
`internal/continuity/request_test.go` checks that a matched completed task is
restored with its rules, failure, acceptance criteria, and context boundary,
without reopening it or creating another task. CLI and MCP integration tests
exercise the same service.

Run the request-resolution benchmark independently of setup and Git overhead:

```sh
go test ./internal/identity -run '^$' -bench BenchmarkRequestResolution -benchtime=5x
```

It resolves a request across 503 recorded tasks using compact indexed identity
metadata. This is a retrieval benchmark, not whole-agent latency.

## Continuity and correctness

The Go suites cover inherited constraints, task baselines, evidence ownership
and recorded successful execution, relevant lessons, reference drift, literal
checks, safe paths, scope ambiguity, incremental edits/deletions, duplicate-ID
rollback, and transactional rebuilds. Synthetic experiment fixtures exercise
representation and lifecycle behavior; they do not measure model quality.

`estimated_tokens` is a human-text approximation, not a model tokenizer. Essential
constraints and acceptance criteria survive context pruning, with explicit
`budget_exceeded` when they cannot fit. Verification validates supplied evidence;
it does not independently execute commands or authenticate authors.

## Historical model evaluations

The current opt-in real-agent identity test is implemented in Go:

```sh
KNODEXA_LIVE_IDENTITY=1 GOCACHE=/tmp/knodexa-go-cache go test ./internal/experiment -run '^TestLiveRequestIdentity$' -v -count=1 -timeout=12m
```

It requires authenticated Codex CLI access and consumes account usage. The
agent implements correct Go handler work and records its own aliases, rules,
scope, and reference; it switches to conflicting image work, then receives a
return request with no task ID or file path. Independent Go checks validate
returned behavior, memory/state preservation, actual resolver use, and baseline
comparison. Each agent phase has a three-minute timeout. Normal checks skip it.
The first live run passed; reports and raw traces are in `audit/identity-live`.

Recorded reports and transcripts remain in `audit/`. Obsolete Python runners
have been removed; captured Python implementations are stored as `.py.txt`
evidence. These artifacts document earlier experiments, not an active runtime
dependency or proof of superiority over Codex.

The completed repeated-switch pair tied at 9/9 design checks; both arms of the
second pair timed out and were unscored. Earlier evaluations supplied known task
identities, so they do not establish request-to-work identity resolution. The
initial live identity run establishes one successful end-to-end example;
general semantic matching and comparative model-level consistency remain to
evaluate.

## Paired real-agent request identity comparison

```sh
KNODEXA_LIVE_COMPARISON=1 GOCACHE=/tmp/knodexa-go-cache go test ./internal/experiment -run '^TestLiveIdentityComparison$' -v -count=1 -timeout=22m
```

This opt-in Go test consumes authenticated Codex usage. Both producers record
memory themselves, switch to unrelated work, and return without a task ID or
path. The native baseline uses modular file memory; Knodexa uses request
resolution. Both receive identical Go requirements and the same independent
behavior checks. Native memory can use prose; it need not follow Knodexa's
schema. Compare common behavior and preservation outcomes, not tool-specific
checks that only one arm implements. One pair cannot establish general advantage.
Reports and raw traces are saved in `audit/identity-comparison`.

Regenerate retained metrics without model calls:

```sh
KNODEXA_IDENTITY_REPORT=1 go test ./internal/experiment -run '^TestSummarizeIdentityComparison$' -v -count=1
```
# Repeated current-version live comparison

`KNODEXA_LIVE_REPEATED=1 go test ./internal/experiment -run
'^TestLiveRepeatedContinuityComparison$' -count=1 -timeout=40m -v` runs three
fresh paired trials with alternating arm order. It consumes authenticated Codex
usage. Results are retained under `audit/continuity-repeated-v2`; the runner
refuses to overwrite an existing run. Archive the whole prior run before a new
evaluation, rather than replacing selected failed trials. The latest completed
run tied on quality in all three pairs; raw traces and metrics remain available.

The opt-in `KNODEXA_LIVE_HARD=1 go test ./internal/experiment -run
'^TestLiveHardContinuity$' -count=1 -timeout=35m -v` captures a known-correct
fixture with real producer agents, then runs four isolated fresh-context pairs.
It refuses to overwrite `audit/hard-continuity-v1`. Independent graders check
behavior, a cross-task compatibility convention, observer panic failures, source
facts, and stale-refactor reporting. Existing completed facts must remain unchanged;
append-only extension records are allowed in either memory representation.

The first run encountered hidden/visible helper-name collisions; retained
artifacts were regraded without model reruns using `KNODEXA_HARD_REGRADE=1 go
test ./internal/experiment -run '^TestRegradeHardContinuity$' -count=1 -v`.
Original and corrected scores/logs remain available. The corrected report is
`audit/hard-continuity-v1/CORRECTED-REPORT.md`.

# Host discovery and shared-process audit

See [real host and concurrent-memory findings](audit/host-and-concurrency-v1/REPORT.md).
Opt-in Go tests `TestLiveHostIntegration` and `TestSharedMemoryAudit` preserve
predeclared model trials and process outputs. Audit collection completing is not
a safety pass: inspect `integration_pass`, `safe`, errors and independent grading.
The live host test consumes authenticated Codex usage; normal checks skip it.
