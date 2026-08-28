# Validation reference

BuildOpt uses focused checks as executable evidence. Run the smallest gate that
owns the changed contract, then add broader composition only when the change
crosses components or platforms.

Every `dev/check-*` command is non-interactive and returns non-zero on failure.
Most preserve the source tree and use private temporary state. Commands that
provision tools or create release artifacts document that effect explicitly.

## Fast repository checks

| Change | Command |
|---|---|
| Markdown, navigation, package docs | `./dev/check-documentation` |
| Required paths and baseline shape | `./dev/check-layout` |
| Normative package structure | `./dev/check-normative-layout` |
| Closed fresh generic predecessor and evidence boundary | `./dev/check-fresh-generic-optimization-plan` |
| Change-aware producer/analyzer fixtures across Gradle and DSL variants | `./dev/check-change-aware-producer-fixtures` |
| Fresh five-family change-aware transition capture | `./dev/check-change-aware-public-capture` |
| Independent change-aware report reconstruction and fixed 3/5 breadth gate | `./dev/check-change-aware-breadth-gate` |
| Change-aware terminal scorecard and typed unmeasured economics | `./dev/check-change-aware-terminal-decision` |
| Request-aligned cause analysis, selected hypothesis and frozen route | `./dev/check-request-aligned-successor-selection` |
| Request-aligned identity and current producer-output matrix | `./dev/check-request-aligned-producer` |
| Request-aligned relevance classifier and Gradle/DSL matrix | `./dev/check-request-aligned-classifier` |
| Generic task/graph producers, typed completeness and deterministic evidence | `./dev/check-sticky-evidence-producers` |
| Fresh five-family cohort, capture bindings, producer completeness and exact outputs | `./dev/check-fresh-generic-capture` |
| Independent fresh action recount and fixed public-breadth gate | `./dev/check-fresh-generic-opportunity-gate` |
| Fresh route terminal scorecard and typed unmeasured outcomes | `./dev/check-fresh-generic-terminal-decision` |
| Superseded sticky-wrapper diagnostic contract | `./dev/check-sticky-wrapper-learning-plan` |
| Sticky-wrapper files, parsers, routing and update contract | `./dev/check-sticky-wrapper-contract` |
| Sticky-wrapper deterministic generator, drift, downgrade, rollback and portable compilation | `./dev/check-sticky-wrapper-generator` |
| Sticky-wrapper checksum bootstrap, safe extraction, atomic cache publication and offline reuse | `./dev/check-sticky-wrapper-bootstrap` |
| Sticky-wrapper portable connection, capability probes, revocation and secret isolation | `./dev/check-sticky-wrapper-connection` |
| Sticky-wrapper Gradle HTTP cache reuse, read-only policy, corruption and outage fallback | `./dev/check-sticky-wrapper-cache` |
| Sticky-wrapper native no-op path, lazy light observation and startup overhead | `./dev/check-sticky-wrapper-noop-overhead` |
| Sticky-wrapper active execution, counterfactuals, suspension and native fallback | `./dev/check-sticky-wrapper-active` |
| CODEOWNERS/workstream mapping | `./dev/check-ownership` |
| Shell and workflow syntax/inventory | `./dev/check-lint-toolchains` |
| Base workflow and immutable pins only | `./dev/check-base-ci --static` |
| Whitespace in the current diff | `git diff --check` |

Run these before a documentation, layout, ownership, or workflow-only commit.

## Toolchains and build foundations

```bash
./dev/check-toolchains-lock
./dev/doctor
./dev/test-doctor
./dev/test-jdk-toolchain
./dev/test-go-toolchain
./dev/test-protobuf-toolchains
./dev/test-lint-toolchains
./dev/test-supply-chain-toolchains
./dev/test-toolchain-lifecycle
./dev/check-golden-lane --static
./dev/run -- ./dev/check-golden-lane --smoke
```

Use `dev/run` when the command must consume a repository-owned toolchain.
`dev/doctor` inspects the host; `buildopt doctor` reports installed runtime
capabilities. They answer different questions.

## Go and launcher paths

Focused language checks:

```bash
./dev/run --toolchain go -- go test -count=1 ./internal/<package> ./cmd/<binary>
./dev/run --toolchain go -- go vet ./internal/<package> ./cmd/<binary>
```

Composition checks:

