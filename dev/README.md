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

## Optional central storage contract validation

Validate the POC boundary between Gradle cache objects and BuildOpt portfolio,
evidence, and checkpoint state:

```bash
./dev/check-central-storage-contract
```

The checker compiles the three Draft 2020-12 state schemas and executes six
language-neutral lifecycle scenarios. It covers ordered immutable publication,
first and later generation CAS, exact idempotent replay, stale/skipped updates,
namespace isolation, independent retention, interruption, outage, and native
fallback. This is contract evidence only; it does not start a server or claim
cross-machine performance.

## Restart-safe central state storage

Validate the concrete typed-state implementation over the existing Shared CAS:

```bash
./dev/check-central-state-storage
```

Validate the owner-operated central TLS/token boundary, including the real
TLS listener, untrusted-client rejection, capability isolation, live
revocation, pending cache-write binding and typed-state CAS:

```bash
./dev/check-central-https-auth
```

Validate the native Gradle producer/consumer path through the local gateway
and central HTTPS object plane:

```bash
./dev/check-central-gradle-cache
```

Validate one-time connection and exact online/offline synchronization of
generated portfolios, evidence and checkpoints:

```bash
./dev/check-central-state-sync
```

Validate automatic pre/post synchronization inside `buildopt optimize`,
cross-commit source-profile revalidation, structural drift rejection and
verified offline lookup:

```bash
./dev/check-central-optimize-integration
```

Validate the complete installed producer/consumer composition in isolated
Docker containers, including server restart, automatic remote profile and
read-only cache reuse, exact outputs, credential containment and outage
fallback:

```bash
./dev/run-poc-central-two-machine /tmp/buildopt-central-two-machine.json
./dev/check-central-two-machine /tmp/buildopt-central-two-machine.json
```

The runner creates and removes its exact Docker network, containers, image and
temporary directory. It needs a working Docker daemon and several minutes for
the producer's eight-pair calibration. The checked-in terminal result can be
revalidated without Docker:

```bash
./dev/check-central-two-machine \
  benchmarks/results/poc-central-two-machine-v1.json
```

The state-storage checker proves private independent `state.sqlite`
migrations, restart persistence, repository/kind isolation, exact CAS replay,
one-winner concurrent promotion, invisible partial publication,
complete-byte corruption rejection and the portfolio/evidence/checkpoint
retention rules. The HTTPS checker then exercises those semantics through a
real trusted TLS listener with scoped credentials. The Gradle-cache checker
adds a clean write-only producer, verified owner commit, independent read-only
consumer with eight `FROM-CACHE` outcomes, exact output hashes, rejected
read-only publication and successful native execution after server loss. It is
a functional proof, not a cross-machine wall-time claim.
The state-sync checker adds first/no-change sync, a clean consumer, exact
interrupted retry, a verified concurrent winner, incompatible-state fallback
and offline snapshot tamper rejection. The central optimize checker then uses
retained public Kafka evidence to prove source-only cross-commit selection
before Gradle, rejects `build.gradle.kts` drift, publishes newly completed
state automatically and reuses only verified offline snapshots. Neither
checker makes a new wall-time claim.
The two-machine checker composes those lower-level guarantees across separate
client filesystems. Its recorded phase durations are diagnostic only; the next
central value experiment owns the equal-opportunity native comparison.

Run the terminal equal-opportunity central experiment with Java 21. It
calibrates Ktor and Beam, publishes the full native workflow once, then gives
the full-graph control and structural candidate the same committed HTTPS cache
objects across eight alternating pairs:

```bash
./dev/run --toolchain temurin-jdk-21 -- \
  ./dev/run-poc-central-end-to-end-value /tmp/central-value.json
./dev/check-central-end-to-end-value /tmp/central-value.json
```

The runner requires network access for the two public repositories and their
Gradle dependencies. It uses separate workspaces, Gradle User Homes and local
BuildOpt caches. Before calibration it runs the complete owner workflow through
native Gradle, records that local cache's object count, bytes and manifest
digest, and imports the verified seed into the private L1 used only by the
isolated calibration arms. The imported L1 is deleted before normal Tier-One
central publication, so this fair native comparator does not widen BuildOpt's
publication policy. The three-hour calibration allowance preserves all eight
pairs and the full public-repository workflows; it does not relax the value
gate. The runner removes its exact temporary root after success and retains
that root only when a failure needs diagnosis. This is a bounded POC
experiment, not a soak.

The retained terminal evidence is:

```bash
./dev/check-central-end-to-end-value \
  benchmarks/results/poc-central-end-to-end-value-v1/summary.json
```

It records Ktor at **82.45% lower wall time** and Beam at **56.41%**, each
with 8/8 positive pairs, exact required outputs and the same committed central
cache opportunity. A Ktor global build-logic change retains native without a
performance claim. The run used the 12-CPU host with a common eight-worker
limit; it is POC evidence, not the contractual golden-runner class.

If a bounded runner ends after one subject has completed all eight pairs, the
retained diagnostic root can resume only that completed subject:

```bash
BUILDOPT_CENTRAL_VALUE_RESUME_ROOT=/tmp/buildopt-central-value.XXXXXX \
  ./dev/run --toolchain temurin-jdk-21 -- \
    ./dev/run-poc-central-end-to-end-value /tmp/central-value.json
```

Resume rejects partial timing observations and rebuilds the installed package
before requiring its canonical payload digest to match the checkpoint exactly
across paths, types, modes, symlink targets and file bytes. A subject stopped
after authoritative native discovery but before its first complete pair may
reuse only its verified checkout, dependency and authority preparation; it
may also retain a complete fail-safe `NATIVE_RETAINED` result whose native
execution succeeded and whose failed calibration accepted zero pairs. It
reruns discovery and all eight pairs. TAR metadata may differ, so the final
evidence retains the archive and payload hashes separately and records whether
checkpoint reuse occurred. Derived statistics are recomputed from the raw
pairs under a deterministic locale during resume.

Measure the lifetime of one central structural profile across a real public
Ktor first-parent commit sequence with Java 21:

```bash
./dev/run --toolchain temurin-jdk-21 -- \
  ./dev/run-poc-profile-lifetime /tmp/profile-lifetime.json
./dev/check-profile-lifetime /tmp/profile-lifetime.json
```

The runner qualifies one Jetty dependency-source profile, advances two
persistent isolated arms through one matching replay, two deliberately
unobserved commits, one unrelated CORS source change and one global
build-logic invalidation. It requires exact owner outputs, identical central
cache opportunity and cumulative economics after calibration. The retained
result records a 112.198-second matching saving, a 220.761-second unrelated
fallback penalty and no observed payback. The networked run is bounded POC
validation, not a soak or production qualification.

Validate the economic precheck that protects a native fallback from unrelated
discovery and calibration:

```bash
./dev/check-economic-prequalification
```

The checker fixes the eight-build theoretical lower bound, the 64-commit
first-parent history cap, both `REJECT` and `MEASURE` paths, exact central
profile replay, and the rule that a rejected candidate emits no discovery
files and performs no calibration. With no argument it also validates the
retained public Ktor result. That run finds two analogous CORS commits, rejects
in 192.442 ms and observes 13.896 seconds of fallback overhead instead of the
preceding run's 220.761 seconds. It uses direct ownership from a verified
generic graph and never branches on a repository name. Qualification still
costs 1,386.764 seconds and remains unpaid in the three-build window.

Run the unchanged zero-manual-file optimize path across the five frozen public
repositories, optionally one repository at a time:

```bash
./dev/run-automatic-breadth-transfer /absolute/evidence/directory
./dev/run-automatic-breadth-transfer /absolute/evidence/directory apache-kafka
```

The runner installs one package, creates a fresh public checkout and private
Gradle home per subject, invokes only `buildopt optimize <workflow>`, preserves
the raw result and generated state, then removes that checkout before the next
row. Partial directories are resumable by repository key. It stops before a
subject when less than 8 GiB is free and never turns a native-retained result
into a qualification.

Validate the retained five-repository result, raw/state hashes, exact outputs,
alternating pairs, fallback, means and break-even calculations with:

```bash
./dev/check-automatic-breadth-transfer
```

The incremental rerun keeps the same frozen five repositories and workflows,
but collects one baseline and eight alternating pairs across seventeen ordinary
invocations. It uses each checkout's Git ignore rules to remove ignored
workspace outputs before each invocation, preserving tracked sources and
`.buildopt`, so clean-workspace materialization is exercised while private
Gradle and BuildOpt state remain available. Capture one repository at a time
to make the long POC restartable, then validate the completed directory:

```bash
./dev/run-automatic-breadth-transfer-v2 /absolute/evidence/directory spring-framework
./dev/run-automatic-breadth-transfer-v2 /absolute/evidence/directory opentelemetry-java-instrumentation
./dev/run-automatic-breadth-transfer-v2 /absolute/evidence/directory apache-kafka
./dev/run-automatic-breadth-transfer-v2 /absolute/evidence/directory micronaut-core
./dev/run-automatic-breadth-transfer-v2 /absolute/evidence/directory apache-groovy
./dev/check-automatic-breadth-transfer-v2 /absolute/evidence/directory/summary.json
```

This is a bounded POC experiment, not a soak. The checker requires all 85
ordinary results, recomputes every balanced pair and preserves safe native
retention when the unchanged value or thirty-build payback gates are not met.

The checked V2 result has zero product failures and zero manual
target-repository files. All five candidates beat optimized native Gradle with
exact outputs. OpenTelemetry, Kafka, Micronaut and Groovy qualify at 14.97%,
54.92%, 66.24% and 75.97% with 8/8 pairs and 19-, 14-, 8- and 2-build payback.
Spring improves 12.71% but remains native at 7/8 and 67-build payback.

The materialization-economics rerun keeps those subjects and gates unchanged,
stores verified outputs in one manifest-bound pack instead of one durable blob
per file, excludes private BuildOpt state from customer-output hashing and
records Gradle, discovery, materialization, verification and remaining wrapper
time for every observation:

```bash
./dev/run-materialization-economics-v2 /absolute/evidence/directory spring-framework
./dev/run-materialization-economics-v2 /absolute/evidence/directory opentelemetry-java-instrumentation
./dev/run-materialization-economics-v2 /absolute/evidence/directory apache-kafka
./dev/run-materialization-economics-v2 /absolute/evidence/directory micronaut-core
./dev/run-materialization-economics-v2 /absolute/evidence/directory apache-groovy
./dev/check-materialization-economics-v2 /absolute/evidence/directory/summary.json
```

The checked terminal bundle is
`benchmarks/results/poc-materialization-economics-v2`. It records 85 ordinary
invocations and 40/40 positive pairs. Spring, OpenTelemetry, Kafka, Micronaut
and Groovy save 9.97%, 14.93%, 38.93%, 59.54% and 75.11% end to end, preserve
exact outputs and pass fallback. One-time learning cost is 2.170–7.357 seconds,
so every row passes the unchanged 30-build payback gate in one to four matching
builds. The runner is long and is not part of normal CI; Base CI validates the
checked-in compact evidence with the strict checker.

Reproduce the current five-repository qualification, portability and public
first-parent lifetime experiment with:

```bash
./dev/run-qualified-lifetime-v2 /absolute/evidence/directory
./dev/check-qualified-lifetime-v2 /absolute/evidence/directory/summary.json
```

The runner may also capture one subject by passing its key as the second
argument. It repeats the unchanged 17-invocation/eight-pair qualification,
publishes the profile and its chunked materialization pack through the central
HTTPS/CAS path, and then compares persistent optimized-native and remote-profile
arms on each preregistered descendant. Every observation must preserve exact
required output bytes; incompatible profiles must retain native execution. The
experiment reports cumulative economics per repository and never averages
unrelated repository percentages. It is bounded POC evidence, not a soak or a
production gate.

If the harness stops after the expensive qualification capture, preserve the
two diagnostic roots printed on stderr and resume that single subject without
repeating its timing samples:

```bash
BUILDOPT_QUALIFIED_LIFETIME_RESUME_AUTO_ROOT=/absolute/buildopt-automatic-breadth.root \
BUILDOPT_QUALIFIED_LIFETIME_RESUME_QUALIFICATION_ROOT=/absolute/qualification-result.root \
  ./dev/run-qualified-lifetime-v2 /absolute/evidence/directory repository-key
```

Resume requires both absolute retained roots plus one repository key. The
runner revalidates the captured result, raw samples, materialization state and
installed binary before continuing; it does not reinterpret or rerun the
qualification.

If all five validated subject results exist but aggregate generation was
interrupted, rebuild and validate only the compact summary without running a
Gradle build:

```bash
./dev/run-qualified-lifetime-v2 /absolute/evidence/directory --summary-only
```

The exact-byte gate also distinguishes native output nondeterminism from a
BuildOpt correctness failure. A mismatch between two native-retained arms
rejects the subject without a performance claim and records the complete
difference manifest. A mismatch after `CENTRAL_PORTFOLIO` selection remains a
hard failure. The runner never normalizes archives or bytecode to manufacture
equivalence.

The checked terminal bundle is
`benchmarks/results/poc-qualified-lifetime-v2`. It records five complete
qualification decisions, four qualified profiles, two portable profiles,
seven exact native-retained descendant builds, zero selected replays, zero
paid-back subjects and zero product failures. Base CI validates the compact
summary plus every subject's raw 17-invocation capture and eight-pair evidence;
it does not rerun the public repositories.

