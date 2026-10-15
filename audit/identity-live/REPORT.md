# Real-agent work identity trial

**Passed.** A real Codex agent established Go handler work, recorded its own
continuity memory, switched to unrelated image work, then recovered the original
work from a natural-language return request with no task ID or file path.

The return request was:

> Add another webhook handler for delivery events. Expose HandleWebhook(value any, service Transformer, observer Observer) (any,error). Keep our established layout and behavior. Add and run relevant tests.

The agent actually ran Knodexa request resolution; its command output identified
`work-17`, the original work. It implemented the new handler by reusing the
existing `Handle` function, preserving the established approach.

Independent Go checks passed for the original and returned implementations:
raw service input, exactly one call, result and error identity, nil observers,
observer ordering, no finish hook after errors, and unchanged panic propagation.
The completed source memory and reference code were unchanged; no unrelated
new task was created. The saved baseline was preserved and Git drift reported.

| Phase | Seconds | Completed command/file items |
|---|---:|---:|
| Establish correct work and capture memory | 78.76 | 18 |
| Switch to inline image normalization | 57.81 | 10 |
| Return without task identity hints | 50.60 | 10 |

The full Go live test finished in 193 seconds, including setup and independent
checks. Model: gpt-6.1-sol, low effort, same conversation across all phases.
Generated background memories were disabled. Source task definitions and
initial constraints were prepared by the harness; aliases, approach rules,
scope, and reference capture were performed by the actual agent. After grading
correct source work, the harness marked it completed and preserved its baseline.

This is one controlled Go repository and one recorded phrase, not a native-memory
comparison, saturated-context test, or proof of general semantic matching. It
establishes that the new request-to-work entry point can work end to end with a
real agent. It does not establish superiority over Codex.

`results.json` contains the measured outcomes and usage. Raw traces are
`establish.jsonl`, `switch.jsonl`, and `return.jsonl`; final source snapshots,
producer memory, returned context, and independent check logs are also retained.

The opt-in runner is `TestLiveRequestIdentity` in
`internal/experiment/live_identity_test.go`; routine Go checks skip paid live runs.
