# Executable specifications

Operational contracts connecting multiple components: CI orchestration, Gradle correlation, Test Optimization integration, PatchBundle, and the capability matrix.

| Specification | Owning item |
|---|---|
| [`ci-orchestration-v1.md`](./ci-orchestration-v1.md) | `F0-030` |
| [`gradle-correlation-v1.md`](./gradle-correlation-v1.md) | `SPK-001` / `GRADLE-CORR-001` |
| [`benchmark-beta-v1.md`](./benchmark-beta-v1.md) | `F0-032` |
| [`test-optimization-integration-v1.md`](./test-optimization-integration-v1.md) | `F0-033` |
| [`patch-bundle-v1.md`](./patch-bundle-v1.md) | `F0-034` |
| [`bandit-policy-v1.md`](./bandit-policy-v1.md) | `F0-035` |
| [`capability-matrix-v1.md`](./capability-matrix-v1.md) | `F0-036` |
| [`data-lifecycle-v1.md`](./data-lifecycle-v1.md) | `F0-037` |
| [`tier-one-cache-policy-v1.md`](./tier-one-cache-policy-v1.md) | `A0-002` |
| [`managed-l1-v1.md`](./managed-l1-v1.md) | `A0-003` |
| [`single-node-shared-storage-v1.md`](./single-node-shared-storage-v1.md) | `A0-004` |
| [`pending-commit-cas-v1.md`](./pending-commit-cas-v1.md) | `A0-005` |
| [`local-authenticated-cache-v1.md`](./local-authenticated-cache-v1.md) | `A0-006` |
| [`jvm-agent-spike-v1.md`](./jvm-agent-spike-v1.md) | `SPK-002` |
| [`hermetic-helper-spike-v1.md`](./hermetic-helper-spike-v1.md) | `SPK-003` |
| [`release-bundle-v1.md`](./release-bundle-v1.md) | `F0-038` / `DEPLOY-001` |
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

The additional materialized contract `golden-lane-runner-v1.json` pins the first runner class, toolchain, image, and checksums consumed by validation scripts. `release-bundle-v1.md` fixes the first verifiable Linux AMD64 distribution without claiming the later install, upgrade, uninstall, revocation, or workflow lifecycle. `walking-skeleton-overhead-v1.md` fixes the first non-promotional baseline-versus-wrapper measurement without replacing the later beta benchmark.