The follow-up recovery bundle keeps Kafka's qualifier and six public descendant
revisions fixed while comparing the implementation before and after selected
replay stops attaching the original full-workflow output observation:

```bash
./dev/check-cross-commit-value-recovery
```

The checker first validates both raw captures with
`check-qualified-lifetime-v2 --subject`, then recomputes the exact selected
revision and source, candidate improvement, attributable replay saving, fallback
delta and cumulative economics. The selected candidate changes from 166.299 to
42.577 seconds and saves 104.975 seconds/71.14% after the repair. The complete
six-build window changes from -22.040 to +66.772 seconds net after 6.762 seconds
of qualification/publication cost. All 4,449 required outputs match and product
failures remain zero.

The after native-retained arms total -31.441 seconds. That uncontrolled delta is
kept in the window total but cannot prove selected-replay value; the strict gate
requires the selected replay itself to be positive. The checked bundle is
`benchmarks/results/poc-cross-commit-value-recovery-v1`. Normal CI validates the
committed evidence and does not rerun the long public-repository capture.

Validate the frozen non-Kafka cross-commit breadth replication without rerunning
the public repositories:

```bash
./dev/check-cross-commit-breadth-replication
```

The checker reconstructs
`benchmarks/results/poc-cross-commit-breadth-replication-v1/summary.json` from
the raw Spring broad, OpenTelemetry JMX and Spring JMS captures. It requires the
observed terminal decisions: Spring broad fails calibration at -0.91% and 1/8,
OpenTelemetry fails ownership before calibration, and Spring JMS qualifies at
11.43% and 8/8 before 14 native output differences reject portability. It also
requires zero selected replays, zero product failures and
`claimBroadened=false`.

Normal CI runs only this deterministic checker. The public builds are not
repeated, native differences are not normalized, and no repository-specific
ownership exception is authorized.

Validate the producer-bound cross-commit breadth V2 evidence without rerunning
OpenTelemetry, Ktor or Groovy:

```bash
./dev/check-cross-commit-breadth-v2 \
  benchmarks/results/poc-cross-commit-breadth-v2
```

The checker validates the frozen three-family specification, revalidates the
raw qualified-lifetime summary, rebuilds the terminal summary byte for byte and
requires the observed 0/3 claim-eligible decision. The full runner is
restartable by repository key; `--summary-only` assembles an already complete
raw matrix. Base CI validates only the compact checked evidence.

Validate the generic workflow-input ownership follow-up without rerunning the
long OpenTelemetry capture:

```bash
./dev/check-workflow-input-ownership
```

The checker binds the frozen public base/target, exact four-path observation,
provider-backed input coverage, 1,027->8-project discovery, zero product
failures and the absence of any performance claim. Base CI separately runs the
full automatic discovery fixture, including provider-backed, consumed,
deleted, global, unsupported and ambiguous cases, plus the Go suite.

The one-time setup observation queries finalized inputs after requested tasks
execute and disables Configuration Cache only for that unmeasured capture.
Standalone Build Impact discovery and all measured or fallback workflows retain
their normal Configuration Cache behavior. No repository identity or filename
extension is used to decide relevance.

Validate producer-atomic native-volatility quarantine without rerunning a public
repository:

```bash
./dev/check-native-volatility-quarantine
```

The checker compares two complete synthetic native observations, quarantines
the complete producer of one differing output, verifies the exact remaining
transported bytes and requires every quarantined output to be rebuilt locally.
It also rejects tampering and retains native for binding, path or producer
ambiguity. The frozen Spring JMS finding remains linked as diagnostic evidence:
14 of 8,385 outputs differed and match two observed producer patterns, but that
revision did not retain a complete producer inventory. Therefore the checker
does not authorize a filtered public pack, selected replay or wall-time claim.
Normal CI executes only this bounded fixture and evidence validation.

Validate the subsequent public Spring qualification and two-descendant lifetime
without rerunning the repository:

```bash
./dev/check-producer-bound-quarantine-lifetime
```

The checker binds the explicit automatic 6/8 policy, raw paired evidence,
independent-root stable pack, 352 locally rebuilt quarantine paths, one selected
84.656-second replay, one optimized-native rejection and +59.550-second
cumulative net. It also reruns the generic qualified-lifetime subject validator
under the derived contract while preserving the historical 7/8 evidence.

Validate the subsequent Micronaut generalization result without rerunning its
public repository builds:

```bash
./dev/check-producer-bound-lifetime-generalization
```

The checker binds the 70-to-2 qualification, 10.33% mean saving, target
producer quarantine and the first descendant's native-retained result. It
requires the exact-output rejection caused by a different generated JAR,
records no descendant performance claim, and motivates the subsequent
cross-revision volatility experiment.

Validate the preregistered cross-revision producer-volatility portfolio and
its fail-closed Go implementation with:

```bash
./dev/check-cross-revision-volatility-portfolio
```

The portfolio stores producer task paths learned only from two independent
optimized-native observations. It never carries historical output bytes into a
new revision; application always partitions a newly observed current-revision
inventory.

Capture the preregistered public Micronaut learning and evaluation revisions
with at least 16 GiB free using:

```bash
./dev/run-cross-revision-volatility-portfolio /new/absolute/evidence/directory
```

The learning revision runs once and cannot claim performance. The later public
revision runs the ordinary eight alternating pairs only after its current
context and complete native output inventory validate against the portfolio.
The checked terminal evidence instead records Wrapper and output-contract
drift: BuildOpt returns structured `NATIVE_RETAINED`, names both incompatible
bindings and stops with zero timing pairs. The learning pair quarantines 476 of
11,187 outputs across five Kotlin producers; the later native pair observes two
different volatile JAR producers, so neither observation is generalized into a
universal producer list.

Validate the portfolio compatibility preflight, its fail-closed Go contract
and the frozen Micronaut evidence with:

```bash
./dev/check-portfolio-compatibility-preflight
```

Capture the rejection fixture and preregister the next compatible public
window with at least 16 GiB free using:

```bash
./dev/run-portfolio-compatibility-preflight /new/absolute/evidence/directory
```

The runner executes one required ordinary customer build. When repository,
workflow, Wrapper or output-contract bindings drift, it stops before cloning
or starting the independent native observation. The checked evidence records
one avoided measurement-only build, zero timing pairs and no performance
claim.

Validate the exactly compatible Micronaut portfolio experiment with:

```bash
./dev/check-compatible-portfolio-value
```

Capture it from scratch with at least 16 GiB free using:

```bash
./dev/run-compatible-portfolio-value /new/absolute/evidence/directory
```

The frozen result passes repository, workflow, Wrapper and runtime
output-contract compatibility, reduces 70 projects to 22 and captures 190
unaffected outputs. It retains native before timing because eight volatile
intermediate producers lack proven transitive lineage to those final outputs.
The structured `PORTFOLIO_PRODUCER_LINEAGE_UNAVAILABLE` result is a successful
fail-closed POC outcome, not a performance percentage.

Validate the lineage correction, its first exact-output recovery and the
terminal eight-pair result without rerunning Micronaut:

```bash
./dev/check-transitive-producer-lineage
```

The checker binds the two implementation revisions, both evidence summaries,
the diagnostic candidate frontier, all 16 timing observations, the single
required-output digest and the POC boundaries. The terminal result proves
lineage-safe replay but rejects value at 65 ms/0.49%, 5/8 positive pairs, an
interval crossing zero and regressive p95.

Validate the follow-up direct-producer frontier experiment without rerunning
Micronaut:

```bash
./dev/check-minimal-quarantine-rebuild-frontier
```

The checker binds the experimental implementation, exact eight-pair summary,
task-shape diagnostic and preceding graph-proven result. The 63-entrypoint
direct frontier keeps one exact output digest and a successful fallback, but
selects the same 52/70 projects and is 709.375 ms/5.60% slower than its own
optimized-native control. The experiment is retained as negative POC evidence;
the launcher keeps the graph-proven lifecycle cover.

Validate the generic Gradle task-operation and dependency-DAG attribution
tooling with:

```bash
./dev/check-gradle-critical-path
```

The checker runs a real Gradle 9.6.1 build, exports the already-resolved task
graph through a diagnostic init script, joins it to Gradle's task build
operations and calculates the longest hard-dependency chain by cumulative task
duration. It keeps `buildSrc` and the main build as separate trace segments and
does not authorize any optimization.

Capture the frozen Micronaut native-versus-quarantine attribution with at
least 16 GiB free using:

```bash
./dev/run-quarantine-critical-path-attribution /new/absolute/evidence/directory
```

The runner preserves the daemon, 12-worker, exact-output, fallback and
eight-alternating-pair contracts. Trace collection is diagnostic and excluded
from wall-time claims; it exists only to distinguish command-line entrypoint
count from executed task and dependency-chain work.

Validate the checked terminal result without rerunning Micronaut:

```bash
./dev/check-quarantine-critical-path-attribution
```

The result joins all 16 trace observations to the unchanged paired wall-time
evidence. The candidate eliminates 110 tasks and 4,731 ms of cumulative task
duration, but none of the eliminated tasks belongs to a control critical path;
the candidate main-build span grows 248.375 ms and its longest hard-dependency
chain grows 178.875 ms. The terminal decision is therefore
`STOP_MICRONAUT_QUARANTINE_LINE`, not another smaller frontier.

Existing raw captures can be reanalyzed without starting Gradle again:

```bash
./dev/run-quarantine-critical-path-attribution \
  /new/absolute/analysis/directory /absolute/existing/raw/directory
```

Validate the corrected OpenTelemetry effective-change replay and its terminal
economic rejection without rerunning the 17 public-repository builds:

```bash
./dev/check-compatible-descendant-discovery
```

The checker binds the raw capture and calibration evidence, requires all eight
50-output pairs to be exact, and confirms that 5/8 with an interval crossing
zero retains native before any descendant timing.

Reproduce the incremental-learning transaction with one exact installed
binary and validate its checked evidence with:

```bash
./dev/run-incremental-learning /absolute/path/to/buildopt /tmp/incremental-learning.json
./dev/check-incremental-learning /tmp/incremental-learning.json
```

The runner performs seventeen useful customer invocations: one full-graph
baseline that captures the output contract in the same Gradle execution, then
eight alternating control/candidate pairs. It requires nine full-graph runs,
eight candidate runs, one baseline observation, stable required-output bytes
and zero measurement-only customer workflows. A small fixture may retain
native Gradle; this gate proves the transaction and unchanged evidence/payback
rules rather than a synthetic speedup. The 12-CPU result remains bounded local
POC evidence, not golden-runner or production evidence.

Reproduce verified unaffected-output materialization in a clean workspace and
validate the checked evidence with:

```bash
./dev/run-verified-output-materialization \
  /absolute/path/to/buildopt \
  /tmp/verified-output-materialization.json
./dev/check-verified-output-materialization \
  /tmp/verified-output-materialization.json
```

The runner executes one full baseline, one unchanged control, one clean
structural candidate and one deliberate corruption fallback. It requires the
candidate to rebuild changed outputs while BuildOpt restores only unaffected
required outputs, compares the complete output set by exact bytes and proves
that corrupt retained state restores the full native graph before candidate
execution. The result is a bounded correctness POC, not a wall-time claim.

Reproduce the generic aggregate-workflow partition and validate its checked
evidence with:

```bash
./dev/run-aggregate-workflow-partition \
  /absolute/path/to/buildopt \
  /tmp/aggregate-workflow-partition.json
./dev/check-aggregate-workflow-partition \
  /tmp/aggregate-workflow-partition.json
```

The runner creates a 66-project Groovy DSL `assemble` workflow with one direct
change owner and 65 transitive consumers. It proves that the old 66-entrypoint
proposal becomes one bounded candidate entrypoint, the 65 unaffected JARs are
materialized from exact revision-bound state and the clean candidate's full
output digest matches the baseline. The runner makes no wall-time claim.

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

## Retired Runtime Tuning evidence

Runtime Tuning failed the later optimized-native value gates, so its evaluator,
workflow, optimizer, activation variables, and fresh-run harnesses have been
removed. The immutable historical result remains revalidated with
`./dev/check-runtime-owner-evaluation`; it is not an active capability.


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

Validate the revision-bound strict-runner baseline without rerunning builds:

```bash
./dev/check-poc-value-baseline
```

The retained capture accepts favorable and unfavorable measurements and is
historical evidence. Its runner was removed with Runtime Tuning; current
experiments use the mechanism-specific benchmark harnesses instead.

Validate the current POC decision, evidence classifications, and open value
gates without rerunning benchmarks:

```bash
./dev/check-poc-value-validation
```

Validate the strict `POC-VALUE-002` decisions without rerunning Gradle:

```bash
./dev/check-poc-value-negative-mechanisms
```

The checked evidence proves effective 4-CPU/16-GiB cgroups, identical outputs,
zero product failures, default fallback to Gradle's native cache, and rejection
of both tested Runtime Tuning candidates. Fresh Runtime Tuning execution is
intentionally unavailable.

Validate the strict `POC-VALUE-003` accelerator coverage without rerunning
Gradle:

```bash
./dev/check-poc-value-coverage
```

