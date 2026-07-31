# Gradle Build Optimization

Monorepo for the autonomous Gradle build optimization platform. The product observes real builds, applies only qualified and reversible optimizations, preserves Gradle behavior as the baseline, and measures net time savings inside the neutral measurement envelope.

The repository has completed **MVP-A0** and is preparing the isolated
**MVP-A1** private beta. It contains the internal managed-cache path, exact
Tier 1 safety and lifecycle gates, causal measurement, locked toolchains, and
a verifiable Linux AMD64 release bundle plus its local install, upgrade,
rollback, and uninstall lifecycle; it does not yet contain the external pilot
or its complete operational readiness profile.

## Sources of truth

1. [Master RFC](./gradle-build-optimization-platform.md): product intent, architecture, invariants, and gates.
2. [Implementation tracker](./implementation-tracker.md): status, dependencies, and execution evidence.
3. `contracts/`, `specs/`, `benchmarks/`, and `adr/`: executable contracts and technical decisions materialized in Phase 0 and extended by later milestones.

An RFC example is not an executable contract. If a contract contradicts an RFC invariant, correct the contract or revise the corresponding decision first.

## Monorepo map

| Path | Artifact or responsibility | Stack | First activating item |
|---|---|---|---|
| `cmd/buildopt/` | Launcher and Local Verifying Cache Gateway | Go | `WS-001` |
| `cmd/buildopt-server/` | Modular monolith for cache, policy, evidence, and export | Go | `WS-005` |
| `internal/` | Private Go packages shared by both binaries | Go | `WS-001` |
| `jvm/gradle-plugin/` | Gradle plugin, adapters, and handshake | Java 17 | `WS-003` |
| `jvm/jvm-agent/` | Opt-in Instrumentation Agent | Java 17 | `SPK-002` |
| `jvm/generated-client/` | Generated control-plane transport bindings | Java 17 | `F0-022` |
| `jvm/patcher/` | Customer-side `PatchBundle` validator and applier | Java 17 | `SPK-004` |
| `rust/hermetic-helper/` | Experimental optional Linux helper | Rust | `SPK-003` |
| `contracts/` | JSON Schema, OpenAPI, Protobuf, and vectors | Multi-language | `F0-010` |
| `specs/` | Executable cross-component specifications | Markdown + fixtures | `F0-030` |
| `benchmarks/` | Workloads, seeds, budgets, and fault matrices | Data + harness | `F0-032` |
| `fixtures/` | Reproducible integration repositories and scenarios | Gradle/Kotlin DSL | `F0-040` |
| `adr/` | Technical decisions that refine the RFC | Markdown | `F0-002` |
| `runbooks/` | Executable recovery and operator procedures | Markdown + shell exercises | `F0-039` |
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
./dev/check-normative-layout
./dev/check-ownership
./dev/check-base-ci --static
./dev/check-generated-code
./dev/check-generated-clients
./dev/check-build-session-schema
./dev/check-experiment-action-schemas
./dev/check-metrics-catalog
./dev/check-build-session-export
./dev/check-protobuf-toolchains
./dev/check-task-events-proto
./dev/check-contract-crypto
./dev/check-http-semantics
./dev/check-state-machines
./dev/check-ci-orchestration
./dev/check-commit-atomicity
./dev/check-beta-benchmark
./dev/check-test-optimization-openapi
./dev/check-test-optimization-integration
./dev/check-patch-bundle-spec
./dev/check-patch-bundle-applier
./dev/check-bandit-policy
./dev/check-capability-matrix
./dev/check-data-lifecycle
./dev/check-tier1-fixtures
./dev/check-tier-one-policy
./dev/check-tier-one-cache-conformance
./dev/check-l1-l2-revocation
./dev/check-gateway-rotation
./dev/check-gateway-spool
./dev/check-shared-commit-recovery
./dev/check-no-hit-overhead
./dev/check-test-cache-isolation
./dev/check-managed-l1
./dev/check-shared-storage
./dev/check-pending-commit
./dev/check-local-authority
./dev/check-gradle-bootstrap-cache
./dev/check-export-gateway
./dev/check-causal-pilot
./dev/check-jvm-agent-spike
./dev/check-hermetic-helper-spike
./dev/check-buildopt-cli
./dev/check-local-gateway
./dev/check-managed-gateway
./dev/check-session-ingest
./dev/check-walking-skeleton-faults
./dev/check-walking-skeleton-overhead
./dev/check-gradle-plugin-handshake
./dev/run -- ./dev/check-gradle-correlation-fixture
./dev/check-toolchains-lock
./dev/doctor
./dev/check-phase-zero
./dev/test-doctor
./dev/test-jdk-toolchain
./dev/test-go-toolchain
./dev/test-protobuf-toolchains
./dev/test-rust-toolchain
./dev/test-lint-toolchains
./dev/test-supply-chain-toolchains
./dev/test-toolchain-lifecycle
./dev/test-golden-lane-container
./dev/run --toolchain go -- ./dev/check-go-toolchain
./dev/check-rust-toolchain --verify-manifest
./dev/check-lint-toolchains
./dev/check-supply-chain-toolchains
./dev/uninstall-toolchains --toolchain go
./dev/check-release-package
./dev/check-deployment-lifecycle
./dev/check-ops-readiness
./dev/check-ops-alerts
./dev/check-github-action
./dev/check-base-runbooks
./dev/run -- ./dev/check-jvm-release
./dev/check-golden-lane --static
```

The commands validate the repository and Phase 0 normative-package layouts, the independent `BUILD_SESSION`, `EXPERIMENT_RESULT`, and `ACTION_RECORD` Draft 2020-12 contracts with positive, negative, and linked lifecycle fixtures, the versioned METRICS-001/MEASURE-001 catalog with fixed units and signs, reproducible project-local Protobuf tooling and the local task-event channel with Go/Java Unix-socket round trips, generated-artifact policy and source/output drift, the real `buildopt run --` passthrough binary and its Linux process/signal contract, the control-plane-independent local bypass and base recovery exercises, the authenticated loopback gateway and managed runner-slot lifecycle, the bounded fully verified gateway GET spool and its fault recovery, locally signed policy and cumulative revocation generations, the current-authority Shared route, Gradle `HttpBuildCache` credential translation and Configuration Cache inputs, the Gradle plugin handshake, Tier 1 default-deny cache policy, and private generation-segmented native L1, authenticated idempotent session ingest into the real `buildopt-server`, preserved failure/cancellation classification with complete invocation cleanup, atomic schema-valid `BUILD_SESSION v1` JSON, bounded deterministic JSONL/stdout delivery, immutable schema-valid partial recovery, and a pre-outcome paired managed-L1 pilot with deterministic bootstrap and a preliminary immutable `EXPERIMENT_RESULT`, the first external neutral-envelope baseline-versus-wrapper overhead report, the strict A0 no-hit overhead budget with authenticated forced misses and a pre-outcome L2-omission short path, the first parallel Gradle correlation fixture, portable toolchain lock, host inventory contract, isolated JDK, Go, ShellCheck, actionlint, Cosign, and Syft provisioning, pinned Go and Rust toolchains, Java 17 JVM artifacts, read-only push/PR base CI for Go, Java 17, and optional Rust, deterministic signed release bundles with SPDX and provenance, fail-closed side-by-side deployment with upgrade/rollback and explicit preserve-or-purge uninstall, the full-SHA/checksum-pinned Linux x64 setup Action, repository shell scripts and workflow fixtures, and golden lane configuration and checksums.

Project-local smoke tests:

```bash
./dev/bootstrap --toolchain temurin-jdk-21
./dev/run -- ./dev/check-golden-lane --smoke
./dev/bootstrap --toolchain go
./dev/run --toolchain go -- ./dev/check-go-toolchain
./dev/bootstrap --toolchain protoc
./dev/bootstrap --toolchain buf
./dev/check-protobuf-toolchains
./dev/bootstrap --toolchain shellcheck
./dev/bootstrap --toolchain actionlint
./dev/check-lint-toolchains
./dev/bootstrap --toolchain cosign
./dev/bootstrap --toolchain syft
./dev/check-supply-chain-toolchains
```

Provisioning is idempotent. `./dev/uninstall-toolchains --toolchain <id>`
removes one lock-owned installation while preserving downloads and build state;
`--all --purge-downloads --purge-state` is the explicit full local cleanup.
See [the development-tool contract](./dev/README.md#cleanup-and-uninstall)
before purging state.

## GitHub Action

Consumers pin the Action itself by its complete commit SHA and pin the release
archive independently:

```yaml
- name: Install BuildOpt
  id: buildopt
  uses: tonyredondo/buildopt@3fe068790878420a2a9e1d84b6ae5fc83f5752c3
  with:
    version: <release-version>
    archive-url: https://example.invalid/buildopt-<release-version>-linux-amd64.tar.gz
    archive-sha256: <lowercase-sha256>

