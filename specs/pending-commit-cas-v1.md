# Pending publication, commit CAS, and reconciliation v1

This specification materializes `A0-005` over the private single-node storage
installed by `A0-004`. It makes pending uploads durable without making them
readable, validates canonical authenticated `CommitDecision` documents,
performs all visibility changes in one first-writer transaction, and blocks
startup until reconciliation completes. `A0-006` now composes these primitives
with locally authenticated policy, revocation-state persistence, and
gateway/server routing without changing this lifecycle contract.

## Durable attempt lifecycle

`cache.sqlite` schema v2 transactionally upgrades schema v1 and adds
`cache_attempts`, `pending_objects`, and `quarantine_records`. `control.sqlite`
adds durable `reconciliation_runs`. Both files retain independent migration
chains with contiguous versions, exact statement checksums, exact
`sqlite_master` inventories, WAL, `synchronous=FULL`, foreign keys, and
integrity checks.

An attempt binds one request and attempt identity to tenant, repository, trust
domain, namespace generation, source state, policy/configuration/cache
contract digests, owner, lease, and an expiry no more than 24 hours away.
Reusing the start identity requires the exact fingerprint. A new pending
object increments the state version; an exact `(attemptId, key, digest, size)`
retry does not. Different bytes under the same pending identity conflict.

PUT streams through the A0-004 bounded content-addressed spool before inserting
pending metadata. The blob may therefore survive a failed metadata write, but
blob presence grants no authority. General reads consult committed metadata
first, so pending and aborted objects are misses. Abort atomically changes the
attempt to `ABORTED` and releases all pending rows; the reconciler later removes
the now-unreferenced immutable blobs.

## Canonical authenticated decision

The implementation accepts only exact JCS bytes for
`COMMIT_DECISION v1`. The decision digest is SHA-256 over the JCS document with
the top-level `decisionDigest` and `authentication.signature` members omitted.
The unpadded base64url Ed25519 signature contains exactly 64 bytes and signs
these unambiguous bytes:

```text
buildopt-cache-commit/v1 NUL keyId NUL sha256:<decision-digest>
```

Verification rejects duplicate or unknown JSON fields, a digest/signature
mismatch, an unknown key, a stale or future revocation epoch, an invalid
decision/validation/grant window, unsupported contracts, unsafe identifiers,
objects above 100 MiB, duplicate or unsorted objects, and any decision that
does not exactly cover the attempt's sorted pending set and immutable
bindings.

The public verifier requires the caller's exact current revocation epoch and
key set. A0-005 does not invent global credentials: only a context
authenticated by the A0-006 boundary can construct a bound HTTP data plane or
submit a verified decision.

## Atomic visibility and CAS

Before opening the visibility transaction, every pending digest and size is
fully verified. One `cache.sqlite` transaction then:

1. rechecks attempt state/version and exact object coverage;
2. proves every `(tenant, namespaceGeneration, key)` is unclaimed;
3. persists the canonical immutable decision;
4. inserts every committed visibility row;
5. changes the attempt to `COMMITTED`; and
6. deletes the pending rows.

The transaction commits once. A fault before that boundary exposes no object.
A competing attempt that finds any claimed identity becomes wholly `ABORTED`
with `CAS_LOST`; it cannot publish its other objects. An exact canonical
decision replay returns the original commit, while changed bytes for that
attempt produce an idempotency conflict.

Only after the cache transaction does the process idempotently add the decision
digest to `control.sqlite`. Failure there is reported as repair required but
does not revoke already-atomic cache authority.

## Bound HTTP data plane

`NewHTTPHandler` implements opaque Gradle-compatible `GET/PUT /cache/{key}` for
one immutable, preauthenticated binding:

- GET returns `200` only after the complete committed file passes size and
  SHA-256 verification; any absent, pending, aborted, corrupt, or unbound
  identity returns `404` without bytes.
- PUT returns `201` for a new durable pending object, `200` for an exact retry,
  and `413` before reading a known-oversized body.
- The handler never redirects and never parses or logs a credential.

The A0-005 handler itself remains context-bound and credential-agnostic.
A0-006 places it behind `buildopt-server` only after establishing local
credentials plus current policy and revocation state. Storage presence alone
still creates no route.

## Fail-closed reconciliation

Every storage open finishes reconciliation before reporting readiness:

- expired pending attempts abort before orphan collection;
- pending objects with missing/corrupt bytes abort their complete attempt;
- one missing or corrupt blob invalidates the whole committed decision and
  makes every object it authorized a miss;
- corrupt bytes move to the private quarantine directory, while missing bytes
  receive a durable quarantine record;
- blobs referenced by neither pending nor committed metadata are deleted;
- missing `control.sqlite` audit rows are derived only from still-durable
  immutable decisions; and
- a complete reconciliation report is recorded in `control.sqlite`.

Reconciliation never creates authority from a blob, audit row, attempt record,
or telemetry.

## Executable evidence

Run:

```bash
./dev/check-pending-commit
```

The checker composes the original ADR model with race-enabled real
filesystem/SQLite tests for pending invisibility, abort, complete coverage,
canonical Ed25519 authorization, two-object atomic commit, transaction
rollback, exact/conflicting replay, CAS loss, corruption/missing quarantine,
control-index repair, startup orphan collection, schema-v1 upgrade, bounded
HTTP PUT, fully verified HTTP GET, and exclusion between final blob publication
and pending metadata while reconciliation waits. It leaves A0 exit gates open
until the later authenticated end-to-end blocks compose these primitives.
