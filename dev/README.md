# Development tools

Reproducible entrypoints for bootstrap, diagnostics, and local execution.

## Toolchain lock

[`toolchains.lock.yaml`](./toolchains.lock.yaml) is the source of truth for downloadable development toolchains on the initial `linux-amd64` platform. It is JSON-compatible YAML 1.2 so the Phase 0 validator can parse it with `jq` before the repository adopts a YAML library.

Every artifact records an exact version, platform, provider, immutable HTTPS URL, SHA-256, adoption status, and the tracker items that require it. Adoption status has these meanings:

- `required`: accepted for the listed tracker items, although provisioning and smoke evidence may still be pending.
- `candidate`: pinned for evaluation but not adopted until its listed decision gate closes.
- `optional`: not required by the core product and provisioned only for its bounded workstream.

Presence in the lock does not close a provisioning item or activate a tool. `dev/bootstrap` will materialize these entries under the repository-local `.tools/` root by default; `dev/doctor` and `dev/run` will verify or consume that state in `ENV-002..012`. These scripts must not use `sudo` or replace global toolchains.

Gradle and the golden container are intentionally delegated to their existing sources of truth:

- `gradle/wrapper/gradle-wrapper.properties` owns the Gradle distribution and checksum.
- `specs/golden-lane-runner-v1.json` owns the golden image and runner contract.

Operating-system capabilities and externally supplied commands such as Docker, Git, `curl`, `jq`, `tar`, `xz`, and `unzip` are host requirements, not downloadable artifacts in this lock. The read-only `dev/doctor` will report them without installing or modifying them.

## Bootstrap

`ENV-003` introduced the exact Temurin JDK 21 artifact required by the golden lane. `ENV-005` added the exact Go toolchain required by the core, and `ENV-010` adds the locked lint tools used by repository scripts and future workflows. Provision a supported target from the repository root:

```bash
./dev/bootstrap --toolchain temurin-jdk-21
./dev/bootstrap --toolchain go
./dev/bootstrap --toolchain shellcheck
./dev/bootstrap --toolchain actionlint
```

The bootstrap downloads the immutable URL from the lock, verifies its SHA-256 before extraction, rejects unsafe archive paths, handles the locked `tar.gz` and `tar.xz` layouts, runs target-specific version and runtime probes, and installs atomically under `.tools/toolchains/`. A second invocation verifies and reuses the existing installation without another download. It never uses `sudo` or modifies global tools.

Set `BUILDOPT_TOOLS_ROOT` to keep the ignored tool state in another local directory. The repository lock remains the source of truth. `ENV-012` will extend this entrypoint to the other adopted toolchains and add the complete cleanup contract.

## Project-local execution

Run a command with the provisioned JDK 21:

```bash
./dev/run -- java -version
./dev/run -- javac -version
./dev/run -- ./dev/check-golden-lane --smoke
```

`dev/run` verifies that the provisioned manifest and Java binaries still match the lock, then sets `JAVA_HOME` and prepends the JDK `bin` directory to `PATH` only for the child process. It preserves the command arguments and exit code. The parent shell and its global `java`/`javac` selection are unchanged.

The optional explicit form is equivalent:

```bash
./dev/run --toolchain temurin-jdk-21 -- java -version
```

Run a command with the provisioned Go toolchain:

```bash
./dev/run --toolchain go -- go version
./dev/run --toolchain go -- ./dev/check-go-toolchain
```

For Go, `dev/run` verifies the locked binary and manifest, disables automatic toolchain switching and the user Go environment file, and supplies project-local module and build caches only to the child process. The parent shell and its global Go selection remain unchanged.

Run either lint tool without depending on a global installation:

```bash
./dev/run --toolchain shellcheck -- shellcheck --version
./dev/run --toolchain actionlint -- actionlint -version
```

For both tools, `dev/run` verifies the provisioned manifest and exact reported version, then prepends only that tool's repository-local `bin` directory for the child process. The parent shell and any global lint installation remain unchanged.

## Go toolchain validation

The root [`go.mod`](../go.mod) declares module `github.com/tonyredondo/buildopt`, the Go 1.26.0 language baseline, and exact `go1.26.5` toolchain. `ENV-005` established that compiler contract before product packages existed; `WS-001` and `WS-002` now provide the first dependency-free launcher behavior without changing the toolchain or module graph.

Run the checker through the isolated toolchain:

```bash
./dev/run --toolchain go -- ./dev/check-go-toolchain
```

The checker requires Linux AMD64, exact locked provenance and version, local-only toolchain selection, disabled user Go configuration, project-local caches, and an unchanged module graph. It then builds and executes a standard-library-only smoke program twice offline and requires identical binaries.

## `buildopt` CLI validation

Build the real `buildopt` binary and run the `WS-001`/`WS-002` launcher integration suite:

```bash
./dev/check-buildopt-cli
```

The checker runs with the locked Go toolchain and offline module resolution. It executes helpers through the built CLI and verifies exact argument boundaries without shell expansion, inherited working directory/environment/standard streams, zero and non-zero child statuses, usage code `64`, cannot-execute code `126`, and command-not-found code `127`.

On Linux, the same suite verifies that the direct child leads a process group separate from the launcher, a nested descendant receives forwarded `SIGINT` and `SIGTERM`, cancellation waits for delayed cleanup without a launcher-owned deadline, handled child statuses remain authoritative, and an unhandled `SIGTERM` becomes status `143`. Other platforms remain outside the current acceptance matrix.

## Rust toolchain validation

The root [`rust-toolchain.toml`](../rust-toolchain.toml) selects `1.93.0-x86_64-unknown-linux-gnu` with Rustup's minimal profile. Rust remains optional for the core, and this pin does not activate the hermetic helper or claim any sandbox capability.

Install the side-by-side toolchain without changing the Rustup default if it is not already available:

```bash
rustup toolchain install 1.93.0-x86_64-unknown-linux-gnu --profile minimal
```

Run the normal offline toolchain and `cargo check` contract:

```bash
./dev/check-rust-toolchain
```

Revalidate the official channel manifest bytes against the repository lock when updating or producing gate evidence:

```bash
./dev/check-rust-toolchain --verify-manifest
```

The checker requires the exact installed compiler, Cargo release, host triple, active repository override, and locked configuration. Its dependency-free Cargo smoke uses temporary `CARGO_HOME` and target directories, disables network access, and leaves the optional helper unimplemented until `SPK-003`. The doctor resolves only an already-installed locked toolchain, so its read-only probe never triggers Rustup auto-installation.

## Lint toolchain validation

Provision both tools and run their integrated smoke:

```bash
./dev/bootstrap --toolchain shellcheck
./dev/bootstrap --toolchain actionlint
./dev/check-lint-toolchains
```

The checker runs exact ShellCheck 0.11.0 over every executable script directly under `dev/`. It then runs exact actionlint 1.7.12 over any existing `.github/workflows/*.yml` or `*.yaml` files and an in-memory valid workflow fixture. actionlint receives the exact repository-local ShellCheck path for embedded `run:` scripts and has opportunistic global Pyflakes discovery disabled.

This is provisioning and lint-smoke evidence for `ENV-010`; it does not create an authoritative CI workflow or close `F0-004`.

## Normative package validation

Validate the namespace skeleton defined by RFC §29.2:

```bash
./dev/check-normative-layout
```

The checker requires all 14 contract, vector, specification, benchmark, and ADR namespaces, their non-empty indexes, and parent directories for the 26 planned normative artifacts. It also preserves the materialized golden-lane ADR, runner contract, and `BUILD_SESSION v1` schema and rejects an empty file at any planned artifact path. F0-010 created the structure; each schema, API, IDL, vector, specification, benchmark, or ADR remains owned by its later tracker item.

## BUILD_SESSION schema validation

Compile the normative `BUILD_SESSION v1` contract and execute every positive and negative fixture:

```bash
./dev/check-build-session-schema
```

The checker runs through the locked Go 1.26.5 toolchain. It uses the exact Draft 2020-12 validator version recorded in [`schema-validator/go.mod`](./schema-validator/go.mod) and its `go.sum`, enables date-time format assertions, and requires each invalid fixture to fail for its intended diagnostic. The isolated test module leaves the product module's offline toolchain smoke unchanged and does not depend on a workstation-global JSON Schema command.

## JVM release validation

Build and inspect the neutral Gradle plugin and JVM agent artifacts:

```bash
./dev/run -- ./dev/check-jvm-release
```

The check compiles both modules with the locked JDK 21, verifies every packaged class is Java 17 bytecode (major 61), validates the agent manifest, and starts a JVM with the packaged no-op agent. It does not activate the plugin handshake or agent instrumentation behavior reserved for later gates.

## Golden container validation

Verify the pinned image and run the build without claiming the contractual runner class:

```bash
./dev/run-golden-lane-container --smoke
```

Produce strict 4-CPU/16-GiB runner evidence on a host with sufficient resources:

```bash
./dev/run-golden-lane-container --require-runner-class
```

The runner resolves the immutable image index by digest, requires its unique Linux AMD64 manifest to equal the recorded platform digest, pulls that exact reference, and verifies the local image operating system, architecture, and repository digest. The subsequent container uses `--pull never`, checks the exact Java patch from the runner specification, and in strict mode verifies effective cgroup v2 CPU and memory limits from inside the container. It never treats the readable source tag as executable identity.

