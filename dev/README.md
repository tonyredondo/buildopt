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

`ENV-003` introduced the exact Temurin JDK 21 artifact required by the golden lane. `ENV-005` added the exact Go toolchain required by the core, `ENV-006` added exact Protobuf tooling, `ENV-010` added the locked lint tools used by repository scripts and future workflows, and `ENV-011` added Cosign and Syft for verifiable releases. Provision a supported target from the repository root:

```bash
./dev/bootstrap --toolchain temurin-jdk-21
./dev/bootstrap --toolchain go
./dev/bootstrap --toolchain protoc
./dev/bootstrap --toolchain buf
./dev/bootstrap --toolchain shellcheck
./dev/bootstrap --toolchain actionlint
./dev/bootstrap --toolchain cosign
./dev/bootstrap --toolchain syft
```

The bootstrap downloads the immutable URL from the lock, verifies its SHA-256 before extraction, rejects unsafe archive paths, handles the locked binary, ZIP, `tar.gz`, and `tar.xz` layouts, runs target-specific version and runtime probes, and installs atomically under `.tools/toolchains/`. A second invocation verifies and reuses the existing installation without another download. It never uses `sudo` or modifies global tools.

Set `BUILDOPT_TOOLS_ROOT` to keep the ignored tool state in another local directory. The repository lock remains the source of truth. Every successful bootstrap marks that root so cleanup cannot be redirected at an arbitrary existing directory.

## Cleanup and uninstall

Remove one locked installation while retaining its verified download and all
project-local state:

```bash
./dev/uninstall-toolchains --toolchain go
```

Remove every toolchain managed directly by `dev/bootstrap`:

```bash
./dev/uninstall-toolchains --all
```

Both forms are idempotent. They derive exact installation names from the lock,
refuse an unmarked or unsafe tools root, and fail before deleting anything when
a selected tool has an active bootstrap lock. Provider-managed state such as
the optional Rustup toolchain is outside this command.

Downloads and build state are preserved by default so an uninstall does not
silently discard caches or evidence. Purge them only with an explicit choice:

```bash
./dev/uninstall-toolchains --all --purge-downloads
./dev/uninstall-toolchains --all --purge-downloads --purge-state
```

`--purge-downloads` removes only the exact archives named by the current lock.
`--purge-state` is accepted only with `--all` and removes `.tools/state/` plus
`.tools/gradle-user-home/`; it does not remove unrelated files in the tools
root. Set the same `BUILDOPT_TOOLS_ROOT` and
`BUILDOPT_TOOLCHAINS_LOCK_FILE` values used for provisioning when operating on
an alternate root or test lock.

Run the complete two-bootstrap, uninstall, reinstall, cache, state, and
concurrency contract without touching the real tools root:

```bash
./dev/test-toolchain-lifecycle
```

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

Run either Protobuf tool without depending on the workstation `PATH`:

```bash
./dev/run --toolchain protoc -- protoc --version
./dev/run --toolchain buf -- buf --version
```

For both tools, `dev/run` verifies the provisioned manifest, upstream layout,
and exact reported version before prepending only that tool's repository-local
`bin` directory for the child process.

Run either lint tool without depending on a global installation:

```bash
./dev/run --toolchain shellcheck -- shellcheck --version
./dev/run --toolchain actionlint -- actionlint -version
```

For both tools, `dev/run` verifies the provisioned manifest and exact reported version, then prepends only that tool's repository-local `bin` directory for the child process. The parent shell and any global lint installation remain unchanged.

Run either supply-chain tool without depending on a global installation:

```bash
./dev/run --toolchain cosign -- cosign version
./dev/run --toolchain syft -- syft version
```

`dev/run` verifies exact Cosign 3.1.2 and Syft 1.50.0 identities before exposing only the selected binary to the child. Release signing uses an explicit local configuration and never treats a public transparency service as an implicit dependency.

## Go toolchain validation

The root [`go.mod`](../go.mod) declares module `github.com/tonyredondo/buildopt`, the Go 1.26.0 language baseline, and exact `go1.26.5` toolchain. `ENV-005` established that compiler contract before product packages existed; `WS-001` and `WS-002` provide the dependency-free process launcher, `WS-003` adds its local handshake receiver, and `WS-004` adds the standard-library-only authenticated loopback gateway without changing the toolchain or module graph.

Run the checker through the isolated toolchain:

```bash
./dev/run --toolchain go -- ./dev/check-go-toolchain
```

