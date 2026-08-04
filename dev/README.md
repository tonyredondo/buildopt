# Development tools

Reproducible entrypoints for bootstrap, diagnostics, and local execution.

New contributors should begin with the
[developer onboarding guide](../docs/getting-started/developer-onboarding.md).
The [validation reference](../docs/reference/validation.md) groups the commands
below by subsystem so a change can run the smallest useful proof.

Validate documentation entry points, local links, referenced commands, and Go
package boundaries with:

```bash
./dev/check-documentation
```

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

The unmanaged compatibility gateway binds only an operating-system-assigned
`127.0.0.1` endpoint and exposes one authenticated readiness route. Its Basic
credential is local-only,
its response binds the current `gatewayConnectionGeneration`, and cache data
routes return `404`. The checker proves that a restart retains endpoint,
credential, and generation; concurrent slots receive distinct identities and
reject each other's credentials; attacker-controlled parent values are
replaced; and the endpoint is gone after the child exits.

## Managed gateway lifecycle validation

Exercise the internal `A0-001` production path with real launcher and detached
gateway processes:

```bash
./dev/check-managed-gateway
```

The race-enabled checker uses private temporary state roots and proves strict
configuration, mode-`0700` directories and mode-`0600` identity state,
current-user control registration, exactly one active invocation per runner
slot, baseline fallback for a busy slot, distinct concurrent slots, and
rejected cross-slot credentials. It also proves that readiness returns `503`
without a current context, an idle process restart retains
endpoint/credential/generation, and an occupied retained port rotates all
three before readiness. Cache data routes and remote credentials remain
absent.

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

## Private-beta managed data lifecycle

Exercise the implemented A1-004/A1-G05 boundary:

    ./dev/check-private-beta-data-lifecycle

The checker proves that BUILD_SESSION exports apply deployment-keyed HMAC
redaction before JSON/JSONL persistence, SUMMARY is the default, wider
profiles require explicit authorization, and DIAGNOSTIC expires within seven
days. It also exercises the operator deletion command across Shared, managed
L1, exports, evidence, and spool: active leases refuse deletion, logical
revocation precedes physical removal, the retained tombstone contains no raw
scope or destination identities, exact retries are idempotent, and the
persisted Shared/L1 generation floors reject stale recreation.

## PatchBundle contract validation

Validate the `F0-016` declarative bundle envelope and its two private-beta
recipe vectors:

```bash
./dev/check-patch-bundle-schema
```

The isolated checker compiles the strict Draft 2020-12 schema, validates exact
UTF-8 replacement blobs and the normative sorted bundle digest, and rejects 13
schema or semantic mutations covering paths, operations, modes, commands,
preimages, blobs, source-state hashing, signature binding, digest binding, and
duplicate ordering.
It does not apply a bundle to Git or execute bundle content; the Java
integration checker below owns that boundary.

Validate the ordered `F0-034` application and recovery plan with:

```bash
./dev/check-patch-bundle-spec
```

The specification checker requires strict trust/source/blob verification,
link-safe staged application, exact pre/postimages, isolated validation, and
new-branch/draft-PR idempotency. Its 15 cases are the executable acceptance
matrix for the Java patcher:

```bash
./dev/check-patch-bundle-applier
```

The composite check first reruns the schema, JCS/Ed25519, and application-plan
contracts. It then executes all 15 cases in temporary real Git repositories:
both recipes, exact replay, source divergence, expired/untrusted authority,
traversal, symlink and gitlink boundaries, postimage rollback, immutable
branch conflict, branch-without-PR recovery, existing draft replay, and
interrupted staging. The checker verifies Java 17 bytecode and that no
customer checkout/index, extra worktree, default branch, or remote is changed.
Because those integration cases require a real host Git, `patcherSpike` stays
outside the standard Gradle `check` lifecycle. The composite checker invokes
it explicitly, and the core base-CI lane invokes that checker; the JDK-only
golden image still performs the standard compile, package, and check lifecycle
without acquiring an unpinned Git installation.

### Full relevant validation gate

Validate the C4-006 composition of local patch correctness and the generated
Test Optimization client with:

```bash
./dev/check-full-relevant-validation
```

The checker proves explicit policy applicability, no-contact NOT_REQUIRED, an
artifact-bound FULL_RELEVANT_VALIDATION request, one exact retry, bounded
deadline-aware polling, and pinned JCS/Ed25519 result verification. Only a
trusted current PASSED result allows promotion; every failure or inconclusive
condition blocks without remote mutation.

### Customer patch workflow

Validate the inert protected-branch workflow and draft-only delivery boundary:

```bash
./dev/check-customer-patch-workflow
```

The checker enforces explicit dispatch, a fork/default-branch guard, pinned
actions, minimal job permissions, one-step token exposure, exact signed-bundle
identity, complete PR narrative, PRELIMINARY labeling, and the absence of
rebase, force-push, default-branch write, ready, or merge operations.

### Patch delivery recovery

Validate exact retries and branch-without-PR recovery with:

```bash
./dev/check-patch-delivery-recovery
```

The six-state contract preserves an exact action branch after PR failure,
creates only the missing draft on retry, reuses an exact draft, and rejects
every identity/head/PR conflict without overwrite, deletion, rebase, or merge.

### Post-merge patch monitoring

Validate contextual/causal post-merge classification and the signed inverse
draft-revert path with:

```bash
./dev/check-post-merge-patch-monitor
```

The checker retains natural failures, runs only budgeted exact inverse controls,
uses deterministic paired intervals plus p95 and correctness guardrails, and
passes an exact MODIFY-only inverse through the production signer, verifier,
path-safe applier, and immutable draft workflow without touching default.

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
JDK 17 and the locked JDK 21, plus Gradle 9.6.1 with the locked JDK 25. Every
row loads the packaged product plugin,
runs a Java 17 cacheable custom task and artifact transform, then proves build
cache and Configuration Cache reuse. Distribution archives are
checksum-pinned and temporary Wrapper/user homes are isolated.

Run the `A0-002` restriction-only policy across the same ten proven rows:

