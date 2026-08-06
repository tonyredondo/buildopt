# Executable specifications

Operational contracts connecting multiple components: CI orchestration, Gradle correlation, Test Optimization integration, PatchBundle, and the capability matrix.

Specifications are normative executable behavior, not the recommended learning
path. Start with the [documentation portal](../docs/README.md) or
[architecture overview](../docs/architecture/overview.md), then return here for
the exact cross-component contract.

| Specification | Owning item |
|---|---|
| [`poc-real-world-compatibility-v1.md`](./poc-real-world-compatibility-v1.md) and [`poc-real-world-compatibility-v1.json`](./poc-real-world-compatibility-v1.json) | `POC-REALWORLD-001` / `POC-REALWORLD-G01` |
| [`poc-real-world-performance-v1.md`](./poc-real-world-performance-v1.md) and [`poc-real-world-performance-v1.json`](./poc-real-world-performance-v1.json) | `POC-REALWORLD-002` / `POC-REALWORLD-G02` |
| [`poc-real-world-diagnostics-v1.md`](./poc-real-world-diagnostics-v1.md) and [`poc-real-world-diagnostics-v1.json`](./poc-real-world-diagnostics-v1.json) | `POC-REALWORLD-DIAGNOSTICS-001` / `POC-REALWORLD-G03` |
| [`poc-public-build-tasks-v1.md`](./poc-public-build-tasks-v1.md) and [`poc-public-build-tasks-v1.json`](./poc-public-build-tasks-v1.json) | `POC-PUBLIC-BUILD-TASKS-001` / `POC-PUBLIC-BUILD-TASKS-G01` |
| [`poc-mockito-test-build-v1.json`](./poc-mockito-test-build-v1.json) | `POC-MOCKITO-TEST-BUILD-001` / `POC-MOCKITO-TEST-BUILD-G01` |
| [`poc-spring-framework-v1.md`](./poc-spring-framework-v1.md) and [`poc-spring-framework-v1.json`](./poc-spring-framework-v1.json) | `POC-SPRING-PREREG-001` / `POC-SPRING-PREREG-G01` |
| [`poc-spring-test-preparation-v2.json`](./poc-spring-test-preparation-v2.json) | `POC-SPRING-TEST-PREPARATION-001` |
| [`poc-otel-test-preparation-v1.json`](./poc-otel-test-preparation-v1.json) | `POC-OTEL-TRANSFER-001` |
| [`poc-stability-validation-v1.md`](./poc-stability-validation-v1.md) and [`poc-stability-validation-v1.json`](./poc-stability-validation-v1.json) | `POC-STABILITY-001` |
| [`poc-pairing-validation-v1.md`](./poc-pairing-validation-v1.md) and [`poc-pairing-validation-v1.json`](./poc-pairing-validation-v1.json) | `POC-PAIRING-001` |
| [`poc-groovy-validation-v1.md`](./poc-groovy-validation-v1.md) and [`poc-groovy-validation-v1.json`](./poc-groovy-validation-v1.json) | `POC-GROOVY-001` |
| [`poc-shared-groovy-validation-v1.md`](./poc-shared-groovy-validation-v1.md) and [`poc-shared-groovy-validation-v1.json`](./poc-shared-groovy-validation-v1.json) | `POC-SHARED-GROOVY-001` |
| [`poc-leaf-kotlin-validation-v1.md`](./poc-leaf-kotlin-validation-v1.md) and [`poc-leaf-kotlin-validation-v1.json`](./poc-leaf-kotlin-validation-v1.json) | `POC-LEAF-KOTLIN-001` |
| [`poc-kotlin-stability-validation-v1.md`](./poc-kotlin-stability-validation-v1.md) and [`poc-kotlin-stability-validation-v1.json`](./poc-kotlin-stability-validation-v1.json) | `POC-KOTLIN-STABILITY-001` |
| [`poc-kotlin-boundary-decision-v1.md`](./poc-kotlin-boundary-decision-v1.md) and [`poc-kotlin-boundary-decision-v1.json`](./poc-kotlin-boundary-decision-v1.json) | `POC-KOTLIN-BOUNDARY-001` |
| [`poc-overhead-attribution-v1.md`](./poc-overhead-attribution-v1.md) and [`poc-overhead-attribution-v1.json`](./poc-overhead-attribution-v1.json) | `POC-OVERHEAD-001` |
| [`poc-breadth-validation-v1.md`](./poc-breadth-validation-v1.md) and [`poc-breadth-validation-v1.json`](./poc-breadth-validation-v1.json) | `POC-BREADTH-001` / `POC-BREADTH-G01` |
| [`poc-value-validation-v1.md`](./poc-value-validation-v1.md) and [`poc-value-validation-v1.json`](./poc-value-validation-v1.json) | `POC-VALUE-001..004` / `POC-VALUE-G01` |
| [`ci-orchestration-v1.md`](./ci-orchestration-v1.md) | `F0-030` |
| [`gradle-correlation-v1.md`](./gradle-correlation-v1.md) | `SPK-001` / `GRADLE-CORR-001` |
| [`benchmark-beta-v1.md`](./benchmark-beta-v1.md) | `F0-032` |
| [`beta-benchmark-harness-v1.md`](./beta-benchmark-harness-v1.md) | `OPS-001/A1` |
| [`beta-disk-faults-v1.md`](./beta-disk-faults-v1.md) | `A1-003` / `A1-G03` |
| [`beta-shared-faults-v1.md`](./beta-shared-faults-v1.md) | `OPS-001/A1` |
| [`beta-system-faults-v1.md`](./beta-system-faults-v1.md) | `OPS-001/A1` / `A1-G04` |
| [`beta-sustained-v1.md`](./beta-sustained-v1.md) | `OPS-001/A1` |
| [`beta-soak-v1.md`](./beta-soak-v1.md) | Historical productization harness; excluded from the active POC |
| [`beta-circuit-breaker-v1.md`](./beta-circuit-breaker-v1.md) | `OPS-001/A1` / `A1-G02` |
| [`beta-gradle-fixtures-v1.md`](./beta-gradle-fixtures-v1.md) | `OPS-001/A1` / `A1-G02` |
| [`private-beta-token-isolation-v1.md`](./private-beta-token-isolation-v1.md) | `A1-002` / `A1-G01` |
| [`onboarding-performance-v1.md`](./onboarding-performance-v1.md) | Product onboarding |
| [`cache-parity-v1.md`](./cache-parity-v1.md) | Safe-cache performance |
| [`private-beta-data-lifecycle-v1.md`](./private-beta-data-lifecycle-v1.md) | `A1-004` / `A1-G05` |
| [`private-beta-operations-v1.md`](./private-beta-operations-v1.md) | `A1-005` |
| [`owner-controlled-pilot-deployment-v1.md`](./owner-controlled-pilot-deployment-v1.md) | `A1-001` |
| [`owner-poc-evaluation-v1.md`](./owner-poc-evaluation-v1.md) | `A1-006` / `A1-G06` |
| [`runtime-owner-evaluation-v1.md`](./runtime-owner-evaluation-v1.md) | `B-G01` / `B-G03` |
| [`task-intelligence-poc-v1.md`](./task-intelligence-poc-v1.md) | `MVP-C1` |
| [`build-impact-manifest-v1.md`](./build-impact-manifest-v1.md) | `C3-001` |
| [`build-impact-declared-graph-v1.md`](./build-impact-declared-graph-v1.md) | `C3-002` |
| [`build-impact-shadow-validation-v1.md`](./build-impact-shadow-validation-v1.md) | `C3-003` |
| [`build-impact-promotion-gate-v1.md`](./build-impact-promotion-gate-v1.md) | `C3-004` / `BIA-002` |
| [`build-impact-selection-v1.md`](./build-impact-selection-v1.md) | `C3-005` |
| [`build-impact-gate-v1.md`](./build-impact-gate-v1.md) | `C3-G01` / MVP-C3 |
| [`build-impact-automatic-v1.md`](./build-impact-automatic-v1.md) | `BIA-F4-001..003` / `BIA-F4-G01` |
| [`build-impact-performance-v1.md`](./build-impact-performance-v1.md) | Build Impact performance |
| [`platform-compatibility-v1.md`](./platform-compatibility-v1.md) | `PLAT-F5-001..004` / `PLAT-F5-G01` |
| [`platform-runtime-parity-v2.md`](./platform-runtime-parity-v2.md) | `PLAT-F6-001..004` / `PLAT-F6-G01` |
| [`edge-cache-config-v1.md`](./edge-cache-config-v1.md) | `C2-001` |
| [`edge-cache-committed-read-v1.md`](./edge-cache-committed-read-v1.md) | `C2-002` |
| [`edge-cache-capacity-slru-v1.md`](./edge-cache-capacity-slru-v1.md) | `C2-003` |
| [`edge-cache-pending-replication-v1.md`](./edge-cache-pending-replication-v1.md) | `C2-004` |
| [`edge-cache-two-node-proxy-v1.md`](./edge-cache-two-node-proxy-v1.md) | `C2-005` |
| [`edge-cache-gate-v1.md`](./edge-cache-gate-v1.md) | `C2-G01` / MVP-C2 |
| [`edge-operability-v1.md`](./edge-operability-v1.md) | `O1-001..004` / `O1-G01` / POC-O1 |
| [`owner-poc-lab-v1.md`](./owner-poc-lab-v1.md) | `O2-001..004` / `O2-G01` / POC-O2 |
| [`build-history-api-v1.md`](./build-history-api-v1.md) | `UX-F1-001` |
| [`build-history-dashboard-v1.md`](./build-history-dashboard-v1.md) | `UX-F1-002` |
| [`custom-task-contract-java-recipe-v1.md`](./custom-task-contract-java-recipe-v1.md) | `C4-004` / `C4-G06` |
| [`test-optimization-integration-v1.md`](./test-optimization-integration-v1.md) | `F0-033` |
| [`full-relevant-validation-gate-v1.md`](./full-relevant-validation-gate-v1.md) | `C4-006` / `C4-G02` |
| [`customer-patch-workflow-v1.md`](./customer-patch-workflow-v1.md) | `C4-007` / `C4-G01` / `C4-G03` / `C4-G04` |
| [`patch-delivery-recovery-v1.md`](./patch-delivery-recovery-v1.md) | `C4-008` |
| [`post-merge-patch-monitor-v1.md`](./post-merge-patch-monitor-v1.md) | `C4-009` / `C4-G05` |
| [`patch-bundle-v1.md`](./patch-bundle-v1.md) | `F0-034` |
| [`bandit-policy-v1.md`](./bandit-policy-v1.md) | `F0-035` |
| [`capability-matrix-v1.md`](./capability-matrix-v1.md) | `F0-036` |
| [`data-lifecycle-v1.md`](./data-lifecycle-v1.md) | `F0-037` |
| [`tier-one-cache-policy-v1.md`](./tier-one-cache-policy-v1.md) | `A0-002` |
| [`tier-one-cache-conformance-v1.md`](./tier-one-cache-conformance-v1.md) | `A0-G01` |
| [`l1-l2-revocation-v1.md`](./l1-l2-revocation-v1.md) | `A0-G02` |
| [`gateway-rotation-v1.md`](./gateway-rotation-v1.md) | `A0-G03` |
| [`gateway-spool-v1.md`](./gateway-spool-v1.md) | `A0-G04` |
| [`shared-commit-recovery-v1.md`](./shared-commit-recovery-v1.md) | `A0-G05` |
| [`shared-capacity-slru-v1.md`](./shared-capacity-slru-v1.md) | `A1-003` / `A1-G03` |
| [`no-hit-overhead-v1.md`](./no-hit-overhead-v1.md) | `A0-G06` |
| [`test-cache-isolation-v1.md`](./test-cache-isolation-v1.md) | `A0-G08` |
| [`managed-l1-v1.md`](./managed-l1-v1.md) | `A0-003` |
| [`single-node-shared-storage-v1.md`](./single-node-shared-storage-v1.md) | `A0-004` |
| [`self-hosted-single-node-config-v1.md`](./self-hosted-single-node-config-v1.md) | `A2-001` |
| [`self-hosted-service-install-v1.md`](./self-hosted-service-install-v1.md) | `A2-002` |
| [`self-hosted-upgrade-restart-v1.md`](./self-hosted-upgrade-restart-v1.md) | `A2-003` |
| [`self-hosted-manual-restore-v1.md`](./self-hosted-manual-restore-v1.md) | `A2-004` |
| [`self-hosted-single-node-gate-v1.md`](./self-hosted-single-node-gate-v1.md) | `A2-G01` / MVP-A2 |
| [`pending-commit-cas-v1.md`](./pending-commit-cas-v1.md) | `A0-005` |
| [`local-authenticated-cache-v1.md`](./local-authenticated-cache-v1.md) | `A0-006` |
| [`gradle-bootstrap-cache-v1.md`](./gradle-bootstrap-cache-v1.md) | `A0-007` |
| [`export-gateway-v1.md`](./export-gateway-v1.md) | `A0-008` |
| [`causal-pilot-v1.md`](./causal-pilot-v1.md) | `A0-009` |
| [`jvm-agent-spike-v1.md`](./jvm-agent-spike-v1.md) | `SPK-002` |
| [`hermetic-helper-spike-v1.md`](./hermetic-helper-spike-v1.md) | `SPK-003` |
| [`release-bundle-v1.md`](./release-bundle-v1.md) | `F0-038` / `DEPLOY-001` |
| [`deployment-lifecycle-v1.md`](./deployment-lifecycle-v1.md) | `DEPLOY-001` |
| [`ops-readiness-v1.md`](./ops-readiness-v1.md) | `OPS-001/A1` |
| [`ops-alerts-v1.md`](./ops-alerts-v1.md) | `OPS-001/A1` |
| [`walking-skeleton-faults-v1.md`](./walking-skeleton-faults-v1.md) | `WS-008` |
| [`walking-skeleton-overhead-v1.md`](./walking-skeleton-overhead-v1.md) | `WS-009` |