The checker requires Linux AMD64, exact locked provenance and version, local-only toolchain selection, disabled user Go configuration, project-local caches, and an unchanged module graph. It then builds and executes a standard-library-only smoke program twice offline and requires identical binaries.

## `buildopt` CLI validation

Build the real `buildopt` binary and run the `WS-001..004` launcher unit and integration suite:

```bash
./dev/check-buildopt-cli
```

The checker runs with the locked Go toolchain and offline module resolution. It executes helpers through the built CLI and verifies exact argument boundaries without shell expansion, inherited working directory/environment/standard streams, fresh reserved rendezvous context and cleanup, zero and non-zero child statuses, usage code `64`, cannot-execute code `126`, command-not-found code `127`, and the early `BUILDOPT_BYPASS=1` path. Unit coverage also validates accepted and rejected v1 frames, invocation matching, size bounds, event authentication, and private socket cleanup.

On Linux, the same suite verifies that the direct child leads a process group separate from the launcher, a nested descendant receives forwarded `SIGINT` and `SIGTERM`, cancellation waits for delayed cleanup without a launcher-owned deadline, handled child statuses remain authoritative, an unhandled `SIGTERM` becomes status `143`, and bypass retains that process-group contract. Other platforms remain outside the current acceptance matrix.

## Authenticated local gateway validation

Exercise the neutral `WS-004` gateway and real launcher with the race detector:

```bash
./dev/check-local-gateway
```

The gateway binds only an operating-system-assigned `127.0.0.1` endpoint and
exposes one authenticated readiness route. Its Basic credential is local-only,
its response binds the current `gatewayConnectionGeneration`, and cache data
routes return `404`. The checker proves that a restart retains endpoint,
credential, and generation; concurrent slots receive distinct identities and
reject each other's credentials; attacker-controlled parent values are
replaced; and the endpoint is gone after the child exits.

## Session ingest validation

Build the real launcher and `buildopt-server`, exercise their authenticated
`WS-005` integration, and run the involved packages under the race detector:

```bash
./dev/check-session-ingest
```

The checker starts the server on an operating-system-assigned loopback port,
delivers successful and failed child sessions through the active gateway, and
requires the server to observe the exact outcomes while the launcher preserves
exit `0` and `37`. It proves that the server token is absent from the child and
diagnostics, unavailable delivery remains fail-open, absent server configuration
performs no ingest contact, and `SIGTERM` shuts down the server
cleanly. Unit coverage adds strict JSON/authentication/size negatives,
generation binding, idempotent replay, conflicting duplicate rejection, and
concurrent acceptance.

## BUILD_SESSION export validation

Execute real successful and failed Gradle 9.6.1 builds through the launcher,
authenticated gateway, and server, then validate both exported documents:

```bash
./dev/check-build-session-export
```

The `WS-006` checker supplies strict pre-outcome tokenized workload context,
requires one authenticated plugin invocation per build, and preserves Gradle
exit `0`/`1`. The server must atomically publish exactly two private mode-`0600`
JSON files without temporary residue or credential leakage. The isolated
`build-session-validator` compiles the normative Draft 2020-12 schema with
format assertions and accepts both documents; focused assertions retain the
requested tasks, outcomes, exits, explicit unavailable observations, and
partial failure timing.

## Experiment and action lifecycle validation

Validate the independent `F0-012` aggregate-result and action-transition
contracts, including their cross-record authorization rules:

```bash
./dev/check-experiment-action-schemas
```

The isolated Draft 2020-12 validator checks every positive and negative
`EXPERIMENT_RESULT` and `ACTION_RECORD` fixture. It then resolves bounded
testdata-only lifecycle references and verifies version ancestry, ordered
windows and intervals, sample reconciliation, transition preconditions, and
exact result linkage. Activation succeeds only when the referenced immutable
result is actually `FINAL` with decision `PROMOTE`; policy-only shadow entry
and safety rollback remain distinct paths. The documents are audit records,
not executable authorization messages.

## Evidence, policy, and resource-profile validation

Validate the `F0-013` evidence, policy, and finite resource-profile contracts:

```bash
./dev/check-foundation-contract-schemas
```

The same isolated validator checks strict positive/negative fixtures and the
four-arm golden-runner catalog. Cross-record checks bind policy to evidence and
the selected profile, verify the policy time window and cgroup headroom, and
reject treatment changes beyond workers and Gradle heap.

## Attempt and atomic commit validation

Validate the `F0-014` attempt, isolated CI request, and `CommitDecision`
contracts:

```bash
./dev/check-attempt-commit-schemas
```