```bash
./dev/check-buildopt-cli
./dev/check-local-gateway
./dev/check-managed-gateway
./dev/check-managed-l1
./dev/check-session-ingest
./dev/check-build-session-export
./dev/check-walking-skeleton-faults
./dev/check-walking-skeleton-overhead
./dev/check-gradle-plugin-handshake
./dev/run -- ./dev/check-gradle-correlation-fixture
```

Use the race-enabled gateway/session gates after lifecycle, concurrency,
signals, spool, authentication, or process cleanup changes.

## Contracts and generated code

Adaptive-fragment identity and selective invalidation:

```bash
./dev/check-adaptive-fragment-contract
./dev/check-adaptive-fragment-state
./dev/check-adaptive-fragment-index
./dev/check-adaptive-fragment-shadow
./dev/check-adaptive-fragment-economics
./dev/check-adaptive-fragment-online
./dev/check-adaptive-fragment-prior
./dev/check-adaptive-fragment-patch-opportunity
./dev/check-adaptive-fragment-planner
./dev/check-adaptive-fragment-composition
```

The static and synthetic `AF-001` gate validates the machine policy, runs the
focused Go identity/compatibility/lifecycle tests and rejects repository-
specific behavior in the fragment package. It makes no timing or activation
claim. The `AF-002` gate compiles four Draft 2020-12 schemas, validates two
linked lifecycle bundles and rejects seven schema, semantic and canonical-digest
mutations. It does not implement persistence or synchronization. The bounded
`AF-003` gate validates the frozen and live 30-decision/five-repository lookup
reports, recalculates latency summaries and rejects tampering. It measures
pre-Gradle decision overhead only, not build-wall-time value.
The `AF-004` gate recomputes the five-repository frozen-history decomposition,
reproduces the original whole-profile selections, reports partial compatibility
separately and rejects any lookahead or report tampering. It remains shadow
evidence and neither executes nor times a fragment.
The `AF-005` gate recomputes immutable signed economics from retained Kafka
composition evidence and synthetic edge vectors. It proves that negative builds
reduce value, asynchronous costs are charged once, recurrence is represented by
exact counts, projections cannot rewrite observations and percentages are never
added. It does not activate or time a fragment.
The `AF-006` gate exercises canonical online checkpoints using only synthetic
requested-build inputs. It rejects measurement-only, binding/cohort drift,
inexact outputs and product failures; exact restart succeeds, insufficient
evidence stays observed/shadow and regression suspends only the affected family
and its transitive dependents. It runs no Gradle build and makes no timing claim.
The `AF-007` gate proves name-independent exploration ordering without
transferring local authority. The `AF-008` gate adds a generic repeated-task
detector, verifies that its proposal remains review-only, exercises temporary
application and exact revert, and binds the accepted exact recipe to 16 frozen
native Gradle pairs with identical outputs. It revalidates existing timing
evidence rather than producing another sample.
The `AF-009` gate recomputes a pure pre-Gradle planner proof. It keeps exact
constituent authorities, requires dependency closure, rejects mutual exclusion,
uses only whole-composition predictions above a fixed net-value floor and
returns native Gradle for seven missing, ambiguous or unsafe vectors. Its economic values
are synthetic and make no build-time or activation claim.
The `AF-010` gate runs six real Gradle 9.6.1 control/candidate scenarios. It
requires exact subgraph/materialization pairs, proves producer-local
invalidation and verified restoration, checks executed producer tasks and
retains the complete native workflow on global, ambiguous or incomplete state.
It validates correctness and activation only; it records no wall times.
The `AF-011` gate validates the immutable 48-pair direct-composition report
and fresh Shared/Edge locality evidence. It accepts either preregistered
outcome; the checked result is `RETAIN_BEST_SINGLE_FRAGMENT` because Kotlin
Build Impact reaches 6/8 positive pairs even though both composed arms are
faster and exact.

```bash
./dev/check-adaptive-fragment-activation
./dev/check-adaptive-fragment-composition
```

```bash
./dev/check-generated-code
./dev/check-generated-clients
./dev/check-build-session-schema
./dev/check-experiment-action-schemas
./dev/check-metrics-catalog
./dev/check-protobuf-toolchains
./dev/check-task-events-proto
./dev/check-contract-crypto
./dev/check-http-semantics
./dev/check-state-machines
./dev/check-ci-orchestration
./dev/check-commit-atomicity
./dev/check-capability-matrix
./dev/check-data-lifecycle
```

When a normative source changes, validate every affected producer and consumer
in addition to the schema/vector checker. Never edit generated files or
descriptors to satisfy drift checks.

## Cache and storage