```bash
./dev/check-tier-one-policy
```

The policy checker gives every scenario its own TestKit and build-cache home.
It proves exact core source-set `JavaCompile` replay, re-execution of `Test`
on the Gradle 9 rows and a custom cacheable task on every row, rejection of a
built-in with an added action, and invocation-wide managed-cache disablement
when a transform is registered. The Gradle 8 empty-test path is not executed
because its framework autoload emits a deprecation under the deliberately
strict warning gate.
The policy plugin never configures or enables a cache; `A0-003` owns that
integration.

Run the closed `A0-G01` HTTP cache compatibility and fault matrix across the
same ten rows:

```bash
./dev/check-tier-one-cache-conformance
```

The checker composes the race-enabled backend and gateway cases with real
Gradle `HttpBuildCache` clients. It proves cold miss/PUT, clean replay,
configuration-cache reuse, early 413 without replacing an existing object,
redirect and timeout normalization, corruption quarantine, retry safety, and
default-deny behavior for modified built-ins, custom tasks, and unknown
transforms. The L2-to-L1 commit/revocation/abort composition is the separate
gate immediately below.

Run the closed `A0-G02` L2-to-L1 revocation and abort lifecycle on the golden
Gradle/JDK lane:

```bash
./dev/check-l1-l2-revocation
```

The checker combines a real authenticated backend commit/revocation/abort
sequence with Kotlin and Groovy TestKit builds. It proves committed L2 replay
populates native L1, same-generation L1 works with remote reads unavailable,
authenticated generation advance forces the next build to miss, and an
aborted pending writer leaves neither a local directory nor a remote hit.
Gateway restart, complete spool faults, production commit fault/recovery, and
physical deletion remain later gates.

Run the closed `A0-G03` managed-gateway restart and rotation gate:

```bash
./dev/check-gateway-rotation
```

The checker starts real concurrent `buildopt` gateway processes with distinct
transient upstream bindings and verifies policy/namespace routing,
cross-credential rejection, and secret-free persistent state. Its golden
Gradle 9.6.1/JDK 21 Kotlin/Groovy TestKit proves Configuration Cache reuse
after a stable gateway restart, one invalidation when endpoint, local
credential, and connection generation rotate together, and reuse after the
rotated entry is stored.

Run the closed `A0-G04` complete verified-spool fault gate:

```bash
./dev/check-gateway-spool
```

The checker proves the gateway receives, bounds, hashes, syncs, and unlinks a
complete upstream hit before downstream `200`. Race-enabled fixtures cover
`ENOSPC`, atomic concurrent byte reservation, partial-body cancellation, a
checksum mismatch discovered after the complete body, and stale-spool cleanup
before a real managed-gateway process registers and serves a verified hit.

Run the closed `A0-G05` production commit and recovery gate:

```bash
./dev/check-shared-commit-recovery
```

The checker cross-validates all 14 Phase 0 atomicity cases against real
filesystem blobs and SQLite WAL. It starts competing commits together, proves
one complete CAS winner and one complete loser, injects a three-object
pre-commit rollback and later control-index failure, repairs audit state by
decision digest, and makes orphaned, missing-blob, and expired-lease states
safe misses.

Run the closed `A0-G06` no-hit overhead gate:

```bash
./dev/check-no-hit-overhead
```

The host path runs two non-qualifying smoke pairs and validates the immutable
strict report. The qualified path runs four alternating native/wrapper pairs
in the pinned 4-CPU/16-GiB container after one warm-up per arm. Every wrapper
long session uses a fresh managed L1 and an authenticated read-only L2 that
returns a controlled `404`; every required JAR is byte-identical, all sessions
last at least five seconds, and nearest-rank p95 must stay within both 500 ms
and 2%. The separate short session omits L2 before execution and proves fewer
than five seconds plus zero remote requests. See
[`no-hit-overhead-v1.md`](../specs/no-hit-overhead-v1.md).

Run the closed `A0-G08` no-grant `Test` cache-isolation gate:

```bash
./dev/check-test-cache-isolation
```

The checker runs the Gradle 9.6.1/JDK 21 Kotlin fixture through an
authenticated observable `HttpBuildCache`. Its unguarded control stores and
restores the root, actual `buildSrc`, and included-plugin `Test` tasks. It then
applies the packaged Tier 1 policy through the init-script boundary and proves
all three execute without a grant, report the explicit no-grant reason, reuse
Configuration Cache, and make exactly zero remote `GET`/`PUT` requests. The
gate does not activate a positive signed grant.

## Managed native L1

Run the `A0-003` launcher/settings-plugin contract across the same eight
Gradle/JDK/DSL rows:

```bash
./dev/check-managed-l1
```

The checker first validates the exact machine contract and race-enabled Go
launcher lifecycle. It then runs Kotlin and Groovy fixtures on Gradle 8.14.3
and 9.6.1 with JDK 17 and 21, proving native cache replay,
default-deny custom-task re-execution, Configuration Cache reuse, and a miss
followed by a hit after `l1SecurityGeneration` rotation. The golden row also
proves malformed-context and L2-writer disablement. A final real
launcher-to-Gradle sequence checks the opaque mode-`0700` directory, exclusive
generation ownership, native cache replay, and rotation without interpreting
Gradle's cache format. The neutral authenticated handshake remains an
independent regression in `./dev/check-gradle-plugin-handshake`.

The checker isolates the native-L1 contract. Compose it with
`./dev/check-local-authority` for signed generations and the managed Shared
route.

## Single-node Shared storage

Validate the `A0-004` server/filesystem substrate:

```bash
./dev/check-shared-storage
```

The checker validates the exact storage contract, then exercises private
same-filesystem SHA-256 publication, concurrent deduplication, complete
read-time integrity, cancellation and oversize cleanup, one server writer,
separate WAL-mode `cache.sqlite`/`control.sqlite` migrations, corruption and
schema-drift refusal, persistence, and clean restart. A final CGO-free real
`buildopt-server` lifecycle verifies the private layout and busy-writer
failure. This substrate checker does not claim publication authority; compose
it with the A0-005 checker below for pending, commit CAS, visibility, and
reconciliation.


