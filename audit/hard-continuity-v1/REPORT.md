# Four harder real-agent continuity cases

**This initial grading contained fixture/helper-name collisions. Use
[the corrected artifact report](CORRECTED-REPORT.md). Original scores and logs
remain retained; no model calls were repeated for the correction.**

| Case | Arm | Scored | Behavior + lesson | Completed memory | Overall pass | Seconds | Actions |
|---|---|---|---|---|---|---:|---:|
| fresh-handoff | native | true | false | false | false | 88.96 | 8 |
| fresh-handoff | knodexa | true | false | true | false | 58.60 | 5 |
| similar-tasks | native | true | false | false | false | 80.45 | 10 |
| similar-tasks | knodexa | true | false | true | false | 68.11 | 15 |
| recorded-failure | native | true | false | false | false | 66.25 | 9 |
| recorded-failure | knodexa | true | false | true | false | 69.71 | 9 |
| refactor | native | true | false | true | false | 71.71 | 8 |
| refactor | knodexa | true | false | true | false | 93.88 | 12 |

One pair per case, same model, alternating order, fresh return contexts and equivalent canonical facts. Producers captured independently verified fixed Go source rather than authoring it. Both arms receive the same cross-task compatibility lesson and observer failure. Refactor changes the reference path, function name and source hash; the original agreement is deliberately retained as history and the agent must flag it. Agent source tests do not determine scoring: independent Go behavior and compatibility tests run after return. Raw traces and before/after memory remain per case.

Scored failures are retained; setup failures and interrupted returns without artifacts are unscored. This controlled suite cannot establish general superiority or forced-compaction behavior. Timings are model phases excluding setup/grading. One observed advantage needs replication on other repositories.
