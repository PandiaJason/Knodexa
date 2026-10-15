# Live task-switch and fresh-agent handoff pilot

2026-10-13. Actual Codex CLI sessions, gpt-6.1-sol, low reasoning effort, one
three-stage run per arm. Controlled synthetic repository and real model-generated
implementations. Generated background memories were disabled.

| Fresh-agent handoff | Native file memory | Knodexa compact resume |
|---|---:|---:|
| Independent acceptance cases | 20/20 | 20/20 |
| Wall time | 44.44 seconds | 31.16 seconds |
| Reported input tokens | 89,938 | 67,539 |
| Cached input tokens | 75,776 | 52,352 |
| Uncached input tokens | 14,162 | 15,187 |
| Output tokens | 973 | 544 |
| Completed command/file tool items | 7 | 4 |

The Knodexa handoff was 29.9% faster and used 24.9% fewer reported input tokens.
It used more uncached input tokens; this is not a demonstrated price saving.
Both implementations satisfied the same independent behavior checks.

## Full workflow, including producing memory and switching tasks

| Entire three-stage run | Native | Knodexa |
|---|---:|---:|
| Wall time | 138.78 seconds | 145.66 seconds |
| Input tokens | 270,354 | 286,689 |
| Cached input tokens | 184,064 | 195,968 |
| Output tokens | 5,034 | 6,486 |

There was no end-to-end workflow win: Knodexa used 6.0% more input tokens and
4.96% more wall time across all three stages. The measured benefit is confined
to the consumer handoff in this pilot.

## Protocol and evidence

1. A producer agent implemented strict expiry validation, ran tests, and emitted
   structured decisions, constraints, acceptance notes, a prior failure, and a
   reusable bool/int lesson. An independent grader confirmed 20/20 cases.
2. The same session switched to an unrelated label-normalization coding task.
   Its implementation passed three independent checks. Both corpora contained
   equivalent unrelated advice for 30 image tasks.
3. The expiry implementation was reset to the same defective code in each arm,
   and source-agent regression tests were withheld. A different fresh session
   returned as sibling task A.2, recovered memory, checked Git drift, and repaired
   the regression. The same 20 hidden edge cases graded each implementation.

Native memory was the producer's structured report in PROJECT_MEMORY.json.
Knodexa stored that producer's report as tasks, a decision, a failure, a lesson,
and evidence records. Fixture serialization was performed by the harness in
both arms; its setup time is not included in model-session wall times. Knowledge
extraction itself is included through the producer turn. Producer outputs and
code edits differed naturally between arms, so this is not a paired replay of
an identical generated memory record.

Raw logs confirm that the Knodexa consumer invoked `resume A.2 --json` once.
The context retained the saved baseline, reported actual HEAD drift, included
the prior sibling-task failure and source A.1 lesson, and preserved acceptance
criteria. Protected memory records were unchanged in both arms; Knodexa could
update only its own resumed task state. Results and usage are in results.json;
raw JSONL, final code, producer memory, and the handoff package are retained.

This is one run per arm, with no randomized ordering, repetitions, statistical
confidence intervals, or equalized cache warmth. It tests file-based native
continuity and a fresh-agent handoff, not enabled Codex background memories,
desktop thread-reading tools, or every internal Codex feature. It cannot justify
a general superiority claim. See ../continuity_eval.py for the full fixture,
execution, and independent grading protocol.