## Self-hosted single-node configuration

Validate the first `A2` slice without installing a service:

```bash
./dev/check-self-hosted-single-node-config
```

The checker enforces the strict mode-`0600` declarative contract and example,
then runs the configuration, production Shared preflight/open, and server
integration packages with the race detector and Go vet. The profile accepts
only canonical loopback, summary export, path-only secrets, beta-token
authentication, disjoint absolute paths, a proven-local filesystem, and the
fixed 20 GiB minimum/500 GiB maximum/50% capacity policy. The production
storage opens before the listener. Installation, upgrade/migration, and manual
restore are covered by the separate `A2-002..004` executable contracts.

## Pending publication, commit CAS, and reconciliation

Validate the `A0-005` durable visibility boundary:

```bash
./dev/check-pending-commit
```

The checker validates the exact lifecycle contract and composes the Phase 0
atomicity model with race-enabled real filesystem/SQLite tests. Pending PUTs
remain misses; canonical JCS/Ed25519 decisions must exactly cover the attempt;
decision plus every committed row use one first-writer transaction; exact
replay is idempotent; abort, lease expiry, transaction failure, and CAS loss
publish nothing. Missing/corrupt bytes quarantine the whole decision, orphan
blobs are collected, and a missing `control.sqlite` audit row is repaired at
startup. The context-bound opaque HTTP handler proves early `413` and
full-verification-before-`200` without making an unauthenticated global route.
`A0-006` composes that handler with locally authenticated policy, revocation
state, and routing in the checker below.

## Local policy, revocation, and authenticated cache routing

Validate the `A0-006` trust and routing boundary:

```bash
./dev/check-local-authority
```

The checker validates the canonical machine contract and runs race-enabled Go
tests for JCS/Ed25519 authority, mode-`0600` no-symlink files, repository and
component binding, monotonic policy/revocation/L1/gateway/namespace state,
schema-v3 Shared registration, current-state Bearer routing, local-to-upstream
credential translation, managed same-UID context removal, safe read misses,
and write failure isolation. It builds both Go binaries with the locked
toolchain and runs the golden Gradle 9.6.1/JDK 21 Kotlin and Groovy fixtures
through real `HttpBuildCache` PUT/GET replay and Configuration Cache reuse.

This block does not claim the complete `A0-G01` fault/redirect/timeout matrix,
production commit-decision finalization, or revoked-directory deletion.

## Gradle dependency and Wrapper bootstrap cache

Validate the `A0-007` signed bootstrap boundary across every supported
Gradle/JDK pair:

```bash
./dev/check-gradle-bootstrap-cache
```

The checker proves that a signed `DEPENDENCY_CACHE` action exposes only a
recursively read-only `modules-2` snapshot while each runner retains an
exclusive private writable `GRADLE_USER_HOME`. It independently verifies the
repository Wrapper JAR and distribution archive, installs through Gradle's
native Wrapper URL-hash layout, resolves a real dependency offline without
copying its artifact into the writable layer, and reuses the distribution
after the source archive is removed. Gradle 8.14.3 and 9.6.1 both run on JDK
17 and 21; invalid, unsafe, busy, corrupt, or unsupported inputs preserve the
unmanaged Gradle baseline.

## Complete, partial, and JSONL export

Validate the `A0-008` export boundary:

```bash
./dev/check-export-gateway
```

The checker proves atomic private complete JSON, a bounded mode-`0600` JSONL
stream with two deterministic events per session, byte-identical at-least-once
replay, conflict detection, final-line crash repair, and immutable
schema-valid partial recovery with exact missing ranges. It also runs real
Gradle success and failure sessions and requires the `buildopt-server export`
stdout bytes to match the durable stream exactly. Aggregate experiment effects,
encrypted delivery retry/DLQ, and remote sinks remain separate gates.

## Causal internal-pilot validation

Validate the `A0-009` paired assignment and aggregate-result path:

```bash
./dev/check-causal-pilot
```

The checker generates one deterministic Java workload and warms both isolated
arms. Control uses the real BuildOpt launcher and Tier 1 policy with caching
disabled; candidate uses the same product path with the native managed L1.
Four pairs alternate order, persist each assignment before execution, remove
outputs and project state, require control compilation and candidate
`FROM-CACHE`, and retain eight schema-valid `BUILD_SESSION` documents. The
required JARs must be byte-identical.

The off-critical-path producer then recomputes a deterministic 4,096-resample
paired bootstrap, requires a positive lower 95% saving bound with no output or
failure regression, and publishes one schema-valid `PRELIMINARY`
`EXPERIMENT_RESULT` as immutable mode-`0600` JSON plus bounded byte-exact
JSONL/stdout. This closes only the internal `A0-G09` proof; no-hit overhead,
beta promotion, external-pilot, feedback, queue, p99, and economics gates
remain open.

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

The checker requires the exact installed compiler, Cargo release, host triple,
active repository override, and locked configuration. Its dependency-free
Cargo smoke uses temporary `CARGO_HOME` and target directories and disables
network access. The doctor resolves only an already-installed locked
toolchain, so its read-only probe never triggers Rustup auto-installation.

## Hermetic helper spike

Run the bounded Linux x86-64 producer probe and fallback fixture:

```bash
./dev/check-hermetic-helper-spike
```

The checker compiles and lints the dependency-free Rust helper offline, probes
real user/mount/PID/network namespaces plus advertised kernel mechanisms, and
validates a closed task-specific producer manifest. Because clock,
randomness, environment, and kernel-policy coverage remain incomplete, the
accepted result is `UNAVAILABLE`: the helper does not execute the candidate,
discards it, aborts pending publication, and then proves the same producer can
complete every coverage marker through the authoritative baseline.

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

## Deployment lifecycle validation

Exercise signed installation, upgrade, rollback, and both uninstall policies:

```bash
./dev/check-deployment-lifecycle
```

