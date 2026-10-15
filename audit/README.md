# Evaluation evidence

This directory preserves reports, metrics, and captured source from audits and
model comparisons. Raw agent session traces are retained locally and excluded
from Git publication; public results include plans, scores, independent grading
logs, source snapshots and memory snapshots. The obsolete Python experiment
runners were removed as part of the Go-only cleanup. Captured implementations
are text snapshots (`.py.txt`), not executable validation scripts.

The current validation workflow is `./scripts/check.sh`, backed by Go tests.
Request identity regressions are in `internal/identity`, `internal/continuity`,
`internal/cli`, and `internal/mcp`. Historical comparisons do not prove the new
request resolver improves model behavior: they supplied task identity upfront.

Current evidence:

- [Current-release paired comparison](release-evaluation-v1/REPORT.md).
- [Real host adoption and original concurrency failures](host-and-concurrency-v1/REPORT.md).
- [Eight-process concurrency fix](host-and-concurrency-v2/REPORT.md).
- [Corrected harder-continuity comparison](hard-continuity-v1/CORRECTED-REPORT.md).

Reports with repaired grading clearly retain and distinguish original scores.
