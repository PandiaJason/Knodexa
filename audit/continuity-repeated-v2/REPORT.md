# Repeated real-agent continuity comparison

| Pair | Independent quality outcome |
|---|---|
| 1 | tie |
| 2 | tie |
| 3 | tie |

Three fresh pairs with alternating order, identical requirements and independent behavior grading. Raw traces, tool counts, timing, token usage and memory snapshots are retained per pair. The same model runs both arms. This tests the current prepare/source-capture workflow, not automatic command hooks or all Codex desktop tools. No forced context compaction. One controlled scenario and three pairs cannot establish general superiority. Timeouts and setup failures remain visible; artifacts are graded when available.


| Pair | Arm | Quality passed | Workflow seconds | Command/file actions | Input tokens |
|---|---|---|---:|---:|---:|
| 1 | native | true | 146.66 | 12 | 464230 |
| 1 | knodexa | true | 155.69 | 20 | 642325 |
| 2 | native | true | 152.53 | 9 | 463926 |
| 2 | knodexa | true | 145.10 | 35 | 771610 |
| 3 | native | true | 130.60 | 11 | 368656 |
| 3 | knodexa | true | 100.93 | 16 | 571420 |

| Arm | Passed trials | Mean workflow seconds | Total command/file actions | Total input tokens | Cached input tokens |
|---|---:|---:|---:|---:|---:|
| native | 3/3 | 143.26 | 32 | 1296812 | 1144576 |
| knodexa | 3/3 | 133.91 | 71 | 1985355 | 1805568 |

Result: all three pairs tied on quality. Knodexa used more command/file actions and cumulative input tokens. Timing varied between pairs; the sample cannot establish a reliable speed advantage. Token totals include cached input and are not billing cost or active context-window size. All 18 agent phases completed. The initial sandbox-blocked attempt made no model progress and remains in the adjacent sandbox-blocked directory.