The `DEPLOY-001` checker packages two real Release Bundle v1 versions from one
clean source revision and consumes them through `dev/manage-deployment`.
Versions remain immutable and side by side, selection is an atomic manifest
rename, and every install or rollback revalidates the complete signed bundle
against an externally supplied public key. The fixture starts the packaged
server, runs the packaged launcher with the packaged Gradle plugin, and proves
that Shared state and immutable exports survive idempotent upgrade, version
upgrade, rollback, default uninstall, and reinstall.

Unsafe roots, unmarked data, a tampered bundle or installed payload, the wrong
trust root, and uninstall while the Shared writer lock is held all fail closed.
Default uninstall preserves the separate data root; `--purge-data` is the
explicit destructive choice. See
[`specs/deployment-lifecycle-v1.md`](../specs/deployment-lifecycle-v1.md) for
the complete contract. Public release publication, online revocation, and the
pilot operational profile remain later work.

## Self-hosted service installation

Compose the signed deployment with the strict isolated configuration and a
deterministic private systemd unit:

```bash
./dev/check-self-hosted-service-install
```

The `A2-002` checker packages and signs the real Linux AMD64 release, installs
it twice from identical clean inputs, requires byte-identical configuration,
unit, and manifest outputs, verifies the unit with `systemd-analyze`, and
rejects permissive secret injection before creating deployment roots. See the
[self-hosted runbook](../runbooks/self-hosted-single-node.md). The manager does
not call `systemctl`; host activation stays an explicit operator action.

## Self-hosted upgrade and restart

Exercise signed side-by-side publication, rollback-safe descriptor composition,
and restart over unchanged persistent data:

```bash
./dev/check-self-hosted-upgrade-restart
```

The `A2-003` checker builds two signed real releases, runs v1 while selecting
v2, injects a post-selection unit-composition failure and proves exact v1
rollback, then restarts with v2. A real pending object remains `404` before,
during, and after the upgrade, readiness returns to `200`, configuration stays
byte-identical, and a same-version retry is idempotent. The manager rewrites
the generated unit but never invokes the supervisor.

## Self-hosted manual restore

Exercise absent-target restore with mandatory authority-generation rotation:

```bash
./dev/check-self-hosted-manual-restore
```

The `A2-004` checker creates a private offline snapshot after a real pending
write, simulates loss of the installed data root, rejects unrotated authority
and a symlink-bearing tree without target mutation, then atomically restores
with strictly higher policy, revocation, L1, and namespace generations. The
packaged server verifies the recovery inputs, reaches readiness `200`, rejects
the old token, and returns a safe miss under the new generation. This is a
manual POC recovery proof, not backup automation or an RPO/RTO claim.

## Self-hosted single-node exit gate

Run the complete current-tree MVP-A2 proof:

```bash
./dev/check-self-hosted-single-node-gate
```

`A2-G01` runs the A2-001 configuration/storage, A2-002 installation, A2-003
upgrade/restart, and A2-004 restore checkers in order and requires the source
tree to remain unchanged. It closes the owner-operated single-node POC only;
systemd mutation, external validation, soak, HA, backup RPO/RTO, enterprise
identity, and productionization remain outside the gate.

## Operational readiness and revocation

Exercise the first `OPS-001/A1` operational slice:

```bash
./dev/check-ops-readiness
```

The server binds its loopback listener before Shared reconciliation, reports
`200` from `/livez`, and keeps `/readyz` plus every product route at `503`
until storage and the externally signed authority are safe. Shutdown disables
readiness before draining.

With authenticated cache routing enabled, the server fingerprints and
revalidates the authority, trust root, and credential files every second. The
race-enabled fixture atomically publishes a higher signed revocation epoch,
observes propagation within the 60-second limit, rejects the old authority
with `401`, and serves the new authority only as a safe miss. Invalid, expired,
or rolled-back state disables readiness. The full benchmark/fault/soak and
profile remains open; see
[`specs/ops-readiness-v1.md`](../specs/ops-readiness-v1.md).

## Operational alert surface

Exercise the ten-class `OPS-001/A1` alert slice:

```bash
./dev/check-ops-alerts
```

The loopback `GET|HEAD /ops/v1/alerts` endpoint remains available while
readiness is false and returns deterministic `OK|FIRING` state without paths,
identities, digests, credentials, or error text. Runtime signals cover
filesystem capacity, quarantine, expired pending leases, SQLite integrity and
probe latency, signed authority reload/freshness, fail-closed routing,
immutable-export backlog, and bounded build-session acceptance errors/p95.
The race-enabled fixture activates and recovers all ten RFC classes. External
paging and the full benchmark/fault/soak evidence are not claimed; see
[`specs/ops-alerts-v1.md`](../specs/ops-alerts-v1.md).

## Shared capacity and byte SLRU

Exercise the `A1-003`/`A1-G03` storage-capacity substrate:

```bash
./dev/check-shared-capacity-slru
```

The checker proves exact and unknown-length admission before body reads,
atomic overlapping reservations, simulated disk exhaustion, tenant/repository/
trust/namespace limits, and HTTP `413` fallback. Real schema-v5 storage then
proves probation-first byte eviction from 85% to 75%, verified-hit promotion,
protected-target demotion, durable 30-day TTL, physical orphan cleanup,
transactional v3 migration, and operational quota signals. Phase-sequenced
closure is supplied by the benchmark-bound disk-fault checker below; the soak
remains separate.

## Private-beta disk faults

Execute the high-watermark and out-of-space rows from the pinned benchmark:

```bash
./dev/check-beta-disk-faults
```

The checker publishes nine real HTTP objects into a 1,000-byte Shared store,
crosses the 850-byte high mark, and proves probation authority and physical
bytes are evicted to 700 bytes below the 750-byte low mark. A separate
loopback PUT sees only 50 bytes available, returns `413` for a declared
60-byte body before the transport reads any body byte, leaves no partial
authority, and recovers to `201` after availability returns. Eight raw private
observations and their summary are bound to the benchmark digest and checked
for strict decoding and tamper rejection. The following Shared slice owns seven
more faults; six restart, network, and revocation rows plus sustained load,
soak, and the complete operational gate remain open afterward.

## Private-beta Shared faults