The checker rejects skipped CAS states, shared candidate/control L1 state, and
inconclusive commit authorization. Linked vectors additionally prove ordered
idempotent transitions, single ownership, terminal abort behavior, positive
validation, and exact coverage of every pending cache object.

## Test Optimization signed-schema validation

Validate the `F0-015` signed grant and validation-result contracts:

```bash
./dev/check-test-optimization-schemas
```

The checker requires closed Ed25519/JCS envelopes, explicit grant selectors
and capabilities, disjoint final status shapes, valid time windows, exact
policy/grant references, and result identity/artifact equality with the
originating F0-014 request. It validates structure and semantic binding;
`F0-020` supplies real cryptographic golden vectors.

## Contract cryptography validation

Validate the shared `F0-020` JCS, SHA-256, UTC timestamp, and Ed25519 vectors:

```bash
./dev/check-contract-crypto
```

The command runs dependency-free Go and Java 17 consumers over the exact same
language-neutral TSV rows. It verifies canonical UTF-8 bytes, UTF-16 member
ordering, IEEE-754 number rendering, lowercase digests, UTC-only RFC 3339
timestamps, and real Ed25519 verification. Negative vectors cover duplicate
keys, malformed UTF-8, unpaired surrogates, non-finite numeric input, changed
payloads, wrong keys, and malformed signatures.

## HTTP failure-semantics validation

Validate the common `F0-021` error, deadline, retry, idempotency, unknown
response, and cancellation contract:

```bash
./dev/check-http-semantics
```

The checker validates the machine-readable stable-error catalog, executes
fault cases for retry success, payload conflict, deadline exhaustion, unknown
state, accepted cancellation, non-idempotent refusal, fail-closed policy
timeout, terminal errors, and capped backoff, then audits every operation in
all three OpenAPI documents against the same allowed codes and outcomes.

## Generated client and compatibility validation

Regenerate and validate the `F0-022` Go and Java 17 control-plane clients:

```bash
./dev/generate-code --artifact openapi-go-client-v1
./dev/generate-code --artifact openapi-java-client-v1
./dev/check-generated-clients
```

The generator extracts all 13 operation IDs, methods, paths, and contract
versions from the three normative OpenAPI documents and binds output drift to
those documents plus the compatibility corpus. Both generated clients perform
one bounded HTTPS attempt without hiding retry policy. The checker regenerates
all tracked outputs in isolation, compiles both clients, exercises real Go TLS
transport/header/path safety, and runs the same N/N-1, incompatible-major, and
unknown-field vectors in Go and Java 17.

## Lifecycle state-machine validation

Validate the executable `F0-023` task-qualification, action-rollout, and
attempt state machines:

```bash
./dev/check-state-machines
```

The checker cross-checks action and attempt states against their normative
JSON Schemas and interprets the language-neutral scenario catalog. The
scenarios prove ordered qualification and promotion, fail-closed
inconclusive results, atomic dependency suspension, rollback, compare-and-swap
rejection, exact idempotent replay after a lost response, payload-conflict
rejection, cancellation, dead-owner reconciliation, and terminal-state
safety.

## CI orchestration validation

Validate the `F0-030` authoritative-job, protected validation queue,
isolation, budget, and recovery contract:

```bash
./dev/check-ci-orchestration
```

The checker cross-checks the attempt lifecycle, inspects the inert GitHub
fixture, and interprets 12 cases. It rejects multiple authoritative arms,
untrusted revisions, shared writable state, daily or weekly overspend, and a
second concurrent lease; cancellation, timeout, unknown task boundaries, and
dead owners release unused reservations without changing the normal job.

## Single-node commit atomicity validation

Validate the executable fault plan for ADR 0002:

```bash
./dev/check-commit-atomicity
```

The checker proves that the immutable `CommitDecision` and every
`COMMITTED` visibility row form one all-or-nothing cache transaction. Thirteen
cases cover replay/conflict, invalid authority, blob and transaction crash
points, corrupt or missing content, first-writer CAS, and independent
`control.sqlite` reconciliation.

## Private-beta benchmark contract validation

Validate the `F0-032` seed, runner, object mix, load phases, Gradle fixture
classes, fault matrix, and required result surface:

```bash
./dev/check-beta-benchmark
```

The checker verifies all immutable input bindings and prints the manifest
digest. It does not execute the 60-minute sustained or eight-hour soak profile;
those measured runs remain part of the later `OPS-001/A1` gate.

## Bounded bandit replay validation

Validate the finite resource catalog and `F0-035` epsilon-greedy policy:

