# Live approach-continuity pilot

Both live Codex agents preserved the established implementation approach. Neither won on quality.

| Return-session outcome | Codex with explicit file memory | Codex with Knodexa |
|---|---:|---:|
| Independent service behavior checks | Pass | Pass |
| AST delegation / no embedded transformation | Pass | Pass |
| Protected reference and memory preserved | Yes | Yes |
| Completed command/file tool items | 5 | 10 |
| Return time, seconds | 38.59 | 33.52 |
| Input tokens | 81,302 | 81,570 |
| Cached input tokens | 66,432 | 72,704 |
| Output tokens | 1,084 | 774 |

Both generated the same implementation: `return service.transform(value)`.

Method: one real producer established a reference implementation, independently graded before reuse. Each arm received identical source and agreements. Each performed unrelated label work, then a separate fresh Codex session implemented a new handler. Native memory used AGENTS.md plus ENGINEERING_MEMORY.md; Knodexa used an explicitly reused approach and resume. Both used gpt-6.1-sol, low effort, CLI tools; generated background memories were disabled. Knodexa record setup was harness-generated and excluded from timing.

The grader checks executable AST and runtime substitution with arbitrary service results, call counts, and exception propagation. It is independent of Knodexa literal checks. This is a small synthetic design contract, not a broad semantic quality benchmark.

Limitations: one pair, fixed native-first order, no native background-memory arm or desktop-tools arm, independently started switch sessions rather than long shared conversations, and no complex implementation or repeated task switches. Generated Python cache files accidentally entered the fixture Git baseline, causing cleanup in both arms; timing is descriptive only. The harness now excludes these caches for future runs; these raw results describe the original run and were not rerun. Evidence supports functioning approach transfer, not superiority over Codex.

Raw session logs, final implementations and metrics are saved beside this report.
