# Executable specifications

Operational contracts connecting multiple components: CI orchestration, Gradle correlation, Test Optimization integration, PatchBundle, and the capability matrix.

| Specification | Owning item |
|---|---|
| [`ci-orchestration-v1.md`](./ci-orchestration-v1.md) | `F0-030` |
| [`gradle-correlation-v1.md`](./gradle-correlation-v1.md) | `SPK-001` / `GRADLE-CORR-001` |
| [`benchmark-beta-v1.md`](./benchmark-beta-v1.md) | `F0-032` |
| [`test-optimization-integration-v1.md`](./test-optimization-integration-v1.md) | `F0-033` |
| `patch-bundle-v1.md` | `F0-034` |
| `capability-matrix-v1.md` | `F0-036` |
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

The additional materialized contract `golden-lane-runner-v1.json` pins the first runner class, toolchain, image, and checksums consumed by validation scripts. `release-bundle-v1.md` fixes the first verifiable Linux AMD64 distribution without claiming the later install, upgrade, uninstall, revocation, or workflow lifecycle. `walking-skeleton-overhead-v1.md` fixes the first non-promotional baseline-versus-wrapper measurement without replacing the later beta benchmark.