```bash
./dev/check-bandit-policy
```

The deterministic replay covers A/A and sample-ratio failure, fixed-cohort
entry, stable-control and exploration floors, shrinkage and tie handling,
headroom, propensity, 24-hour delayed and duplicate outcomes, drift/epoch
reset, kill switch, and guardrail rollback.

## Capability matrix validation

Validate the evidence-backed `F0-036` Tier 1 target/status matrix:

```bash
./dev/check-capability-matrix
```

The checker requires all ten Gradle/JDK/DSL combinations and validates exact,
approximated, and unavailable records with their methods, reasons, evidence,
and safe fallbacks. Untested rows cannot inherit the golden-lane profile.

## Data lifecycle and redaction validation

Validate the `F0-037` retention, profile authorization, keyed redaction,
JSON/JSONL, bounded spool, and managed-deletion contract:

```bash
./dev/check-data-lifecycle
```

The checker recomputes exact HMAC tokens for four export profiles, scans every
managed golden output for raw sensitive values, validates at-least-once
deduplication and partial sequence recovery, rejects changed event reuse, and
executes eight deletion-order/boundary cases.

## PatchBundle contract validation

Validate the `F0-016` declarative bundle envelope and its two private-beta
recipe vectors:

```bash
./dev/check-patch-bundle-schema
```

The isolated checker compiles the strict Draft 2020-12 schema, validates exact
UTF-8 replacement blobs and the normative sorted bundle digest, and rejects 12
schema or semantic mutations covering paths, operations, modes, commands,
preimages, blobs, signature binding, digest binding, and duplicate ordering.
It does not apply a bundle to Git or execute bundle content; those parser,
worktree, symlink/submodule, idempotency, and recovery proofs remain with
`F0-034`/C4.

Validate the ordered `F0-034` application and recovery plan with:

```bash
./dev/check-patch-bundle-spec
```

The specification checker requires strict trust/source/blob verification,
link-safe staged application, exact pre/postimages, isolated validation, and
new-branch/draft-PR idempotency. Its 15 cases become the executable acceptance
matrix for the Java patcher spike.

## BuildOpt OpenAPI validation

Validate the `F0-017` BuildOpt control and internal cache-control APIs:

```bash
./dev/check-buildopt-openapi
```

The isolated contract module loads both OpenAPI 3.1 documents with external
JSON Schema references enabled, performs full document validation, checks the
TLS/bearer/contract-version/idempotency/precondition/deadline/cancellation/
retry/error policy on every operation, and rejects any opaque cache-payload
route in the control API. Its in-process mock validates every request and
response, exercises all nine operations, proves byte-equivalent exact replay,
and returns a schema-valid conflict when a key is reused with another payload.

The validator is pinned in `dev/schema-validator/go.mod`; it does not add a
dependency to the product module. Complete fault/retry vectors, generated
clients, N/N-1 compatibility, and durable queue/cache transactions remain with
their later tracker items.

## Test Optimization OpenAPI validation

Validate the `F0-018` producer/consumer API:

```bash
./dev/check-test-optimization-openapi
```

The isolated checker loads the OpenAPI 3.1 contract and its signed grant/result
schemas, enforces the common TLS, bearer, deadline, cancellation, retry,
idempotency, and stable-error policy, and exercises grant resolution, current
grant status, delayed validation submission, and final polling through a
request/response-validating mock. It also proves exact replay and rejects an
idempotency key reused with another payload.

## Test Optimization integration validation

Run the full shared producer/consumer conformance surface for `F0-033`:

```bash
./dev/check-test-optimization-integration
```

Sixteen fixtures cover current/N-1 grants, missing/expired/untrusted authority,
revocation and status failures, delayed polling, exact and conflicting
retries, corrupt or caller-path artifacts, failed/inconclusive results, and an
incompatible major. The command composes those cases with the OpenAPI,
signed-schema, crypto, HTTP-semantics, and generated-client suites.

## Metrics catalog validation

Validate the machine-readable `F0-024` catalog and its private-beta measurement
policy:

```bash
./dev/check-metrics-catalog
```

The dependency-free validator requires all 35 core session, effect, overhead,
tail, driver, correctness, and coverage definitions. Each definition carries
the complete RFC §22.9 governance fields and may use only bounded dimensions,
explicit `COMPLETE | PARTIAL | UNAVAILABLE` state, and
`EXACT | APPROXIMATED | UNAVAILABLE` methods. Negative fixtures reject missing
or duplicate metrics, reversed saved/delta signs, zero-filled unavailable
values, high-cardinality dimensions, and MEASURE-001 policy drift. The same
catalog version is emitted by the `BUILD_SESSION` producer.