Each specification must link fixtures or conformance tests and the RFC decision it refines. `F0-010` reserves these paths without creating empty specifications.

`ci-orchestration-v1.json` is the machine-readable scheduling, isolation,
budget, and recovery corpus consumed by the F0-030 conformance checker.
`commit-atomicity-v1.json` is the F0-031 transaction fault/replay plan backing
ADR 0002.
`test-optimization-integration-v1.json` is the shared F0-033
producer/consumer scenario corpus.
`patch-bundle-v1.json` is the ordered F0-034 application and recovery plan
consumed by the Java patcher spike.
`bandit-policy-v1.json` is the deterministic F0-035 policy/replay corpus.
`capability-matrix-v1.json` is the current evidence-backed F0-036 Tier 1
status matrix.
`data-lifecycle-v1.json` is the F0-037 retention, redaction, buffering, and
deletion contract.
`tier-one-cache-policy-v1.json` is the restriction-only A0-002 runtime,
task/action, transform, and fallback allowlist.
`tier-one-cache-conformance-v1.json` is the A0-G01 backend, gateway, Gradle
client, retry, corruption, and default-deny compatibility matrix.
`l1-l2-revocation-v1.json` is the A0-G02 committed-L2/native-L1 generation,
authenticated revocation, miss/rotation, and aborted-writer isolation
contract.
`gateway-rotation-v1.json` is the A0-G03 stable process restart, complete local
identity rotation, Configuration Cache, transient upstream authority, and
concurrent-slot isolation contract.
`gateway-spool-v1.json` is the A0-G04 complete pre-200 verification, bounded
reservation, disk/cancellation/checksum fault, and managed-process crash
cleanup contract.
`shared-commit-recovery-v1.json` is the A0-G05 real filesystem/SQLite WAL
contract for concurrent commit CAS, all-object visibility atomicity, digest
audit repair, and safe orphan/missing/expired recovery.
`shared-capacity-slru-v1.json` is the A1-003/A1-G03 hard-quota, durable TTL,
byte-weighted SLRU, conservative reservation, and high/low watermark contract.
`edge-cache-config-v1.json` is the C2-001 private single-node configuration,
origin transport, bounded storage, and immutable Shared-only authority
contract. It deliberately opens no Edge route yet.
`edge-cache-committed-read-v1.json` is the C2-002 authenticated Shared hit,
complete durable publication, exact current-revocation offline-read, corruption
fallback, and restart contract.
`edge-cache-capacity-slru-v1.json` is the C2-003 hard byte quota,
conservative reservation, durable TTL, byte-SLRU pressure, and schema migration
contract.
`edge-cache-pending-replication-v1.json` is the C2-004 exact-attempt pending
visibility, authenticated asynchronous Shared replication, durable
retry/restart, TTL, and no-local-promotion contract.
`edge-cache-two-node-proxy-v1.json` is the C2-005 IPv4-loopback route,
committed-first and exact-attempt fallback, and owner-controlled two-Edge
central-collision proof.
`edge-cache-gate-v1.json` is the C2-G01 current-tree constituent, invariant,
boundary, and final two-node runtime composition contract.
`edge-operability-v1.json` is the POC-O1 standalone process, signed hot reload,
private aggregate status, signed-bundle, reproducible-systemd-unit, and
graceful-lifecycle composition contract.
`owner-poc-lab-v1.json` composes the synthetic Gradle lane, repeated Shared
fault evidence, two-Edge Shared collision proof, and complete Edge operability
gate into one JSON-reporting POC-O2 command that base CI runs without owner
repositories, external partners, or the deferred soak.
`build-history-api-v1.json` is the UX-F1-001 authenticated loopback read model
over immutable redacted BUILD_SESSION exports, including exact list/detail
operations, filters, stable cursor pagination, private-file limits, and the
explicit boundary that leaves Test Optimization unchanged.

