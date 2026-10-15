---
id: L-bound-verification-pipes
title: Bound command pipe draining in every verification path
status: PROPOSED
source_task_id: "1"
summary: Share command execution across run, record and inherited acceptance checks; bound pipe draining with exec.Cmd.WaitDelay.
applicability: When executing shell commands whose descendants may retain stdout or stderr.
tags: [verification, subprocess, architecture]
related_files:
  - internal/continuity/command.go
  - internal/continuity/command_test.go
  - internal/continuity/run.go
  - internal/continuity/record.go
  - internal/continuity/behavior.go
---

CommandContext alone cannot bound waiting for inherited output pipes after the
shell exits. Centralize execution so all verification paths use the same bounded
output, exit reporting and WaitDelay. A pipe-drain error must fail verification.
WaitDelay closes pipes; it does not terminate background descendants.

Regression: TestCommandBoundsInheritedOutputPipes starts a background sleep that
inherits output, and requires ErrWaitDelay, a failing outcome, and a bounded return.
Full local validation: ./scripts/check.sh (vet, race tests and standalone CLI).
No agent-quality improvement or superiority claim follows from this cleanup.
