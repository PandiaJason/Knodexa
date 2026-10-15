# Behavioral quality contracts and reviewed refactors

Knodexa now executes explicitly recorded acceptance checks before verified capture.
The implementation checks below passed. A fresh real agent also produced correct
refactored code **and persisted a valid extension agreement**, while preserving the
completed parent's facts and baseline.

## Deterministic regression checks

| Candidate/workflow | Weak caller verification | Recorded quality gate |
|---|---|---|
| Trim raw input before delegation | Passed compilation/smoke test | Rejected |
| Swallow service error | Passed compilation/smoke test | Rejected |
| Invoke service twice | Passed compilation/smoke test | Rejected |
| Change observer ordering | Passed compilation/smoke test | Rejected |
| Remove/change protected acceptance tests | Not accepted | Rejected before capture |
| Erase acceptance definitions during verification | Command completed | Rejected before capture |
| Replace acceptance command with a weaker one in `record` | Not executed | Explicit agreement review required |
| Omit old checks from a successful routine refresh | Passed | Old checks retained for future runs |
| Correct implementation | Passed | Accepted and indexed |
| Reviewed source relocation/rename | Passed | Accepted, versioned child binding saved |

These are controlled Go regressions, not model-quality samples. They show that
compiling code and passing a weak caller-selected test no longer suffice when a
stronger behavioral contract has been recorded. Protected input hashes guard the
recorded tests/specification, and routine capture retains existing checks. Explicit
acceptance changes use versioned `approach set` with a reason, followed by actual
verification. Coverage still depends on the tests and commands the project records.

[Regression test output](regression-tests.log). Tests also cover inherited
obligations, `run` enforcement, compact context, failed-write atomicity, invalid
refactor reviews, CLI JSON/help and MCP execution. The omission/weakening guards
were hardened and tested offline after the live trial; its child-first recording
path was unchanged. Ordinary validation skips authenticated model calls.

## Fresh-agent workflow acceptance

The [plan](plan.json) declared one attempt: a producer records a known-correct
fixed handler, the fixture owner installs protected independent acceptance test
templates, the source is relocated/renamed after its checkpoint, and a fresh
agent implements an extension using the installed continuity protocol.

The test templates exist before the return and are inspectable, as normal
project acceptance specifications. The recorded command copies those protected
templates into the test package and executes them. Independent post-return grading
then verifies behavior and the cross-task SDK compatibility marker. New-memory
capture is a separate required endpoint, checked against canonical child records,
current source hashes and valid inherited reference reconciliation before cleanup.

| Endpoint | Result |
|---|---|
| Independent behavior and SDK lesson | Passed |
| Current reference preserved | Passed |
| Completed parent facts and baseline preserved | Passed |
| Stale reference/checkpoint explicitly reported | Passed |
| Inherited acceptance executed successfully during `record` | Passed |
| Child agreement and current source identity captured | Passed |

The return agent completed in **79.46 seconds / 12 tool actions**. Producer setup
and capture are separate. The full live Go test took about 124 seconds. There is
one producer and one fresh return agent, using gpt-6.1-sol at low reasoning effort.
The plan's baseline commit identifies pre-change history; this trial ran the
working-tree quality implementation, with subsequent retention hardening covered
by offline tests. No scored live attempt was retried.

- [All endpoint results](results.json)
- [Successful record and named behavioral outcomes](refactor/knodexa/record-result.json)
- [Independent Go grading](refactor/knodexa/independent-checks.log)
- [Agent's final response](refactor/knodexa/return.final.txt)
- Source snapshots and before/after completed parent memory are retained alongside them.
- [Original live execution log](execution.log)

Raw local agent JSONL/stderr remain excluded from Git publication. The narrow
record-result artifact is extracted from the actual CLI output; it records
successful inherited and child-local behavioral checks.

## Scope of the improvement

This adds executable quality enforcement and removes manual hash assembly from an
explicitly reviewed refactor. It does not make free-form design rules executable,
intercept every host's edit tools, or infer semantic equivalence from a reason.
Commands execute locally only in opted-in `record`/`run` workflows. Failed checks
leave code/file edits in place; they prevent verified capture rather than roll back
an agent's edits. Direct canonical edits remain possible, so this is not an
adversarial security boundary.

The live workflow passed the new capture endpoint that a prior refactor agent
failed to complete. This is acceptance evidence for the new workflow, not a
controlled before/after quality comparison: instructions, fixtures and agent draws
differ. Existing [six quality ties and two unscored setup failures](../superiority-evaluation-v1/REPORT.md)
remain unchanged. Higher general coding quality than organized memory is still
not established by these tests.
