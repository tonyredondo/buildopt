# Architecture overview

This document explains the architecture implemented by the current repository.
The [master RFC](../../gradle-build-optimization-platform.md) remains the source
of product decisions and invariants; executable details live in
[contracts](../../contracts/README.md), [specifications](../../specs/README.md),
and [ADRs](../../adr/README.md).

## System at a glance

BuildOpt sits around an existing Gradle invocation. It does not replace the
repository's Wrapper or build logic.

```text
CI job or developer shell
        |
        v
 buildopt gradle build
        |
        +---- bypass ------------------------------> original Gradle command
        |
        v
 Launcher and local verifying gateway
        |              |                 |
        |              |                 +--------> buildopt-server
        |              |                             sessions, policy,
        |              |                             Shared cache, history
        |              |
        |              +---- optional Edge --------> Shared cache authority
        |
        v
 Gradle Wrapper + BuildOpt settings/project plugin
        |
        +---- managed native L1
        +---- authenticated loopback L2
        +---- task and build evidence
        |
        v
 Original artifacts, stdout/stderr, and exit status
```

The normal Gradle outcome remains authoritative. Optimization setup and
telemetry delivery are fail-open where they must not break a correct baseline;
cache publication, policy activation, task qualification, Build Impact, and
patch application fail closed because accepting unproven work could change the
result.

### Sticky-wrapper entrypoint and central cache POC

The POC includes a thin repository-committed entrypoint above the implemented
launcher. Deterministic generation, verified native-package bootstrap and
neutral Gradle process passthrough remain available. The active experiment now
uses that entrypoint to key evidence by the exact ordinary request a customer
repeats; it does not substitute a different Gradle command:

```text
committed buildoptw / buildoptw.bat
        |
        +-- verify/cache pinned BuildOpt distribution
        +-- read exact local signed decision snapshot
        +-- optionally synchronize typed state over HTTPS
        |
        v
existing buildopt launcher -> existing Gradle Wrapper
```

The closed `REQUEST_ALIGNED_RECURRENT_CLOSURE_V1` route has an observation-only
producer, adjacent-transition classifier and fresh public-capture boundary.
It canonically binds exact Gradle arguments and requested tasks, Gradle and
Wrapper identity, portable JDK facts, a safe environment aggregate, build
logic and the finalized requested graph. Current outputs are repository-
relative and owned by a unique producer; missing, ambiguous or outside-graph
ownership is typed unavailable. The classifier attaches adjacent-revision
changes, derives an exact affected producer closure only from finalized inputs
and binds every omitted current output. Unsafe or full-graph cases emit no
action. Nothing in this path authorizes execution or timing, so the original
optimized-native invocation remains authoritative.

The public-capture runner applies that same generic boundary to a frozen exact
request over chronological first-parent histories. It records raw base/target
observations, changed paths and independently reproducible classifier reports.
The 110-transition capture completes Groovy and Spring but not Kafka,
Micronaut or OpenTelemetry. The independent gate ignores the aggregate summary,
reconstructs every report and confirms 2/5 complete/action families against
the frozen 5/5-complete and 3/5-action requirements. Its terminal decision
preserves five economic criteria as unmeasured and authorizes no speedup or
successor. No runtime optimizer, candidate activation or timing was authorized.