```bash
./dev/check-tier1-fixtures
./dev/check-tier-one-policy
./dev/check-tier-one-cache-conformance
./dev/check-l1-l2-revocation
./dev/check-gateway-rotation
./dev/check-gateway-spool
./dev/check-shared-commit-recovery
./dev/check-no-hit-overhead
./dev/check-test-cache-isolation
./dev/check-shared-storage
./dev/check-pending-commit
./dev/check-local-authority
./dev/check-gradle-bootstrap-cache
./dev/check-managed-l1
```

Storage changes should include the owning package tests, corruption/recovery
gate, and platform compatibility if filesystem or file-lifecycle code changed.

## Historical Runtime Tuning evidence

```bash
./dev/check-causal-pilot
./dev/check-runtime-owner-evaluation
./dev/check-retired-poc-mechanisms
./dev/check-no-hit-overhead
./dev/check-walking-skeleton-overhead
```

The runtime checker validates immutable historical evidence only. The
retirement checker proves the rejected mechanisms have no active CLI,
launcher, plugin, workflow, or runner surface. Do not alter reference data or
thresholds to rehabilitate a failed mechanism.

## Build Optimization performance scorecard

```bash
./dev/check-poc-value-validation
./dev/check-poc-value-negative-mechanisms
./dev/check-poc-value-coverage
./dev/check-poc-value-combined
./dev/check-poc-breadth
./dev/check-poc-overhead
./dev/check-poc-stability
./dev/check-poc-real-world-compatibility
./dev/check-poc-real-world-performance
./dev/check-build-optimization-performance
./dev/check-poc-qualified-profile
./dev/check-poc-qualified-profile-matrix-v1-result \
  benchmarks/results/poc-qualified-profile-matrix-v1/summary.json
./dev/check-automatic-breadth-transfer
./dev/check-automatic-breadth-transfer-v2 \
  benchmarks/results/poc-automatic-breadth-transfer-v2/summary.json
```

The commands validate the final POC decision, strict no-value/no-action
evidence, accelerator breadth, and the combined public path. The last preserves
the earlier mechanism-development scorecard and validates the clean
OpenTelemetry composition plus the unchanged Apache Kafka transfer as the
current substantial public paths. None reruns Gradle or adds percentages from
different workloads.
Use the owning benchmark runner only when the relevant implementation or
fixture changes.

`./dev/check-source-ownership-compatibility` validates the latest bounded
ownership result: a complete public Gradle model, typed task-consumer
attribution, explicit native retention for unproven configuration inputs and
3,890 exact Groovy outputs. Its paired qualification rejection is retained as
negative evidence and is not rerun until favourable.

`./dev/check-configuration-input-binding` validates the independent Gradle
8/9 and Groovy/Kotlin result. Add `--fixture` to execute the four real
Configuration Cache cases; the default Base CI check validates the immutable
contract and result without recompiling Kotlin DSL from a cold home.

`./dev/check-aggregate-output-closure` validates the following generic output
coverage step. Its immutable four-case result requires exact custom aggregate
outputs, one changed producer entrypoint, verified stable-output
materialization, no stable-producer execution and zero product failures. The
owning runner is used only when the implementation or fixture changes; normal
Base CI does not repeat the Gradle matrix.

`./dev/check-structural-profile-rebinding` validates the canonical
cross-revision profile identity, its five drift dimensions, four incomplete
evidence failures and the real central selection/refresh path. It makes no
timing claim and never treats a compatible structure as authority to reuse
stale output bytes.

`./dev/check-ordinary-learning-economics` validates that only requested
ordinary builds supply duration, graph, portability, volatility and outcome
evidence. It enforces a five-match payback horizon, rejects four unsafe evidence
classes and preserves the separate eight-pair robust qualification gate. Its
synthetic durations test decisions and do not claim repository performance.

`./dev/check-lifetime-breadth-v3 check` recomputes the terminal five-repository
ordinary-build experiment from every subject result, qualification capture and
the Kafka calibration evidence. It verifies one executable SHA, requested-build
accounting, exact outputs, zero product failures, descendant eligibility,
selection, cumulative economics and the frozen 3/5 repository plus 50%
selection coverage gates:

```bash
./dev/check-lifetime-breadth-v3 check \
  ./specs/poc-lifetime-breadth-v3.json \
  ./benchmarks/results/poc-lifetime-breadth-v3
```

Apply the frozen terminal POC gate and verify its retained decision with:

```bash
./dev/check-functional-coverage-decision
```

