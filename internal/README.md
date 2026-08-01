# Internal Go packages

Private implementation shared by `buildopt` and `buildopt-server`.

`launcher/` contains the dependency-free `WS-001` command passthrough, the
`WS-002` Linux process-group and signal contract, the `WS-003` plugin handshake,
and the neutral `WS-004` authenticated local rendezvous used by `cmd/buildopt`.
It forwards `SIGINT`/`SIGTERM` to the child group, preserves child status, owns
the private event socket and loopback readiness gateway, and consumes the
`F0-039` local bypass before creating either service or parsing server
configuration. The bypass uses the same process/signal contract and removes all
reserved launcher state from the child. Grace-period escalation remains with
the invoking CI environment. `A0-001` adds the opt-in managed runner-slot path:
a current-user private
state root, exclusive invocation and gateway leases, a detached idle-bounded
process, UID-authenticated invocation registration, context-gated readiness,
restart-stable identity, and complete rotation when the endpoint cannot be
recovered. `A0-003` adds the launcher-owned native L1 lifecycle: opaque
tenant/repository/trust/compatibility scoping, generation-segmented private
directories, an exclusive child-lifetime lease, and local-cache disablement
for pending L2 writers. `A0-006` authenticates canonical local policy and
cumulative revocation state before Gradle, persists anti-rollback generations,
derives L1 authority from the signed state, and gives the gateway an
invocation-only Shared credential over its same-UID control channel. The
gateway translates Gradle's local Basic credential, rejects redirects, and
routes no cache request without current context. A1-004 adds a shared
deployment-lifecycle lease and rejects L1 generations below a completed
deletion tombstone.

`sessioningest/` contains the provisional `WS-005` gateway-to-server record,
strict authenticated HTTP transport, and concurrency-safe in-memory acceptance
store. Its optional `WS-006` handoff carries only predeclared tokenized context
and facts from an authenticated Gradle invocation.

`buildsession/` is the dependency-free producer for the normative
`BUILD_SESSION v1` schema and the atomic local-file exporter. It derives only
deterministic manifest/baseline digests, declares unobserved metrics
unavailable, publishes mode-`0600` immutable complete/partial JSON, and owns a
bounded private JSONL stream with deterministic at-least-once replay and
startup recovery. A1-004 applies keyed repository/trust/task redaction before
either durable form and enforces explicit bounded profile authorization.
Runtime schema conformance remains with the isolated
validator under `dev/schema-validator/`.

`datalifecycle/` owns the isolated private-beta profile policy, HMAC
tokenization, shared lifecycle leases, durable logical revocation, coordinated
known-component removal, tokenized tombstones, and post-deletion generation
floors consumed by Shared, managed L1, and export.

`localauthority/` owns the A0-006 canonical JCS/Ed25519 authority, pinned
trust-root and private-file boundary, strict semantic validation, and durable
monotonic policy/revocation state used independently by launcher and Shared.

`sharedcache/` owns the A0-004..A0-006 single-node storage and publication
boundary used by `buildopt-server`: private same-filesystem SHA-256 blobs, a
process-lifetime writer lease, independently migrated WAL-mode
`cache.sqlite`/`control.sqlite`, durable pending attempts, canonical Ed25519
decisions, atomic first-writer visibility, context-bound opaque HTTP GET/PUT,
quarantine, startup reconciliation, and current local-authority records. It
verifies complete bytes before returning a hit, persists no raw data-plane
credential, rejects stale/rolled-back authority, and never derives authority
from blob presence.

`edgecache/` owns the optional MVP-C2 boundary. C2-001 adds the strict private
single-node configuration, loopback listener and authenticated Shared-origin
rules, bounded local-storage declaration, and immutable Shared-only commit and
collision authority. C2-002 adds authenticated Shared-only committed read-
through, complete content-address verification before SQLite publication, and
per-read exact current-revocation authorization across offline restart. Later
C2 blocks own byte SLRU, pending replication, and the executable two-node proxy
proof; no Edge server route exists yet.

`neutralenvelope/` owns the strict `WS-009` observation and report contract. It
pairs externally timed native and optimization-off wrapper executions,
reconciles required-output digests, retains signed differences, and binds the
runner, metric catalog, envelope, launcher, server, and plugin inputs.

`buildimpact/` owns the C3 conservative graph-omission boundary. C3-001 loads
only bounded, strict, repository-contained customer manifests bound to one
repository and pipeline class, digests their canonical form, and rejects
inferred entrypoints, ambiguous ownership, unsafe paths, symlinks, and any
unknown-change policy other than `FULL_GRAPH`. C3-002 binds a complete
Gradle-declared graph to that digest, maps source changes through transitive
reverse dependents, and predicts only customer-enumerated alternatives that
cover every affected project, artifact, and Build-owned check. Test-owned work
is preserved and execution remains on the original entrypoints. C3-003 records
strict manifest/graph/adapter-bound full-baseline and paired-control
observations, classifies candidate build/content/check/project divergence as a
false negative, and keeps infrastructure or invalid baselines inconclusive.
C3-004 aggregates only current-binding results through the unchanged BIA-002
minimum window, sample, coverage, per-stratum controls, and exact one-sided
false-negative bounds. Binding drift resets the sample, insufficient evidence
is inconclusive, and one known false negative suspends. Every validation and
promotion result still disables selection. C3-005 owns the sole active
boundary: it revalidates loaded digests, recalculates BIA-002 from bound
observations, selects only a customer-manifest alternative, and restores the
original entrypoints for every disabled, bypassed, killed, drifted, unknown,
global, unqualified, or invalid path while preserving Test-owned checks.

No type in this directory replaces the normative schemas, OpenAPI, or Protobuf definitions in `contracts/`.

`generated/openapi/` contains the checked-in Go transport binding derived from
the normative OpenAPI documents. It is regenerated through
`./dev/generate-code --artifact openapi-go-client-v1` and never edited
manually.