The four generated committed files contain only bootstrap identity and portable
non-secret configuration. Runtime credentials remain private. The implemented
connection boundary hashes the committed project scope independently of the
checkout path, additionally binds server/cache namespace and generation, and
proves `CACHE_READ` plus `STATE_READ` with live revocation. Missing or rejected
credentials retain native Gradle and never reach the child process. A local exact
decision keeps `NATIVE_NOOP` out of a blocking server lookup; expired,
incompatible or absent state uses native Gradle. When the connection is valid,
the wrapper now configures Gradle's native HTTP cache through an invocation-local
verifying gateway, enables `--build-cache` unless explicitly disabled, and keeps
the central path read-only for ordinary developer and pull-request builds. The
managed local L1 remains private and BuildOpt-controlled; the central L2 follows
Gradle's native cache policy so arbitrary cacheable tasks are not silently
excluded. The existing Gradle-cache data plane and typed state control plane
remain separate. `SWL-007` now defines the control-plane records and
persistence adapters for observations, actions, trials, signed decisions and
economics; `SWL-008` consumes a verified local snapshot through a read-only
selector. Its lookup is bounded and never performs synchronous network I/O; it
only retains native Gradle today, while active actions remain deferred to a
later POC block. `SWL-008A` connects that retention boundary to the committed
sticky wrapper: an invocation with no server credential or explicit BuildOpt
integration skips gateway, plugin-handshake, managed-L1 and central-cache
setup, while preserving the native Gradle process contract. Its ordinary
observer defaults to lazy `light` mode and computes the executable digest
concurrently when possible; `full` source-revision lookup and `0` disablement
are explicit choices. `SWL-009` adds a separate observation plane
for ordinary Wrapper invocations: it records phase timing, provenance, outcome and
Configuration Cache state without granting authority or changing the Gradle
result. `SWL-010` adds a trusted-CI trial plane that runs isolated candidate
and native commands with separate checkout, Gradle/cache/daemon and BuildOpt
state roots; it is budgeted, order-balanced and exact-output checked. Its first
result is negative for value, so that historical trial remains diagnostic.
`internal/stickyactive` proves isolated execution/suspension and
`internal/durablecatalog` keeps patches review-only. `SWL-014B` now composes
observation through signed active decision and economics behind the committed
wrapper. Its only composition root is `internal/launcher/sticky_learning.go`; `internal/launcher/run.go`
delegates to it, and shared conservative value statistics live in
`internal/stickyvalue`. Trusted learning additionally requires mode `auto`, a
non-zero committed trial budget, explicit `BUILDOPT_STICKY_LEARNING=1` and an
owner token with state-write/cache-read/state-read authority. This boundary is
frozen in the
[Fresh Generic Optimization POC Tracker](../plans/fresh-generic-optimization-poc-tracker.md)
and its [fresh machine contract](../../specs/poc-fresh-generic-optimization-v1.json).
Every candidate, native and counterfactual child remains owned by the launcher
process supervisor through explicit active/trial executor adapters.
It builds on deterministic generation, verified package bootstrap, exact
Gradle passthrough, authenticated portable connection and read-only central
cache integration. The decision store grants no production authority and does
not make a standalone wall-time claim. Native, candidate and counterfactual
children still use the launcher's existing process supervisor; executor
adapters in `stickyactive` and `stickytrial` must not bypass WS-002 signal or
exit semantics.

The original longitudinal v1 harness is not architecture evidence for that
composition. It configured no central identity and zero trial budget and gave
`--build-cache` only to control. The closed fresh route required symmetric cache
opportunity through separate per-arm writable namespaces that start empty,
plus explicit lifecycle/action/ledger records before a campaign
can become terminal-ready. Only the existing task-contract and
declared-graph-scope detectors may enter the public opportunity gate.
`SWL-FRESH-001` must first implement complete generic public producers for
both; incomplete input blocks the experiment and cannot count as absence of
opportunity.

The installed two-machine proof exercises this boundary with a trusted
producer and a clean read-only consumer in separate containers. The wrapper
preloads and verifies one archive by version/platform/SHA-256, keeps pending
Gradle cache objects invisible until owner commit, then restores cacheable
tasks over HTTPS after a service restart. During an outage it executes the
ordinary Gradle Wrapper and reproduces the same outputs. The central cache
credential and optional POC CA path are launcher inputs and are scrubbed before
Gradle starts; neither can authorize a BuildOpt decision.

## Components

