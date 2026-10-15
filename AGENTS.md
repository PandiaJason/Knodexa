<!-- knodexa:begin -->
## Knodexa continuity and consistency

Use Knodexa before project implementation, when switching work, and before
reporting implementation success. Preserve the user's scope and existing project
instructions. Selected agreements govern the work; unrelated lessons are advisory.

### Restore before coding

Use configured Knodexa MCP tools or run the CLI from the repository root.
Prefer `knodexa` on PATH; in the Knodexa source checkout use `./bin/knodexa`.
If unavailable, report that limitation and read existing project agreements
directly. Do not claim checks ran when the integration was unavailable.

- For a request without a task ID, use `knodexa_prepare_work` or
  `knodexa prepare "<request>" --text`. Inspect identity evidence, selected work,
  `needs_review`, agreement sources, baseline and context boundary.
- For a known task, use `knodexa_resume_task` or
  `knodexa resume <task-id> --json`. Completed sources stay completed; reopen
  only for an intentional reopening.
- Resolve ambiguity using actual scope and files; clarify user intent if needed.
  Never choose authority by recency or task-name prefixes. Create independent
  work explicitly; link related extensions to the matched parent or reused
  agreement. Continue useful independent work while clarification is pending.
- Read constraints, acceptance criteria, past failures, reference code and
  required skills. Expand truncated references before relying on them. Investigate
  baseline mismatch and stale references; never silently refresh their hashes.
  Set aside conventions from unrelated prior work.

### Preserve the agreement

Follow the selected design. Record consequential decisions and failed approaches
with rationale and resolution. Intentional departures require a specific
`change_reason` and verification evidence, retaining the previous agreement.
A reason records intent; it does not prove correctness or expand authorization.

### Check before success

- Run relevant behavioral tests plus `knodexa approach check <task-id> --json`
  (MCP: `knodexa_check_approach`). Declare actual touched files on the task so
  reusable `@touched` checks cover the extension. Review free-form design rules
  separately; structural success is not semantic proof.
- For an executable agent workflow, use
  `knodexa run "<request>" --verify='<test command>' -- <agent-command>`.
  The agent must read `KNODEXA_CONTEXT_FILE`. The runner checks restored
  agreements and new Go files in declared scopes before capturing identity.
  Failed checks do not roll back file edits.
- Capture verified work with `knodexa record <task-id> --file=agreement.json
  --verify='<test command>'` (MCP: `knodexa_record_work`). Include aliases,
  files and `approach` with summary, rules, scopes, references and relevant checks.
  Replacement requires `approach.change_reason`. Preserve completed ancestors
  and their baselines; reviewed refactor bindings belong on the related child.
- Extract reusable lessons with affected files and tags. Attach actual evidence
  before verifying decisions or lessons. Refresh human-readable snapshots with
  `knodexa memory export` when useful; canonical YAML/Markdown stays authoritative.
  When verified knowledge becomes obsolete, use `knodexa supersede <old-id>
  --by=<verified-replacement> --reason='<why>'` (MCP:
  `knodexa_supersede_knowledge`). Keep historical content; do not let obsolete
  advice compete with the current agreement. Expand deferred memory on demand.

Report passed checks, unresolved drift and integration limitations. A missing
agreement or unrun check is not verified consistency. This applies to repository
coding work, not unrelated conversation. Instructions depend on the host loading
and following them; they do not intercept every agent's tools automatically.

<!-- knodexa:end -->

## Knodexa repository requirements

- The product goal is preserving work identity and implementation agreements
  across unrelated work and fresh contexts. Task-name prefixes never establish
  inheritance; use explicit relationships or unambiguous file scopes.
- Keep canonical Markdown/YAML authoritative and SQLite rebuildable. Preserve
  the deterministic zero-LLM core. Use Go for implementation and required tests.
- Build a missing or stale CLI with
  `go build -o bin/knodexa ./cmd/knodexa`. Run `./scripts/check.sh` for Go vet,
  race tests and standalone validation. Do not introduce extra experiment scripts.
- Record consequential decisions, failures and test evidence through Knodexa.
  Extract reusable lessons with touched files and tags before completion.
- Use declared scopes and touched files for consistency checks. Restore known
  previous-focus boundaries when useful; focus supplies exclusions, never authority.
- Structural checks require separate semantic review. Use real agent trials for
  agent-quality claims; synthetic tests and successful retrieval do not establish
  superiority over competent scoped memory.
