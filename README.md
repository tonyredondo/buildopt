# Gradle Build Optimization

Monorepo for the autonomous Gradle build optimization platform. The product observes real builds, applies only qualified and reversible optimizations, preserves Gradle behavior as the baseline, and measures net time savings inside the neutral measurement envelope.

The repository is in **Phase 0**. This initial skeleton defines technical ownership and module boundaries; it does not yet contain active optimizations or distributable artifacts.

## Sources of truth

1. [Master RFC](./gradle-build-optimization-platform.md): product intent, architecture, invariants, and gates.
2. [Implementation tracker](./implementation-tracker.md): status, dependencies, and execution evidence.
3. `contracts/`, `specs/`, `benchmarks/`, and `adr/`: executable contracts and technical decisions as they materialize during Phase 0.

An RFC example is not an executable contract. If a contract contradicts an RFC invariant, correct the contract or revise the corresponding decision first.

## Monorepo map

| Path | Artifact or responsibility | Stack | First activating item |
|---|---|---|---|
| `cmd/buildopt/` | Launcher and Local Verifying Cache Gateway | Go | `WS-001` |
| `cmd/buildopt-server/` | Modular monolith for cache, policy, evidence, and export | Go | `WS-005` |
| `internal/` | Private Go packages shared by both binaries | Go | `WS-001` |
| `jvm/gradle-plugin/` | Gradle plugin, adapters, and handshake | Java 17 | `WS-003` |
| `jvm/jvm-agent/` | Opt-in Instrumentation Agent | Java 17 | `SPK-002` |
| `jvm/patcher/` | Customer-side `PatchBundle` validator and applier | Java 17 | `SPK-004` |
| `rust/hermetic-helper/` | Experimental optional Linux helper | Rust | `SPK-003` |
| `contracts/` | JSON Schema, OpenAPI, Protobuf, and vectors | Multi-language | `F0-010` |
| `specs/` | Executable cross-component specifications | Markdown + fixtures | `F0-030` |
| `benchmarks/` | Workloads, seeds, budgets, and fault matrices | Data + harness | `F0-032` |
| `fixtures/` | Reproducible integration repositories and scenarios | Gradle/Kotlin DSL | `F0-040` |
| `adr/` | Technical decisions that refine the RFC | Markdown | `F0-002` |
| `dev/` | Bootstrap, doctor, local execution, and validation | Shell | `ENV-001..012` |
| `.github/` | CI and protected workflows | GitHub Actions | `F0-004` |

Rust is not a core requirement. The JVM Agent provides instrumentation, not a sandbox. `Test` tasks retain Test Optimization policy and ownership.

## Build conventions

- Run Gradle exclusively through the Gradle Wrapper pinned by the golden lane; do not depend on a global installation.
- Resolve toolchains and utilities from `dev/toolchains.lock.yaml` and run them through `dev/run`; do not replace global toolchains.
- JVM components produce Java 17 bytecode and are validated with at least JDK 17, 21, and 25 according to the capability matrix.
- Go contains the launcher, gateway, and control/data plane. Rust remains isolated in the optional hermetic helper.
- Never edit generated code manually, and require CI to detect drift.
- Do not announce or activate a capability before closing its gate and linking evidence in the tracker.

Concrete versions, the Gradle Wrapper, and build manifests are introduced by `F0-002` and `ENV-*`; `F0-001` does not pin them prematurely.

## Available validation

From the repository root:

```bash
./dev/check-layout
./dev/check-toolchains-lock
./dev/doctor
./dev/test-doctor
./dev/test-jdk-toolchain
./dev/test-go-toolchain
./dev/test-rust-toolchain
./dev/test-golden-lane-container
./dev/run --toolchain go -- ./dev/check-go-toolchain
./dev/check-rust-toolchain --verify-manifest
./dev/run -- ./dev/check-jvm-release
./dev/check-golden-lane --static
```

The commands validate the repository layout, portable toolchain lock, host inventory contract, isolated JDK and Go provisioning, the pinned Go module/toolchain, Java 17 JVM artifacts, and golden lane configuration and checksums.

Project-local smoke tests:

```bash
./dev/bootstrap --toolchain temurin-jdk-21
./dev/run -- ./dev/check-golden-lane --smoke
./dev/bootstrap --toolchain go
./dev/run --toolchain go -- ./dev/check-go-toolchain
```

Smoke test inside the pinned image:

```bash
./dev/run-golden-lane-container --smoke
./dev/run-golden-lane-container --require-runner-class
```

The runner resolves the immutable image index, verifies its Linux AMD64 platform digest, inspects the pulled image, and executes it without tag fallback. Only `--require-runner-class` also enforces and verifies the contractual 4 vCPU/16 GiB development configuration.