| Component | Runtime artifact | Responsibility |
|---|---|---|
| Repository BuildOpt Wrapper | `buildoptw`, `buildoptw.bat` | Verifies and bootstraps a pinned BuildOpt distribution, then preserves the repository Gradle Wrapper process contract; bypass runs Gradle before configuration or bootstrap |
| CI Launcher | `buildopt` | Preserves argv, environment boundaries, streams, process tree, signals, and child exit status |
| Local Verifying Cache Gateway | inside `buildopt` | Gives Gradle an invocation-local endpoint, verifies policy and objects, and hides upstream credentials |
| Gradle plugin | `jvm/gradle-plugin` JAR | Performs the authenticated handshake, configures managed cache inputs, and emits bounded task/build events through public Gradle APIs |
| Optimization service and Shared backend | `buildopt-server` | Accepts sessions, exports immutable evidence, serves authenticated cache state, stores control metadata, and exposes local build history |
| Edge Cache | `buildopt-edge` | Provides a bounded nearby read-through/pending-write cache while Shared retains commit and collision authority |
| Build Impact | `buildopt-impact` | Discovers the declared Gradle graph and verifies repository-owned generated impact state |
| Adaptive fragment model | `internal/adaptivefragment` | Defines path-independent fragment identity, immutable state/economics, non-authorizing structural priors and conflict-aware pre-Gradle composition plans; no runtime activation yet |
| Ordinary-build observation | `internal/stickyobservation` | Appends private, canonical observations with phase timing, provenance, outcome and Configuration Cache evidence; no decision authority |
| Budgeted sticky trials | `internal/stickytrial`, `cmd/sticky-trial-benchmark` | Runs trusted-CI-only alternating candidate/native pairs in eight distinct private roots, charges observed compute and requires exact output hashes; no activation authority |
| Patch engine | `jvm/patcher` JAR | Verifies signed exact bundles, applies them in a detached worktree, and supports draft-only delivery and exact revert |
| Windows service host | `buildopt-service.exe` | Runs server or Edge under Windows SCM with the supplied private config |

The optional JVM Agent and Rust helper are bounded experiments. Neither is a
source of authority and both report `UNAVAILABLE` where their evidence cannot
prove safe task behavior.

## Invocation lifecycle

### 1. Preflight and bypass

The launcher accepts raw `buildopt run -- <argv>`, packaged
`buildopt gradle <args>`, and the owner-invoked `buildopt optimize <args>` POC
entrypoints. The optimize path owns discovery, calibration, repository-scoped
profile storage and exact pre-Gradle replay; GitHub/GitLab replace only the
ephemeral checkout-path scope with provider repository identity. Every other
state binding remains exact. It evaluates
`BUILDOPT_BYPASS=1` before starting any BuildOpt service. In bypass, reserved
BuildOpt variables are removed and the original argv is executed directly.

Without bypass, the launcher validates each configured group atomically. An
incomplete optional group is not partially exposed to Gradle: that capability
falls back while the original command remains runnable.

### 2. Invocation-local trust context

The launcher creates a fresh attempt identity and authenticated plugin event
channel. It starts or reconnects the local loopback gateway and registers the
current invocation. Linux uses peer credentials on Unix sockets for managed
control; macOS and Windows use a loopback control endpoint with a fresh random
credential where peer-UID sockets are not available.

Upstream cache and ingest credentials remain launcher/gateway inputs. They are
never placed in Gradle's environment, event payloads, exported sessions, or
diagnostics.

### 3. Gradle startup and handshake

The repository's own Wrapper starts Gradle. The BuildOpt plugin authenticates
the local context, verifies the gateway generation, and sends one producer
handshake. Configuration Cache reuse does not reuse an invocation identity; a
new handshake occurs for every build.

The settings plugin may configure:

- a generation-segmented native `DirectoryBuildCache` as L1;
- the local gateway as Gradle's HTTP L2;
- signed policy digests and restriction-only Tier 1 task policy;
- a read-only dependency and Wrapper bootstrap cache.

Unknown combinations, missing authority, or occupied state leases preserve the
baseline or a safe private-L1 path. They never borrow policy from a different
repository, compatibility class, or security generation.

### 4. Cache and evidence flow

A cache read is returned only after authorization, complete-byte checksum
verification, current revocation checks, and committed-state lookup. A write
enters a durable pending attempt and is not generally readable until an
authenticated `CommitDecision` selects it. Abort, expiry, process death, or
conflict leaves it invisible and eligible for reconciliation.

Build and task observations are bounded and capability-labelled. `EXACT`,
`APPROXIMATED`, and `UNAVAILABLE` are distinct states; unavailable evidence is
never converted to zero or success. The server can publish immutable private
`BUILD_SESSION v1` files and a deterministic JSONL stream.

### 5. Outcome and cleanup

The launcher waits for the complete Gradle process tree, preserves its status,
and ends the invocation registration. Signal handling covers descendants:
Unix uses process groups and Windows uses a Job Object. Session delivery and
export errors remain diagnostic unless a stricter operator contract explicitly
requires startup to fail closed.

## Control plane, data plane, and authority

```text
                  signed policy / revocation
                +-----------------------------+
                |                             v
Launcher ---- invocation context ----> Local gateway
   |                                      |
   | evidence                             | verified GET / pending PUT
   v                                      v
Server control state                 Edge (optional) ----> Shared blobs
   |                                      |                    |
   +---- CommitDecision / scopes ---------+--------------------+
```