The checked matrix contains eight alternating pairs for each Kotlin/Groovy
Build Impact and exact reviewed custom-task/Patch workload. The native control
keeps Build Cache, Configuration Cache, parallelism, and daemon enabled. To
reproduce the bounded run inside the pinned 4-CPU/16-GiB image:

```bash
GRADLE_USER_HOME=.tools/gradle-user-home/local \
  ./dev/run-poc-value-coverage-container \
    /tmp/poc-value-coverage.json
```

This qualifies only the two synthetic workload classes and exact reviewed
recipe represented by the fixtures. It does not qualify other Patch recipes,
production readiness, or Test Optimization behavior.

Validate the strict `POC-VALUE-004` combined public-path decision without
rerunning Gradle:

```bash
./dev/check-poc-value-combined
```

The checked report contains eight alternating pairs in the same four
Kotlin/Groovy workload cells. The candidate uses the installed
`buildopt-impact check` and `buildopt gradle` commands plus the packaged plugin;
the control uses optimized native Gradle. Unproven Safe Cache and Runtime
profiles remain disabled. Reproduce it inside the pinned 4-CPU/16-GiB image
with:

```bash
GRADLE_USER_HOME=.tools/gradle-user-home/local \
  ./dev/run-poc-value-combined-container \
    /tmp/poc-value-combined.json
```

The resulting `CONTINUE` decision is bounded to the owner-controlled synthetic
classes. It is not a universal-savings, production-readiness, soak, design
partner, or Test Optimization claim.

Generalize that bounded claim across realistic change shapes without rerunning
the strict benchmark:

```bash
./dev/check-poc-breadth
./dev/check-poc-overhead
./dev/check-poc-stability
./dev/check-poc-pairing
./dev/check-poc-groovy
./dev/check-poc-shared-groovy
./dev/check-poc-leaf-kotlin
./dev/check-poc-kotlin-stability
./dev/check-poc-kotlin-boundary
./dev/test-poc-pairing
```

The checked report covers no-change, leaf-source, shared-source, and global
build-logic changes in Kotlin and Groovy. Each cell contains eight alternating
pairs, exact task-selection or full-fallback counts, Configuration Cache state,
byte-identical required outputs, and zero product-attributable failures. Create
fresh revision-bound evidence inside the pinned 4-CPU/16-GiB image with:

```bash
GRADLE_USER_HOME=.tools/gradle-user-home/local \
  ./dev/run-poc-breadth-container /tmp/poc-breadth.json
```

Both a pass and a fail are valid evidence. A fail retains the already qualified
narrow POC claim and identifies the workload class that must not be generalized.
The checked repeat currently fails breadth with 4/8 qualifying cells: no-change
Kotlin preserves parity, and leaf Kotlin, shared Kotlin, and leaf Groovy clear
the accelerator threshold. The diagnostic phase report proves the measured
candidate is native-only and loads no init/project plugin. Remaining Groovy and
global-build-logic failures are order-sensitive and stay outside the claim.

`check-poc-stability` compares two revision-bound strict batches. Each batch
runs control and candidate in separate 4-CPU/16-GiB containers with independent
writable workspaces, Gradle homes, and daemons; the second batch reverses which
complete arm runs first. Each workload cell keeps its private daemon and Gradle
cache after the same single warm-up used by the original breadth contract.
Fixture, mutation sequence, tasks, eight samples, outputs, and thresholds remain
unchanged. Create the two reports with:

```bash
GRADLE_USER_HOME=.tools/gradle-user-home/local \
  ./dev/run-poc-stability-container \
    /tmp/poc-stability-control-first.json control-first CONTROL_FIRST
GRADLE_USER_HOME=.tools/gradle-user-home/local \
  ./dev/run-poc-stability-container \
    /tmp/poc-stability-candidate-first.json candidate-first CANDIDATE_FIRST
./dev/check-poc-stability \
  /tmp/poc-stability-control-first.json \
  /tmp/poc-stability-candidate-first.json \
  benchmarks/results/poc-stability-v1-decision.json
```

`check-poc-pairing` removes the remaining long runner-time separation without
sharing writable state. Two strict containers stay alive concurrently, one per
arm. Every workload cell is warmed privately and then measured as eight
consecutive cross-container pairs with alternating arm order. A second batch
reverses the starting order. Every pair records timestamps and must keep the
inter-arm idle gap at or below five seconds. Tasks, fixture, samples, outputs,
thresholds, and correctness guardrails remain unchanged.

Run the two strict batches, assemble their decision, and validate them with:

```bash
./dev/run-poc-pairing-container \
  /tmp/poc-pairing-control-first.json paired-control-first CONTROL_FIRST
./dev/run-poc-pairing-container \
  /tmp/poc-pairing-candidate-first.json paired-candidate-first CANDIDATE_FIRST
./dev/assemble-poc-pairing-decision \
  /tmp/poc-pairing-decision.json \
  /tmp/poc-pairing-control-first.json \
  /tmp/poc-pairing-candidate-first.json
./dev/check-poc-pairing \
  /tmp/poc-pairing-control-first.json \
  /tmp/poc-pairing-candidate-first.json \
  /tmp/poc-pairing-decision.json
```

A stable negative classification is useful POC evidence and authorizes work on
that reproduced value gap. A classification mismatch remains a measurement
failure and does not authorize a product change.

The validator recomputes every observation and verifies the checked decision.
Classification agreement is the stability gate, not a prerequisite for keeping
honest evidence. The checked pairing batches reproduce six of eight cells;
build-logic Kotlin and shared Kotlin remain blocked as measurement mismatches.
The reproduced no-change and shared-source Groovy failures are the only Groovy
cells authorized for product experiments.

`POC-GROOVY-001` uses the `GROOVY_VALUE` profile to test the no-change fix and
preserve leaf value. The generic fixture remains unchanged; an init script
raises only its existing deterministic verification rounds for both arms. Both
arms execute consecutively inside one strict container, while their workspaces,
installations, writable state, Gradle homes, and daemons remain separate. Run
and validate the two opposite-order batches with:

```bash
BUILDOPT_POC_PAIRING_PROFILE=GROOVY_VALUE \
  ./dev/run-poc-pairing-container \
    /tmp/poc-groovy-control-first.json groovy-control-first CONTROL_FIRST
BUILDOPT_POC_PAIRING_PROFILE=GROOVY_VALUE \
  ./dev/run-poc-pairing-container \
    /tmp/poc-groovy-candidate-first.json groovy-candidate-first CANDIDATE_FIRST
BUILDOPT_POC_PAIRING_PROFILE=GROOVY_VALUE \
  ./dev/assemble-poc-pairing-decision \
    /tmp/poc-groovy-decision.json \
    /tmp/poc-groovy-control-first.json \
    /tmp/poc-groovy-candidate-first.json
./dev/check-poc-groovy \
  /tmp/poc-groovy-control-first.json \
  /tmp/poc-groovy-candidate-first.json \
  /tmp/poc-groovy-decision.json
```

The checked reports reproduce no-change parity and leaf acceleration in both
orders with unchanged thresholds, byte-identical outputs, and zero product
failures. Percentages are kept separate by batch.

Measure the shared-source Groovy cell on the same calibrated boundary with:

```bash
BUILDOPT_POC_PAIRING_PROFILE=GROOVY_SHARED_VALUE \
  ./dev/run-poc-pairing-container \
    /tmp/poc-shared-groovy-control-first.json \
    shared-groovy-control-first CONTROL_FIRST
BUILDOPT_POC_PAIRING_PROFILE=GROOVY_SHARED_VALUE \
  ./dev/run-poc-pairing-container \
    /tmp/poc-shared-groovy-candidate-first.json \
    shared-groovy-candidate-first CANDIDATE_FIRST
BUILDOPT_POC_PAIRING_PROFILE=GROOVY_SHARED_VALUE \
  ./dev/assemble-poc-pairing-decision \
    /tmp/poc-shared-groovy-decision.json \
    /tmp/poc-shared-groovy-control-first.json \
    /tmp/poc-shared-groovy-candidate-first.json
./dev/check-poc-shared-groovy \
  /tmp/poc-shared-groovy-control-first.json \
  /tmp/poc-shared-groovy-candidate-first.json \
  /tmp/poc-shared-groovy-decision.json
```

The checked reports require five control verifications, two affected candidate
verifications, unchanged thresholds, identical outputs, and classification
agreement across both starting orders.

Measure the leaf-source Kotlin cell on its equivalent calibrated boundary with:

```bash
BUILDOPT_POC_PAIRING_PROFILE=KOTLIN_LEAF_VALUE \
  ./dev/run-poc-pairing-container \
    /tmp/poc-leaf-kotlin-control-first.json \
    leaf-kotlin-control-first CONTROL_FIRST
BUILDOPT_POC_PAIRING_PROFILE=KOTLIN_LEAF_VALUE \
  ./dev/run-poc-pairing-container \
    /tmp/poc-leaf-kotlin-candidate-first.json \
    leaf-kotlin-candidate-first CANDIDATE_FIRST
BUILDOPT_POC_PAIRING_PROFILE=KOTLIN_LEAF_VALUE \
  ./dev/assemble-poc-pairing-decision \
    /tmp/poc-leaf-kotlin-decision.json \
    /tmp/poc-leaf-kotlin-control-first.json \
    /tmp/poc-leaf-kotlin-candidate-first.json
./dev/check-poc-leaf-kotlin \
  /tmp/poc-leaf-kotlin-control-first.json \
  /tmp/poc-leaf-kotlin-candidate-first.json \
  /tmp/poc-leaf-kotlin-decision.json
```

The checked reports require five control verifications, one affected candidate
verification, unchanged thresholds, identical outputs, and classification
agreement across both starting orders.

Recheck the two remaining Kotlin classification mismatches with:

```bash
BUILDOPT_POC_PAIRING_PROFILE=KOTLIN_STABILITY_VALUE \
  ./dev/run-poc-pairing-container \
    /tmp/poc-kotlin-stability-control-first.json \
    kotlin-stability-control-first CONTROL_FIRST
BUILDOPT_POC_PAIRING_PROFILE=KOTLIN_STABILITY_VALUE \
  ./dev/run-poc-pairing-container \
    /tmp/poc-kotlin-stability-candidate-first.json \
    kotlin-stability-candidate-first CANDIDATE_FIRST
BUILDOPT_POC_PAIRING_PROFILE=KOTLIN_STABILITY_VALUE \
  ./dev/assemble-poc-pairing-decision \
    /tmp/poc-kotlin-stability-decision.json \
    /tmp/poc-kotlin-stability-control-first.json \
    /tmp/poc-kotlin-stability-candidate-first.json
./dev/check-poc-kotlin-stability \
  /tmp/poc-kotlin-stability-control-first.json \
  /tmp/poc-kotlin-stability-candidate-first.json \
  /tmp/poc-kotlin-stability-decision.json
```

The checked evidence deliberately retains the failed stability gate: both
shared-source and global-build-logic change classification between batches,
despite exact correctness and execution guardrails. It authorizes no product
change and requires a measurement-boundary decision before another rerun.

Validate the terminal measurement-boundary decision without collecting a new
sample:

```bash
./dev/check-poc-kotlin-boundary
```

The checker binds the decision to the exact `E-163` and `E-167` report hashes,
recomputes four-batch classification and threshold histories for both cells,
and requires `STOP_RETAIN_BOUNDED_CLAIM`. No further unchanged replication,
threshold movement, pair discard, or product tuning is authorized inside the
current POC.

## Public-repository compatibility

Validate the pinned repository manifest without network access:

```bash
./dev/check-poc-real-world-compatibility --spec-only
```

Create strict compatibility evidence by cloning the three exact public
revisions and running their representative tasks through both native Gradle and
the installed BuildOpt entry point:

```bash
./dev/run-poc-real-world-compatibility-container \
  /tmp/poc-real-world-compatibility.json
```

The runner uses disposable checkouts and separate empty homes on the pinned
4-CPU/16-GiB image. It removes CI, Android, signing, cache, and scan credentials
from the child environment; disables build scans; rejects publishing tasks; and
requires byte-identical non-empty outputs. The run resolves only public source,
plugins, wrappers, and dependencies. It records compatibility, not performance.
Only a passing checked document authorizes the paired real-world performance
matrix.

## Public-repository performance replication

Validate the preregistered matrix before observing any timings:

```bash
./dev/check-poc-real-world-performance --spec-only
```

Run the complete paired matrix in the pinned 4-CPU/16-GiB image:

```bash
./dev/run-poc-real-world-performance-container \
  /tmp/poc-real-world-performance.json
```

The runner performs an unmeasured online preflight, disconnects both arm
containers, warms their private persistent daemons offline, and records eight
temporally paired observations for `NO_CHANGE` and `LEAF_SOURCE_CHANGE` in
each compatible public repository. It never adds percentages across cells and
accepts a negative terminal decision as valid evidence.

Validate the checked-in result without rerunning Gradle:

```bash
./dev/check-poc-real-world-performance
```

The terminal evidence qualifies Mockito but not Spotless or SpotBugs, so it
retains the bounded synthetic claim and authorizes no unchanged rerun, product
tuning, or broader public-repository claim.

Validate and print the historical mechanism-development scorecard without
rerunning a benchmark:

```bash
./dev/check-build-optimization-performance
```