## Walking-skeleton fault validation

Exercise the complete `WS-008` bypass, failure/cancellation, and cleanup
contract with real binaries:

```bash
./dev/check-walking-skeleton-faults
```

The checker records an ordinary exit `37` as `BUILD_FAILURE`, forwards
`SIGTERM` through a two-process child group whose cleanup exit is `42`, records
that invocation as `CANCELLED`, and verifies the child tree, plugin attempt
directory/socket, and loopback gateway are all gone. The optimization-off
walking skeleton has no cache data route, so it creates no cache lease. A final
early-bypass case returns exit `38` without adding a server record. The same
suite runs inside `dev/golden-lane-build` with the host-built locked binaries.

## Walking-skeleton overhead validation

Measure the complete optimization-off wrapper against native Gradle from the
same external envelope:

```bash
./dev/check-walking-skeleton-overhead
```

The `WS-009` checker runs four real `neutralProbe` pairs in alternating order,
removes the required output before every arm, requires byte-identical
deliverables, and retains the first pair plus signed negative differences. The
wrapper arm includes launcher, authenticated plugin/gateway rendezvous, server
ingest, and export; the native arm includes none of them. The generated report
binds all measurement inputs by SHA-256 and always keeps
`promotionGateActive: false`.

A host run is a non-qualifying smoke. The strict golden-container run records
the evidence report once under `benchmarks/results/`; subsequent runs validate
that historical report without rewriting it. See
[`walking-skeleton-overhead-v1.md`](../specs/walking-skeleton-overhead-v1.md).

## Gradle plugin handshake validation

Build the packaged `dev.buildopt` plugin and exercise its neutral authenticated
`WS-003`/`WS-004` rendezvous through the real Gradle 9.6.1 Wrapper:

```bash
./dev/check-gradle-plugin-handshake
```

The checker compares deterministic task output against a direct baseline,
requires the plugin to authenticate gateway readiness and the Unix event
channel before one accepted v1 `ProducerHello` on two invocations, and proves
that the second invocation reuses Configuration Cache while its task is
up-to-date. It also verifies that inherited rendezvous values are replaced, a
missing rendezvous leaves the Gradle build successful, an intentional Gradle
failure retains exit code `1`, and, when Git is available, the working tree is
unchanged. The same check runs inside `dev/golden-lane-build`; the pinned image
does not contain Git, so the host run owns that last read-only assertion.

## Gradle correlation fixture validation

Run the first `F0-040` fixture with the locked JDK 21 and real Gradle 9.6.1 Wrapper:

```bash
./dev/run -- ./dev/check-gradle-correlation-fixture
```

The checker uses an isolated project cache, output tree, and local build cache. On the initial empty-cache run, two equivalent cacheable tasks in separate projects must execute concurrently and expose the same native Gradle cache key. After cleaning their outputs, the second identical run must reuse Configuration Cache and restore both tasks from the local build cache with byte-identical outputs.

Run the closed `SPK-001` matrix on the golden version:

```bash
./dev/run -- ./dev/check-gradle-correlation-spike --gradle-9-only
```

Run the complete Gradle 9.6.1 and checksum-pinned 8.14.3 matrix:

```bash
./dev/run -- ./dev/check-gradle-correlation-spike --full
```

The spike starts a loopback HTTP Build Cache, exercises direct tasks, both
Worker API isolation modes, a real child JVM, cache hit/miss, failure,
cancellation, and Configuration Cache, then correlates Gradle operation IDs
structurally with native keys and observed HTTP `PUT` requests. Cold Kotlin DSL
work emits non-task stores, so the accepted result is the fail-closed
`UNAVAILABLE` capability: every `UNATTRIBUTED` store aborts the complete
attempt. See [`specs/gradle-correlation-v1.md`](../specs/gradle-correlation-v1.md).

## Tier 1 fixture matrix

Run the `F0-040` consumer repositories through both TestKit and a real
Wrapper:

```bash
./dev/check-tier1-fixtures
```

The checker executes Kotlin and Groovy DSL on Gradle 8.14.3 and 9.6.1 with
JDK 17 and the locked JDK 21. Every row loads the packaged product plugin,
runs a Java 17 cacheable custom task and artifact transform, then proves build
cache and Configuration Cache reuse. Distribution archives are
checksum-pinned and temporary Wrapper/user homes are isolated. JDK 25 remains
explicitly unproven until its lock-owned runtime is provisioned.

