# Executable specifications

Operational contracts connecting multiple components: CI orchestration, Gradle correlation, Test Optimization integration, PatchBundle, and the capability matrix.

| Specification | Owning item |
|---|---|
| [`ci-orchestration-v1.md`](./ci-orchestration-v1.md) | `F0-030` |
| [`gradle-correlation-v1.md`](./gradle-correlation-v1.md) | `SPK-001` / `GRADLE-CORR-001` |
| [`benchmark-beta-v1.md`](./benchmark-beta-v1.md) | `F0-032` |
| [`beta-benchmark-harness-v1.md`](./beta-benchmark-harness-v1.md) | `OPS-001/A1` |
| [`test-optimization-integration-v1.md`](./test-optimization-integration-v1.md) | `F0-033` |
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
| [`no-hit-overhead-v1.md`](./no-hit-overhead-v1.md) | `A0-G06` |
| [`test-cache-isolation-v1.md`](./test-cache-isolation-v1.md) | `A0-G08` |
| [`managed-l1-v1.md`](./managed-l1-v1.md) | `A0-003` |
| [`single-node-shared-storage-v1.md`](./single-node-shared-storage-v1.md) | `A0-004` |
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
`beta-benchmark-harness-v1.json` is the OPS-001/A1 executable smoke contract
for all phase/client strata, real Shared HTTP publication/read paths, private
raw observations, digest-bound summaries, and explicit non-qualification.
`managed-l1-v1.json` is the A0-003 launcher/settings-plugin contract for
opaque scope binding, native retention, generation directories, exclusive
leases, and L2-writer local disablement.
`single-node-shared-storage-v1.json` is the A0-004 server/filesystem contract
for private immutable blobs, one process writer, separate WAL-mode
cache/control schemas, and fail-closed startup.
`pending-commit-cas-v1.json` is the A0-005 lifecycle contract for durable
pending attempts, canonical Ed25519 decisions, atomic first-writer visibility,
context-bound opaque HTTP GET/PUT, quarantine, and startup reconciliation.
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