This checker validates the V3 source evidence, regenerates the eight decision
criteria, compares the terminal JSON byte for byte and rejects false-continue
or threshold-drift fixtures.

`check-automatic-breadth-transfer` validates the immutable V1 unchanged
zero-manual-file run across Spring Framework, OpenTelemetry Java
Instrumentation, Apache Kafka, Micronaut Core and Apache Groovy. It recomputes
the raw/state digests, alternating pairs, output identity, fallback, graph
reduction and synchronous learning payback.

`check-automatic-breadth-transfer-v2` validates the current terminal result
after incremental learning, verified output materialization and aggregate
partitioning are composed. It checks 85 ordinary invocations, exact executable
and repository bindings, state trees, all output manifests and recomputed
economics. OpenTelemetry, Kafka, Micronaut and Groovy qualify; Spring improves
but retains native under the unchanged gates.

`check-incremental-learning` validates the successor transaction: one baseline
plus eight exact-bound control/candidate pairs accumulated across 17 useful
invocations, with stable required outputs, full fallback and zero
measurement-only workflows. The checked fixture retains native Gradle because
its weak timing does not pass the unchanged gates.

`check-poc-real-world-compatibility` binds Spotless, Mockito, and SpotBugs to
exact released commits plus wrapper, settings, and distribution hashes. The
paired performance gate is now complete: Mockito qualified, while Spotless and
SpotBugs did not clear the unchanged thresholds. The checked decision retains
the bounded synthetic claim; it does not authorize a general public-repository
claim or another unchanged run.

The later substantial-repository path qualified Spring Framework test
preparation at 28.72% over eight pairs, then transferred the unchanged
mechanism to OpenTelemetry. That transfer stopped without qualification after
the optimized native control failed in pair 7. Validate the terminal record
with `./dev/check-poc-otel-test-preparation-result`; six completed positive
pairs remain diagnostic-only and are not a substitute for the preregistered
eight-pair gate.

The later installed-path iteration is separate from that unchanged transfer.
After task attribution, typed graph reduction and exact-bound hot state, the
explicit standard-`Jar` adapter passed the new four-pair OpenTelemetry gate at
39.92% mean saving with 4/4 positive pairs and exact outputs. The subsequent
clean composition removes the regressive hot-state mechanism and saves 50.40%
or 5,361.25 ms with 4/4 positive pairs, the same 125 outputs, and safe global
fallback. Validate the historical adapter evidence with
`./dev/check-poc-otel-optimization-v2-result` and the clean evidence with
`./dev/check-poc-otel-clean-composition-v1-result`; these results qualify only
their POC workload and do not widen production cache policy.

The unchanged clean profile later qualified on Apache Kafka 4.3.1, a distinct
64-project Java/Scala/generated-source workload. Native root `testClasses`
averaged 4,609.5 ms and installed BuildOpt 2,070 ms, saving 55.09% with 4/4
positive pairs, 4,062 exact outputs and full fallback. Validate it with
`./dev/check-poc-third-repository-transfer-v1-result` and its negative fixtures
with `./dev/test-poc-third-repository-transfer-v1`. The result remains
output-scoped POC transfer evidence.

`check-poc-qualified-profile` validates the usability layer added after that
transfer: strict repository config, candidate-only standard-`Jar` activation,
the machine-readable plan emitted before Gradle, rejected mechanism expansion,
and native/full-graph fallback. It does not rerun or reinterpret any timing.
The matrix result checker then recomputes the three independent installed cells
and terminal specialization decision: Spring retains native at 7/8 positive
pairs, OpenTelemetry retains native with zero accepted observations, and only
Kafka qualifies at 81.85% with 8/8 positive pairs and complete safety proofs.

`check-poc-breadth` accepts both a passing and a failing decision document and
recomputes all 64 observations. The post-attribution repeat retains the narrow
claim: 4/8 realistic change/DSL cells qualify, despite correct selection and
byte-identical outputs. `check-poc-overhead` independently validates the
non-overlapping diagnostic phases and confirms that traced timings never gate
performance. `check-poc-stability` recomputes two strict batches whose control
and candidate arms have independent writable state, private persistent daemon
lifecycles, and opposite execution order. It validates either a stable result
or an explicit negative decision without weakening the breadth thresholds. The
checked result is `MEASUREMENT_UNSTABLE`: 0/8 cells qualified control-first,
4/8 qualified candidate-first, and four classifications changed with global
order.

## Task Intelligence

```bash
./dev/check-jvm-agent-spike
./dev/check-hermetic-helper-spike
./dev/check-task-intelligence-poc
```

