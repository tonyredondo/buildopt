# Shared commit and recovery v1

This contract closes `A0-G05` by running the accepted Phase 0 atomicity model
against the real single-node Shared filesystem plus `cache.sqlite` and
`control.sqlite` WAL databases.

## Visibility transaction

`cache.sqlite` is the sole visibility authority. One serializable transaction
persists the immutable canonical `CommitDecision`, every committed object,
the attempt's `COMMITTED` state, and release of all pending rows. A fault
immediately before its commit rolls back all four effects. The transaction
never consults or writes `control.sqlite`.

Two attempts start their commit calls concurrently for the same tenant,
namespace, and key. Exactly one publishes its complete two-object set. The
other atomically becomes `ABORTED/CAS_LOST`; neither its colliding object nor
its distinct object becomes visible. Reconciliation deletes both losing
content-addressed blobs.

## Faults and startup recovery

The executable production matrix proves:

- a three-object pre-commit fault persists no decision or committed row and
  leaves the complete pending attempt retryable;
- failure of the later `control.sqlite` index write does not revoke the
  complete cache commit, returns `requiresReconcile`, and startup repairs the
  missing audit row from the immutable decision digest;
- a durable blob with no pending or committed metadata is deleted and remains
  a miss;
- a committed record whose blob is missing invalidates the complete decision,
  records quarantine evidence, and makes every object in that decision a
  miss; and
- an expired lease becomes `ABORTED/LEASE_EXPIRED`, releases pending metadata,
  deletes the orphaned blob, and never becomes a hit.

The existing happy commit, exact replay, conflicting replay, incomplete,
expired and revoked decision, corruption, and crash-boundary cases remain
cross-checked against all 14 scenarios in
[`commit-atomicity-v1.json`](./commit-atomicity-v1.json).

Run:

```bash
./dev/check-shared-commit-recovery
```

This gate does not prove the overhead budget (`A0-G06`), root/composite Test
isolation without a `TestCacheGrant` (`A0-G08`), quota/TTL/SLRU
(`A1-G03`), or the wider fail-closed beta restart profile (`A1-G04`).
`MANAGED_SHARED_CACHE` remains unavailable until the outstanding A0 safety
gate is closed.