Execute cancellation, integrity, SQLite, lease, and pending/commit death rows:

```bash
./dev/check-beta-shared-faults
```

The checker uses real loopback HTTP and fresh Shared roots to prove that
cancelled PUT/GET streams never become partial hits, truncated or same-length
corrupt blobs become byte-free quarantined misses, an external SQLite write
lock exposes no partial commit, expired pending authority is aborted and
collected, and process death preserves a miss until the verified decision is
applied after restart. Seventeen mode-`0600` observations are manifest- and
digest-bound. Six restart, network, and revocation faults plus sustained load,
soak, and the complete operational gate remain open.

## Private-beta system faults

Execute the gateway/server restart, network, and revocation rows:

```bash
./dev/check-beta-system-faults
```

The checker builds and launches real `buildopt` and `buildopt-server`
executables. It proves stable managed-gateway identity and complete hits across
restart, liveness without readiness during server reconciliation, bounded
deadline and byte-free fail-open behavior through the real gateway, and signed
policy/grant revocation that rejects stale routes and late commits while
rotating generation and clearing pending bytes. Eighteen mode-`0600`
observations are manifest- and digest-bound. Together with the disk and Shared
slices, all 15 fault rows and `A1-G04` are closed. The sustained checker below
owns the 60-minute load slice; soak, `A1-G02`, and the complete operational
gate remain separate.

## Private-beta sustained load

Exercise the real data plane with the CI-safe non-qualifying trial:

```bash
./dev/check-beta-sustained
```

Run the exact one-hour qualification only in the pinned 4-CPU/16-GiB cgroup:

```bash
./dev/check-beta-sustained --qualify
```

Both paths use the real managed gateway, authenticated Shared storage, the
exact 70/20/8/2 object mix, and strict raw/result validation. The trial emits
300 observations, marks its runner unverified, and must be rejected by the
production validator. The qualifying path runs 1/8/32 clients for 1,200
seconds each, emits 30,000 observations, and enforces the golden-runner and
boundary-specific p95 targets. A green qualification closes only the
60-minute sustained slice; the eight-hour soak, Gradle fixture preservation,
`A1-G02`, and complete `OPS-001/A1` gate remain open.

## Private-beta eight-hour soak

Exercise the long-lived real data plane with the CI-safe non-qualifying trial:

```bash
./dev/check-beta-soak
```

Run the exact eight-hour qualification only in the pinned 4-CPU/16-GiB cgroup:

```bash
./dev/check-beta-soak --qualify
```

Both paths retain one managed gateway, Shared store, and signed authority for
the whole execution. The trial emits 300 scaled observations, marks its runner
unverified, and must be rejected by the production validator. Qualification
runs 1/8/32 clients for 9,600 seconds each, emits 30,000 exact-size
observations, and enforces the golden runner, zero-error, duration, and
boundary-specific p95 contracts. A green run closes only the eight-hour soak;
the already bounded Gradle fixture and circuit-breaker matrices remain separate
evidence inputs. With those inputs green, the soak is the final evidence needed
for `A1-G02` and the complete `OPS-001/A1` gate.

## Private-beta circuit breaker

Exercise the bounded circuit and Gradle-preservation matrix:

```bash
./dev/check-beta-circuit-breaker
```

The checker injects verified-spool quota exhaustion, `ENOSPC`, and an
oversized-object `413`, proves private durable per-runner cooldown state and
automatic recovery, and runs the real Gradle 9.6.1/JDK 21 Kotlin and Groovy
fixtures with Shared omitted. Each fallback build succeeds, writes the native
managed L1, and replays `FROM_CACHE` with Configuration Cache reused. This
closes only the circuit-breaker slice; the eight-hour qualification and
small/medium/large Gradle fixture matrix remain separate boundaries.

## Private-beta Gradle fixture sizes

Exercise the bounded small, medium, and large golden-lane build matrix:

```bash
./dev/check-beta-gradle-fixtures
```

The checker materializes three private Kotlin DSL multi-project repositories
with 8, 64, and 384 Java sources and linear critical paths of 2, 8, and 24
`compileJava` tasks. Gradle 9.6.1/JDK 21 must produce each exact known output,
then restore every critical-path task `FROM_CACHE` through the native managed
L1 while reusing Configuration Cache. It writes a mode-`0600` result, validates
it against the benchmark manifest, and removes it. This closes only the
fixture-size matrix; the eight-hour soak remains before `A1-G02` and the full
operational profile can close.

## Private-beta operational closure

Exercise the complete bounded `A1-005` operator path:

```bash
./dev/check-private-beta-operations
```

The composite checker validates the isolated-profile operations contract and
runbook, then runs the real readiness/revocation, ten-class local alert,
runner-circuit/Gradle-preservation, and base bypass/kill-switch/rollback/
uninstall drills. The runbook defines preflight, admission, triage, authority
rotation, circuit cooldown, shutdown/restart, and recovery stop conditions
without manual SQLite, authority, or circuit-state edits. Every child exercise
must leave the working tree unchanged.
This closes only `A1-005`; the owner-controlled pilot deployment is exercised
and recorded independently below.

## Owner-controlled pilot deployment

Validate the immutable `A1-001` deployment record:

```bash
./dev/check-owner-controlled-pilot-deployment
```

The checker binds the public synthetic pilot revision to the exact signed
BuildOpt release, repository workload, two authenticated installed runs, two
schema-valid `BUILD_SESSION` records, reproducible deliverables, eight native
managed-L1 `compileJava` replay hits, and custom-task default deny. It also
retains both the hosted workflow's initial account-billing block and its
successful public-runner retry without misclassifying either as a code failure.
The checker never contacts
GitHub or private machine state and makes no causal, signed-Shared-authority,
external-user, or eight-hour-soak claim. `A1-006` and `A1-G06` remain open.

## Task Intelligence pilot evaluation

Run the focused state, coverage, exact-correlation, quarantine, source-patch,
and accepted-pilot evidence contract with:

```bash
./dev/check-task-intelligence-poc
```