The scorecard keeps each mechanism attributable and never adds percentages
from different workloads. It retains the historical OpenTelemetry profile and
also prints the qualified clean composition separately, so a regressive
included mechanism cannot be hidden by a faster terminal arm. Current
activation and continuation decisions come from `check-poc-value-validation`;
the combined result is a separately measured path, not the sum of the
mechanisms. Fresh cache and Build Impact evidence can be created with
`run-cache-parity-benchmark` and `run-build-impact-performance`; each has a
matching `check-*` command. See [the benchmark index](../benchmarks/README.md#build-optimization-scorecard).

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

Validate the revision-bound public-workflow profiles and their generic
hypothesis decision without rerunning the long diagnostic:

```bash
./dev/check-poc-real-world-diagnostics
```

The fresh capture command is
`./dev/run-poc-real-world-diagnostics-container <output.json>`. It runs exact
upstream workflows in disposable checkouts on the digest-pinned 4-CPU/16-GiB
runner. The output is diagnostic only: task durations can overlap and are
never summed into a savings claim.

Validate the corrected build-task ownership boundary and frozen Spotless and
Mockito test-build value experiments with:

```bash
./dev/check-poc-public-build-tasks
```

This check does not run a benchmark. It binds the unchanged diagnostic,
Mockito's Tier 1 `compileTestJava` eligibility, both public revisions, the
optimized native controls, exact outputs and unchanged value thresholds before
any new timing is observed.

Validate the completed Spotless exact-workflow experiment without rerunning it:

```bash
./dev/check-poc-spotless-impact
```

The checked result stops this alternative: it saved 296.375 ms/4.22% on
average, but missed the fixed 500-ms floor and its 95% interval crossed zero.
To create a fresh preregistered capture from a clean checkout, use
`./dev/run-poc-spotless-impact-container <output.json>`.

Validate the completed Mockito test-build experiment without rerunning it:

```bash
./dev/check-poc-mockito-test-build
```

The checked result stops this alternative: BuildOpt restored the exact same
1,260 test-class outputs and averaged 11.80% faster, but saved only 281.375 ms
and its 95% interval crossed zero. The gated complete Mockito workflow was not
run. To create a fresh preregistered capture from a clean checkout, use
`./dev/run-poc-mockito-test-build-container <output.json>`.

Validate the Spring Framework value-iteration contract before running its
long baseline or observing any local timing:

```bash
./dev/check-poc-spring-framework
```

The contract pins a public revision with green upstream CI, Gradle 9.6.1,
Temurin 25, the 12-CPU local runner, and separate `assemble`, `testClasses`,
and `check` cells. Spring's native Build Cache and parallel execution remain
enabled. No result may be transferred to OpenTelemetry unless it clears the
unchanged 500-ms, 2%, and positive-lower-bound gate with identical outputs and
all requested tests preserved.

The post-diagnostic Checkstyle Runtime Tuning hypothesis is historical. It did
not clear the POC value floor and its activation boundary has been removed.

Capture the revision-bound Spring diagnostic on the local 12-CPU POC host:

```bash
./dev/run-poc-spring-diagnostic /absolute/path/to/result.json
```

The runner downloads and verifies the pinned archive, performs an unmeasured
online preflight, freezes the resulting native Gradle cache, and restores that
same cache seed before the offline `assemble`, `testClasses`, and `check`
cells. It records wall clock, Gradle profile phases, critical tasks, memory,
required outputs, and exact test-case outcomes. The diagnostic authorizes
follow-up hypotheses but never turns overlapping task durations into a savings
claim. Spring's four implicit `buildSrc` tests are retained in every cell; the
top-level `assemble` and `testClasses` requests still select no product `Test`
task. Validate checked-in evidence without rerunning Spring with:

```bash
./dev/check-poc-spring-diagnostic
```

After the Runtime Tuning candidate and measurement contract are committed,
run the accepted full-verification comparison with:

```bash
./dev/check-poc-spring-value /absolute/path/to/result.json
```

The runner performs one unmeasured warmup per arm and then eight alternating
pairs. Before every measured arm it removes outputs, restores the same native
cache seed from the original source, and applies the fixed source mutation.
Both arms must execute `checkstyleNohttp`; cached or failed pairs are rejected,
not discarded.

The Checkstyle candidate stopped before accepted pairs: its 3.41-second
isolated saving could not reach 2% of the 546-second control warmup, which also
exposed an upstream `java24Test` timing failure. The next authorized Spring
experiment is test-process concurrency with the complete test set unchanged;
the Checkstyle mechanism is not eligible for OpenTelemetry transfer.

Run the non-gating test-runtime discovery with:

```bash
./dev/run-poc-spring-test-runtime-discovery /absolute/path/to/result.json
./dev/check-poc-spring-test-runtime-discovery /absolute/path/to/result.json
```

The discovery executes three complete offline `check` cells from the same
native cache seed: 12 native workers, 6 native workers, and 6 workers with two
forks for each `Test` task that retained Gradle's one-fork default. Explicit
repository parallelism remains unchanged. The result selects a candidate for a
later paired experiment; it cannot itself support a savings claim.

The checked discovery rejected global two-fork tuning. The stable native
6-worker cell completed 41,276 tests in 523.858 seconds. Global two-fork tuning
retained the same test count but failed after 649.929 seconds, a 24.07%
regression. The next discovery may parallelize only large default-fork suites
through a generic predeclared size rule; the failed global policy is not a
product candidate.

The frozen selective follow-up is:

```bash
./dev/run-poc-spring-selective-test-runtime-discovery /absolute/path/to/result.json
```

It compares a fresh six-worker native cell with six workers plus two forks only
for conventional `test` tasks whose project contains at least 500 Java, Groovy,
or Kotlin files below `src/test`. It preserves additional test suites and every
explicit repository fork value. The threshold and two-fork value are fixed
before execution; this discovery is still non-gating and cannot claim savings.

The checked selective discovery rejected that final fork hypothesis. Native
Gradle completed 41,276 tests in 541.782 seconds; the candidate retained the
same cases and byte-identical 23,680-file output manifest but failed
`:spring-test:test` after 550.310 seconds, an 8.528-second/1.57% regression.
Do not retry or tune the threshold. The next Spring block is the independent,
build-owned `testClasses` graph; no Gradle `Test` task may be changed or run by
that Build Impact experiment.

The next experiment is frozen in
`specs/poc-spring-test-preparation-v2.json`. Validate its ownership, exact
entrypoints, eight-pair design, outputs, and unchanged gate with:

```bash
./dev/check-poc-spring-test-preparation
./dev/check-build-impact-automatic
```

The discovery adapter resolves the root-build Gradle task graph, detects any
real root-build `Test`, then disables every requested/dependency task before the
discovery action runs. Only a complete, root-Test-free graph may compare native
`testClasses` with `:spring-jms:testClasses`; unknown or test-containing graphs
remain full-build. A pre-measurement dry run found Spring's unavoidable
`:buildSrc:test` task outside that root graph. Protocol v2 therefore requires
that common build-logic task to remain `UP-TO-DATE` in both arms while still
forbidding every root-build `Test`; no accepted pair or value threshold existed
when that scope correction was frozen.

After committing the runner so its SHA is immutable, capture the eight paired
observations on the declared 12-CPU host with:

```bash
./dev/run-poc-spring-test-preparation /absolute/path/to/result.json
./dev/check-poc-spring-test-preparation-result /absolute/path/to/result.json
```

The installed Spring breadth experiment is frozen in
`specs/poc-spring-impact-breadth-v1.json`. It compares two additional selective
output scopes against optimized native Gradle and separately proves that a
global build-logic change retains the full graph:

```bash
./dev/run-poc-spring-impact-breadth /absolute/path/to/result.json
./dev/check-poc-spring-impact-breadth /absolute/path/to/result.json
```

The runner uses an isolated installed package, all 12 host CPUs, one unmeasured
preflight, four alternating pairs per selective cell, byte-identical declared
outputs and the unchanged 500-ms/2%/positive-bound gate. It is a bounded POC
experiment, not a production or Test Optimization path.

The runner downloads the fixed Spring revision, performs networked dependency
preparation outside the measured region, stops that daemon, and then measures
offline control/candidate arms with separate Gradle homes and the same restored
native cache seed. It executes no root-build Gradle `Test`, requires the same
`:buildSrc:test UP-TO-DATE` outcome in both arms, compares the non-empty
`spring-jms/build/classes` manifests byte for byte, retains all eight pairs,
and applies the unchanged 500 ms, 2%, positive-lower-bound value gate.

The unchanged transfer to OpenTelemetry Java Instrumentation is frozen in
`specs/poc-otel-test-preparation-v1.json`. It retains OpenTelemetry's native
Build Cache, parallelism, parallel Configuration Cache, JDK 21, and all 12
workers. The control is the native unqualified `testClasses` selector; the
candidate is only
`:instrumentation:spring:spring-boot-autoconfigure:testClasses`. Both arms use
separate writable Gradle homes plus the same immutable read-only dependency
cache and restored native build-cache seed. Every measured invocation is
offline, executes no Gradle `Test`, and must produce the same non-empty affected
class manifest.

After this runner and contract are committed, capture the eight alternating
pairs with:

```bash
./dev/run-poc-otel-test-preparation /absolute/path/to/result.json
./dev/check-poc-otel-test-preparation-result
```

If preparation or an arm fails, the runner preserves only its diagnostic logs
beside the requested output as `<result>.failure-logs.tar.gz` before removing
the multi-gigabyte temporary workspace. No failed or partial pair is promoted
to evidence. OpenTelemetry's Thrift generator creates four UID-`nobody` source
files; before each native `clean`, the runner makes only that pinned output
directory removable through the network-disabled cleanup image.

The fixed 500 ms, 2%, positive-lower-bound gate and all eight observations are
retained regardless of outcome. This is a POC value transfer only: it makes no
production, soak, design-partner, universal-performance, or Test Optimization
claim.

The accepted run stopped at pair 7 when the optimized native control failed in
upstream OpenTelemetry Byte Buddy processing with `zip file closed`. Six
completed pairs remain descriptive-only; they are not promoted through an
incomplete eight-pair gate. The checked terminal evidence records that failure
and the no-retry decision, so that exact experiment remains terminal.

A separate experiment on the stable OpenTelemetry `v2.30.0` release is frozen
in `specs/poc-otel-spring-family-v2.json`. Its root `testClasses` probe was
rejected after the fixed 20-minute budget without accepting a timing. The
replacement control is the 53-task Spring instrumentation family already
separated by the upstream pull-request workflow; it completed an offline
preflight in Gradle-reported `2m45s` without executing a Gradle `Test` task.
Validate the immutable revision, hashes, task list, graph, four-pair protocol,
fallback and unchanged value gate with:

```bash
./dev/check-poc-otel-spring-family-v2
```

The runner and result checker can reproduce the fixed installed-path
experiment with:

```bash
./dev/run-poc-otel-spring-family-v2 /absolute/path/to/result.json
./dev/check-poc-otel-spring-family-v2-result /absolute/path/to/result.json
```

The runner packages and installs the current revision, downloads only the
pinned public source archive, prepares dependencies once outside timing, then
runs all measured arms offline. A failed warmup, arm, output comparison,
fallback or timing-gap assertion stops the run; completed observations are not
substituted.

For owner-operated overhead attribution, `buildopt impact` also accepts
`--timings-file PATH`. The repository-relative destination receives a private
canonical report whose top-level phases reconcile exactly and whose nested
planner phases separate manifest, graph, generated-state and evaluation work.
The option does not change selection or authorization. Validate the report
with `./dev/check-build-impact-phase-timings PATH`; its contract is documented
in `specs/build-impact-poc-phase-timings-v1.md`.

The accepted terminal evidence is checked in at
`benchmarks/results/poc-otel-spring-family-v2.json`. Native Gradle averaged
14,961.5 ms and BuildOpt 13,464.5 ms, but the frozen value gate failed because
one of four pairs regressed and the paired interval crossed zero. Validate that
the complete evidence, exact outputs, global fallback and retain decision have
not drifted with:

```bash
./dev/check-poc-otel-spring-family-v2-result \
  benchmarks/results/poc-otel-spring-family-v2.json
```

Protocol revision 3 corrects a pre-warmup graph-hash stop: the first runner
constructed different manifest IDs and one extra global path from the exact
manifest used during preregistration. Zero warmups and zero observations had
started. The runner now reproduces the pinned graph byte for byte and reports
the safe unknown-path fallback reason exactly; the experiment and thresholds
are unchanged.

This is a new stable-release and control-boundary experiment. It neither
retries nor reinterprets the earlier terminal result, and it changes no test
selection, production authority, public release, soak or design-partner scope.

The optimized terminal protocol keeps that native control and frozen value
gate, but replaces the aggregate candidate with four typed compile producers
and requires an exact-bound hot-state hit in every measured candidate arm.
Validate the preregistration before timing with:

```bash
./dev/check-poc-otel-optimization-v1
```

The original Hot State runner has been retired. Validate its retained terminal
measurement with:

```bash
./dev/check-poc-otel-optimization-v1-result /absolute/path/to/new-result.json
```

The runner stops on any arm failure, output mismatch, missing hot-state hit,
unsafe fallback or excessive inter-arm gap. It never discards a measured pair
or moves the frozen 500-ms, 2%, positive-bound and 4/4-pair gate.

The follow-up repeated-work POC enables native-cache eligibility only for an
exact unmodified standard Gradle `Jar` producer:

```bash
./dev/check-poc-standard-jar-cache
```

Its TestKit proof requires the standard `Jar` to replay byte-identically while
a custom `Jar` with an extra action and a `Copy` task execute normally. This
adapter is explicit on `buildopt impact`; it does not widen the managed Tier 1
safe-cache policy.

The historical terminal protocol measured that adapter over one shared hot
Gradle runtime. Fresh runs of the Hot State composition are retired:

```bash
./dev/check-poc-otel-optimization-v2
./dev/check-poc-otel-optimization-v2-result /absolute/path/to/new-result.json
```

The control shares the same checkout, Gradle user home and daemon but remains
unable to consume the candidate-only standard-Jar entry. The unchanged gate
still requires 500 ms, 2%, a positive paired lower bound, 4/4 positive pairs,
exact outputs, zero product failures and the complete global fallback.

The qualified full-path ablation re-executes the frozen Spring and
OpenTelemetry protocols and records all six logical arms without inventing
zero effects for mechanisms that are not authorized for a workload:

```bash
./dev/check-poc-full-path-ablation /absolute/path/to/poc-full-path-ablation-v1
./dev/test-poc-full-path-ablation
```

Each executable arm retains its own contemporaneous optimized-native control.
Cross-protocol differences are descriptive only, percentages are never added,
and a failed source protocol stops the run without discarding a pair.

The clean OpenTelemetry composition removes exact-bound hot-state reuse from
the candidate while retaining Build Impact, the exact standard-`Jar` adapter,
the same native control, outputs, fallback, and frozen value gate:

```bash
./dev/check-poc-otel-clean-composition-v1
./dev/run-poc-otel-clean-composition-v1 /absolute/path/to/new-result.json
./dev/check-poc-otel-clean-composition-v1-result /absolute/path/to/new-result.json
./dev/test-poc-otel-clean-composition-v1
```

The checked repository evidence qualifies the clean composition at 5,361.25 ms
or 50.40% mean saving over optimized native Gradle. All four pairs are positive,
the paired interval is +4,334.25..+5,937 ms, all 125 required outputs are
identical, every candidate restores the exact standard `Jar` producer from
cache, and no measured invocation enables exact-bound hot state. The negative
test proves that the checker rejects both hot-state contamination and output
drift.

The installed qualified-profile matrix remeasures the public `buildopt poc`
path independently on Spring, OpenTelemetry and Kafka. It enables only the
mechanisms already qualified for each fixed scope and never averages repository
percentages:

```bash
./dev/check-poc-qualified-profile-matrix-v1
./dev/run-poc-qualified-profile-matrix-v1 \
  /absolute/path/to/result-directory \
  /absolute/path/to/installed/buildopt \
  /absolute/path/to/kafka-source.tar.gz
./dev/check-poc-qualified-profile-matrix-v1-result \
  /absolute/path/to/result-directory/summary.json
```

The three complete cell documents are retained next to the summary. A broad
continuation requires at least two independently qualifying repository
families; every failed cell retains its optimized native Gradle path. The
checked terminal result qualifies only Kafka: 28,523.25 ms/81.85% saved with
8/8 positive pairs. Spring saves 1,895 ms/14.33% but has only 7/8 positive
pairs, and OpenTelemetry records a zero-observation impact-discovery
preparation failure. The resulting `SPECIALIZE_QUALIFIED_PROFILES` decision
does not average repository percentages and keeps native Spring/OpenTelemetry
paths active.

The deterministic discovery gate then reconstructs only that retained Kafka
profile from the checked matrix, manifest, graph, generated state, trace/input
digests, and reviewed contract:

```bash
./dev/check-poc-profile-discovery
```

It builds the installed CLI, proves two invocations are byte-identical, compares
the embedded profile with the reviewed v2 fixture, checks Spring and
OpenTelemetry native fallbacks, and runs the Go drift/uncertainty matrix. It
does not run Gradle or create a new performance observation.

The trace-gated hypothesis checker then recomposes the terminal decision from
the immutable installed synthetic and Spring attribution traces:

```bash
./dev/check-poc-trace-hypothesis-v1
```

It verifies the exact input and composer digests, required-output/failure
boundaries, three family summaries and five candidate phases. No phase is both
causally recoverable and at least 500 ms in two families, so the byte-stable
result is `NO_ACTIONABLE_HYPOTHESIS`. The command collects no timing and does
not change product code or activation authority.

The terminal portfolio checker then recomposes the final POC decision from the
installed matrix, Kafka cell, deterministic discovery result and trace result:

```bash
./dev/check-poc-portfolio-decision-v1
```

It requires the exact SHA-256-bound inputs, revalidates every upstream result,
forbids cross-repository averages and additive mechanism effects, and emits
`SPECIALIZE_BOUNDED_KAFKA_PROFILE`. Only the explicit reviewed Kafka profile is
retained; Spring and OpenTelemetry stay on optimized native Gradle. It runs no
benchmark and creates no production, soak, design-partner, or Test Optimization
authority.

The completed task-tail experiment adds one exact standard-`Copy` adapter
selected from the same OpenTelemetry trace. It deliberately measures isolated and
cumulative effects rather than adding component percentages:

```bash
./dev/check-poc-standard-copy-cache
./dev/check-poc-standard-copy-cascade
```

The three independent comparisons are Copy-only versus optimized native,
Copy's incremental value on top of Build Impact plus the qualified standard
`Jar` adapter, and the complete composed profile versus optimized native. Each
comparison has its own four alternating pairs and frozen value gate. This
direct complete-profile measurement is the only accepted evidence for a
cascade effect; individual percentages are never summed.

The checked result confirms that cascade. The directly measured complete
Impact + Jar + Copy profile saves 4,377 ms/52.89%, has 4/4 positive pairs and a
+4,130.25..+4,653.25-ms interval. Copy-only and incremental Copy show favorable
means of 27.05% and 24.90%, but both have only 3/4 positive pairs and intervals
that cross zero. All twelve observations preserve the same 21,818-file output
digest and the global fallback succeeds. The terminal decision is therefore
`RETAIN_STANDARD_COPY_EVIDENCE_ONLY`: keep Copy disabled while retaining the
qualified whole-profile cascade evidence.

Before adding another normal-build adapter, review the remaining retained real
task tails and their source hashes:

```bash
./dev/check-poc-normal-build-tail-expansion
./dev/test-poc-normal-build-tail-expansion
```

The checked review adds no timing claim and finds zero actionable candidates.
Standard `Jar` is already qualified, standard `Copy` failed its direct
incremental gate, custom `ShadowJar` is below the value floor and already
served by native cache, configured `JavaExec` is below the floor with broader
process effects, and Spring selected no new standard task. Its terminal
decision is `STOP_NORMAL_BUILD_TASK_ADAPTER_EXPANSION_NO_ACTIONABLE_TAIL`;
another adapter requires a materially different dominant tail.

The follow-up test-build experiment executes the same filtered Spring Test
task in both arms and changes only exact standard-JAR eligibility:

```bash
./dev/check-poc-spring-test-build-optimization
./dev/run-poc-spring-test-build-optimization /absolute/path/to/new-result.json
./dev/check-poc-spring-test-build-optimization /absolute/path/to/new-result.json
```

The checked result is a stop, not an activation. BuildOpt restores three exact
`testFixturesJar` producers and preserves the same 8 tests and 15 JARs, but
regresses the complete workflow by 735.25 ms/11.31%, with 0/4 positive pairs
and a -1,449..-113-ms interval. The direct `buildopt gradle` experiment option
remains diagnostic-only for reproducibility; the standard-JAR adapter is not
promoted beyond the Build Impact scope where it independently qualified.

The follow-up overhead ablation separates the complete native workflow from
loading the packaged init/plugin classpath without an optimization and from
the exact standard-JAR adapter. It uses four rotated three-arm rounds, one
warm-up per arm, independent Gradle homes, offline measured execution, the
same eight tests, and byte-identical JARs:

```bash
./dev/check-poc-optimization-overhead-ablation
./dev/run-poc-optimization-overhead-ablation /absolute/path/to/new-result.json
./dev/check-poc-optimization-overhead-ablation /absolute/path/to/new-result.json
```

The retained result keeps native Gradle for this workflow. Init/plugin-only
averaged 1,061.75 ms slower than native; the three exact hits recovered 449.5
ms relative to that arm but the complete adapter remained 612.25 ms/9.53%
slower than native, with only 2/4 positive rounds and an interval crossing
zero. These phase differences are diagnostic under the recorded rotated
orders; only the end-to-end native comparison controls activation.

The final bounded Runtime Tuning experiment tested one trace-selected resource
hypothesis: Spring `testClasses` with 12 native workers versus a six-worker
cap. Its fresh-run harness is retired; validate the immutable evidence with:

```bash
./dev/check-poc-runtime-research
```

The retained result closes that hypothesis without activation. Native Gradle
at 12 workers averages 9,556.75 ms; the six-worker candidate averages 9,748.25
ms, losing 191.5 ms/2.00%. Only 2/4 pairs favor the candidate and the paired
interval is -973.5..+590.5 ms. All pairs preserve 378 outputs and exact sorted
task outcomes. The terminal decision is `RETAIN_NATIVE_12_WORKERS`; do not
search another worker value for this trace.

The controlled remote-cache experiment compares the same Gradle 9.6.1
`HttpBuildCache` client and committed Shared object set through two read paths:
direct Shared access over a frozen 80-ms/20-MiB/s modeled WAN, and a prewarmed
BuildOpt Edge on loopback. The eight cacheable producers restore 32 MiB in
every arm; local cache, Configuration Cache and measured writes are disabled:

```bash
./dev/check-poc-remote-cache-value
./dev/run-poc-remote-cache-value /absolute/path/to/new-result.json
./dev/check-poc-remote-cache-value /absolute/path/to/new-result.json
```

### Real-repository remote-cache transfer

`POC-REMOTE-CACHE-TRANSFER-001` moves the unchanged authenticated Shared/Edge
read path from the deterministic fixture to Apache Kafka 4.3.1
`:clients:testClasses`. Before any cache comparison, three sequential downloads
of Kafka's fixed source archive derive and freeze a 337-ms/6,994,831-B/s link.
Measured builds use the same native Gradle HTTP client and committed objects,
disable local and Configuration caches, preserve the 4,062 required outputs,
and differ only in direct Shared versus prewarmed Edge locality. Because
Gradle's `--offline` also disables its HTTP build cache, JVM proxy properties
block every non-loopback dependency request while leaving the two loopback
cache endpoints available.

```bash
./dev/check-poc-remote-cache-transfer-v1
./dev/run-poc-remote-cache-transfer-v1 \
  /absolute/path/to/new-result.json \
  /absolute/path/to/kafka-source.tar.gz
./dev/check-poc-remote-cache-transfer-v1-result \
  /absolute/path/to/new-result.json
```

The network profile and unchanged 500-ms/2%/positive-bound/4-of-4 gate are
preregistered. No failed pair, dependency preparation, cache seed or Edge
warm-up enters the timing, and no post-result profile search is permitted.

The checked result qualifies Edge locality under that profile. Direct Shared
reads average 8,885.25 ms and Edge averages 7,534 ms, saving 1,351.25
ms/15.21%. All four pairs are positive, the interval is
+788.25..+1,883 ms, outputs and `FROM-CACHE` outcomes are exact, and Edge
makes zero measured Shared reads. It cannot claim that Shared storage itself
is faster than another remote origin.

### Qualified Kafka remote composition

`POC-QUALIFIED-REMOTE-COMPOSITION-001` preregistered one end-to-end effect
instead of adding independent Kafka percentages. Its candidate assumed that
the required client artifact was produced by exact standard `:clients:jar`
before reading the same committed objects through prewarmed Edge.

```bash
./dev/check-poc-qualified-remote-composition-v1
./dev/run-poc-qualified-remote-composition-v1 \
  /absolute/path/to/new-result.json \
  /absolute/path/to/buildopt \
  /absolute/path/to/kafka-source.tar.gz
./dev/check-poc-qualified-remote-composition-v1-result \
  /absolute/path/to/new-result.json
```

The seed invalidated that premise before timing: Gradle reported
`:clients:jar SKIPPED`, custom `:clients:shadowJar` produced the required
artifact, and its digest differed from the historical standard-`Jar` premise.
The checked terminal result records zero Shared objects, no Edge server, no
warm-up and zero measured pairs. This is a successful fail-closed POC outcome,
not a performance percentage. A corrected experiment may compose only
Kafka-qualified Build Impact and Edge locality.

### Kafka Build Impact and Edge composition

`POC-KAFKA-IMPACT-EDGE-COMPOSITION-001` corrects the disproved standard-Jar
premise without changing the Kafka workload or value threshold. The optimized
native control runs the full root `assemble` through the modeled Shared origin;
the installed candidate selects the already qualified three-project packaging
scope and reads the same committed native Gradle cache object through a
prewarmed Edge. Both arms must restore custom `:clients:shadowJar` from cache
and reproduce the seed artifact exactly:

```bash
./dev/check-poc-kafka-impact-edge-composition-v1
./dev/run-poc-kafka-impact-edge-composition-v1 \
  /absolute/path/to/new-result.json \
  /absolute/path/to/buildopt \
  /absolute/path/to/kafka-source.tar.gz
./dev/check-poc-kafka-impact-edge-composition-v1-result \
  /absolute/path/to/new-result.json
```

The standard-Jar adapter, Runtime Tuning, Safe Cache, Hot State and Test
Optimization remain disabled. The protocol contains zero timing observations
until the fixed experiment completes; its result must report the directly
measured combined effect rather than adding the earlier Build Impact and Edge
percentages.

The terminal run produced four positive diagnostic pairs: native full
`assemble` through Shared averaged 43,345 ms and installed Build Impact through
Edge averaged 7,423 ms. The performance gate would pass, but the composition
does not qualify. Under forced Edge failure Gradle disabled remote cache and
completed, while custom `shadowJar` rebuilt different bytes. Validate the
retained terminal decision with the result checker above.

### Kafka shadow JAR reproducibility

`POC-KAFKA-SHADOWJAR-REPRODUCIBILITY-001` isolates that fallback failure with
five fresh source trees and Gradle homes. The baseline uses Kafka's explicit
non-reproducible archive settings; the normalized builds change only the two
`AbstractArchiveTask` properties. The final build injects HTTP 503 through a
loopback fixture and requires Gradle's local rebuild to reproduce the same
normalized artifact:

```bash
./dev/check-poc-kafka-shadowjar-reproducibility-v1
./dev/run-poc-kafka-shadowjar-reproducibility-v1 \
  /absolute/path/to/new-result.json \
  /absolute/path/to/kafka-source.tar.gz
./dev/check-poc-kafka-shadowjar-reproducibility-v1-result \
  /absolute/path/to/new-result.json
```

The checked result proves equal payloads, baseline metadata drift, two
byte-identical normalized rebuilds and one byte-identical HTTP-503 fallback.
It captures no performance claim and does not modify Kafka upstream.

### Qualified Kafka Build Impact and Edge composition rerun

`POC-KAFKA-IMPACT-EDGE-COMPOSITION-002` binds the archive normalization above
before dependency preparation and derives seed, control, and candidate from
the same source. It then collects four fresh alternating pairs and repeats the
global-change and HTTP-503 safety paths:

```bash
./dev/check-poc-kafka-impact-edge-composition-v2
./dev/run-poc-kafka-impact-edge-composition-v2 \
  /absolute/path/to/new-result.json \
  /absolute/path/to/buildopt \
  /absolute/path/to/kafka-source.tar.gz
./dev/check-poc-kafka-impact-edge-composition-v2-result \
  /absolute/path/to/new-result.json
```

The checked evidence qualifies the composition at 82.35% mean savings:
42,992.75-ms native + Shared versus 7,587.25-ms installed Impact + Edge, with
4/4 positive pairs and interval +30,162..+42,487.75 ms. All measured outputs
and the HTTP-503 local rebuild match `3ffd994e...3349`; the global change uses
the full graph and candidate reads cause zero measured Shared traffic. The
claim is limited to the fixed Kafka change and modeled network profile.

### Repository-owned Kafka composition profile

Validate the v2 `buildopt poc` profile that exposes the qualified Kafka Build
Impact plus read-only Edge composition with:

```bash
./dev/check-poc-kafka-composition-usability
```

The checker validates the exact normalized-source SHA precondition, CLI plan,
ambient-endpoint masking, read-only loopback Edge hit, native full-graph
fallbacks and exact local execution after HTTP 503. It references the existing
82.35% composition result and deliberately records no new timing or production
claim.

### Installed Kafka profile value

`POC-KAFKA-INSTALLED-PROFILE-VALUE-001` measures the exact repository-owned v2
profile through the packaged `buildopt poc` command rather than experiment-only
selection wiring:

```bash
./dev/check-poc-kafka-installed-profile-value-v1
./dev/run-poc-kafka-installed-profile-value-v1 \
  /absolute/path/to/new-result.json \
  /absolute/path/to/installed/buildopt \
  /absolute/path/to/kafka-source.tar.gz
./dev/check-poc-kafka-installed-profile-value-v1-result \
  /absolute/path/to/new-result.json
```

Preparation and warm-up are unmeasured. Eight new alternating pairs compare
optimized native Gradle plus Shared with the installed Build Impact plus
read-only Edge profile. The result must preserve the normalized shaded JAR,
global full-graph fallback and HTTP-503 local fallback. Earlier composition
timings cannot be reused.

The checked result qualifies the installed path: optimized native Gradle plus
Shared averages 34,347.25 ms and the packaged profile 6,694.375 ms, saving
27,652.875 ms/**80.51%**. All eight pairs are positive and the corrected
bootstrap interval is +24,826.5..+29,903.625 ms. Exact output, zero candidate
origin requests, native global fallback and byte-identical HTTP-503 fallback
all pass. Revalidate the archived evidence with:

```bash
./dev/check-poc-kafka-installed-profile-value-v1-result
```

The Build Impact generalization protocol broadens the real Spring matrix across
compilation, test preparation, build-owned verification, packaging, and source
distribution. Structural discovery freezes selective execution only where the
declared graph is complete; unknown verification or distribution relationships
must retain the original full selector. The checker accepts a pending result so
the protocol can be committed before timing:

```bash
./dev/check-poc-impact-generalization
./dev/run-poc-impact-generalization /absolute/path/to/new-result.json
./dev/check-poc-impact-generalization /absolute/path/to/new-result.json
```

The measurement uses the installed package, optimized native Gradle control,
four alternating pairs per selective cell, byte-identical non-empty outputs,
and separate build-logic/global fallback cases. It executes no root-build
Gradle `Test` task and leaves Hot State, Runtime Tuning, Safe Cache, Shared/Edge
Cache, and Test Optimization disabled.

The checked result generalizes only shared test preparation: native Gradle
averages 13,971.75 ms and BuildOpt 11,333.75 ms, saving 2,638 ms/18.88% with
4/4 positive pairs and a +1,516..+3,275.5-ms interval. Leaf compilation and
packaging fail the unchanged gate and remain native. Verification and source
distribution retain exact-output full-graph execution because their generated
graphs are incomplete. The negative suite rejects output drift, fallback-reason
drift, and false qualification.

### Third substantial repository transfer

Preregister and execute the unchanged clean BuildOpt profile against the pinned
Apache Kafka 4.3.1 Java/Scala test-preparation workload:

```bash
./dev/check-poc-third-repository-transfer-v1
./dev/run-poc-third-repository-transfer-v1 /absolute/path/to/new-result.json
./dev/check-poc-third-repository-transfer-v1-result /absolute/path/to/new-result.json
./dev/test-poc-third-repository-transfer-v1
```

The installed candidate combines only generic Build Impact and exact standard
`Jar` caching. It compares root `testClasses` (64 reached projects) with
`:clients:testClasses` (three reached projects) after the fixed production
source mutation. Four alternating offline pairs share the checkout, Gradle
home, daemon, dependency cache, and candidate-augmented native cache seed. The
runner compares `clients` main/test classes and resources byte for byte, rejects
any Gradle `Test` execution, and proves that a `gradle.properties` change
restores the full graph. Hot State, Runtime Tuning, Copy, remote caches, managed
runtime, Test Optimization, soak, and production claims remain disabled.

The checked result qualifies the transfer: native Gradle averages 4,609.5 ms
and BuildOpt 2,070 ms, saving 2,539.5 ms/55.09%. All four pairs are positive,
the interval is +1,625.5..+4,093 ms, and all 4,062 required outputs match. The
negative suite rejects hot-state activation, output drift, and a weakened
fallback.

### Qualified POC profile

Validate the short repository-owned activation that follows the performance
roadmap:

```bash
./dev/check-poc-qualified-profile
```

The check exercises candidate selection, native/full-graph fallback, strict
configuration decoding, disabled-mechanism rejection, pre-execution JSON plan
output and child-environment isolation. It does not rerun the performance
matrix or create a production policy.

Validate the captured installed-package replay separately:

```bash
./dev/check-poc-qualified-profile-adoption
```

The adoption contract binds the repository-owned fixtures, native package and
fixed OpenTelemetry/Kafka revisions. It checks candidate-only standard-`Jar`
replay, historical output digests, and global full-graph fallback. The captured
record intentionally contains no durations and does not replace either
repository's performance experiment.

Validate the preregistered Kafka packaging experiment before timing:

```bash
./dev/check-poc-kafka-packaging-v1
```

This checker binds Kafka 4.3.1, its complete generated graph, the installed
three-project client-Jar candidate, the root `assemble` control, exact output,
four alternating pairs and the unchanged value gate. Preregistration contains
no accepted timings.

After provisioning the fixed source archive, dependency cache and installed
package, execute the immutable comparison with:

```bash
./dev/run-poc-kafka-packaging-v1 \
  /absolute/result.json \
  /absolute/install/bin/buildopt \
  /absolute/kafka-source.tar.gz \
  /absolute/gradle-home-seed
```

The runner creates a private disposable checkout, uses separate persistent
Gradle homes for the two arms, performs unmeasured warm-ups, records exactly
four alternating offline pairs, verifies the client JAR after every arm, and
executes the global fallback after timing.

Validate the generic Gradle verification/archive graph extension and the
preregistered Spring verification experiment before timing:

```bash
./dev/check-poc-verification-distribution-graph-v1
```

The checker binds public Gradle `VerificationTask` and
`AbstractArchiveTask` contracts to complete Spring `checkstyleMain` and
`sourcesJar` graphs. It retains arbitrary tasks, any `Test`, and unattributed
scheduled work as fail-closed full-graph cases. Only the Checkstyle scope is
authorized for timing. Run the installed comparison with:

```bash
./dev/run-poc-verification-distribution-graph-v1 \
  /absolute/result.json \
  /absolute/install/bin/buildopt \
  /absolute/spring-framework-source.tar.gz \
  /absolute/gradle-home-seed
```

The runner uses separate homes, a shared immutable native-cache seed, one
unmeasured warm-up and four alternating offline pairs. It requires the exact
non-empty Spring MVC Checkstyle report after every arm and verifies global
full-graph fallback after timing.

The retained result is validated with:

```bash
./dev/check-poc-verification-distribution-graph-v1-result
```

Native Checkstyle averages 33,916 ms and BuildOpt 33,812.25 ms. The
103.75-ms/0.31% mean is positive in only 2/4 pairs and has a -5,158-ms lower
bound, so the result deliberately keeps verification on the full native graph.
Source distribution remains graph-complete capability evidence without a
timing claim.

Attribute that neutral verification result without adding a fifth performance
sample with:

```bash
./dev/check-poc-verification-overhead-attribution-v1
./dev/run-poc-verification-overhead-attribution-v1 \
  /absolute/attribution.json \
  /absolute/install/bin/buildopt \
  /absolute/spring-framework-source.tar.gz \
  /absolute/gradle-home-seed
```

The diagnostic uses one warm observation per arm plus Gradle operation traces
and BuildOpt phase timings. Trace durations cannot change the retained
performance decision. At most one generic correction is allowed, and only if
a named candidate-specific phase exposes at least 500 ms of recoverable
critical-path cost.

Validate the retained attribution with:

```bash
./dev/check-poc-verification-overhead-attribution-v1-result
```

The trace records 143 native and 51 candidate task operations. Although the 92
omitted rows total 4,249 ms, most are cache hits or no-op outcomes and their
durations overlap. BuildOpt's largest own phase is only 1.238233 ms, so no
generic correction is authorized and verification remains on native Gradle.

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

The POC generalization checks separate capability from value. Run
`./dev/check-poc-opportunity-analysis` to prove deterministic structural
candidate discovery and fail-closed uncertainty across distinct real graph
fixtures. Run `./dev/check-poc-general-build-value` to bind the three direct
whole-profile measurements, recalculate their results from SHA-bound evidence
and confirm that neither mechanism effects nor repository percentages are
aggregated.

The fresh structural-transfer protocol is validated with
`./dev/check-poc-structural-transfer-v1 --contract`. Its runner,
`./dev/run-poc-structural-transfer-v1 /absolute/path/to/new-result.json`,
packages and installs the current BuildOpt revision, downloads the exact
Micronaut source archive, performs excluded online preparation, then measures
the frozen offline native-versus-structural comparison. Once evidence exists,
`./dev/check-poc-structural-transfer-v1` independently recalculates its result.

The subsequent source-ownership correction is checked with
`./dev/run --toolchain go -- ./dev/check-poc-source-ownership`. It proves that
cyclic components retain their production conservative boundary, direct POC
attribution uses only original member roots, equal-specificity ownership falls
back, and malformed owned roots are rejected. The unchanged Micronaut protocol
may be replayed only after this checker passes and every direct owner is covered.

`./dev/check-poc-structural-profile` validates the generic qualification bridge.
It builds the real CLI, analyzes a checked graph, materializes the v4 profile
twice, proves byte-for-byte determinism, rejects evidence/source drift, and
executes the focused runtime-precondition fallback test. The qualifier contains
no repository-name branches and never writes or activates a profile itself.

The fresh installed adoption is frozen by
`./dev/check-poc-structural-profile-adoption-v1 --contract` and executed with
`./dev/run-poc-structural-profile-adoption-v1 /absolute/path/to/result.json`.
The runner materializes the profile before timing, then compares only
`buildopt poc` with optimized native Gradle over eight alternating pairs. The
independent checker recomputes the result, revalidates the source qualification
and requires both global-change and profile-drift full-graph fallbacks. With no
argument it validates the committed terminal evidence, which records 72.16%
mean savings, 8/8 positive pairs and exact required outputs.

`./dev/check-poc-generic-evaluation` validates the one-decision installed POC
surface. It exercises a complete generic opportunity, invalid evidence, no
profile write on fallback, repository-relative atomic output policy, and the
absence of automatic or production authorization. It does not run a timing
experiment or modify Test Optimization.

`./dev/check-poc-generic-measurement` builds the real CLI and creates a
two-revision external Git repository. It proves two independent source clones,
two Gradle homes, independently restored native-cache seeds, alternating
eight-pair order, exact required outputs, qualified evidence, full-graph
fallback, atomic profile evaluation and invalid-evidence rejection. The fixture
uses a deterministic fake Wrapper so Base CI validates orchestration rather
than publishing a new performance percentage.

`./dev/check-generic-profile-onboarding` creates an external two-project Git
repository with no BuildOpt JSON and drives the installed `buildopt profile
propose` command from one exact change and one required-output glob. It checks
the generated manifest, graph, binding, fallback and proposal byte for byte,
feeds them to `profile analyze` and no-evidence `profile evaluate`, and verifies
that a custom executable workflow retains native full graph without writing
candidate state. This gate measures onboarding correctness, not build-time
value.

`./dev/run-generic-profile-realworld /absolute/evidence/directory` packages the
current CLI, shallow-fetches the frozen Apache Groovy and Micronaut revisions,
applies their already qualified one-file changes, prepares dependencies outside
measurement and runs the accepted proposal pass offline. The bounded online
preflight is capped at 20 minutes per repository. The runner writes proposal,
analysis, manifest, graph, generated binding and digest evidence, then invokes
`./dev/check-generic-profile-realworld`. The checker can also validate the
committed bundle without network access and requires exact equality with both
retained structural plans. It deliberately makes no new timing claim.

`./dev/check-poc-apache-groovy-classes-v1` independently validates the first
fresh public-repository result produced by that generic pipeline. It recomputes
all eight Apache Groovy pairs and their deterministic paired interval, checks
the manifest/graph/change digests and 66 byte-identical class outputs, rebuilds
the current CLI, reproduces the exact qualified profile, and proves that
tampered evidence writes no profile. The checked result is 50.06% faster for
the fixed `groovy-json` classes scope; no other Apache Groovy scope is enabled.

`./dev/run-generic-profile-matrix /absolute/evidence/directory [specification] [repository-key]` packages the
current CLI and applies the same generic structural-only proposal, measurement,
and evaluation flow to the frozen Spring, OpenTelemetry, Kafka, Micronaut, and
Groovy revisions. Every cell keeps its repository-declared workflow and output
globs; OpenTelemetry supplies all 53 original entrypoints through repeated
`--entrypoint` arguments. Preparation is excluded, while each accepted cell
uses eight alternating isolated pairs, exact outputs, and full-graph fallback.

`./dev/check-generic-profile-matrix [evidence-directory] [specification]
[repository-key]` validates either a frozen terminal bundle or one atomic
subject without network access. The terminal v3 bundle qualifies structural
Build Impact for Kafka (**84.11%**), Micronaut (**41.74%**), and Groovy
(**73.85%**). Spring saves **17.94%** but retains native because one negative
pair fails the frozen 8-of-8 rule. The separately preregistered OpenTelemetry
v4 subject qualifies at **14.43%**, with 8/8 positive pairs, exact 125-file
outputs and scheduling-equivalent full-graph fallback. The checker rejects
partial timing, repository percentage averages, and attribution of retained
Jar/Edge results to Build Impact.

`./dev/check-statistical-qualification-v2` validates the frozen balanced
qualification contract before any new observation is captured. Run each of the
five subjects twice through `./dev/run-generic-profile-matrix`, using the
unchanged v4 capture contract and the same committed BuildOpt revision. Place
the two immutable capture directories under
`benchmarks/results/poc-statistical-qualification-v2/<key>/capture-{1,2}` and
assemble the matrix with:

```bash
./dev/assemble-statistical-qualification-v2 \
  "$PWD/benchmarks/results/poc-statistical-qualification-v2" \
  2026-08-12T12:00:00Z
```

`./dev/check-statistical-qualification-v2-result` independently recomputes
every completed aggregate and preserves unavailable captures. It requires two
different source documents with identical repository, revision, plan,
executable and options; 16 raw pairs become eight AB/BA blocks. The checker
does not average repository percentages or permit a failed observation to be
discarded.

`./dev/check-new-family-transfer --spec-only` validates the preregistered Ktor
JVM JAR transfer before the owner workflow or BuildOpt proposal runs.
After that contract is committed,
`./dev/run-new-family-transfer /absolute/repository/evidence/directory`
collects two fresh installed captures and aggregates them with the same
balanced 500-ms/2%, 6-of-8-block, positive-bound and non-regressive-p95 gate.
Both arms run Ktor's public unqualified `jvmJar` selector with the exact
`ktor-http` JAR output and optimized-native Gradle options.
`./dev/check-new-family-transfer [directory]` revalidates each source capture
and recomputes the aggregate offline. `./dev/test-new-family-transfer-result`
also proves that summary and aggregate tampering fail closed. Unsupported,
weak or unstable results retain native Gradle; the runner cannot add a
Ktor-specific product decision.

`./dev/check-new-family-change-breadth --spec-only` validates the frozen Ktor
follow-up before any new proposal or timing. Its three selective cells cover
an upstream dependency source, a JVM resource and a two-module mixed-source
change; a fourth root-configuration cell must return an untimed native
full-graph decision. `dev/run-generic-profile-matrix` accepts the new
`changes[]` form while preserving the previous `changePath` form, so this
tests multiple paths without adding Ktor logic to the product.

`./dev/run-new-family-change-breadth /absolute/repository/evidence/root
[cell-key] [capture-index]` records resumable cells through that generic
runner. `dev/assemble-new-family-change-breadth` creates the terminal matrix,
`dev/check-new-family-change-breadth-result` recomputes every completed
qualification from its captures, and `dev/test-new-family-change-breadth-result`
proves that summary or qualification tampering fails closed.

The terminal Ktor matrix qualifies dependency source at 85.80%, a JVM
resource at 86.51%, and a two-module mixed-source edit at 77.98% lower wall
time. All 48 raw pairs and 24 reciprocal blocks improve with exact outputs and
six fallbacks; two global proposals retain native Gradle without timing.

The accepted captures all propagate the complete preregistered Gradle option
list. An earlier diagnostic matrix that omitted the repository's Develocity
property is preserved under `incidents/` and contributes zero terminal pairs.

`./dev/check-new-family-calibration-economics --spec-only` validates the
preregistered Ktor economics study and all immutable terminal bindings before
fresh timing. The resumable runner is:

```bash
./dev/run-new-family-calibration-economics \
  "$(pwd)/benchmarks/results/poc-new-family-calibration-economics-v1" \
  /absolute/path/to/preregistered/buildopt [cell-key] [capture-index]
```

It times cold discovery and exact replay separately, rejects an option-drift
replay, then invokes `profile measure --calibration-only` for candidate cache
seed, base-daemon and bounded target stabilization. The assembler reports
installed and exact-replay break-even per cell against unchanged terminal
savings. The result checker recomputes the summary and the negative fixture
must reject summary or log tampering. The global-configuration cell is never
timed.

Before the original-workflow preflight, the runner prepares the exact Wrapper
distribution selected by that repository, reusing only a matching cached
distribution and allowing at most three bounded network attempts. It runs no
owner task. The successfully used distribution is then copied into each
private measurement Gradle home. Proposal, warm-ups, timed pairs and fallback
therefore perform no Gradle distribution download; caches and daemons remain
isolated between arms.

Proposal and measurement explicitly disable Configuration Cache. The temporary
output-contract preflight inspects repository-owned task outputs and is not a
configuration-cache-compatible workload; using the same explicit mode prevents
repository defaults from changing whether that generic preflight can run.

The terminal bundle contains two fresh captures per Ktor cell. Installed cold
discovery plus candidate stabilization repays after 7 builds for dependency
source, 10 for a JVM resource and 8 for the mixed-source edit. Exact proposal
replay itself takes 0.321–0.376 seconds; including fresh stabilization, replay
evaluation repays after 2, 4 and 3 builds. Revalidate it with:

```bash
./dev/check-new-family-calibration-economics
./dev/check-new-family-calibration-economics-result
./dev/test-new-family-calibration-economics-result
```

The checker binds every structured capture and recomputes `summary.json`; the
negative fixture rejects summary or structured-calibration tampering. Raw
process logs are not published; their hashes remain in the phase records. The
global-configuration cell remains untimed native full graph.

`./dev/run-generic-workflow-value /absolute/evidence/directory [workflow-key]`
uses the same installed generic runner with the frozen public workflow-family
contract. It measures Groovy JAR packaging, Kafka Checkstyle verification,
Kafka fat-JAR distribution, and Spring test-class preparation independently
against their unchanged optimized-native workflows. The companion
`./dev/check-generic-workflow-value [evidence-directory] [workflow-key]`
requires complete eight-pair evidence for measured subjects and explicit
fail-closed classification for subjects that cannot satisfy the output or task
evidence contract. The terminal public bundle retains native for all four
families: Spring records a positive **18.47% / 2.695 s** signal but only 7/8
positive pairs, while Groovy/Kafka stop on generated time, absolute workspace
paths, or archive timestamps. The checker also validates the root-cause record
and proves that failed observations were not discarded or normalized.

`./dev/check-generic-output-equivalence` validates the versioned semantic
contract before any public timing. Byte identity remains implicit; the only
reviewed exceptions are isolated-root relocation in bounded UTF-8 text,
canonical ZIP content, and exact volatile Java-properties keys inside exact
ZIP entries. The Go conformance tests change undeclared findings, payloads and
properties and require fail-closed rejection. The frozen public rerun uses two
fresh eight-pair captures for Groovy `jar`, Kafka `checkstyleMain`, and Kafka
`shadowJar`, then evaluates eight reciprocal blocks per workflow under the
current material-value, tail, shape, fallback, and zero-failure gates.
If Gradle emits a task line before its terminal outcome, the evidence retains
the ordered transitions as diagnostics, uses the last emission for counters,
and fingerprints terminal task outcomes only. No task is discarded.

```bash
./dev/run-generic-output-equivalence \
  "$(pwd)/benchmarks/results/poc-generic-output-equivalence-v1"
./dev/check-generic-output-equivalence-result
```

The terminal bundle qualifies all three subjects independently. Groovy `jar`
reduces 72,318.625 ms to 19,454.5 ms (**73.10%**), Kafka Checkstyle reduces
82,835.375 ms to 58,208.5 ms (**29.73%**), and Kafka `shadowJar` reduces
40,727.8125 ms to 13,624.9375 ms (**66.55%**). All 48 raw pairs improve,
semantic outputs match, p95 improves, final warm-up and measured task shapes
are stable, both full-graph fallbacks pass per subject, and product failures
remain zero. The complete collection incidents remain indexed beside the
terminal evidence and contribute no timing to the result.

`./dev/check-generic-change-breadth` validates the next preregistered matrix:
six selective leaf/shared-source cells and four untimed build-logic/global
fallback cells across the same Groovy/Kafka workflow families. The generic
proposal roots lifecycle candidates at the reviewed owners of the declared
outputs; graph discovery must still prove that those tasks cover the changed
source. Each selective cell receives two eight-pair captures and each fallback
receives two independent installed-path proposals.

```bash
./dev/run-generic-change-breadth \
  "$(pwd)/benchmarks/results/poc-generic-change-breadth-v1"
./dev/check-generic-change-breadth-result
```

Fallback cells emit no wall-time percentage. They pass only when both captures
validate the owner workflow and return
`NATIVE_FULL_GRAPH / GLOBAL_CHANGE_REQUIRES_FULL_GRAPH` without candidate
documents.

The terminal matrix qualifies all six selective cells independently. Groovy
`jar` saves 73.54% for a leaf edit and 65.80% for a shared-source edit; Kafka
Checkstyle saves 28.00% and 30.10% for two distinct source edits; Kafka
`shadowJar` saves 66.64% for a clients edit and 79.54% for a generator edit.
All 96 raw pairs and 48 reciprocal blocks improve, candidate p95 is lower,
semantic outputs and measured shapes match, 12 selective fallbacks pass, and
product failures remain zero. All eight build-logic/global captures retain the
native graph and emit no candidate artifacts. The terminal result is
[versioned beside the benchmark](../benchmarks/results/poc-generic-change-breadth-v1/README.md).

`./dev/check-calibration-economics` validates the preregistered follow-up that
separates first-run calibration work from the already-qualified steady-state
value. The runner writes two fresh phase captures for each of the six selective
change-breadth cells and keeps proposal/preflight artifacts beside their
SHA-256-bound phase record:

```bash
./dev/run-calibration-economics \
  "$(pwd)/benchmarks/results/poc-calibration-economics-v1"
./dev/check-calibration-economics-result
```

The summary reports three cell-specific break-even views. Checkout stays
visible but is excluded as shared native/BuildOpt work; public-release download
is not measured. The runner removes each temporary public checkout after its
capture so the full matrix remains bounded by one active subject plus the
installed exact BuildOpt revision.

The terminal bundle reports installed-workflow break-even of 10–11 builds for
Groovy JAR, 27–31 for Kafka Checkstyle, and 14–15 for Kafka `shadowJar`.
Complete comparative-POC break-even is 20–22, 49–55, and 27–28 builds. Validate
the versioned [phase artifacts and summary](../benchmarks/results/poc-calibration-economics-v1/README.md)
with the two commands above.

`./dev/check-calibration-efficiency` validates the preregistered follow-up that
fuses structural discovery, permits only exact digest-bound proposal replay,
and stops candidate stabilization after two matching exact task fingerprints.
Fresh captures are assembled and checked with:

```bash
./dev/run-calibration-efficiency \
  "$(pwd)/benchmarks/results/poc-calibration-efficiency-v1" \
  /absolute/path/to/the/contract-bound/buildopt
./dev/assemble-calibration-efficiency-result \
  "$(pwd)/benchmarks/results/poc-calibration-efficiency-v1" \
  2026-08-14T21:07:28Z
./dev/check-calibration-efficiency-result
./dev/test-calibration-efficiency-result
```

The terminal result lowers cold discovery by 8.01%–21.08% and improves
installed break-even from 10–31 to 9–26 builds across all six cells. Exact
replay takes 0.281–1.261 seconds, but repeat evaluation still includes two
candidate target warm-ups. The checker verifies hashes, byte-identical
artifacts, drift rejection, adaptive fingerprints, immutable terminal savings
and deterministic recomputation.

The installed-profile adoption replay is frozen by
`specs/poc-installed-profile-replay-v1.json` and
`specs/poc-installed-profile-replay-v1.md`. The capture runner downloads and
installs immutable public `v0.3.1`, reconstructs all six qualified
Groovy/Kafka changes in clean external checkouts, executes `buildopt poc`, and
then changes one digest-bound manifest byte to require native fallback:

```bash
./dev/run-installed-profile-replay /absolute/path/to/new-capture
```

The committed bundle is validated without network access or Gradle execution:

```bash
./dev/check-installed-profile-replay-result
./dev/test-installed-profile-replay-result
```

The Ktor public-package replay is frozen separately by
`specs/poc-new-family-installed-profile-replay-v1.json`. It reconstructs the
three terminal dependency-source, resource and mixed-source changes, installs
the pinned public release and proves both exact selection and invocation-option
drift fallback:

```bash
./dev/run-new-family-installed-profile-replay /absolute/path/to/new-capture
```

The committed result is checked without downloading the release or executing
Gradle:

```bash
./dev/check-new-family-installed-profile-replay-result
./dev/test-new-family-installed-profile-replay-result
```

This replay creates no new timing claim. It preserves each terminal Ktor
qualification and compares only contemporary candidate and native-fallback
outputs from the same reconstructed checkout. The committed public `v0.3.2`
bundle passes all three selections, option-drift fallbacks and exact output
comparisons.

The one-command onboarding contract is validated separately:

```bash
./dev/check-magic-onboarding-contract
./dev/test-magic-onboarding-contract
./dev/run --toolchain go -- go test -count=1 ./internal/launcher -run '^TestOptimize'
./dev/check-magic-auto-discovery-contract
./dev/test-magic-auto-discovery-contract
./dev/check-magic-auto-discovery
./dev/check-magic-calibration-contract
./dev/test-magic-calibration-contract
./dev/check-magic-profile-portfolio-contract
./dev/test-magic-profile-portfolio-contract
./dev/check-magic-auto-replay-contract
./dev/test-magic-auto-replay-contract
./dev/check-magic-ci-onboarding
./dev/check-magic-wow-report
./dev/check-magic-calibration
./dev/check-magic-end-to-end-value
./dev/check-magic-end-to-end-value-v2
./dev/check-magic-calibration-cost
```

The command contract fixes `buildopt optimize build`, private atomic
state/result files, exact checkpoint reuse, bounded calibration options,
human/JSON output, Gradle exit preservation and the POC authority boundary.
The automatic-discovery checker then exercises the real binary on packaging,
verification, distribution and test-preparation Gradle workflows, requiring
zero manual BuildOpt files and private generated evidence. Unsupported,
global-change and ambiguous-base fixtures must retain native Gradle. The
automatic-calibration checker then proves eight balanced pairs, exact outputs,
full fallback, break-even evaluation, private evidence, exact reuse without
remeasurement and an under-budget no-claim result. A qualified result is
materialized as a private v4 profile under a digest-bound family portfolio.
The same real checker proves exact profile reuse, one-entry idempotence,
fail-closed repair after profile tampering and no profile under an insufficient
pair budget. Unit and contract fixtures cover dependency-source, resource,
leaf-source and mixed-source families plus coexistence/replacement semantics.
An exact second invocation now validates eleven repository, revision,
executable, Wrapper, workflow, option, graph, output, evidence and profile
bindings before Gradle, selects the smaller qualified graph without another
flag or calibration, and records the decision in nanoseconds. The same checker
proves that a tampered profile runs native, repairs only from still-valid
evidence, and becomes selectable on the following exact invocation.

The v1 end-to-end value checker preserves the first automatic
public-repository matrix and its retained raw `result.json` files as
diagnostic history. It deliberately expects that historical capture to remain
incomplete rather than rewriting it after later improvements.
The later calibration-cost checker independently recomputes the retained Beam
preflight: authoritative read-only dependency binding, one immutable native
cache seed, measured task-shape stabilization, eight positive alternating
pairs, exact outputs, successful fallback, 109.402-second comparable cost
reduction and 26-build payback. It does not rewrite the earlier matrix or claim
that its different protocol has the same absolute learning cost.
The v2 terminal checker binds public `v0.6.1` and recomputes two fresh
install-to-decision results. Ktor reduces 133 to ten projects and measures
38.810 to 7.830 seconds (79.82%, 26-build payback); Beam reduces 316 to six and
measures 65.081 to 24.958 seconds (61.65%, 28-build payback). It checks all
eight pair durations, output hashes, task-shape stability, p95, fallback,
package/result/evidence digests and the zero-manual-file boundary. It also
requires the separate Ktor global-build-logic case to retain a successful
native build without calibration or a timing claim. Rejected `v0.6.0`
environment and Configuration Cache attempts remain preserved but contribute
no timing observation.
The CI onboarding checker then proves one command on GitHub and GitLab,
provider-bound path-independent exact checkpoint reuse, cross-repository and
revision rejection, exact argv, private checksummed review artifacts and child
failure preservation. Provider cache contents remain untrusted and no service
or hand-authored BuildOpt file is required.
Every completed optimize invocation also emits private `value-report.md` and
`value-report.json` files. The value-report checker recomputes project
reduction, installed-path mean saving, paired uncertainty, per-arm p95,
calibration break-even, exact-replay projection, selection overhead and native
fallback. Individual reports still keep expected useful lifetime unavailable
unless they contain cross-commit evidence. The separate Ktor lifetime result
now demonstrates one profile-specific observed window; it is not a universal
default for other profiles.

`specs/poc-cross-date-output-equivalence-v1.json` and
`specs/poc-cross-date-output-equivalence-v1.md` freeze the follow-up for the
remaining Groovy date boundary. The reviewed fixture adds exactly `BuildDate`
beside `BuildTime`. The runner builds the two frozen Groovy cells, applies a
controlled date-only mutation to each real JAR, and requires the old contract
to reject it while the reviewed contract matches. It also requires an
undeclared property change and a class-payload change to remain visible. Four
natural Kafka cross-capture matches are carried forward from the digest-bound
public replay; all six historical timing qualifications remain unchanged.

```bash
./dev/run-cross-date-output-equivalence /absolute/path/to/new-capture
./dev/check-cross-date-output-equivalence-result /absolute/path/to/new-capture
./dev/test-cross-date-output-equivalence-result
```

The controlled date mutation is reproducible semantic-equivalence evidence,
not a second timed build on another date. This block adds no timing claim,
automatic activation, production authority, soak, design-partner dependency,
or Test Optimization scope.

The checker rebuilds the aggregate from cell plans, outputs, log hashes and
the prior immutable qualification. It requires 6/6 selective plans, 6/6
`PROFILE_PRECONDITION_FAILED` fallbacks and 6/6 same-replay semantic-output
matches. Historical terminal digests are diagnostic because the Groovy JAR
qualification used the earlier contract that excluded `BuildTime` but not its
date-dependent `BuildDate`; no new timing is measured or inferred.

`./dev/check-generic-profile-ci-replay` validates the manual clean-runner
replay without cloning public repositories. It reconstructs retained Action
artifacts for all five subjects, requires five semantic and byte-level graph
matches, changes one graph deliberately to prove explicit `DRIFT`, and proves
that a four-of-five aggregate is incomplete. The hosted
`.github/workflows/profile-proposal-replay.yml` uses
`./dev/prepare-generic-profile-ci-replay` to create the exact external
checkouts, `./dev/evaluate-generic-profile-ci-replay` to write each durable
verdict, and `./dev/compose-generic-profile-ci-replay` to publish the terminal
summary. This replay performs no timing and writes no qualified profile.

`./dev/check-generic-output-contract` exercises the generic Gradle output
preflight with a synthetic repository whose build directory is redirected from
`build` to `target`. It proves discovery without declarations, successful
confirmation, early rejection of an empty conventional glob and native
fallback for cross-project ownership ambiguity. The same preflight is embedded
in `profile propose`; failed output contracts cannot write graph documents or
enter measurement.

`./dev/check-generic-output-contract-evidence` validates the frozen Hibernate
observation against `specs/poc-generic-output-contract-v1.json`. The original
`hibernate-core/build/libs/**` declaration is empty, Gradle exposes owned JAR
candidates under `hibernate-core/target/libs`, and proposal stops at
`NATIVE_FULL_GRAPH` with no warm-up, timing or profile.

`./dev/check-generic-owner-input` converts a validated output contract into
the shared `.buildopt/profile.json` only after explicit confirmation. It proves
deterministic validation, automatic base-to-HEAD Git change derivation,
owner-input digest binding, and a real Gradle output move from `target` to
`dist`. The drift returns `NATIVE_FULL_GRAPH / REQUIRED_OUTPUTS_EMPTY`, reports
the new owned JAR candidate and writes no candidate graph. The same schema is
exercised by `./dev/check-generic-profile-ci`; the frozen five-repository
replay continues to validate the former CI-only schema for compatibility.

`./dev/check-generic-workflow-breadth [result.json]` exercises that unchanged
owner-input path across packaging, typed verification, distribution, and
build-owned test preparation. It executes each original workflow, derives the
exact changed-project candidate, rebuilds the required output from a clean
state and compares bytes while rejecting any Gradle `Test` execution. The
unsupported executable fixture must stop at
`NATIVE_FULL_GRAPH / ORIGINAL_WORKFLOW_UNSUPPORTED` before structural state or
timing. `./dev/check-generic-workflow-breadth-result result.json` validates the
specification, runner, fixture and terminal decisions without rerunning Gradle.

`./dev/run-generic-holdout /absolute/evidence/directory` packages the current
committed CLI and applies the preregistered generic proposal, isolated
measurement and evaluation path to the frozen Hibernate ORM holdout. The
runner records the proposal even when it retains native Gradle and measures
only a complete structural candidate. `./dev/check-generic-holdout
[evidence-directory]` validates the exact source, workflow, change, outputs,
control, proposal, eight-pair evidence, evaluation and fallback without
network access; use `--spec-only` before evidence capture. The retained v1
attempt demonstrates fail-closed handling of an invalid owner-declared output.
The retained v2 correction points to Hibernate's repository-defined
`target/libs` directory and changes no measured condition or value gate. Its
terminal bundle completes all eight pairs at 7.80% mean savings and 7/8
positive pairs, validates exact outputs plus full fallback, and retains native
Gradle under the unchanged repeatability rule. The default runner now follows
the preregistered v3 diagnostic correction: one cache-seed and one daemon-
stabilization warm-up per private arm, followed by the same eight pairs while
recording task-outcome summaries and log digests. Validate that preregistration
before capture with `./dev/check-generic-holdout --spec-only
specs/poc-generic-holdout-v3.json`; no v2 observation is reused.

`./dev/run-generic-holdout-crossover /absolute/evidence/directory` executes
the separately preregistered v5 recovery experiment. It captures two complete
fresh batches, each with two exact-target stability observations and eight
alternating raw pairs, then combines only adjacent opposite-order pairs into
eight reciprocal blocks. `./dev/check-generic-holdout-crossover [directory]`
recomputes the aggregate from both checked batches and validates exact task
paths, outputs, fallbacks and the unchanged value gate. Use `--spec-only`
before capture. Version 4 timings are never inputs to the v5 result.

The retained terminal bundle is
`benchmarks/results/poc-generic-holdout-v5`. It qualifies at 12.733 seconds/
5.88% mean savings with interval +6.808..+19.859 seconds and 8/8 positive
reciprocal blocks. Validate it without cloning Hibernate:

```bash
./dev/check-generic-holdout-crossover \
  benchmarks/results/poc-generic-holdout-v5
```

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