## JVM Agent spike

Run the bounded agent against a real Gradle daemon and Configuration Cache:

```bash
./dev/check-jvm-agent-spike
```

The checker uses the locked JDK 21 and Gradle 9.6.1 Wrapper. It executes all
six access classes, then verifies the accepted `UNAVAILABLE` result: class
loads are not method-level access evidence, so every report is incomplete and
pending publication aborts. Capacity overflow and transformer conflict retain
the task output; an injected `premain` crash fails only its isolated diagnostic
daemon and a fresh uninstrumented daemon reproduces the baseline. The printed
warm timing is descriptive and does not activate a promotion gate.

## Local task-event Protobuf validation

`F0-019` materializes the correlation result as the normative
[`task_events.proto`](../contracts/proto/local-events/v1/task_events.proto) and
[ADR 0003](../adr/0003-local-task-event-channel.md). `ENV-006` provisions exact
`protoc` 35.1 and Buf 1.72.0 from their immutable locked artifacts.

Provision and validate the complete toolchain plus protocol round trip:

```bash
./dev/bootstrap --toolchain protoc
./dev/bootstrap --toolchain buf
./dev/check-protobuf-toolchains
```

The integrated checker resolves both tools only from `.tools/`, runs Buf
`STANDARD` lint, compares the source descriptor produced by Buf and `protoc`
byte-for-byte, compiles the Java peer with the locked JDK as Java 17 bytecode,
and runs the standard-library Go peer with locked Go. The two directions
exchange conventional varint-length-delimited messages over real Unix sockets.
They cover exact attribution, `UNATTRIBUTED`, attempt-wide `UNAVAILABLE`, atomic
whole-attempt abort, acknowledgements, invalid semantic combinations, and the
1 MiB frame bound. Generated control-plane clients and their compatibility
vectors are validated separately by `./dev/check-generated-clients`.

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

The checker runs exact ShellCheck 0.11.0 over executable scripts under `dev/`, `.github/actions/`, and `fixtures/github-actions/`. It then runs exact actionlint 1.7.12 over any existing `.github/workflows/*.yml` or `*.yaml` files and an in-memory valid workflow fixture. actionlint receives the exact repository-local ShellCheck path for embedded `run:` scripts and has opportunistic global Pyflakes discovery disabled.

`ENV-010` owns this reproducible lint tooling. `F0-004` consumes it in the
authoritative base workflow described below.

## Base CI validation

The read-only [base workflow](../.github/workflows/base-ci.yml) runs on pushes
to `main`, pull requests, and manual dispatch. Validate its immutable Action
pins, exact Ubuntu runner, Java/Rust versions, triggers, permissions, and lane
commands without executing a build:

```bash
./dev/check-base-ci --static
```

The core lane requires the locked JDK 21, Go, protoc, ShellCheck, and
actionlint installations plus a checksum-primed Gradle Wrapper. It runs layout,
ownership, lock, shell/workflow lint, generated-artifact drift detection,
race-enabled Go tests, Go vet, the nested schema validator module, Java 17
bytecode verification, and the JVM agent on both Java 21 and the exact Java
17.0.19 compatibility runtime:

```bash
./dev/run -- ./gradlew --no-daemon --version
BUILDOPT_JAVA17_HOME=/path/to/jdk-17 \
    ./dev/check-base-ci --lane core
```

The separately named Rust lane keeps the helper optional to the core while
still making its accepted compiler baseline a required check whenever the
workflow runs:

```bash
rustup toolchain install \
    1.93.0-x86_64-unknown-linux-gnu \
    --profile minimal \
    --no-self-update
./dev/check-base-ci --lane rust
```

Both lanes require an unchanged tracked working tree. This workflow closes
only base `F0-004` validation; protected scheduling, validation queues,
cross-run lifecycle, budgets, and recovery remain with `CI-ORCH-001`.

## Supply-chain and release validation

Provision the exact release tools and exercise the offline local-key profile:

```bash
./dev/bootstrap --toolchain cosign
./dev/bootstrap --toolchain syft
./dev/check-supply-chain-toolchains
./dev/test-supply-chain-toolchains
```

The `ENV-011` checker uses a temporary test key to sign and verify a blob with Cosign 3.1.2, requires a Sigstore bundle v0.3 with no transparency-log or timestamp entries, and generates an SPDX 2.3 document with Syft 1.50.0. The synthetic suite proves checksum-verified binary and `tar.gz` provisioning, normalized layouts, idempotency, isolation from global tools, metadata drift rejection, usage codes, and child failure propagation without contacting upstream services.

