# Continuity implementation

Five sequential changes are implemented in the Go core:

1. `identity capture <id> --files=...` records a stable work ID, source hashes,
   Go functions/types and imports. Resolution can use source symbols without
   repeating the old title. Git rename evidence and unchanged content relocate
   sources; competing work remains ambiguous. Source parsing uses a hash-validated
   SQLite cache. Import matches are advisory and do not establish identity alone.
2. `run` prepares context before invoking a compatible command and captures source
   identity only after explicit verification and approach checks pass. Commands
   receive a JSON file through `KNODEXA_CONTEXT_FILE`. This is opt-in integration;
   arbitrary client edit tools are not intercepted. Failed verification leaves
   code changes available for inspection and does not advance memory.
3. Go AST checks support `require-call`, `forbid-call`, `forbid-import` and
   `require-function`. Comments cannot satisfy required calls; import aliases
   normalize to import paths. These are syntactic checks, not type resolution,
   control-flow proof or a guarantee of semantic design quality.
4. Explicit approach replacement increments its version and archives the previous
   agreement. `approach history <id>` exposes both. Changed/missing references and
   opposing explicit checks require review. Free-form rule conflicts require
   review by a person or agent. Proven file relocation does not automatically
   authorize rewriting a versioned agreement's reference paths.
5. `prepare` / `knodexa_prepare_work` combine identity, checkpoint comparison,
   constraints, acceptance criteria, agreement checks and bounded reference
   excerpts. Mandatory agreements remain available even when they exceed a tiny
   budget; the budget is a text estimate, not an absolute JSON token limit.

Example (substitute actual project paths and the agent command):

```sh
knodexa identity capture webhook --files=handlers/webhook.go
knodexa approach set webhook --summary='Delegate to the existing handler' \
  --rule='Preserve service injection' --scope=handlers/ \
  --reference=handlers/webhook.go \
  --ast='handlers/adapter.go::require-call::HandleWebhook' \
  --reason='Record agreed adapter architecture'
knodexa prepare 'Extend HandleWebhook' --json
knodexa run 'Extend HandleWebhook' --verify='go test ./...' -- agent-command
knodexa approach history webhook
```

Evidence: `GOCACHE=/private/tmp/knodexa-go-cache ./scripts/check.sh` runs Go vet,
race tests and a fresh-binary standalone smoke test. Focused tests cover actual Git
renames, symbol collisions, subprocess context delivery, successful and failed
verification, stale agreements, AST false positives and version preservation.

Implementation behavior is validated. Three fresh live comparison pairs using
the current prepare/source-capture workflow all tied against Codex with scoped
file memory. See [repeated comparison](continuity-repeated-v2/REPORT.md).
This controlled scenario does not establish general superiority or test forced
compaction, cross-agent transfer, or command-hook integration.
