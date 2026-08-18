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
| `internal/launcher` | Command passthrough, packaged Gradle discovery, signals, gateway lifecycle, L1, authority handoff, bootstrap cache, central state sync and remote profile revalidation | `buildopt`; launcher, cache and central optimize specs |
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