Exercise the complete `F0-038` bundle twice and run tamper cases:

```bash
./dev/check-release-package
```

The checker snapshots the current source into a temporary clean Git repository, builds the Linux AMD64 launcher/server and versioned Java 17 plugin/agent twice, and requires byte-identical TAR, SPDX, SLSA provenance, release manifest, and checksum manifest. Both Cosign signatures must bind the same digest. Verification rejects modified payloads, modified signatures, an unpinned public key, and extra files. See [`specs/release-bundle-v1.md`](../specs/release-bundle-v1.md) for the production command and trust boundary.

## GitHub Action validation

Exercise the Action metadata, immutable fixture lock, offline installer, and
manual workflow:

```bash
./dev/test-github-action-install
./dev/check-github-action
```

The installer suite regenerates the synthetic Release Bundle v1 TAR, intercepts
the HTTPS request without network access, and proves checksum-before-extract,
exact entries/types/modes, atomic publication, matching-install reuse, GitHub
outputs/environment/PATH, literal argv, exit `37`, checksum mismatch, platform
rejection, mutable-install rejection, and invalid input handling.

The integrated checker proves that every `uses:` reference is a full 40-hex
commit, the BuildOpt commit contains the exact Action and archive, the raw
archive URL is bound to that commit, the SHA-256 matches its Git object, the
workflow is manual-only with `contents: read`, and locked
ShellCheck/actionlint accept all scripts and the hosted fixture. The hosted run
is dispatched separately for GitHub-runner evidence. This closes `WS-007`, not
`F0-004`, `CI-ORCH-001`, release publication, token/fork policy, or the full
`DEPLOY-001` lifecycle.

## Base recovery runbook validation

Execute every `F0-039` base recovery drill:

```bash
./dev/check-base-runbooks
```

The checker builds the real launcher and proves that
`BUILDOPT_BYPASS=1` is consumed before invalid server context can be parsed or
any plugin/gateway context reaches the child. It also clears the CI-style kill
switch to recover the normal rendezvous, runs bypassed process-tree signal
cleanup under the race detector, rejects a candidate Action archive with a bad
digest, restores the checksum-pinned known-good fixture, and exercises
uninstall with preserved state and with an explicit purge. Every file operation
is confined to a guarded temporary directory, and the Git working tree must
remain unchanged.

The operator procedure and Phase 0 limitations are in
[`runbooks/base-recovery.md`](../runbooks/base-recovery.md). Online revocation,
durable attempt/lease cleanup, and the complete install/upgrade/uninstall
lifecycle remain with `OPS-001/A1`, `WS-008`, and `DEPLOY-001`.

## Normative package validation

Validate the namespace skeleton defined by RFC §29.2:

```bash
./dev/check-normative-layout
```

The checker requires all 14 contract, vector, specification, benchmark, and ADR namespaces, their non-empty indexes, and parent directories for the 26 planned normative artifacts. It also preserves the materialized golden-lane and local-channel ADRs, runner and Gradle-correlation specifications, `BUILD_SESSION v1` schema, and task-event Protobuf IDL, and rejects an empty file at any planned artifact path. F0-010 created the structure; each schema, API, IDL, vector, specification, benchmark, or ADR remains owned by its later tracker item.

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

The check compiles both modules with the locked JDK 21, verifies every packaged class is Java 17 bytecode (major 61), validates the `dev.buildopt` plugin marker and agent manifest, and starts a JVM with the packaged no-op agent. Handshake behavior is exercised separately by `dev/check-gradle-plugin-handshake`; agent instrumentation remains reserved for a later gate.

## Golden container validation

Verify the pinned image and run the build without claiming the contractual runner class:

```bash
./dev/run-golden-lane-container --smoke
```

Produce strict 4-CPU/16-GiB runner evidence on a host with sufficient resources:

```bash
./dev/run-golden-lane-container --require-runner-class
```

