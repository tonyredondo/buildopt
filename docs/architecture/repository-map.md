# Repository map

This document is the bridge between the conceptual architecture and the
source tree. Use it to find the owning layer before changing behavior.

## Top-level structure

```text
buildopt/
├── cmd/             Go executable entrypoints
├── internal/        Private Go implementation packages
├── jvm/             Gradle plugin, JVM agent, generated client, patcher
├── rust/            Optional experimental Linux helper
├── contracts/       Normative schemas, APIs, Protobuf, metrics, vectors
├── specs/           Executable cross-component behavior
├── adr/             Durable architecture decisions
├── fixtures/        Synthetic repositories and fault scenarios
├── benchmarks/      Workloads, budgets, and immutable results
├── packaging/       Native Linux, macOS, and Windows packages/services
├── runbooks/        Operator recovery and lifecycle procedures
├── dev/             Bootstrap, checks, generators, packaging, labs
├── .github/         GitHub Actions, ownership, and CI workflows
├── .gitlab/         GitLab component and verified installer
├── docs/            Explanatory user, developer, and architecture guides
├── gradle-build-optimization-platform.md
└── implementation-tracker.md
```

The repository is a monorepo because contracts, producers, consumers, fixtures,
and evidence must change together. It is not a collection of independently
versioned microservices.

## Active POC ownership

The current experiment is specified by
[`poc-fresh-generic-optimization-v1`](../../specs/poc-fresh-generic-optimization-v1.md)
and ordered by the
[`Fresh Generic Optimization POC Tracker`](../plans/fresh-generic-optimization-poc-tracker.md).
It reuses the sticky-wrapper implementation described below but accepts no
historical BuildOpt evidence as an experiment input.
Its generated repository-facing files are owned by `internal/stickywrapper`;
`SWL-001..006` own their contract, generator, verified bootstrap, passthrough,
portable connection and native Gradle HTTP cache integration. `SWL-007` owns
the canonical decision/evidence records and local/central adapters in
`internal/stickydecision`; `SWL-008` adds its read-only selector and
pre-Gradle native-retention budget; `SWL-008A` connects the sticky-wrapper
supervised native fast path and lazy observer in `internal/launcher`; `SWL-009` adds
ordinary-build observations in `internal/stickyobservation`. The existing
`internal/sharedcache` and
`internal/launcher` packages own the central Gradle HTTP cache, typed state and
launcher primitives that later blocks will reuse.
`internal/stickyactive` owns the generic active-runtime boundary: it revalidates
signed decisions, runs direct candidate/native commands, hashes required
outputs, samples native counterfactuals and suspends regressive profiles. It is
an execution-control POC, not a repository-specific optimization catalog.
`internal/durablecatalog` owns the review-only SWL-012 catalog: structural
task-contract and graph-breadth detectors, digest-bound recipes and exact
apply/revert proofs. Its proposals never merge or authorize runtime actions;
accepted task recipes leave plain native Gradle in charge. No third cache or
state service is planned.
`internal/stickywrapper/status.go` owns the SWL-013 read-only customer report.
It loads only validated local observations and decision snapshots, renders one
model as human text or JSON, and keeps missing evidence and native fallback
explicit. The generated wrapper templates own the unambiguous `--buildopt`
management routing; no report path grants action authority.
The SWL-014 installed proof is owned by `dev/run-poc-central-two-machine`,
`dev/central-two-machine-client` and `dev/check-sticky-wrapper-two-machine`,
with its normative contract in `specs/poc-sticky-wrapper-two-machine-v1.*` and
its checked result in `benchmarks/results/sticky-wrapper-two-machine-v1.json`.
The historical `SWL-014A` route is preserved by the diagnostic tracker,
`specs/poc-sticky-wrapper-longitudinal-v2.*`, the preflight-only
`dev/run-sticky-wrapper-longitudinal-v2`, its checker and fixture test. The
checked zero-pair result lives at
`benchmarks/results/sticky-wrapper-longitudinal-v2-preflight.json`. Final
campaign/result-directory behavior belonged to the old `SWL-015` route. The
active replacement is owned by `specs/poc-fresh-generic-optimization-v1.*`,
`docs/plans/fresh-generic-optimization-poc-tracker.md` and
`dev/check-fresh-generic-optimization-plan`.

