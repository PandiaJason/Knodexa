# Retained-artifact grading correction

| Case | Arm | Scored | Behavior + lesson | Completed facts | Overall pass | Seconds | Actions |
|---|---|---|---|---|---|---:|---:|
| fresh-handoff | knodexa | true | true | true | true | 58.60 | 5 |
| fresh-handoff | native | true | true | true | true | 88.96 | 8 |
| recorded-failure | knodexa | true | true | true | true | 69.71 | 9 |
| recorded-failure | native | true | true | true | true | 66.25 | 9 |
| refactor | knodexa | true | true | true | true | 93.88 | 12 |
| refactor | native | true | true | true | true | 71.71 | 8 |
| similar-tasks | knodexa | true | true | true | true | 68.11 | 15 |
| similar-tasks | native | true | true | true | true | 80.45 | 10 |

The initial hidden grader reused visible fixture helper names, causing compilation collisions. Corrected helper names are validated against the original visible tests and deliberately broken implementations. No model calls were repeated. Original scores/logs remain beside corrected results. Completed-memory preservation checks every original value (including status, baseline and agreement) rather than byte-identical serialization; append-only extension records are allowed equally in both arms.

One fixed-source producer and one fresh return per arm/case, same model, equivalent knowledge, alternating case order. The transferred compatibility requirement appears in shared memory, not the initial handler. Observer panic tests verify the recorded failure. Refactor changes source path/name/hash and requires an explicit stale-reference report. Setup/grading time is excluded from phase timings. This is controlled evidence, not general superiority.

All four pairs tied on the stated behavioral/preservation criteria, with no model timeouts. Knodexa was faster on fresh handoff and similar-task selection, slower on recorded failure and refactor. Total return-phase command/file actions: Knodexa 41, native 35. These totals exclude producer capture.

Additional workflow limitation: the Knodexa refactor agent's `record webhook-extension` command was rejected because its inherited reference no longer existed. It retained an unindexed agreement JSON for review. This is not covered by the primary functional pass criteria and must not be hidden by the quality tie; safely recording reconciled extension agreements remains unresolved.