- name: Run the existing Gradle command
  shell: bash
  run: >-
    buildopt run --
    ./gradlew
    --init-script "$BUILDOPT_GRADLE_INIT_SCRIPT"
    build
```

The Action supports GitHub-hosted or self-hosted Linux x64 runners, requires
HTTPS for the archive and redirects, verifies the complete SHA-256 and exact
Release Bundle v1 TAR layout before extraction, and installs under
`runner.temp`. It adds only the packaged `bin/` directory to subsequent-step
`PATH`; the server, plugin, agent, installation root, and pinned init script are
also exposed as outputs and non-secret environment paths.

The SHA-256 value is expected to come from a separately authenticated Release
Bundle v1 publication. The Action does not turn an unauthenticated checksum
supplied by the same download location into a trust root. See the
[WS-007 fixture](./fixtures/github-actions/README.md) for the immutable test
pins and scope boundaries.

For immediate recovery, set `BUILDOPT_BYPASS=1` on the launcher invocation.
The launcher consumes that value and runs the original command without the
plugin, gateway, or configured server path. The
[base recovery runbook](./runbooks/base-recovery.md) covers the CI kill switch,
immutable rollback, uninstall, and explicit state preservation or purge.

Smoke test inside the pinned image:

```bash
./dev/run-golden-lane-container --smoke
./dev/run-golden-lane-container --require-runner-class
```

The runner resolves the immutable image index, verifies its Linux AMD64 platform digest, inspects the pulled image, and executes it without tag fallback. Only `--require-runner-class` also enforces and verifies the contractual 4 vCPU/16 GiB development configuration.
