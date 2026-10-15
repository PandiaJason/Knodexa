# Reviewed reference reconciliation

The refactor capture gap is fixed by persisting a child-local reference binding.
The completed ancestor's agreement, status and Git baseline are retained.

Record input adds `approach.reference_reconciliations`:

```json
{
  "source_task": "work-prod",
  "source_version": 1,
  "original_path": "handlers/reference.go",
  "original_sha256": "<64-character original SHA-256>",
  "current_path": "handlers/transport.go",
  "current_sha256": "<64-character reviewed current SHA-256>",
  "reason": "Reviewed authorized refactor and verified equivalent behavior"
}
```

This object is an element of the reconciliation array. Supply the ordinary
summary/rules/references and run `record <child-id> --file=agreement.json
--verify='go test ./...'`. Recording checks both the original inherited agreement
version/hash and the actual current file hash. Source file checks follow the
reviewed path, retaining their expressions. Rules, skills, constraints and
acceptance criteria remain inherited. The binding persists in the child's YAML,
appears in restored context and exports, and applies to descendants.

Validation rejects unrelated source tasks, self-bindings, invalid paths, duplicate
bindings, missing reasons, stale versions/hashes and later current-file drift.
Reference reconciliation does not approve changed AST rules or other semantic
obligations; those checks can still fail after a function rename.

Evidence:

- `TestReplayRefactoredExtensionCapture` replays the actual retained refactor
  source and completed parent agreement from `hard-continuity-v1`. Independent
  behavior and SDK compatibility checks passed, the extension was indexed, and
  completed parent bytes stayed unchanged. No new model calls were made.
- `TestChildReconcilesRefactoredReferenceWithoutChangingParent` covers persistent
  restoration, inherited acceptance criteria, repeated recording, rejected wrong
  hashes, no partial writes and later drift detection.
- `TestReconciliationRejectsStaleOrUnrelatedBindings` covers source/version/path
  validation and verifies that inherited rules/check expressions are retained.
- `GOCACHE=/private/tmp/knodexa-go-cache ./scripts/check.sh` validates Go vet,
  race tests and standalone CLI smoke.

This fixes a specific continuity workflow failure. Prior agent comparisons remain
unchanged; general superiority over native scoped memory remains unproven.