`SWL-014B` added the only launcher-owned composition root at
`internal/launcher/sticky_learning.go`, plus shared deterministic value
statistics in `internal/stickyvalue`. It adapts
`internal/stickyobservation`, `internal/stickytrial`,
`internal/stickydecision`, `internal/stickyactive`,
`internal/durablecatalog` and the economic ledger. `SWL-014C` then owns
`internal/durablecatalog/public_screen.go` and the public opportunity gate;
`SWL-014D` owns the installed active-value command and checker. The exact file
manifests and commands are locked in the v2 machine contract. The composed
fixture and status evidence are checked by
`dev/check-sticky-wrapper-learning-lifecycle`; the old public opportunity
screen is diagnostic-only. `SWL-FRESH-001` next owns complete generic public
evidence producers.

Run `./dev/check-fresh-generic-optimization-plan` to validate current planning
authority and `./dev/check-sticky-wrapper-learning-plan` to preserve the
superseded route. As implementation begins, this map must name the concrete
owning paths in the same commit that adds them.

## Executables and their implementation

| Binary | Entrypoint | Main internal packages | User-facing documentation |
|---|---|---|---|
| `buildopt` | `cmd/buildopt/` | `launcher`, `sessioningest`, `buildsession`, `localauthority` | [CLI reference](../reference/cli.md), [component README](../../cmd/buildopt/README.md) |
| `buildopt-server` | `cmd/buildopt-server/` | `sharedcache`, `buildhistory`, `datalifecycle`, `selfhosted`, `githubqueue` | [Product workflows](../guides/product-workflows.md), [component README](../../cmd/buildopt-server/README.md) |
| `buildopt-edge` | `cmd/buildopt-edge/` | `edgecache` | [Operations](../guides/operations.md), [Edge runbook](../../runbooks/edge-cache.md) |
| `buildopt-impact` | `cmd/buildopt-impact/` | `buildimpact` | [Build Impact workflow](../guides/product-workflows.md#build-impact) |
| `buildopt-service.exe` | `cmd/buildopt-service/` | server and Edge entrypoints | [Windows operations](../guides/operations.md#windows) |
| `neutral-envelope` | `cmd/neutral-envelope/` | `neutralenvelope` | [component README](../../cmd/neutral-envelope/README.md) |
| Evaluation CLIs | `cmd/task-intelligence-evaluation/`, `cmd/beta-benchmark/` | corresponding internal packages | [Validation reference](../reference/validation.md) |

Files in `cmd/` should remain thin composition roots. Reusable behavior belongs
in `internal/`; cross-process representations belong in `contracts/` first.

## Go package map

| Package | Architectural responsibility | Closest executable/specification |
|---|---|---|
| `internal/launcher` | Command passthrough, packaged Gradle discovery, signals, native sticky-wrapper fast path, lazy observation, gateway lifecycle, L1, authority handoff, bootstrap cache, central state sync and remote profile revalidation | `buildopt`; launcher, cache and central optimize specs |
| `internal/adaptivefragment` | Canonical fragment family/revision identity, declared-binding compatibility, lifecycle, immutable typed state, discardable pre-Gradle compatibility index, non-authorizing chronological shadow replay, signed economics, requested-build online checkpoints, repository-independent hypothesis ranking, review-only task-contract opportunity detection, conflict-aware exact-composition planning and producer-local Build Impact activation | AF-001..AF-011 specs; `check-adaptive-fragment-contract`, `check-adaptive-fragment-state`, `check-adaptive-fragment-index`, `check-adaptive-fragment-shadow`, `check-adaptive-fragment-economics`, `check-adaptive-fragment-online`, `check-adaptive-fragment-prior`, `check-adaptive-fragment-patch-opportunity`, `check-adaptive-fragment-planner`, `check-adaptive-fragment-activation`, `check-adaptive-fragment-composition` |
| `internal/sessioningest` | Strict authenticated provisional session transport | `buildopt` and `buildopt-server`; `WS-005` |
| `internal/buildsession` | `BUILD_SESSION v1` production, immutable JSON, JSONL, recovery | server export; data lifecycle specs |
| `internal/buildhistory` | Redacted immutable history read model, API, embedded dashboard | `buildopt-server`; UX-F1 specs |
| `internal/localauthority` | Canonical signed policy/revocation verification and anti-rollback state | launcher and Shared; local authority spec |
| `internal/sharedcache` | Shared blobs, pending/commit CAS, independently governed Gradle-cache and typed BuildOpt-state SQLite lifecycles, quota, scoped central tokens, HTTPS handlers, reconciliation | `buildopt-server`; A0/A1 cache and central-state/HTTPS specs |
| `internal/edgecache` | Edge config, store, read-through, pending replication, quota, runtime status | `buildopt-edge`; C2/O1 specs |
| `internal/taskintelligence` | Qualification state, trace coverage, quarantine evidence | task-intelligence specs |
| `internal/buildimpact` | Manifest, graph discovery, validation, promotion, active selection | `buildopt-impact`; C3/BIA specs |
| `internal/profilediscovery` | Read-only, digest-bound POC profile derivation with native fallback | `buildopt profile discover`; profile-discovery spec |
| `internal/neutralenvelope` | Paired measurements, pilot assignment, reports, no-hit evidence | `neutral-envelope`; measurement specs |
| `internal/betabenchmark` | Synthetic load and disk/shared/system fault evidence | `beta-benchmark`; beta specs |
| `internal/datalifecycle` | Redaction profiles, deletion, leases, tombstones | `buildopt-server`; private-beta lifecycle specs |
| `internal/selfhosted` | Strict single-node config and storage preflight | `buildopt-server`; A2 specs |
| `internal/githubqueue` | Bounded GitHub workflow-job webhook adapter | server; GitHub queue spec |
| `internal/filelock` | Portable non-blocking advisory locks | shared by persistent components |
| `internal/platformfs` | Platform-specific no-link/reparse traversal checks | storage and private files |
| `internal/contractcrypto` | Shared canonical JSON and cryptographic primitives | contract vectors |
| `internal/metricscatalog` | Versioned metric catalog validation | metrics validator |
| `internal/stickydecision` | Canonical sticky-wrapper observations, action state, trials, signed decisions, economic ledger, local/central CAS stores, cache/state separation and the read-only native-retention selector | `SWL-007..008`; sticky decision-store and no-op specifications |
| `internal/stickyobservation` | Private append-only ordinary-build observations, lazy recorder creation, phase reconciliation, provenance and Configuration Cache evidence; no action authority | `SWL-009`; `poc-sticky-wrapper-observation-v1` |
| `internal/stickytrial` | Trusted-CI-only paired-trial scheduler, direct command runner, isolation digests, output equivalence and budget accounting; no action authority | `SWL-010`; `poc-sticky-wrapper-trial-v1` |
| `internal/stickyactive` | Revalidated active runtime profiles, native counterfactuals, exact-output checks, regression suspension and fail-closed native fallback; no shell or repository-specific rules | `SWL-011`; `poc-sticky-wrapper-active-v1` |
| `internal/durablecatalog` | Generic task-contract and graph-breadth opportunity detection, digest-bound reviewable recipes and exact apply/revert transactions; no automatic merge or runtime authority | `SWL-012`; `poc-sticky-wrapper-durable-catalog-v1` |
| `internal/stickywrapper` | Repository wrapper generation/bootstrap and read-only status/explanation reports; no credential exposure or action authorization | `SWL-001..004, SWL-013`; `poc-sticky-wrapper-status-v1` |
| `internal/generated` | Checked-in generated transport clients | generated-code manifest |

Package comments describe these boundaries in Go documentation. A package may
depend on another `internal/` package, but it must not redefine a normative
schema owned by `contracts/`.

## JVM modules

| Module | Purpose | Boundary |
|---|---|---|
| `jvm/gradle-plugin` | Settings/project plugins, handshake, managed L1/L2 inputs, Tier 1 policy | Public Gradle APIs and Configuration Cache compatibility |
| `jvm/jvm-agent` | Opt-in instrumentation experiment | Observational only; not a sandbox or qualification authority |
| `jvm/generated-client` | Generated OpenAPI transport bindings | Regenerated from the normative IDL; never edited manually |
| `jvm/patcher` | Signed bundle verification, exact recipes, isolated application, validation, draft/revert workflow | Never executes bundle content, rebases, force-pushes, or merges |

The root Gradle build includes all four modules and emits Java 17-compatible
bytecode using the pinned Wrapper and repository-local JDK 21.

## Contract-to-code relationships

| Normative source | Primary producers | Primary consumers | Conformance evidence |
|---|---|---|---|
| `specs/poc-sticky-wrapper-contract-v1.*`, `poc-sticky-wrapper-generator-v1.*`, `poc-sticky-wrapper-bootstrap-v1.*`, `poc-sticky-wrapper-passthrough-v1.*`, `poc-sticky-wrapper-connection-v1.*` and `poc-sticky-wrapper-cache-v1.*` | `internal/stickywrapper`; `internal/launcher`; `buildopt wrapper` CLI; embedded POSIX/Windows templates; `jvm/gradle-plugin` managed cache settings | generated repository wrappers, user-cache distributions, private central credentials and native Gradle HTTP cache objects | `check-sticky-wrapper-contract`, `check-sticky-wrapper-generator`, `check-sticky-wrapper-bootstrap`, `check-sticky-wrapper-passthrough`, `check-sticky-wrapper-connection`, `check-sticky-wrapper-cache`; portable fixtures, deterministic generation, bootstrap integrity, process parity, scope/capability/revocation, cache producer/consumer/outage and secret-isolation tests |
| `specs/poc-sticky-wrapper-decision-store-v1.*` and `contracts/jsonschema/sticky-wrapper-decision-store.v1.schema.json` | `internal/stickydecision`; owner signing/decision tooling | sticky-wrapper selector and future observation/trial consumers; local files and existing central `EVIDENCE` state | `check-sticky-wrapper-decision-store`; JCS/digest, transition, signature, evidence-reference, ledger, CAS, expiry, revocation, corruption and plane-separation vectors |
| `specs/poc-sticky-wrapper-noop-v1.*` | `internal/stickydecision` and `cmd/sticky-noop-benchmark` | read-only native-retention decision path before Gradle | `check-sticky-wrapper-noop`; signed local snapshot, missing/corrupt/incompatible fallback, refresh coalescing and p50/p95 budget |
| `specs/poc-sticky-wrapper-noop-overhead-v1.*` | `internal/launcher`; `dev/run-sticky-wrapper-noop-overhead` | sticky-wrapper native no-op execution and lazy light-observation cost | `check-sticky-wrapper-noop-overhead`; 20-sample interleaved overhead result, child-environment scrubbing, asynchronous digest and bounded p95 guardrails |
| `specs/poc-sticky-wrapper-observation-v1.*` and `contracts/jsonschema/sticky-wrapper-observation.v1.schema.json` | `internal/launcher` and `internal/stickyobservation` | ordinary Wrapper timing/provenance records and the checked-in observation dataset | `check-sticky-wrapper-observation`; real Wrapper invocations, append/load/tamper vectors, phase reconciliation and Configuration Cache reuse |
| `specs/poc-sticky-wrapper-trial-v1.*` and `contracts/jsonschema/sticky-wrapper-trial.v1.schema.json` | `internal/stickytrial`, `cmd/sticky-trial-benchmark` | bounded candidate/native reports and exact required-output hashes | `check-sticky-wrapper-trial`; alternating order, trusted-CI budget, eight-root isolation, cancellation/concurrency fixtures and four exact-output pairs |
| `specs/poc-sticky-wrapper-active-v1.*` and `contracts/jsonschema/sticky-wrapper-active.v1.schema.json` | `internal/stickyactive`, `cmd/sticky-active-benchmark` | active execution records, counterfactual comparison and suspension/fallback reasons | `check-sticky-wrapper-active`; negative qualification, direct-command execution, exact outputs, regression, drift, expiry, revocation, bypass and failure vectors |
| `specs/poc-sticky-wrapper-durable-catalog-v1.*` and `contracts/jsonschema/sticky-wrapper-durable-catalog.v1.schema.json` | `internal/durablecatalog`, `cmd/sticky-durable-catalog-benchmark` | review-only native task/graph proposals, exact source recipes and isolated transaction evidence | `check-sticky-wrapper-durable-catalog`; two DSL families, eight paired task-contract measurements, exact output hashes, apply/revert and structural graph proposals |
| `specs/poc-sticky-wrapper-two-machine-v1.*` | `dev/run-poc-central-two-machine`, `dev/central-two-machine-client` | installed wrapper archive, central Gradle cache objects and separate producer/consumer credentials | `check-sticky-wrapper-two-machine`; isolated HTTPS restart, owner-commit visibility, read-only cache hit, exact output equality and native outage fallback |
| `specs/poc-sticky-wrapper-longitudinal-v1.*` and `specs/poc-sticky-wrapper-longitudinal-v2.*` | v1 diagnostic runner/checker today; v2 runner/checker in `SWL-014A` | historical cache-asymmetric no-op evidence versus the future cache-symmetric, lifecycle-aware campaign | v1 sample remains `DIAGNOSTIC_ONLY`; v2 must reject unequal cache policy, missing action/ledger evidence and sample-count-only readiness |
| `contracts/jsonschema/adaptive-fragment*.v1.schema.json` and `specs/poc-adaptive-state-portability-v1.*` | adaptive learner and local state writer | `internal/adaptivefragment`; `internal/launcher` HTTPS state adapter | `check-adaptive-fragment-state`, `check-adaptive-fragment-index`, `check-adaptive-state-portability` |
| `contracts/jsonschema/build-session.v1.schema.json` | `internal/buildsession` | server history/export tooling | `check-build-session-schema`, `check-build-session-export` |
| `contracts/proto/local-events/v1/` | JVM Gradle plugin | launcher event channel | `check-task-events-proto`, plugin handshake/correlation checks |
| `contracts/openapi/buildopt-cache-control.v1.yaml` | server/control implementation | generated Go/Java clients | generated-client and compatibility checks |
| `contracts/jsonschema/patch-bundle.v1.schema.json` | patch signer/workflow | Java patch verifier/applier | patch bundle, candidate, workflow, and recovery checks |
| `contracts/jsonschema/central-state-*.v1.schema.json` | central-state clients and storage | `internal/sharedcache` typed state, HTTPS routes and `internal/launcher` sync/revalidation client | `check-central-storage-contract`, `check-central-state-storage`, `check-central-https-auth`, `check-central-state-sync`, `check-central-optimize-integration` |
| `specs/build-impact-*.json` | Build Impact packages/CLI | CI and selection flow | Build Impact gate and automatic discovery checks |
| `specs/platform-runtime-parity-v2.json` | native runtime and packaging | native CI | `check-platform-compatibility` |

Use [GENERATED_CODE.md](../../GENERATED_CODE.md) for the source-to-generated
manifest and exact regeneration commands.

## Tests, fixtures, and checks

The repository uses three evidence layers:

1. language-level unit and integration tests beside the code;
2. synthetic repositories and fault inputs under `fixtures/`;
3. `dev/check-*` composition gates that build real binaries and validate
   cross-component behavior.

`benchmarks/results/` contains immutable bounded evidence, not mutable golden
answers. A check may validate a result but must not rewrite it merely to pass.

Use the [validation reference](../reference/validation.md) to select the
smallest gate that owns a change. Base CI combines layout, ownership,
documentation, generated-code, Go, Java, and optional Rust checks. Native CI
adds macOS ARM64 and Windows AMD64 runtime/package/service coverage.

## Operational artifacts

| Path | Role |
|---|---|
| `dev/toolchains.lock.yaml` | Immutable development toolchain identities and checksums |
| `dev/package-release`, `dev/verify-release` | Signed reproducible Linux release bundle |
| `install.sh`, `install.ps1`, `.github/workflows/release.yml` | Public native-package download and tagged publication |
| `packaging/linux/` | Linux archive and receipt-based install/uninstall |
| `packaging/macos/` | Native archive, receipt-based install/uninstall, launchd agents |
| `packaging/windows/` | Native ZIP, receipt-based install/uninstall, SCM definitions |
| `specs/self-hosted-*.json` | Strict server config and lifecycle contracts |
| `specs/edge-cache*.json` | Edge config, authority, storage, and operability contracts |
| `runbooks/` | Human procedures paired with executable exercises |

Runtime state belongs outside the source tree. `.tools/` and `.buildopt/` are
ignored development state; deployment state roots must be absolute, private,
dedicated, and compatible with the platform storage policy.

## Where a change belongs

| If you are changing... | Start in | Also inspect |
|---|---|---|
| Product invariant or scope | Master RFC | tracker, affected contract/spec |
| Wire shape or persisted document | `contracts/` | generated clients, every producer/consumer |
| Cross-component behavior | `specs/` | fixture and owning `dev/check-*` |
| CLI parsing/composition | `cmd/<binary>/` | internal package and CLI reference |
| Reusable Go behavior | owning `internal/<package>/` | package tests and architecture docs |
| Gradle behavior | `jvm/gradle-plugin/` | Tier 1 fixtures and capability matrix |
| Patch recipe | `jvm/patcher/` | bundle/recipe specs and exact revert proof |
| CI installation | `.github/`, `.gitlab/`, `action.yml` | native packages, release workflow, and recovery runbook |
| Packaging/service lifecycle | `packaging/`, `dev/` | platform spec and operations guide |
| Explanation or onboarding | `docs/` or closest README | documentation gate and linked source |

If a change crosses rows, name all owning workstreams and validate the shared
contract rather than testing only the final binary.
