# Shared-task concurrency fix

The original failure artifacts in `host-and-concurrency-v1` are unchanged.
This run uses the same eight-process scenarios, plus an atomic shared-task update
case. No new model calls were required.

| Case | Before | After |
|---|---|---|
| Public CLI alias updates | 4 successful commands; 3 updates retained; 4 SQLITE_BUSY failures | 8 successful commands; all 8 updates retained; no errors |
| Full-task writes from the same old snapshot | All 8 reported success; only 1 update survived | 1 saved; 7 explicitly rejected as stale; no successful write lost |
| Atomic shared-task updates | Not available | All 8 saved and retained |
| Distinct-task canonical writes | All 8 saved and retained | All 8 saved and retained |

Completion state, saved baseline and established agreement were preserved in all
cases. [Exact outcomes and worker logs](concurrency/results.json) include the seven
expected stale-write errors. These errors are not hidden or counted as successful
updates. The snapshot safety criterion now distinguishes explicit conflict
rejection from silent loss; the original criterion required all eight updates to
survive. Alias/atomic-update cases still require eight successes and eight retained
updates. This does not mean every conflicting full-record replacement can succeed.

## Implementation

- OS-backed process locks serialize cooperating task writers, SQLite initialization
  and index synchronization. Waits are bounded to five seconds. Lock ownership is
  released by the OS on process exit; persistent lock files must not be deleted.
- `ReadTask`, `ListTasks` and indexed `GetTaskData` supply a nonserialized snapshot
  fingerprint. `SaveTask` compares it against current canonical bytes while locked;
  stale or blind replacement returns `storage.ErrConflict` before writing. Callers
  must reload and review/reapply their intended change. Fingerprints do not modify
  task YAML or public JSON.
- `UpdateTask` applies mergeable changes to fresh canonical task data while locked.
  Alias additions use it, preserving intervening aliases and other metadata.
  No-op callbacks and failed callbacks do not rewrite canonical memory. Callbacks
  must not invoke other mutation operations.
- SQLite configures busy handling before WAL, and cooperating processes serialize
  schema initialization and cache transactions. Verification commands execute
  outside task locks; agreement recording rejects a stale task snapshot at save.

## Verification

The opt-in audit was rerun into a separate directory:

```sh
GOCACHE=/private/tmp/knodexa-go-cache KNODEXA_SHARED_MEMORY_AUDIT=1 \
  KNODEXA_SHARED_MEMORY_AUDIT_OUT=audit/host-and-concurrency-v2/concurrency \
  go test ./internal/experiment -run '^TestSharedMemoryAudit$' -count=1 -v
```

Full Go vet, race tests and standalone CLI smoke passed using `scripts/check.sh`.
Additional storage race tests verify stale-write rejection, independent Storage
instances merging changes, no partial writes, bounded contention, lock release
after a callback failure and lock reacquisition after killing the holder process.

Runtime verification is on this macOS host. The Windows lock implementation is
cross-compiled separately, not runtime-certified here. Locks coordinate clients
using these APIs; direct external file edits and concurrent mutations of decisions,
lessons or project manifests are not covered by the task fingerprint protocol.
This fixes the demonstrated shared-task loss and CLI contention, not every possible
multi-agent race or comparative memory-quality limitation.
