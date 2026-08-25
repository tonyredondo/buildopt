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

## Components

| Component | Runtime artifact | Responsibility |
|---|---|---|
| CI Launcher | `buildopt` | Preserves argv, environment boundaries, streams, process tree, signals, and child exit status |
| Local Verifying Cache Gateway | inside `buildopt` | Gives Gradle an invocation-local endpoint, verifies policy and objects, and hides upstream credentials |
| Gradle plugin | `jvm/gradle-plugin` JAR | Performs the authenticated handshake, configures managed cache inputs, and emits bounded task/build events through public Gradle APIs |
| Optimization service and Shared backend | `buildopt-server` | Accepts sessions, exports immutable evidence, serves authenticated cache state, stores control metadata, and exposes local build history |
| Edge Cache | `buildopt-edge` | Provides a bounded nearby read-through/pending-write cache while Shared retains commit and collision authority |
| Build Impact | `buildopt-impact` | Discovers the declared Gradle graph and verifies repository-owned generated impact state |
| Adaptive fragment model | `internal/adaptivefragment` | Defines path-independent fragment identity, immutable state/economics, non-authorizing structural priors and conflict-aware pre-Gradle composition plans; no runtime activation yet |
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
| Edge blobs and metadata | Edge | Bounded local store, durable pending replication, TTL and byte-SLRU |
| Build sessions | Server exporter | Atomic mode-`0600` immutable JSON plus bounded deterministic JSONL |
| Patch staging | Java patcher | Private detached Git worktree; exact branch/ref publication only after all postimages match |

### Optional cross-machine state boundary

The adaptive POC now has four storage-neutral documents: fragment generations,
append-only observations, portfolio snapshots and economic-ledger snapshots.
They use repository scope, exact generation links and external RFC 8785 JCS
digests. `AF-002` defines and validates those bytes only. Local persistence and
reuse through the existing HTTPS state plane remain deferred to `AF-012`; an
unavailable or incompatible document therefore grants no current runtime
authority.

`AF-009` adds a storage-neutral planner over those documents. It accepts only
same-scope, unexpired qualified generations and exact whole-composition
economic predictions. Dependencies must be closed, conflicts are symmetric and
every constituent correctness authority remains present. Missing or ambiguous
input, absent joint economics or a prediction below the fixed net-value floor
produces a native Gradle plan before Gradle starts. Runtime activation remains
deferred to `AF-010`.

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