Agent/helper coverage remains bounded and may be `UNAVAILABLE`; the safe
fallback must pass along with the positive evaluation path.

## Patch Autopilot

```bash
./dev/check-patch-bundle-spec
./dev/check-patch-bundle-applier
./dev/check-archive-reproducibility-recipe
./dev/check-patch-candidate-validation
./dev/check-full-relevant-validation
./dev/check-customer-patch-workflow
./dev/check-patch-delivery-recovery
./dev/check-post-merge-patch-monitor
./dev/check-patch-autopilot-recipes
./dev/check-patch-autopilot-validation-revert
```

Recipe changes require exact positive, rejection, idempotency, rollback, and
revert coverage. No checker may mutate a real remote repository or weaken the
draft-only boundary.

## Build Impact

```bash
./dev/check-build-impact-manifest
./dev/check-build-impact-declared-graph
./dev/check-build-impact-shadow-validation
./dev/check-build-impact-promotion-gate
./dev/check-build-impact-selection
./dev/check-build-impact-gate
./dev/check-build-impact-automatic
./dev/check-build-impact-performance
```

Run the automatic check after CLI/discovery changes and the gate after any
selection, fallback, ownership, or Test Optimization boundary change.

## One-command POC

```bash
./dev/check-magic-wow-report
./dev/check-magic-ci-onboarding
./dev/check-magic-calibration
```

The focused value-report check is fast and recomputes the human/JSON contract.
The CI fixture proves private checksummed publication on both providers. The
calibration fixture is the slower end-to-end proof: eight real balanced pairs,
full fallback, exact replay, tamper recovery, cumulative economics and the
under-budget no-claim path. Its intentional delay validates the protocol; it
is not public-repository performance evidence.

## CI, releases, and deployment

```bash
./dev/check-github-action
./dev/check-gitlab-ci
./dev/check-ci-orchestration
./dev/check-release-package
./dev/check-deployment-lifecycle
./dev/check-self-hosted-service-install
./dev/check-self-hosted-upgrade-restart
./dev/check-self-hosted-manual-restore
./dev/check-self-hosted-single-node-gate
./dev/check-platform-compatibility
```

Native lifecycle proof requires the hosted macOS/Windows matrix; a successful
cross-build on Linux proves file format and compilation, not native service,
filesystem, or cancellation behavior.

## Edge and operations

```bash
./dev/check-edge-cache-config
./dev/check-edge-cache-committed-read
./dev/check-edge-cache-capacity-slru
./dev/check-edge-cache-pending-replication
./dev/check-edge-cache-two-node-proxy
./dev/check-edge-cache-gate
./dev/check-edge-operability
./dev/check-edge-service
./dev/check-ops-readiness
./dev/check-ops-alerts
./dev/check-base-runbooks
./dev/check-private-beta-operations
```

## POC and benchmark evidence

```bash
./dev/check-materialization-economics-v2 \
  benchmarks/results/poc-materialization-economics-v2/summary.json
./dev/check-poc-value-validation
./dev/check-owner-poc-lab
./dev/check-beta-benchmark-harness
./dev/check-beta-disk-faults
./dev/check-beta-shared-faults
./dev/check-beta-system-faults
./dev/check-beta-sustained
./dev/check-beta-circuit-breaker
./dev/check-beta-gradle-fixtures
```

The historical eight-hour soak harness is outside the active POC. It is not a
quickstart, owner-lab, CI, or value-gate requirement; reconsider it only after
the POC proves enough net value to justify productization.

## Adaptive fragment longitudinal evidence

```bash
./dev/check-adaptive-fragment-longitudinal
```

This is a static-plus-Go recomputation gate for historical `AF-013` evidence.
It reads the frozen Spring, OpenTelemetry, Kafka, Micronaut and Groovy source
evidence, verifies source hashes and exact signed observations, regenerates the
canonical report and rejects result or threshold tampering. It does not rerun
Gradle or make a fresh performance claim, and it is not the current adaptive
implementation scorecard. The AF-014A..D current campaign will have separate
contracts, results and checkers.

## Current installed longitudinal harness

```bash
./dev/check-current-longitudinal-harness \
  "$PWD/benchmarks/results/current-longitudinal-harness-v1.json"
```