Invalid usage exits `64`, an unavailable daemon or image/build verification failure exits `1`, and a host that cannot enforce the strict runner class exits `2`. The child container's other nonzero status is preserved.

## Doctor

`dev/doctor` inventories the active workstation without changing files, installing packages, starting services, or downloading artifacts. It reports:

- detected operating system, architecture, kernel, CPU, memory, workspace filesystem, and available bytes;
- required host-command paths and versions;
- Docker client/daemon state, versions, storage driver, cgroup version, and resource-limit capabilities;
- observable cgroup, user-namespace, seccomp-process, and Landlock-securityfs state;
- the active `PATH` winner and observed version for each locked toolchain.

The capability fields are inventory only. They do not qualify Landlock, seccomp, or complete process-tree enforcement for C1. Toolchain states are `MATCH`, `MISMATCH`, `MISSING`, or `UNPROBED`; until their own `ENV-*` gates close, they do not make the doctor fail.

Use the human-readable report interactively:

```bash
./dev/doctor
```

Use the versioned JSON report for automation:

```bash
./dev/doctor --json
```

The report schema version is `buildopt.dev/doctor-report/v1`. Exit codes are stable:

| Code | Meaning |
|---:|---|
| `0` | The report was generated and every required host check passed; deferred checks may warn. |
| `1` | The report was generated and at least one required host check failed. |
| `64` | Invalid command-line usage. |
| `70` | The report could not be generated because its lock or required JSON machinery was unavailable or invalid. |

The doctor deliberately probes the active `PATH` and repository state; it does not search arbitrary home directories or infer that an inactive installation is usable. `dev/run` will supply the project-local `PATH` when `ENV-003` and later provisioning items materialize it.

## Validation

Run the lock and doctor contract tests from the repository root:

```bash
./dev/check-normative-layout
./dev/check-build-session-schema
./dev/check-buildopt-cli
./dev/check-toolchains-lock
./dev/test-doctor
./dev/test-jdk-toolchain
./dev/test-go-toolchain
./dev/test-rust-toolchain
./dev/test-lint-toolchains
./dev/test-golden-lane-container
```

The validator rejects malformed schema versions, duplicate identities or URLs, unknown platforms, non-HTTPS sources, invalid SHA-256 values, unsupported artifact kinds, and missing or malformed tracker references.

The doctor tests exercise successful and failed host reports, JSON shape, exit codes `0`, `1`, `64`, and `70`, JDK `java`/`javac` probes, and the read-only working-tree invariant.

The JDK toolchain tests use a synthetic archive and isolated tool root. They exercise checksum and manifest-drift rejection, atomic provisioning, idempotency, project-local `JAVA_HOME`/`PATH`, global-Java isolation, missing-tool behavior, usage errors, and child exit-code propagation without downloading or changing the workstation JDK.

The Go toolchain tests use a synthetic archive and isolated tool root. They exercise atomic provisioning, idempotency, exact-version selection, project-local `GOROOT`, `GOPATH`, module/build caches, disabled automatic toolchain switching and user configuration, global-Go isolation, missing-tool behavior, and child exit-code propagation without downloading or changing the workstation Go installation.

The Rust toolchain tests use synthetic Rustup, rustc, Cargo, and channel-manifest fixtures. They exercise the exact repository override, offline isolated Cargo state, locked manifest verification, missing/mismatched tools, configuration drift, usage errors, and Cargo failure propagation without installing a toolchain or touching the global default.

The lint toolchain tests use synthetic ShellCheck and actionlint archives in their real upstream layouts. They exercise checksum-verified `tar.xz` and `tar.gz` provisioning, atomic installation, idempotency, exact-version selection, repository-local `PATH`, global-tool isolation, manifest drift, lint failure propagation, usage errors, and missing-tool behavior without downloading or changing global tools.

The golden container tests use a synthetic Docker client and deterministic host-resource probes. They verify index-to-platform digest binding, exact pull and run arguments, local image identity, strict cgroup settings, mutable-reference rejection, daemon/resource failures, and child exit-code propagation without contacting a registry or starting a container.

## Update policy

Toolchain updates are atomic repository changes:

1. Select an exact upstream release from the official project or its official release repository; moving aliases such as `latest` are forbidden.
2. Record the new version, platform, provider, immutable URL, and upstream SHA-256 in the same change.
3. Keep local paths, usernames, package-manager locations, credentials, mirrors, and workstation-specific state out of the lock.
4. Verify the downloaded bytes against the recorded SHA-256 before provisioning or changing adoption state.
5. Run `./dev/check-toolchains-lock` and every smoke test affected by the tool before updating tracker evidence.

Adding a platform or changing an adopted provider requires explicit compatibility evidence. A checksum-only change for the same immutable URL is treated as a supply-chain conflict and must not be accepted without resolving the upstream discrepancy.