The diagnostic Agent and Linux helper remain fail-closed `UNAVAILABLE`; the
reviewed source-patch route is the only active pilot route. Revalidate the exact
Java recipe separately with `./dev/check-custom-task-contract-recipe`.

## Edge Cache configuration

Validate the C2-001 owner-operated single-node Edge boundary with:

```bash
./dev/check-edge-cache-config
```

The checker loads the checked-in example through the production parser and
rejects permissive files, unknown fields, trailing documents, remote clear-text
Shared origins, unsafe or overlapping paths, excessive capacity/TTL/object
limits, and any attempt to move commit, collision, offline-read, or
offline-write authority away from the fixed fail-closed policy. This block
does not open a listener or claim committed reads and replication.

## Edge Cache committed read-through

Run the C2-002 real Shared-to-Edge and offline-restart proof with:

```bash
./dev/check-edge-cache-committed-read
```

The checker commits a real pending Shared object through the canonical Ed25519
decision and SQLite transaction, reads it through the scoped token route, and
requires Edge to verify framing, decision binding, length, and SHA-256 before
durable metadata publication. It then shuts Shared down, reopens Edge, and
serves the same bytes only with exact current signed revocation authority.
Negative cases cover pending/malformed/corrupt responses, policy or revocation
drift, TTL, local corruption, and a competing writer. SLRU and writes remain
separate later blocks.

## Edge Cache capacity and byte-SLRU

Run the C2-003 hard-capacity and local pressure proof with:

```bash
./dev/check-edge-cache-capacity-slru
```

The checker composes strict Edge configuration with the complete Edge race
suite and the real Shared committed-read regression. Deterministic reduced-size
fixtures prove exact reservations, no hard-quota oversubscription, new-entry
probation, hit promotion, protected-byte demotion, probation-first 85/75
pressure eviction, logical-before-physical TTL cleanup, and metadata v1-to-v2
migration. Pending writes and replication remain a separate later block.

## Edge Cache pending replication

Run the C2-004 attempt-private write and real Shared replication proof with:

```bash
./dev/check-edge-cache-pending-replication
```

The checker verifies signed write-authority projection, exact reservation before
body reads, idempotent same-byte retry, different-byte rejection, cross-attempt
misses, shared committed-plus-pending quota accounting, pending TTL, durable
exponential retry, interrupted-claim restart recovery, and the asynchronous
worker. A real read-write beta token then proves the object reaches Shared only
as pending and remains absent from both committed paths across Edge restart.
The loopback proxy and two-node collision proof remain in C2-005.

## Edge Cache two-node proxy

Run the C2-005 real loopback and owner-controlled collision proof with:

```bash
./dev/check-edge-cache-two-node-proxy
```

The checker opens two actual IPv4-loopback proxy listeners over independent
Edge roots. Both accept different bytes for the same signed attempt/key, serve
only their own candidate while Shared is unavailable, and then replicate to a
real Shared attempt. Shared accepts the first and rejects the collision; after
the signed central commit, both Edge nodes return the accepted bytes online and
continue returning the same committed bytes offline. Negative tests reject
non-loopback listeners, read-only PUT, unsafe paths/methods, and unknown-length
writes before body consumption.

## Edge Cache gate

Run the final C2-G01 current-tree composition with:

```bash
./dev/check-edge-cache-gate
```

The gate validates the exact C2-001..005 contracts and runs their nested final
runtime proof without changing the source tree. One invocation covers strict
private configuration, verified committed read-through and offline restart,
hard byte quota and SLRU, exact-attempt pending replication/recovery, real
loopback HTTP, and the owner-controlled two-node central-collision winner.
Soak, external validation, OS service management, authority hot reload, and
production hardening remain explicitly outside this POC gate.

## Edge Cache operability

Run the complete owner-operated process, reload, status, packaging, and service
proof with:

```bash
./dev/check-edge-operability
```

The gate builds `buildopt-edge`, starts a real IPv4-loopback runtime under the
race detector, durably replicates a pending object, fails closed during an
invalid authority replacement, recovers only with a verified monotonic signed
generation, checks redacted mode-`0600` aggregate status and graceful shutdown,
and validates a byte-reproducible hardened systemd unit. The command is also
included in the signed Linux AMD64 release bundle. See the
[Edge operator runbook](../runbooks/edge-cache.md). Linking or starting the
unit, HA/backups, enterprise identity, other platforms, external validation,
and the eight-hour soak remain outside `POC-O1`.

## Synthetic owner POC lab

Run the complete owner-independent POC proof and emit its JSON result on
standard output with:

```bash
./dev/run-owner-poc-lab
```

Use `--output ABSOLUTE_PATH` to publish the same result atomically. The lab
builds the synthetic golden-lane Gradle project, repeats the Shared fault
evidence three times under `-race`, proves the two-Edge Shared collision path,
and runs the complete Edge operability gate. `./dev/check-owner-poc-lab`
strictly validates the contract and result; base CI runs it on a clean checkout.
It does not run the eight-hour soak, use external design partners, or authorize
production promotion.

## Build history API

Run the UX-F1-001 contract and real-server proof with:

```bash
./dev/check-build-history-api
```

When `buildopt-server` has an export directory, setting the separate
`BUILDOPT_HISTORY_API_TOKEN` activates authenticated `GET` operations at
`/api/v1/build-sessions` and `/api/v1/build-session?id=...`. The collection is
newest-first, filters exact redacted repository/outcome values, and uses a
stable opaque cursor with a default 25 and maximum 100 items. The detail
operation returns the immutable already-redacted BUILD_SESSION document.

The route is absent without the independent read token. It never accepts the
ingest credential, mutates exports, reconstructs raw identities, or changes
the separate Test Optimization integration.

## Build history dashboard

Run the UX-F1-002 embedded-interface contract with:

```bash
./dev/check-build-history-dashboard
```

When the history API is configured, `/buildopt/` serves a dependency-free
responsive dashboard from the existing loopback server. The credential view
keeps the independent read token only in page memory; the loaded view provides
real summary counts, exact repository/outcome filters, an explicitly
loaded-row-only session/revision search, cursor loading, and immutable build
detail.

