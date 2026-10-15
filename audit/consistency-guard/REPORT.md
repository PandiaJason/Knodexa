# Consistency guard validation

The opt-in runner now evaluates the restored agreement, rather than mutable
post-execution memory. It checks declared touched files plus newly discovered Go
files within restored scopes before capturing identity. Run results expose the
structural check report. Agreement replacement through CLI/MCP set requires an
explicit reason; no invented reason is supplied.

The regression fixture restores a handler agreement prohibiting println, then
runs a command that adds a compiling extension using println and removes the
saved check. Ordinary go test succeeds. The restored guard rejects the extension
and does not capture it. This exercises execution and file changes, not a paid
model trial. Another regression rejects unexplained agreement replacement.

Usage: prepare the relevant work, declare scopes and AST checks on its agreement,
then use run "<request>" --verify='go test ./...' -- <agent-command>.
For deliberate design changes, review them and record a new agreement with
change_reason; run does not authorize replacement during execution.

Limits: no automatic interception of other clients, no rollback of source or
metadata edited by commands, and no semantic proof from free-form rules or AST
checks. Agreement files remain writable by repository users. This establishes a
specific detection capability, not general superiority over Codex memory files.

Validation: GOCACHE=/private/tmp/knodexa-go-cache ./scripts/check.sh
(Go vet, full race tests, standalone CLI smoke).
