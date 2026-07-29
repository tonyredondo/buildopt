# Executable specifications

Operational contracts connecting multiple components: CI orchestration, Gradle correlation, Test Optimization integration, PatchBundle, and the capability matrix.

| Specification | Owning item |
|---|---|
| `ci-orchestration-v1.md` | `F0-030` |
| [`gradle-correlation-v1.md`](./gradle-correlation-v1.md) | `SPK-001` / `GRADLE-CORR-001` |
| `benchmark-beta-v1.md` | `F0-032` |
| `test-optimization-integration-v1.md` | `F0-033` |
| `patch-bundle-v1.md` | `F0-034` |
| `capability-matrix-v1.md` | `F0-036` |
| [`release-bundle-v1.md`](./release-bundle-v1.md) | `F0-038` / `DEPLOY-001` |

Each specification must link fixtures or conformance tests and the RFC decision it refines. `F0-010` reserves these paths without creating empty specifications.

The additional materialized contract `golden-lane-runner-v1.json` pins the first runner class, toolchain, image, and checksums consumed by validation scripts. `release-bundle-v1.md` fixes the first verifiable Linux AMD64 distribution without claiming the later install, upgrade, uninstall, revocation, or workflow lifecycle.
