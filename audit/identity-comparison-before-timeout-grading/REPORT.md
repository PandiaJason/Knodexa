# Paired real-agent work identity comparison

Outcome: **inconclusive**.

Both arms use the same Go requirements, actual producer memory capture, unrelated image work, and a same-conversation return with no task ID or path. The native arm uses an index and modular JSON memory files with native file tools; Knodexa uses its request resolver and canonical task records.

| Arm | Common quality checks | Workflow seconds | Command/file items | Input tokens |
|---|---|---:|---:|---:|
| native | Unscored | 288.16 | 12 | 184908 |
| knodexa | Passed | 233.40 | 41 | 879301 |

Common quality means independent returned behavior passes and original reference and completed memory are preserved. Native memory retrieval does not require a Knodexa tool call. Knodexa-specific routing/baseline checks are reported separately, not used to make the native quality criteria stricter.

One pair, unrandomized arm order, a controlled Go repository, one recorded phrase, different generated implementations and histories, no forced compaction, and generated background memories disabled. This does not establish general superiority over Codex or compare all desktop tools. Timeouts remain unscored. Workflow timings include the three agent phases, exclude harness setup/grading, and depend on server scheduling. Tokens are cumulative tool-call usage, not active context-window occupancy. Do not treat a tie as a quality win.

Raw traces, final source snapshots, producer memories, and independent checks remain in each arm directory. Plan and aggregate metrics are in plan.json and results.json.

Harness interruption: the initial native producer passed behavior checks, but a typed parser rejected its valid prose approach notes. Its interrupted traces remain in native-setup-error. The parser was corrected; the fresh Knodexa arm completed, then a fresh native arm was attempted. Its return phase timed out after writing code and passing agent-authored tests; independent return checks were not reached. The interrupted setup attempt was not a scored quality failure, and its usage is excluded from the paired workflow figures.

Unscored-arm usage and tool counts may be partial: a timed-out return has no completed-turn token usage. Do not compare its displayed token total with a completed arm or interpret the timeout as a demonstrated continuity error.
