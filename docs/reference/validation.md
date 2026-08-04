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

## Runtime Optimizer and evidence

```bash
./dev/check-causal-pilot
./dev/check-runtime-resource-profiles
./dev/check-runtime-rollout-control
./dev/check-runtime-validation-isolation
./dev/check-runtime-owner-evaluation
./dev/check-no-hit-overhead
./dev/check-walking-skeleton-overhead
```

Use immutable paired inputs and preserve `INCONCLUSIVE` outcomes. Do not alter
reference data, thresholds, or held-out results merely to obtain promotion.

## Build Optimization performance scorecard

```bash
./dev/check-poc-value-validation
./dev/check-poc-value-negative-mechanisms
./dev/check-poc-value-coverage
./dev/check-poc-value-combined
./dev/check-poc-breadth
./dev/check-build-optimization-performance
```

The commands validate the final POC decision, strict no-value/no-action
evidence, accelerator breadth, and the combined public path. The last also
preserves the earlier mechanism-development scorecard for historical
comparison. None reruns Gradle or adds percentages from different workloads.
Use the owning benchmark runner only when the relevant implementation or
fixture changes.

`check-poc-breadth` accepts both a passing and a failing decision document and
recomputes all 64 observations. The checked result retains the narrow claim:
only 2/8 realistic change/DSL cells qualify, despite correct selection and
byte-identical outputs.

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
