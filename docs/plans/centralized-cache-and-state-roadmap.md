# Centralized Gradle cache and BuildOpt state POC roadmap

## Objective

BuildOpt should offer one optional owner-operated HTTPS service that lets
developer workstations and CI runners share two complementary forms of state:

1. Gradle Build Cache objects, consumed through Gradle's native remote-cache
   protocol; and
2. BuildOpt discovery, calibration, profile and evidence state, consumed by
   `buildopt optimize`.

This service is not a replacement for Gradle's local cache or for a CI
provider's cache. It is an independent central option for teams that want one
cache and optimization memory shared by every build machine, regardless of CI
provider. The supported local POC path must continue to work without a server.

The implementation should extend the existing `buildopt-server`, Shared Cache,
local verifying gateway and content-addressed storage. It must not create a
second cache implementation or store BuildOpt control documents as opaque
Gradle cache entries.

## Current status

The storage contract is complete. The three versioned state schemas, exact
HTTP/state semantics and executable lifecycle vectors are defined in the
[central storage contract](../../specs/poc-central-storage-contract-v1.md).
They keep Gradle objects opaque and evictable while portfolios, evidence and
checkpoints use typed immutable manifests plus one CAS head. No central-state
server, external listener, credential flow or synchronization exists yet.

The next block is `POC-CENTRAL-STATE-STORAGE-001`: implement those exact state
semantics on the existing content-addressed files and SQLite metadata, then
prove restart, corruption, concurrency and partial-publication behavior before
adding HTTPS.

## Target experience

An owner prepares one server host:

```bash
buildopt-server init \
  --state-dir /var/lib/buildopt \
  --public-url https://buildopt.example.com

buildopt-server serve --config /etc/buildopt/server.json
```

Each workstation or runner connects once:

```bash
buildopt connect https://buildopt.example.com --token-file ./buildopt.token
```

The normal customer command remains unchanged:

```bash
buildopt optimize build
```

After connection, BuildOpt should configure the invocation-local gateway,
Gradle's remote HTTP cache and compatible profile synchronization
automatically. Upstream credentials remain launcher/gateway inputs and are
never passed to Gradle or committed to the repository.

## Architecture

```text
Developer or CI runner
        |
        | local Gradle cache first
        v
BuildOpt launcher and verifying gateway
        |
        | HTTPS
        v
+--------------------------------------------------+
|                  buildopt-server                 |
|                                                  |
| Gradle cache data plane                          |
|   /cache/<gradle-cache-key>                      |
|                                                  |
| BuildOpt state control plane                     |
|   /api/v1/repositories/<scope>/portfolios        |
|   /api/v1/repositories/<scope>/evidence          |
|   /api/v1/repositories/<scope>/checkpoints       |
+--------------------------------------------------+
        |
        +-- immutable content-addressed blobs
        +-- cache metadata database
        +-- separate BuildOpt control metadata database
```

One process and storage root may serve both planes, but the protocols,
authorization scopes, metadata and lifecycle rules remain separate.

## State ownership and retention

| State | Owner | Central representation | Retention rule |
|---|---|---|---|
| Task and transform outputs | Gradle | Opaque immutable objects keyed by Gradle cache key | Evictable by byte quota and recency |
| Local cache mirror | Gradle | Gradle User Home on each machine | Gradle-owned cleanup |
| Profile portfolio | BuildOpt | Versioned immutable manifest plus CAS head | Current plus 30 days after supersession |
| Calibration evidence | BuildOpt | Immutable digest-bound manifest and objects | While referenced, then 30 days |
| Calibration checkpoint | BuildOpt | Immutable resumable state plus CAS head | 24 hours from creation |
| Resumable checkpoints | BuildOpt | Exact input-bound temporary documents | Short-lived and safe to discard |
| Applicability and invalidation | BuildOpt | Repository/workflow/graph/tool/output bindings | Durable while every binding matches |

Evicting a Gradle object causes an ordinary cache miss. Evicting or replacing
BuildOpt authority changes an optimization decision, so control state must use
explicit generations, immutable documents and compare-and-swap updates rather
than the Gradle cache's opaque `GET`/`PUT` semantics.

## Cross-commit reuse

Evidence and applicability have different identities:

- evidence remains immutably bound to the exact commits and executable that
  produced it;
- a qualified profile may apply to later commits only when its structural
  repository, Wrapper, Gradle, JDK, BuildOpt, graph, option and output-contract
  bindings still match and the current change belongs to that qualified
  family; and
- build logic, Wrapper, global configuration, output-contract or unsupported
  graph drift marks the profile stale before Gradle starts.

The applicability key must therefore not be only a commit SHA. Ordinary source
changes inside a qualified family are expected to cross commits. Structural
drift retains optimized native Gradle and may start a new calibration
generation.

## Cache population policy

The POC default is deliberately conservative:

- trusted clean CI builds may read and publish Gradle cache objects;
- pull requests and developer workstations read the remote Gradle cache and
  continue writing their local cache, but do not publish remotely by default;
- BuildOpt evidence is accepted only after the existing correctness and value
  gates pass;
- a profile update uses the exact previous generation as a precondition; and
- a rejected, aborted or interrupted attempt never becomes readable
  optimization authority.

The policy may be relaxed explicitly for an owner-operated experiment, but the
client must always report whether it has read-only or read-write authority.

## Network and security boundary

- External clients use HTTPS with a certificate trusted by the build JVM and
  BuildOpt client.
- Plain HTTP remains limited to the invocation-local loopback gateway and test
  fixtures.