This `AF-014A` gate validates the installed package, source archive and
executable digests; separate control/candidate state; 18 alternating timed
learning observations; exact selected, forward native-retained and bypass
outputs; and reconciliation of external wall time with non-overlapping
pre-execution, Gradle, finalization and unattributed phases. It also rejects
contract or timing tampering. The controlled omitted workload exists only to
exercise every state transition reliably, so this gate makes no repository
performance claim. `AF-014B` owns cohort freezing before public timing.

## Frozen current longitudinal cohorts

```bash
./dev/check-current-longitudinal-cohorts \
  "$PWD/benchmarks/results/current-longitudinal-cohorts-v1.json"
```

This `AF-014B` gate validates five contiguous first-parent chains containing
100 primary commits and 50 ordered reserves. It recomputes changed-path digests
and generic change shapes, binds JDK/workflow/output scope, and rejects reorder,
scope drift, unknown fields or timing before `AF-014C`. It performs no build.

## Current installed longitudinal campaign

```bash
./dev/check-current-longitudinal-campaign \
  "$PWD/benchmarks/results/current-longitudinal-raw-v1.json" \
  "$PWD/benchmarks/results/current-longitudinal-report-v1.json"
```

The `AF-014C` checker recomputes signed and cumulative net value from raw
control/candidate pairs, binds every attempted revision to the frozen primary
and reserve order, requires exact output bytes and verifies that candidate
state `N` consumes exactly the state produced by accepted observation `N-1`.
It also verifies that unmeasured dependency preparation is reported for the
anchor and each attempted revision and is limited to modules, verification
keyrings and Wrapper distributions shared symmetrically between the arms.
It also rejects any attempt to relabel the installed whole-profile runtime as
adaptive-fragment activation.

The committed campaign closes at 100 comparable exact-output pairs with zero
product failures. It records 25 positive and 75 negative pairs, 100 native
retentions, zero activations and -368.623 seconds cumulative signed value.
One Groovy primary dependency/native failure remains an explicit exclusion and
the next frozen reserve supplies the twentieth comparable pair.

## Current longitudinal attribution

```bash
./dev/check-current-longitudinal-attribution
```

The `AF-014D` gate validates the attribution contract, revalidates the
`AF-014C` raw/report pair, recomputes every repository, change-shape, workflow,
decision-reason and mechanism row, and rejects edited output. Its equations
separate recorded candidate-side BuildOpt cost from residual Gradle/runner
variation. A positive pair cannot create attributable saving when the raw
record shows no selected profile or activated fragment.

The committed outcome is `CURRENT_VALUE_NOT_ATTRIBUTABLE`: 100 native
retentions record 179.029 seconds of BuildOpt path cost, the residual is
-189.593 seconds, and activated-mechanism saving is zero. This is a static Go
recomputation over current public-repository evidence; it does not rerun Gradle
or create a production claim.

## Adaptive-fragment terminal decision

```bash
./dev/check-adaptive-fragment-terminal-decision
```

The `AF-015` gate first revalidates AF-014C/D, binds raw, report, attribution,
campaign protocol and terminal contract by SHA-256, and recomputes all 15
frozen criteria. It retains all repository rows, exclusions, negative builds,
paired-bootstrap lower bounds and cumulative checkpoints at 1/5/10/15/20.
Edited output and post-result threshold movement fail closed.

The committed decision is `STOP_ADAPTIVE_FRAGMENT_POC`: 9 criteria pass and 6
fail. Activation is 0/71 eligible builds, positive breadth and confidence are
0/5, portfolio value is -368.623 seconds, no family repays and native-retention
cost is 0.531 seconds p50 / 8.656 seconds p95. AF-013 is context only and no
Gradle build is rerun by this static decision gate.

## Complete lanes

- `./dev/check-phase-zero` composes the historical Phase 0 gates.
- `./dev/check-base-ci --lane core` reproduces the broad Go/Java base lane and
  requires the exact Java 17 compatibility runtime plus provisioned tools.
- `./dev/check-base-ci --lane rust` validates the optional Rust helper.
- `./dev/check-platform-compatibility` is the local portability gate; hosted
  `.github/workflows/platform-ci.yml` supplies native execution.
- `./dev/check-owner-poc-lab` validates the source-preserving one-command POC
  result.

Do not run every gate by default. Broad lanes cost more and make diagnosis
slower. Use them when a shared contract, build foundation, package, release, or
cross-platform boundary changed.

## Discover maintained checks

The source tree is authoritative. List all current check entrypoints with:

```bash
find dev -maxdepth 1 -type f -name 'check-*' -printf '%f\n' | sort
```

If a new executable gate has user or contributor value, add it to the closest
section here and link its owning specification.