`build-history-dashboard-v1.json` is the UX-F1-002 embedded local interface
contract, including memory-only API authentication, exact/loaded-only filter
semantics, source-backed summaries/details, security headers, responsive and
accessible states, and the explicit absence of fabricated analytics or Test
Optimization behavior.

`no-hit-overhead-v1.json` is the A0-G06 paired strict-runner contract for
authenticated read-only L2 misses, fresh L1/output state, long-session p95
budgets, and pre-outcome L2 omission with zero short-session requests.
`test-cache-isolation-v1.json` is the A0-G08 fail-closed no-grant contract for
root, actual `buildSrc`, and included-plugin `Test` tasks, including a usable
authenticated remote-cache control and exact zero-request guarded proof.
`deployment-lifecycle-v1.json` is the DEPLOY-001 contract for externally
verified immutable versions, atomic selection, persistent data, and the
install/upgrade/rollback/uninstall lifecycle.
`ops-readiness-v1.json` is the first OPS-001/A1 slice for live-before-ready
startup, fail-closed application routing, shutdown draining, and signed
authority reload within 60 seconds.
`ops-alerts-v1.json` is the OPS-001/A1 ten-class local alert contract for
bounded storage, authority, export, and acceptance signals plus deterministic
activation/recovery without exposing sensitive values.
`private-beta-token-isolation-v1.json` is the A1-002/A1-G01 contract for
hashed 30-day credentials, exact repository/namespace/plane/operation scopes,
per-request revocation, remote TLS, gateway-only token handling, and GitHub
fork isolation.
`private-beta-data-lifecycle-v1.json` is the A1-004/A1-G05 contract for
pre-persistence HMAC redaction, explicit bounded diagnostic profiles,
logical-before-physical whole-deployment deletion, active managed leases,
tokenized downstream obligations, and enforced Shared/L1 generation floors.
`private-beta-operations-v1.json` is the A1-005 composition contract for the
isolated profile's readiness/revocation, ten-class local alert surface,
runner circuit fallback, and bypass/rollback/uninstall procedures. Its bounded
exercise explicitly excludes the eight-hour soak and external pilot evidence.
`owner-controlled-pilot-deployment-v1.json` binds the private synthetic pilot
repository, signed installed release, deterministic workload, authenticated
managed-L1 replay, schema-valid sessions, and explicit non-causal boundary
that closes A1-001 without closing A1-006 or A1-G06.
`owner-poc-evaluation-v1.json` binds the two immutable public pilot revisions,
the paired alternating design, exact required distribution, causal lower bound,
p95 limit, and zero-divergence/failure acceptance used to close A1-006/A1-G06.
`beta-benchmark-harness-v1.json` is the OPS-001/A1 executable smoke contract
for all phase/client strata, real Shared HTTP publication/read paths, private
raw observations, digest-bound summaries, and explicit non-qualification.
`beta-disk-faults-v1.json` is the exact benchmark-bound high-watermark and
out-of-space fault slice with raw trigger/recovery observations, zero-body-read
admission rejection, byte eviction to low, and tamper-evident validation.
`beta-shared-faults-v1.json` is the exact benchmark-bound cancellation,
integrity, SQLite contention, lease-expiry, and pending/commit process-death
slice with 17 private trigger/recovery observations.
`beta-system-faults-v1.json` is the exact benchmark-bound gateway/server
restart, network latency/loss, and signed policy/grant revocation slice with
18 private trigger/recovery observations.
`beta-sustained-v1.json` is the exact one-hour 1/8/32-client benchmark slice
through the real managed gateway and Shared data plane, with 30,000 private
observations, strict golden-runner qualification, and boundary-specific p95
targets.
`beta-soak-v1.json` is the exact eight-hour 1/8/32-client stability slice
through one long-lived managed gateway, Shared store, and authority, with
30,000 private observations and the same strict runner and p95 boundaries.
`beta-circuit-breaker-v1.json` is the A1-G02 flood, oversized-object, and
disk-pressure circuit slice: private per-slot state suppresses Shared between
invocations, preserves writable managed L1, and proves Kotlin/Groovy Gradle
fallback and replay without claiming the separate soak or fixture-size matrix.
`beta-gradle-fixtures-v1.json` is the benchmark-bound small/medium/large
Kotlin DSL build matrix: deterministic multi-project repositories prove exact
known outputs, ordered critical paths, managed-L1 replay, and Configuration
Cache reuse without claiming performance qualification or the separate soak.
`managed-l1-v1.json` is the A0-003 launcher/settings-plugin contract for
opaque scope binding, native retention, generation directories, exclusive
leases, and L2-writer local disablement.
`single-node-shared-storage-v1.json` is the A0-004 server/filesystem contract
for private immutable blobs, one process writer, separate WAL-mode
cache/control schemas, and fail-closed startup.
`self-hosted-single-node-config-v1.json` is the A2-001 strict declarative
configuration and pre-listener storage-preflight contract for the isolated
single-node profile.
`self-hosted-service-install-v1.json` is the A2-002 signed-release, private
layout, path-only secret, deterministic systemd-unit, and reproducible fresh
installation contract.
`self-hosted-upgrade-restart-v1.json` is the A2-003 serialized signed-upgrade,
rollback-safe descriptor composition, unchanged persistent-data restart, and
pending-object invisibility contract.
`self-hosted-manual-restore-v1.json` is the A2-004 absent-target offline
snapshot, cryptographic recovery-authority comparison, strict generation
rotation, atomic publication, and fail-closed admission contract.
`self-hosted-single-node-gate-v1.json` is the A2-G01 current-source composite
that closes MVP-A2 only when configuration, installation, upgrade/restart, and
manual restore all pass together.
`build-impact-manifest-v1.json` is the C3-001 strict customer-authority
boundary for repository/pipeline binding, enumerated original and alternative
entrypoint sets, required artifacts/checks, global paths, and mandatory
`FULL_GRAPH` fallback.
`build-impact-declared-graph-v1.json` is the C3-002 strict
manifest-digest-bound Gradle graph and shadow decision contract: affected
projects include reverse dependents, every required artifact/Build-owned check
must remain reachable, Test-owned checks stay untouched, and all unknown or
global cases run the original full graph.
`build-impact-shadow-validation-v1.json` is the C3-003 immutable observation
and result contract for full-baseline shadow evidence, isolated paired
controls, exact project/artifact/check comparison, explicit false negatives,
and infrastructure/baseline `INCONCLUSIVE` outcomes; every result keeps active
selection disabled.
`build-impact-promotion-gate-v1.json` is the C3-004 exact BIA-002 evidence
gate: current manifest/graph/adapter binding, 30-day and 3,000-decision
minimums, 99% validation coverage, 100 controls per mandatory stratum,
one-sided zero-failure confidence bounds, and immediate suspension on one false
negative. Current checked-in evidence remains honestly `INCONCLUSIVE`, and a
qualified report still cannot activate selection.
`build-impact-selection-v1.json` is the C3-005 sole active omission boundary:
it rechecks canonical manifest/graph digests and bound BIA-002 observations,
selects only a customer-authorized graph alternative, preserves Test-owned
checks separately, and restores original entrypoints on disablement, bypass,
kill switch, drift, insufficient/suspended evidence, or conservative graph
fallback.
`pending-commit-cas-v1.json` is the A0-005 lifecycle contract for durable
pending attempts, canonical Ed25519 decisions, atomic first-writer visibility,
context-bound opaque HTTP GET/PUT, quarantine, and startup reconciliation.
`build-impact-gate-v1.json` is the C3-G01 current-tree composite that closes
owner-operated MVP-C3 only when manifest, graph, shadow/control, unchanged
BIA-002, fail-closed selection, and the offline synthetic full-versus-selected
proof all pass together without mutating source.
`local-authenticated-cache-v1.json` is the A0-006 trust and routing contract
for canonical local authority, monotonic policy/revocation generations,
current-state Shared authorization, gateway credential translation, and the
managed Gradle `HttpBuildCache`.
`gradle-bootstrap-cache-v1.json` is the A0-007 launcher contract for signed
read-only dependency snapshots, per-runner writable homes and leases,
independent Wrapper checksum verification, and native distribution reuse.
`export-gateway-v1.json` is the A0-008 complete/partial BUILD_SESSION contract
for private bounded JSONL, deterministic at-least-once replay, startup
recovery, and stdout export.
`causal-pilot-v1.json` is the A0-009 pre-outcome paired-assignment, neutral
observation, deterministic bootstrap, preliminary result, and internal
net-savings gate contract.

The additional materialized contract `golden-lane-runner-v1.json` pins the first runner class, toolchain, image, and checksums consumed by validation scripts. `release-bundle-v1.md` fixes the first verifiable Linux AMD64 distribution; `deployment-lifecycle-v1.md` owns its local install, upgrade, rollback, and uninstall behavior without claiming publication or online revocation. `walking-skeleton-overhead-v1.md` fixes the first non-promotional baseline-versus-wrapper measurement without replacing the later beta benchmark.