The runner resolves the immutable image index by digest, requires its unique Linux AMD64 manifest to equal the recorded platform digest, pulls that exact reference, and verifies the local image operating system, architecture, and repository digest. It also builds the walking-skeleton launcher, server, signal helper, metrics validator, and isolated schema validator with the locked Go toolchain and `CGO_ENABLED=0` into a temporary read-only mount, so the JDK-only image can execute the real authenticated rendezvous, session ingest, cancellation cleanup, metric-catalog checks, and `BUILD_SESSION v1` validation without adding an unpinned compiler or `jq`. The subsequent container uses `--pull never`, checks the exact Java patch from the runner specification, and in strict mode verifies effective cgroup v2 CPU and memory limits from inside the container. It never treats the readable source tag as executable identity.

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
./dev/check-protobuf-toolchains
./dev/run -- ./dev/check-gradle-correlation-fixture
./dev/check-toolchains-lock
./dev/test-doctor
./dev/test-jdk-toolchain
./dev/test-go-toolchain
./dev/test-protobuf-toolchains
./dev/test-rust-toolchain
./dev/test-lint-toolchains
./dev/test-supply-chain-toolchains
./dev/check-release-package
./dev/test-github-action-install
./dev/check-github-action
./dev/test-golden-lane-container
```

The validator rejects malformed schema versions, duplicate identities or URLs, unknown platforms, non-HTTPS sources, invalid SHA-256 values, unsupported artifact kinds, and missing or malformed tracker references.

The doctor tests exercise successful and failed host reports, JSON shape, exit codes `0`, `1`, `64`, and `70`, JDK `java`/`javac` probes, and the read-only working-tree invariant.

The JDK toolchain tests use a synthetic archive and isolated tool root. They exercise checksum and manifest-drift rejection, atomic provisioning, idempotency, project-local `JAVA_HOME`/`PATH`, global-Java isolation, missing-tool behavior, usage errors, and child exit-code propagation without downloading or changing the workstation JDK.

The Go toolchain tests use a synthetic archive and isolated tool root. They exercise atomic provisioning, idempotency, exact-version selection, project-local `GOROOT`, `GOPATH`, module/build caches, disabled automatic toolchain switching and user configuration, global-Go isolation, missing-tool behavior, and child exit-code propagation without downloading or changing the workstation Go installation.

The Protobuf toolchain tests use a synthetic upstream-layout ZIP and raw binary in an isolated tool root. They exercise checksum verification, standard-include layout validation, atomic installation, idempotency, exact-version selection, repository-local `PATH`, global-tool isolation, manifest drift, missing tools, usage errors, and child exit propagation without downloading or changing workstation Protobuf tools.

The Rust toolchain tests use synthetic Rustup, rustc, Cargo, and channel-manifest fixtures. They exercise the exact repository override, offline isolated Cargo state, locked manifest verification, missing/mismatched tools, configuration drift, usage errors, and Cargo failure propagation without installing a toolchain or touching the global default.

The lint toolchain tests use synthetic ShellCheck and actionlint archives in their real upstream layouts. They exercise checksum-verified `tar.xz` and `tar.gz` provisioning, atomic installation, idempotency, exact-version selection, repository-local `PATH`, global-tool isolation, manifest drift, lint failure propagation, usage errors, and missing-tool behavior without downloading or changing global tools.

The supply-chain tool tests use synthetic Cosign and Syft artifacts. They exercise checksum-verified binary and `tar.gz` provisioning, atomic installation, idempotency, project-local selection, global-tool isolation, local sign/verify and SPDX paths, metadata drift, checksum rejection, usage errors, and child failure propagation. The release-package checker then uses the real locked tools and build toolchains for deterministic positive and tamper fixtures.

The GitHub Action tests use a deterministic synthetic release archive and fake HTTPS transport to exercise the setup boundary without downloading or executing untrusted product bytes. The integrated fixture then binds the committed Action, archive, checkout dependency, runner, permissions, and manual workflow to exact identities.

The golden container tests use a synthetic Docker client and deterministic host-resource probes. They verify index-to-platform digest binding, exact pull and run arguments, local image identity, strict cgroup settings, mutable-reference rejection, daemon/resource failures, and child exit-code propagation without contacting a registry or starting a container.

## Update policy

Toolchain updates are atomic repository changes:

1. Select an exact upstream release from the official project or its official release repository; moving aliases such as `latest` are forbidden.
2. Record the new version, platform, provider, immutable URL, and upstream SHA-256 in the same change.
3. Keep local paths, usernames, package-manager locations, credentials, mirrors, and workstation-specific state out of the lock.
4. Verify the downloaded bytes against the recorded SHA-256 before provisioning or changing adoption state.
5. Run `./dev/check-toolchains-lock` and every smoke test affected by the tool before updating tracker evidence.

Adding a platform or changing an adopted provider requires explicit compatibility evidence. A checksum-only change for the same immutable URL is treated as a supply-chain conflict and must not be accepted without resolving the upstream discrepancy.