- The gateway holds the upstream credential and exposes only a fresh local
  invocation credential to Gradle.
- Tokens are scoped independently for cache read, cache write, state read and
  state write. The POC may use one owner-issued token containing several
  scopes; it does not require production RBAC or workload identity.
- Repository, compatibility, trust and generation scopes remain part of every
  server lookup. Blob presence alone is never authorization.
- Logs, reports and exported sessions never contain tokens or raw credential
  material.

## Availability and fallback

Central persistence must not make a correct Gradle build depend on the server:

- a remote Gradle cache miss or outage falls through to the local cache and
  normal task execution;
- an unavailable BuildOpt state service may use a previously verified local
  snapshot only while every binding still matches;
- no local compatible snapshot selects optimized native Gradle;
- failed uploads remain visible diagnostics but do not replace the Gradle exit
  status; and
- reconnect and replay are idempotent and never convert partial state into a
  qualified profile.

## Relationship to CI-provider caches and Edge

The central service is optional and additive:

- GitHub Actions, GitLab or another provider may still cache Wrapper and
  dependency state;
- teams may stop archiving the local Gradle Build Cache in CI once the remote
  service proves faster and more reusable, but BuildOpt does not require that
  choice;
- the same server remains available to developer machines, self-hosted runners
  and multiple CI providers; and
- Edge Cache remains an optional locality layer in front of the committed
  Gradle object plane. It does not own BuildOpt profile authority.

## Ordered implementation blocks

| Order | Tracker block | Deliverable | Acceptance criterion |
|---:|---|---|---|
| 0 | `POC-CENTRAL-CACHE-STATE-ROADMAP-001` | Freeze this optional central-persistence architecture and its POC boundaries. | The tracker, one-command roadmap and documentation index agree on ownership, sequencing and non-goals. |
| 1 | `POC-CENTRAL-STORAGE-CONTRACT-001` | Define versioned HTTP/state schemas, namespaces, generations, CAS, retention and failure semantics. | Executable contract vectors distinguish Gradle blobs, portfolios, evidence and checkpoints and reject cross-namespace access. |
| 2 | `POC-CENTRAL-STATE-STORAGE-001` | Extend existing content-addressed files and SQLite control metadata for BuildOpt portfolios, evidence and checkpoints. | Restart, idempotency, concurrent-generation, corruption and eviction tests preserve immutable state and never promote partial work. |
| 3 | `POC-CENTRAL-HTTPS-AUTH-001` | Add externally reachable HTTPS plus scoped owner-operated POC credentials while retaining the loopback gateway boundary. | A second machine can authenticate through a trusted certificate; HTTP outside loopback, wrong scopes and secret disclosure fail closed. |
| 4 | `POC-CENTRAL-GRADLE-CACHE-001` | Route Gradle's native remote Build Cache through the local gateway to the centralized object plane. | A trusted producer publishes once, a clean consumer receives `FROM-CACHE`, bytes match, read-only clients cannot publish and outage rebuilds normally. |
| 5 | `POC-CENTRAL-STATE-SYNC-001` | Add `buildopt connect` and exact local/remote synchronization for portfolios, evidence and checkpoints. | First sync, no-change sync, concurrent update, interruption, offline verified snapshot and incompatible-state fallback all pass without manual internal files. |
| 6 | `POC-CENTRAL-OPTIMIZE-INTEGRATION-001` | Integrate compatible remote profile lookup and publication into `buildopt optimize`. | A later compatible commit selects a remotely learned profile automatically; structural drift selects native before Gradle and starts a separate generation. |
| 7 | `POC-CENTRAL-TWO-MACHINE-001` | Exercise one server, one trusted producer and one clean consumer in isolated machines or containers. | The consumer reuses both Gradle objects and BuildOpt state, exact outputs match, credentials stay private and server loss proves local/native fallback. |
| 8 | `POC-CENTRAL-END-TO-END-VALUE-001` | Compare the complete installed centralized path against optimized native Gradle using the same remote Gradle cache opportunity. | At least two substantial Gradle families show equivalent outputs and a net wall-time win attributable to structural reduction plus cache reuse; an honest non-winning case retains native. |

The local one-command roadmap remains the primary critical path. Blocks 1–3
may establish the reusable server boundary after the `buildopt optimize` state
contract exists. State synchronization and automatic replay require the local
profile portfolio and replay blocks; centralized persistence must not delay or
become mandatory for local POC value.

## POC success scorecard

Each end-to-end result must report:

- server setup commands and time;
- client inputs beyond the normal Gradle command;
- local and remote cache hits, misses, bytes and transfer time;
- profile source generation and selection overhead;
- optimized-native and BuildOpt end-to-end wall time;
- graph reduction and executed task outcomes;
- exact or reviewed semantic output equivalence;
- calibration cost, expected matching lifetime, break-even and cumulative
  saving;
- behavior with empty state, stale state, unauthorized write, server outage
  and corrupted data; and
- whether CI-provider cache functionality was enabled, disabled or irrelevant.

Cache-hit percentages, graph reduction and component timings are diagnostic.
The decision gate remains net customer-visible wall time against optimized
native Gradle under an equal remote-cache opportunity.

## POC boundaries

This roadmap does not require production high availability, multi-tenancy,
RBAC administration, billing, hosted control plane, KMS/HSM integration,
cross-region replication, disaster-recovery objectives, an eight-hour soak or
a design partner. It does not authorize Test Optimization or production
promotion.

It does require real HTTPS between machines, exact namespace isolation,
credential containment, restart-safe state, native fallback, equivalent
outputs and measured value beyond what the same Gradle remote cache already
provides.