Static assets contain no credential or history data, make no external
requests, use safe DOM text insertion for returned values, and ship with a
same-origin-only CSP, frame denial, no-referrer, and no-store headers. The
interface covers authentication, loading, empty, ready, and error states and
does not add fabricated analytics or Test Optimization behavior.

## Build Impact manifest

Validate the C3-001 customer-owned manifest boundary with:

```bash
./dev/check-build-impact-manifest
```

The checker loads the repository fixture through the production parser and
rejects repository/pipeline mismatches, inferred or unsafe Gradle entrypoints,
unsafe paths and symlinks, ambiguous ownership, unknown fields, trailing
documents, and every unknown-change policy except `FULL_GRAPH`. Passing this
checker does not authorize selection or satisfy the separate `BIA-002` gate.

## Build Impact declared graph

Run the C3-002 conservative decision matrix with:

```bash
./dev/check-build-impact-declared-graph
```

The checker composes C3-001 with a manifest-digest-bound Gradle graph and
proves reverse-dependent expansion, complete artifact/check coverage,
Test-owned check preservation, static/manifest/adapter global fallbacks, and
strict rejection of missing security fields, cycles, unknown references, and
Test-containing alternatives. Execution stays on the original entrypoints;
the calculated alternative is shadow-only.

## Build Impact shadow validation

Run the C3-003 observation and comparison matrix with:

```bash
./dev/check-build-impact-shadow-validation
```

The checker validates exact manifest/graph/adapter-bound shadow and paired
control fixtures. It requires the full original baseline, exact project reach,
required artifact digests/sizes and every owned check. Candidate build or
content/check/project divergence is a false negative; infrastructure and
invalid baseline evidence remain `INCONCLUSIVE`. No result authorizes selection.

## Build Impact promotion gate

Run the C3-004 BIA-002 aggregation and suspension matrix with:

```bash
./dev/check-build-impact-promotion-gate
```

The checker composes C3-001..004 and binds every result to the current
repository, pipeline, manifest, graph, and adapter. It proves the unchanged
30-day, 3,000-decision, 99%-coverage, 100-control-per-stratum thresholds and
exact one-sided 95% false-negative bounds. Binding drift resets the sample,
invalid or insufficient evidence is `INCONCLUSIVE`, and one false negative is
`SUSPENDED`. The checked-in observations remain inconclusive; only the
deterministic threshold corpus qualifies, and neither state authorizes active
selection.

## Build Impact active selection

Run the C3-005 fail-closed selection matrix and real synthetic Gradle proof:

```bash
./dev/check-build-impact-selection
```

The checker composes C3-001..005, then copies the owner-controlled
three-project fixture into two temporary workspaces. Production selection
recomputes BIA-002 and chooses only the customer-authorized service-a
alternative for a library-c change. Offline full and selected builds must emit
the same service-a JAR and independently executed Test-owned marker, while the
selected workspace must omit unrelated service-b. Disablement, bypass, kill
switch, binding drift, suspended/inconclusive evidence, and global/unknown
changes all restore the original entrypoints.

## Build Impact composite gate

Close the owner-operated MVP-C3 current-tree proof with:

```bash
./dev/check-build-impact-gate
```

The gate composes C3-001..005 in one source-preserving invocation. It retains
the two checked-in observations as `INCONCLUSIVE` and requires the full
synthetic qualification, fail-closed matrix, and real offline Gradle proof.

## Runtime owner evaluation

On the exact 4-CPU/16-GiB runner, execute real A/A and resource-profile
autotuning evidence:

```bash
./dev/run-runtime-owner-evaluation /tmp/buildopt-runtime-evidence
```

The command compiles an eight-project, 2,048-source parallel Gradle fixture in
four alternating pairs. It also drives 200 durable pre-outcome A/A assignments
and one-hour delayed exactly-once rewards through the production cohort/bandit
engine. The result must preserve p95/p99, queue, OOM, compute, and exact ZIP
guardrails; a cache hit or synthetic reward alone cannot pass.
The hosted run is retained under `benchmarks/results` and revalidated without
rerunning the builds with `./dev/check-runtime-owner-evaluation`.


## Owner-operated causal POC evaluation

Run both immutable public pilots through the paired causal harness:

```bash
./dev/run-owner-poc-evaluation \
  /tmp/buildopt-owner-poc-evidence \
  ../buildopt-pilot \
  ../buildopt-pilot-groovy
```

Each repository receives four pre-assigned alternating pairs with isolated
control/candidate workspaces. The command requires positive mean savings and a
positive lower 95% paired-bootstrap bound, non-regressive p95, identical ZIPs,
and zero product/build-failure divergence. The hosted workflow repeats the
measurement on `ubuntu-24.04`; `./dev/check-owner-poc-evaluation` validates the
checked-in evidence without rerunning the expensive builds.


## Private-beta benchmark harness

Exercise the non-qualifying real-data-plane smoke profile:

```bash
./dev/check-beta-benchmark-harness
```

The harness strictly loads `benchmarks/beta-v1.yaml`, runs all four named
phases with 1/8/32 concurrent workers through real Shared HTTP PUT/GET paths,
commits cold pending objects with an exact Ed25519 decision, and emits 1,200
private raw JSONL observations before its digest-bound JSON summary. The smoke
cycle keeps the exact 70/20/8/2 distribution and 70% read-hit target while
scaling sizes and omitting qualifying durations, Gradle fixtures, and faults.
Validation rejects result or raw-stream tampering; the full operational gate
remains open.

## Private-beta token isolation

Exercise the complete `A1-002`/`A1-G01` negative gate:

```bash
./dev/check-private-beta-token-isolation
```

