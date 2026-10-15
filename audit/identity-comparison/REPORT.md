# Paired real-agent work identity comparison

Outcome: **tie**.

Both arms use the same Go requirements, actual producer memory capture, unrelated image work, and a same-conversation return with no task ID or path. The native arm uses an index and modular JSON memory files with native file tools; Knodexa uses its request resolver and canonical task records.

| Arm | Common quality checks | Workflow seconds | Command/file items | Input tokens |
|---|---|---:|---:|---:|
| native | Passed | 174.27 | 13 | 376498 |
| knodexa | Passed | 167.82 | 20 | 717521 |

Common quality means independent returned behavior passes and original reference and completed memory are preserved. Native memory retrieval does not require a Knodexa tool call. Knodexa-specific routing/baseline checks are reported separately, not used to make the native quality criteria stricter.

One pair, unrandomized arm order, a controlled Go repository, one recorded phrase, different generated implementations and histories, no forced compaction, and generated background memories disabled. This does not establish general superiority over Codex or compare all desktop tools. Generated artifacts are independently graded even after a return-response timeout; timeouts without artifacts remain unscored. Response completion is a separate metric. Workflow timings include the three agent phases, exclude harness setup/grading, and depend on server scheduling. Tokens are cumulative tool-call usage, not active context-window occupancy. Do not treat a tie as a quality win.

Raw traces, final source snapshots, producer memories, and independent checks remain in each arm directory. Plan and aggregate metrics are in plan.json and results.json.

Response completion: native=true, Knodexa=true. Token usage may be incomplete for any timed-out response even when its artifact is graded.
