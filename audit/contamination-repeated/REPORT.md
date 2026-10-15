# Repeated task-switch contamination evaluation

Predeclared plan: two balanced pairs, two intervening tasks per arm, same conversation throughout, nine independent handler-design checks. The order is randomized between native-first and Knodexa-first pairs. Model: gpt-6.1-sol, low effort; generated background memories disabled.

Native baseline: AGENTS.md plus modular, scoped memory files and the identical reference implementation. Knodexa: AGENTS.md plus recorded scoped agreements, automatic resume routing, and the identical reference. No arm is instructed to ignore its memory. The final request does not restate the hidden acceptance checks.

Intervening work requires inline normalization, then coercion/retries/exception fallbacks. Return work must preserve raw service delegation, result identity, exception identity, exactly one call, observer ordering, falsey observers, and delegation for malformed-looking inputs. The grader checks AST and injected-service runtime behavior independently of agent-written tests. The reference passes all checks; six intentionally broken implementations are rejected (grader-self-check.json).

| Pair | Arm | Design checks | Return tool items | Return seconds | Status |
|---|---|---:|---:|---:|---|
| 1 | knodexa | 9/9 | 5 | 40.51 | Completed |
| 1 | native | 9/9 | 3 | 35.93 | Completed |
| 2 | native | Unscored | 1 | 240 | Return timed out after 240 seconds; no completed implementation or quality score. Exact earlier phase timings lost on harness interruption; raw usage retained. |
| 2 | knodexa | Unscored | — | — | Establish stage timed out after 240 seconds; no return implementation or quality score. |

Aggregate results:

```json
{
  "native": {
    "planned": 2,
    "recorded": 2,
    "valid": 1,
    "all_checks_passed": 1,
    "passed_checks": 9,
    "total_checks": 9,
    "median_return_seconds": 35.93,
    "return_tool_items": 3,
    "invalid": 1,
    "workflow_seconds": 156.84,
    "workflow_input_tokens": 656862,
    "workflow_cached_input_tokens": 595712
  },
  "knodexa": {
    "planned": 2,
    "recorded": 2,
    "valid": 1,
    "all_checks_passed": 1,
    "passed_checks": 9,
    "total_checks": 9,
    "median_return_seconds": 40.51,
    "return_tool_items": 5,
    "invalid": 1,
    "workflow_seconds": 184.75,
    "workflow_input_tokens": 893825,
    "workflow_cached_input_tokens": 784000
  },
  "paired_outcomes": [
    {
      "pair": 1,
      "outcome": "tie"
    },
    {
      "pair": 2,
      "outcome": "invalid"
    }
  ],
  "quality_superiority_demonstrated": false
}
```

This sample does not establish general superiority. Two pairs cannot substantiate a reliable model-level advantage, even if both favored one arm. Invalid trials remain in the record. There is no forced compaction or saturated context-window claim; large cumulative token usage is not the same as active window occupancy. This is a controlled synthetic Python workflow, not an evaluation of all Codex desktop tools or optional background memories. Metadata capture is harness-prepared and excluded from workflow timing. Intermediate tests are agent-authored; the final design grader is independent. Different generated histories and server scheduling limit speed comparisons. Aggregate workflow totals cover completed trials only; when counts differ due to invalid trials, totals must not be compared across arms. Preserved approach outcomes, rather than reduced tool calls alone, are the primary metric.

Every raw session transcript, generated final implementation, returned Knodexa context, and metric is saved here. Re-run the harness to add evidence; do not select only runs that favor Knodexa.