The checker proves that only domain-separated token hashes reach
`control.sqlite`; repository, namespace, generation, plane, and operation
crossings fail closed; read-only tokens cannot `PUT`; expiry and a live
`buildopt-server token revoke` take effect before the next request. It also
proves that the remote token differs from the signed local authority
credential, stays inside the gateway, requires TLS outside loopback, and is
removed before Gradle. The inert GitHub composition gives forks no token,
same-repository pull requests only stable reads, and protected `main` the
distinct stable read-write token. This closes only `A1-002` and `A1-G01`.

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
is dispatched separately for GitHub-runner evidence. The immutable WS-007
fixture remains pinned to its historical Action bytes; the current init script
is additionally exercised with a real packaged Gradle invocation by
`dev/check-deployment-lifecycle`. This closes `WS-007`, not `F0-004`,
`CI-ORCH-001`, release publication, token/fork policy, or operational
readiness.

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
durable attempt/lease cleanup, and the complete pilot fault/soak profile remain
with `OPS-001/A1` and `WS-008`; the signed local deployment lifecycle is
covered separately by `dev/check-deployment-lifecycle`.

## Normative package validation

Validate the namespace skeleton defined by RFC §29.2:

```bash
./dev/check-normative-layout
```

The checker requires all 16 contract, vector, specification, benchmark, and ADR
namespaces, their non-empty indexes, and parent directories for the 99 planned
normative artifacts. It also preserves every materialized contract, including
the deployment lifecycle, and rejects an empty file at any planned artifact
path. F0-010 created the structure; each schema, API, IDL, vector,
specification, benchmark, or ADR remains owned by its tracker item.

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

The runner resolves the immutable image index by digest, requires its unique Linux AMD64 manifest to equal the recorded platform digest, pulls that exact reference, and verifies the local image operating system, architecture, and repository digest. It also builds the walking-skeleton launcher, server, signal helper, no-hit miss helper, metrics validator, and isolated schema validator with the locked Go toolchain and `CGO_ENABLED=0` into a temporary read-only mount. The JDK-only image receives those binaries plus the official static `jq` 1.7.1 binary after its SHA-256 is verified, so it can execute the real authenticated rendezvous, session ingest, cancellation cleanup, metric-catalog checks, `BUILD_SESSION v1` validation, and no-hit gate without adding a compiler or package manager to the image. The subsequent container uses `--pull never`, checks the exact Java patch from the runner specification, and in strict mode verifies effective cgroup v2 CPU and memory limits from inside the container. It never treats the readable source tag as executable identity.

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

## Phase 0 exit gate

Run the complete environment exit gate on a Linux AMD64 workstation with a
functional Docker daemon and enough memory to enforce the golden runner:

```bash
./dev/check-phase-zero
```

The gate validates the lock and deterministic doctor contract, requires a
passing live host report, exercises every adopted project-local toolchain
through its owning checker, and then runs the digest-pinned image with strict
4-CPU/16-GiB cgroups. The live report continues to expose workstation-global
`PATH` drift; it does not require two development computers to share global
installations because the executable toolchain checks consume the verified
repository-local copies.

## Automatic Build Impact

Generate the conservative Gradle model, reviewable graph, and derived manifest
for a repository with a customer-owned Build Impact manifest:

```bash
./dev/run --toolchain go -- go run ./cmd/buildopt-impact generate \
  --repository fixtures/build-impact/synthetic-repository \
  --manifest fixtures/build-impact/synthetic-repository/buildopt-impact-manifest.json \
  --repository-id buildopt-synthetic \
  --pipeline-class pull-request
```

Use `buildopt-impact check` with the same arguments in CI. It regenerates both
documents and fails on drift. Ambiguity, global changes, incomplete coverage,
unsupported task shapes, included builds, stale generated state, or an
unqualified `BIA-002` result all preserve `FULL_GRAPH`. Run the complete
fixture and real Gradle equivalence proof with:

```bash
./dev/check-build-impact-automatic
```

This command does not select, prioritize, shard, retry, or otherwise optimize
tests.

## macOS and Windows compatibility

The supported portable CLI inventory is `buildopt` plus `buildopt-impact` for
macOS and Windows on AMD64 and ARM64. Produce checksummed, self-contained user
packages with:

```bash
./packaging/macos/package.sh --version 0.1.0 --output dist
pwsh -NoProfile -File packaging/windows/package.ps1 -Version 0.1.0 -Output dist
```

The matching install scripts are idempotent across reinstall and upgrade, keep
an exact private receipt, and the uninstall scripts remove only receipt-owned
files. macOS uses POSIX process groups; Windows uses a Job Object for complete
descendant cleanup. Linux retains its peer-credential Unix socket, while
macOS/Windows use authenticated loopback TCP for the Gradle event channel.
Persistent gateway/L1/bootstrap services and server/Edge processes have native
package and lifecycle coverage on macOS and Windows; platform differences stay
visible through `buildopt doctor` and native CI.

Run the local portability inventory, cross-build, package syntax, and workflow
gate with:

```bash
./dev/check-platform-compatibility
```

The hosted `.github/workflows/platform-ci.yml` matrix is the native acceptance
gate for install, upgrade, Gradle handshake and Build Impact, cancellation,
cleanup, uninstall, and artifact publication.

## Build Optimization performance

Validate the current POC decision, evidence classifications, and open value
gates without rerunning benchmarks:

```bash
./dev/check-poc-value-validation
```

Validate and print the current safe-cache, Runtime Tuning, and Build Impact
scorecard without rerunning a benchmark:

```bash
./dev/check-build-optimization-performance
```

The scorecard keeps each mechanism attributable and never adds percentages
from different workloads. It does not close the combined product gate. Fresh
cache and Build Impact evidence can be created
with `run-cache-parity-benchmark` and `run-build-impact-performance`; each has
a matching `check-*` command. See
[the benchmark index](../benchmarks/README.md#build-optimization-scorecard).

## Historical onboarding performance

Validate the checked-in local observations without running Gradle:

```bash
./dev/check-onboarding-performance
```

The fresh-run command, required immutable pilot inputs, two controls, warming,
four-pair alternating design and interpretation boundary are documented in
[the benchmark index](../benchmarks/README.md#historical-v02-public-onboarding-performance).
The manual `Onboarding performance` workflow installs release `0.2.0`, runs
the same harness on `ubuntu-24.04`, validates the JSON and uploads it. It is
manual because performance sampling is intentionally separate from ordinary
push/PR CI and from the deferred soak.

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
