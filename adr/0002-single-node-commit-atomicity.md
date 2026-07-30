# ADR 0002: Single-node commit atomicity

- Status: accepted
- Date: 2026-07-29
- Items: `F0-031`, `CACHE-008`, `STORAGE-001`

## Context

Pending Gradle cache uploads must never become general hits before a complete,
current, authenticated `CommitDecision`. The private-beta implementation uses
content-addressed files plus separate WAL-mode `cache.sqlite` and
`control.sqlite` databases. A distributed transaction between those stores
would create an availability dependency without improving cache visibility.

The safety property is narrower and stronger: the immutable decision and every
object visibility record it authorizes must become durable together in the
same cache-metadata transaction. The control ledger is an audit/index consumer
of that already-durable decision, not a second commit authority.

## Decision

An authorized commit follows this order:

1. stream every pending object to a private spool while calculating SHA-256;
2. enforce exact size/checksum and make each immutable blob durable at its
   content-addressed path;
3. validate one canonical `CommitDecision` against attempt, source, policy,
   grants, epochs, verdict, expiration, and the complete sorted object set;
4. open one `cache.sqlite` transaction;
5. persist the verified decision by digest, perform first-writer-wins CAS for
   every `(tenant, namespaceGeneration, key)`, and insert all `COMMITTED`
   visibility records;
6. commit that transaction once; only then may a general GET return a hit;
7. idempotently index the decision digest in `control.sqlite`.

Steps 4–6 are one atomic unit. A constraint failure or crash before the commit
rolls back the decision and all visibility records. There is no partial
visibility and no query to `control.sqlite` inside this transaction.

Exact replay of the same decision digest returns the existing committed state.
Reusing its identity with changed canonical bytes is an idempotency conflict.
A competing attempt that loses any key CAS commits nothing for that attempt.
Replacing poisoned data requires another namespace generation.

## Recovery

- A crash before blob durability leaves no authoritative record.
- Durable blobs without a committed decision are orphans and remain misses
  until the reconciler deletes them.
- A decision or visibility record whose blob is absent, truncated, wrong-sized,
  or digest-mismatched is quarantined and served as a miss.
- A crash after the cache transaction but before the control-ledger write does
  not revoke the objects. The reconciler derives the missing audit index from
  the immutable decision digest.
- An expired lease or invalid/revoked/incomplete decision aborts pending and
  cannot open the visibility transaction.
- Startup readiness stays false until blob/metadata reconciliation and
  revocation/tombstone loading finish.

If `control.sqlite`, the deployment key, or monotonic epochs are unrecoverable,
actions remain disabled and policy, namespace, and L1 security generations
rotate. Authorization is never reconstructed from blob presence or telemetry.

## Rejected alternatives

- A distributed transaction across both SQLite files: it makes an audit-index
  failure affect already-safe cache visibility and couples their lifecycles.
- Writing `COMMITTED` rows before persisting the decision: readers could
  observe authority that cannot be audited or replayed.
- One transaction per object: a multi-object attempt could become partly
  visible.
- Copying pending objects into stable aliases after commit: interruption would
  create partial visibility; immutable digest blobs plus metadata CAS avoid
  that copy boundary.
- Reconstructing decisions from blobs after metadata loss: bytes do not prove
  policy, grant, epoch, validation, or object-set completeness.

## Consequences

`cache.sqlite` is the sole single-node visibility authority. `control.sqlite`
may lag and be repaired without changing object validity. Blob durability may
precede authorization, so orphan collection is required and must never infer
authorization.

The concrete schema/table names remain private implementation details. Any
replacement backend must preserve the same all-or-nothing decision/object-set
transaction, CAS, replay, quarantine, and reconciliation observations.

## Validation

[`specs/commit-atomicity-v1.json`](../specs/commit-atomicity-v1.json) defines
14 fault and replay cases. Run:

```bash
./dev/check-commit-atomicity
```

The executable model proves happy commit, exact replay, changed-payload
conflict, incomplete/expired/revoked decisions, crashes at each durability
boundary, missing/corrupt blobs, first-writer CAS loss, and a failed
`control.sqlite` write repaired by digest. This is the Phase 0 transaction test
plan; the A0 storage implementation must run equivalent cases against the real
WAL databases and filesystem before closing `A0-G05`.
