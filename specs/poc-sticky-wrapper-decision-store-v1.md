# Sticky-wrapper decision store v1

This is the `SWL-007` contract for the owner-operated sticky-wrapper learning
POC. It defines the control-plane records needed to remember what BuildOpt has
observed, what it tried, and which action (if any) may be selected later. It is
an experiment contract, not a production authorization service.

## Boundary

The store contains five immutable record families:

| Record | Purpose |
| --- | --- |
| `STICKY_OBSERVATION` | One ordinary requested build and its bounded evidence |
| `STICKY_ACTION` | One qualification or rollout state transition |
| `STICKY_TRIAL` | An isolated candidate/control comparison |
| `STICKY_DECISION` | A signed, expiring selection for one exact binding |
| `STICKY_ECONOMIC_LEDGER` | Signed wall-time saving and BuildOpt cost entries |

`STICKY_STATE_HEAD` is the single mutable pointer to the newest immutable
record. `STICKY_REVOCATION` is the local revocation input. Every record is
canonical RFC 8785 JSON and is addressed by the SHA-256 of its exact bytes.
Unknown fields, versions, scopes, timestamps, and state transitions are
rejected.

Gradle cache objects are deliberately outside this protocol. A cache key, hit,
blob digest, or cache manifest can never authorize a BuildOpt action. The
central adapter stores sticky records as typed `EVIDENCE` artifacts in the
existing `state.sqlite` lifecycle, while Gradle objects remain in the existing
cache/blob lifecycle.

## Binding and authority

Every record binds the repository scope, workflow, source revision, Gradle
version, Wrapper digest, invocation options, required-output contract,
BuildOpt executable digest, and revocation epoch. A decision additionally binds
an action generation, policy digest, cache-contract digest, evidence digests,
an issue/expiry interval, and an Ed25519 signature. The signature covers the
canonical unsigned decision digest under the domain
`buildopt-sticky-decision/v1`.

`ACTIVE_RUNTIME_PROFILE` and `ACTIVE_DURABLE_PATCH` are valid only for a
`QUARANTINE_VALIDATED` action in `ACTIVE_IN_CI` or `ACTIVE_LOCALLY`, with
evidence and a non-expired signature. `NATIVE_NOOP` has no action generation.
Expired or revoked decisions are never selected; the caller must use native
Gradle. This POC does not grant production authority and does not perform
automatic merges or customer-branch mutation.

## State machines

Qualification and rollout are independent, as required by the RFC:

```text
UNKNOWN -> OBSERVING -> CONTRACT_QUALIFIED -> QUARANTINE_VALIDATED
   \______________________________> REJECTED
Any non-terminal state -> SUSPENDED

PROPOSED -> SHADOW -> CI_CANARY -> ACTIVE_IN_CI -> ACTIVE_LOCALLY
Any state -> SUSPENDED; ACTIVE_* -> ROLLED_BACK -> RETIRED
```

The executable transitions are `PROPOSE`, `BEGIN_SHADOW`,
`BEGIN_CI_CANARY`, `ACTIVATE_IN_CI`, `ACTIVATE_LOCALLY`, `SUSPEND`,
`ROLLBACK`, and `RETIRE`. An `INCONCLUSIVE` trial can be recorded but can
never be evidence for activation. Every action ID has a contiguous sequence.

## Persistence semantics

Both adapters implement the same operation:

1. decode and validate canonical bytes and the exact repository scope;
2. verify decision signature, expiry, and revocation when a key registry is
   available;
3. require the next generation and current head digest;
4. publish the immutable record before moving the head;
5. commit the new head using an idempotency key and compare-and-swap; and
6. return the original result for an exact replay, or a conflict for a changed
   request using the same idempotency key.

The filesystem adapter keeps `records/`, `requests/`, `head.json`,
`revocation.json`, and a separate `cache/` directory below one private root.
The central adapter uses `state.sqlite`'s immutable object/manifest/CAS APIs;
its `EVIDENCE` namespace is not a Gradle cache namespace. A failed CAS may leave
an unreachable immutable object, but it cannot advance the head or authorize
execution.

## POC limitations

The central adapter's revocation epoch is process-local until the existing
owner-operated authority endpoint is wired into the sticky wrapper. There is no
distributed transaction between `cache.sqlite` and `state.sqlite`, no HA,
multi-tenancy, KMS/HSM, billing, or SLO claim. These are deliberately deferred
until the POC demonstrates value. The next block (`SWL-008`) consumes only an
exact local snapshot and must preserve a non-blocking native no-op path.

## Conformance

The JSON Schema 2020-12 union is
[`contracts/jsonschema/sticky-wrapper-decision-store.v1.schema.json`](../contracts/jsonschema/sticky-wrapper-decision-store.v1.schema.json).
The implementation and lifecycle vectors live in
[`internal/stickydecision`](../internal/stickydecision). Run:

```bash
./dev/check-sticky-wrapper-decision-store
```

The checker runs canonical/JCS and Ed25519 tests, every valid transition,
replay and conflict behavior, stale generations, expiration, revocation,
corruption, cross-scope and cross-plane negatives, and both local and central
adapters. Schema fixtures are validated independently with the pinned
Draft-2020-12 validator.