- **Control plane:** policy, revocation generations, attempts, commit
  decisions, rollout state, validation evidence, and data-lifecycle state.
- **Data plane:** content-addressed cache blobs and bounded transfer paths.
- **Authority:** signed policy plus explicit repository, namespace, trust,
  operation, and generation bindings. Blob presence or a checksum alone is not
  authorization.

The sticky-wrapper experiment adds typed observations, trials, action decisions
and a cumulative economic ledger to the control plane. It does not encode them
as Gradle cache entries. `ACTIVE` remains an exact, expiry-bound decision; cache
availability alone can never move an action from observation or trial into
active execution.

In the owner-operated POC, Shared and its local signing key are part of the
trusted computing base. The deferred hardened profile separates attestation
authority from the data plane and adds workload identity, multi-tenancy,
KMS/HSM-backed keys, HA, and recovery objectives.

## Persistence model

| State | Owner | Durability rule |
|---|---|---|
| Gateway identity and slot lease | Launcher/gateway | Private current-user files; identity rotates as a unit |
| Managed L1 | Launcher and Gradle plugin | Repository/trust/compatibility/generation segmented; exclusive invocation lease |
| Shared blobs | Server | Immutable content-addressed files on a proven local filesystem |
| Cache and control metadata | Server | Separate SQLite databases in WAL and `FULL` synchronous mode |
| BuildOpt portfolio/evidence/checkpoint metadata | Server | Independent `state.sqlite`; immutable manifests plus exact-generation CAS head |
| Sticky-wrapper decisions, actions, trials and economics | `internal/stickydecision` | Immutable JCS records plus one generation-CAS head; local files or existing central `EVIDENCE` state; no Gradle-cache authority |
| Ordinary-wrapper observations | `internal/stickyobservation` | Private append-only JSONL, per-record canonical validation and an exclusive append lock; unavailable phases remain unavailable |
| Edge blobs and metadata | Edge | Bounded local store, durable pending replication, TTL and byte-SLRU |
| Build sessions | Server exporter | Atomic mode-`0600` immutable JSON plus bounded deterministic JSONL |
| Patch staging | Java patcher | Private detached Git worktree; exact branch/ref publication only after all postimages match |

### Optional cross-machine state boundary

The adaptive POC has four storage-neutral documents: fragment generations,
append-only observations, portfolio snapshots and economic-ledger snapshots.
They use repository scope, exact generation links and external RFC 8785 JCS
digests. `AF-012` now persists those exact bytes in a private local immutable
generation behind one optimistic-CAS head. The same bytes travel through the
existing repository-scoped HTTPS `EVIDENCE` and `PORTFOLIO` manifests and are
fully revalidated before a clean second machine restores them.

`AF-009` adds a storage-neutral planner over those documents. It accepts only
same-scope, unexpired qualified generations and exact whole-composition
economic predictions. Dependencies must be closed, conflicts are symmetric and
every constituent correctness authority remains present. Missing or ambiguous
input, absent joint economics or a prediction below the fixed net-value floor
produces a native Gradle plan before Gradle starts. `AF-010` now consumes that
plan for Build Impact only: each producer needs separate compatible subgraph
and exact-output-materialization fragments. Unaffected producers restore
verified bytes, changed producers rebuild locally, and global, ambiguous or
incomplete state retains the complete original workflow before execution.
The activation implementation remains storage-neutral. `AF-011` directly measured the active
composition and retained independently qualified fragments because Kotlin
Build Impact missed its frozen isolated repeatability gate. Portable state does
not change those authorities: a verified offline generation may be reused,
while absent, corrupt, incompatible or unlinked state retains native Gradle.
Gradle cache objects remain on the independent cache protocol and never store
adaptive policy documents.

The local POC remains service-independent. `buildopt-server` can now expose one
TLS 1.3 endpoint with two logically isolated planes:

| Plane | Contents | Visibility and failure |
|---|---|---|
| Gradle cache | Opaque, evictable objects addressed by native Gradle cache keys | `GET`/`PUT`; eviction or outage is an ordinary miss and grants no optimization authority |
| BuildOpt state | Typed portfolio, evidence and checkpoint artifacts under a repository/kind namespace | Immutable artifact and manifest upload followed by exact next-generation head CAS; invalid or unavailable state retains optimized native Gradle |

