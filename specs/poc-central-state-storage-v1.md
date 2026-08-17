# Restart-safe typed central state storage

## Outcome

The POC now persists BuildOpt portfolios, evidence and resumable checkpoints
on the same private content-addressed files already used by Shared Cache. A
separate `state.sqlite` database owns all typed visibility, generations,
references and retention. Gradle cache metadata remains in `cache.sqlite` and
cannot authorize a BuildOpt state read.

This is a local storage implementation, not yet a network service.

## Durable layout

```text
<state-root>/
  blobs/sha256/<first-two-hex>/<remaining-hex>
  cache.sqlite
  control.sqlite
  state.sqlite
  writer.lock
```

Identical bytes may occupy one physical CAS file. They still need an explicit
`state_objects` row for the exact repository scope and state kind before a
manifest may reference them. A cache entry, another repository or another
state kind therefore cannot cross the logical visibility boundary.

`state.sqlite` uses the existing CGO-free SQLite driver with WAL,
`synchronous=FULL`, foreign keys, a five-second busy timeout, exact migration
checksums and private-file validation.

## Publication and reads

Publication follows the contract order:

1. stream and verify every namespaced object into the physical CAS;
2. canonicalize and validate the immutable manifest;
3. prove every artifact row, complete artifact byte stream and evidence
   reference still matches; and
4. advance the sole repository/kind head by exactly one generation in the
   same SQLite transaction that records idempotency.

Objects and complete manifests without step four remain invisible. Two
concurrent writers starting from the same head produce one winner and one
precondition failure. Retrying the exact request returns the original head;
reusing its idempotency key with another request is rejected.

Every current-state read recalculates the canonical head and manifest SHA-256,
checks their bindings, walks repository/kind-scoped metadata and reads every
artifact completely through the verified blob boundary. Corrupt or missing
state is rejected; the caller must retain optimized native Gradle.

## Restart and retention

Opening the existing storage root validates all three databases and performs
typed-state retention before orphan-blob reconciliation. The reconciler now
counts both Gradle and BuildOpt metadata, so cleanup in either plane cannot
delete bytes still owned by the other.

- Unpublished objects and manifests expire after 24 hours.
- The current portfolio remains; superseded portfolios remain for 30 days.
- Evidence remains while any retained portfolio references it, then for 30
  more days.
- Checkpoints expire exactly 24 hours after creation.

These rules are independent from Gradle cache SLRU and quotas.

## Executable evidence

Run:

```bash
./dev/check-central-state-storage
```

The checker first revalidates the three central-state schemas and lifecycle
vectors, then runs the concrete storage suite with Go's race detector. It
proves restart, repository/kind isolation, exact replay, concurrent CAS,
invisible partial publication, staged cleanup, artifact and manifest
corruption rejection, skipped generation rejection, referenced-evidence
retention and checkpoint expiry.

## POC boundary

This storage block itself created no listener, token, client connection, remote
selection or performance claim. The subsequent
[`POC-CENTRAL-HTTPS-AUTH-001`](./poc-central-https-auth-v1.md) block now exposes
the same boundaries through scoped TLS without changing their persistence
semantics. Local `buildopt optimize` remains independent of this storage. State
remains untrusted input that requires exact local revalidation,
`productionAuthorized=false`, soak and design partners are not required, and
Test Optimization remains out of scope.