The executable [central storage contract](../../specs/poc-central-storage-contract-v1.md)
fixes namespaces, retention, CAS and fallback. Its
[local storage implementation](../../specs/poc-central-state-storage-v1.md)
now uses the existing physical CAS plus independent `state.sqlite` metadata,
reverifies every artifact on publication/read and preserves typed state across
restart. The [central HTTPS contract](../../specs/poc-central-https-auth-v1.md)
adds a real external TLS listener and four independent owner-issued
capabilities. Only token digests persist and revocation is checked on every
request. Metadata, authorization and visibility do not cross planes. Remote
Gradle-cache gateway forwarding and typed-state client synchronization are now
implemented as separate POC proofs. `buildopt connect` persists one private
repository-scoped connection and `buildopt sync` verifies canonical
portfolio/evidence/checkpoint bundles, optimistic generation conflicts,
interrupted retry and offline snapshots. A connected `buildopt optimize` now
performs the same sync automatically, revalidates remote ancestry, build logic,
graph ownership, family, workflow, tools, outputs, preconditions and evidence
before selection, and publishes completed local state afterward. Native Gradle
remains authoritative on any drift or service failure.

Linux checks a local filesystem allowlist. macOS requires `MNT_LOCAL` and a
same-device boundary. Windows requires one local volume and rejects reparse
traversal. Atomic file replacement is used on every platform; Unix additionally
syncs directories where Windows cannot open them for equivalent flushing.

## Optimization safety model

BuildOpt separates actions by the proof they require:

- **Direct bounded actions:** an exact qualified task adapter or read-only Edge
  route within a repository-owned profile.
- **Proof-gated runtime actions:** native Configuration Cache and cache routes
  promoted only from compatible evidence and controls.
- **Repository changes:** Patch Autopilot emits an exact signed bundle, applies
  it in isolation, runs full relevant validation, and creates only a draft
  change. It does not force-push or merge.
- **Less work:** Build Impact selects only alternatives predeclared by a
  repository-owned manifest and proven against the current generated graph.
  Unknown or global changes use the full graph. Test-owned checks are never
  omitted by BuildOpt.
- **Reviewable specialization:** POC profile discovery binds a qualified
  matrix cell to the exact graph, generated state, trace/input digests, output,
  and reviewed contract. It emits a profile only when every binding remains
  valid; it never activates the result and otherwise emits native full graph.

Every active action has a disablement, fallback, or exact revert path. Passing
one build is not treated as proof of correctness.

## Deployment profiles

| Profile | Intended use | Current status |
|---|---|---|
| Per-invocation local | Launcher, handshake, and neutral loopback gateway | Implemented |
| Owner-operated POC | Private single-node Shared, managed L1, evidence, optional Edge | Implemented and synthetically exercised |
| Self-hosted single node | Explicit private config and OS service lifecycle | Implemented |
| Hardened multi-tenant | Workload identity, external attestation, HA, KMS/HSM, customer SLOs | Deferred |

The POC is not silently promoted to the hardened profile. `buildopt doctor`,
the capability matrix, and the tracker report exact evidence boundaries.

## Platform equivalence

The same functional contracts run on Linux, macOS, and Windows, but the
implementation uses native primitives:

| Concern | Linux | macOS | Windows |
|---|---|---|---|
| File locking | `flock` | `flock` | `LockFileEx` |
| Child isolation | Process group | Process group | Process group + Job Object |
| Managed control | Same-UID Unix socket | Authenticated loopback | Authenticated loopback |
| Service manager | systemd | launchd user agent | Windows SCM wrapper |
| Resource isolation | cgroup v2 when available | Report-only | Job Object |

No platform emulates a stronger capability than it can enforce. See the
[platform parity specification](../../specs/platform-runtime-parity-v2.md).

## Where the implementation lives

The [repository map](./repository-map.md) maps every component above to its
source, contracts, fixtures, checks, and operational documentation. The most
important dependency rule is:

```text
RFC/ADR decision
    -> normative contract or executable specification
        -> internal implementation
            -> cmd/ or JVM entrypoint
                -> fixture/check/CI evidence
```

Generated clients are the exception: their normative IDL is edited first and
their checked-in source is regenerated. Never edit generated output directly.
