# Gradle Build Optimization — Implementation Tracker

**Overall status:** `DOING` — `F0-019` encoded exact/`UNATTRIBUTED` correlation and whole-attempt abort in the local Protobuf channel; `ENV-006` is unblocked next<br>
**Current phase:** Phase 0 — contracts, fixtures, and walking skeleton<br>
**Private beta functional target:** `A1 + B + C1 + C4`<br>
**Last updated:** 2026-07-29<br>
**Master RFC:** [gradle-build-optimization-platform.md](./gradle-build-optimization-platform.md)<br>
**RFC baseline SHA-256:** `e97b068433128a51cab509f2f799efdf872b6950056bce308b80cbd1470ef81d`

---

## 1. How to use this tracker

This file tracks implementation; the RFC retains product decisions, invariants, and contracts. If they diverge:

1. correct status, ownership, or evidence here;
2. decide a scope, architecture, safety-posture, or gate change in the RFC first;
3. then update this tracker, its date, and its changelog.

### Allowed states

| State | Meaning |
|---|---|
| `TODO` | Defined work that has not started |
| `DOING` | Active work with an owner |
| `WAITING` | Waiting for a normal dependency that is not closed yet |
| `BLOCKED` | An unresolved impediment requires action or a decision |
| `DONE` | Criterion completed with linked evidence |
| `DEFERRED` | Outside the current critical path by explicit decision |
| `UNAVAILABLE` | A spike proves that the capability is not viable for that combination; safe degradation must exist |

### Update rules

- Do not use subjective percentages. Phase progress is `DONE gates / total gates`.
- No item moves to `DONE` without evidence: a file, test, build, benchmark, ADR, release, or pilot result.
- An `Accepted` RFC decision does not mean its implementation is complete.
- `UNAVAILABLE` closes a spike only when the specified fallback is implemented and tested.
- Every `DOING` item must have one accountable owner, even when several people collaborate.
- Every `BLOCKED` item must also appear in the blocker register in section 13.
- When the RFC changes, verify its SHA-256 and reconcile this tracker's dependencies, gates, and baseline.

---

## 2. Executive status

### 2.1 Milestones

| Milestone | Objective | State | Gate progress | Dependency |
|---|---|---:|---:|---|
| Preparation | RFC, beta scope, and tracker closed | `DONE` | 2/2 | — |
| Phase 0 | Toolchains, executable contracts, fixtures, spikes, and walking skeleton | `TODO` | 1/6 | Preparation |
| MVP-A0 | Foundation and internal pilot | `WAITING` | 0/9 | Phase 0 |
| MVP-A1 | Autonomous Cache in an isolated private beta | `WAITING` | 0/6 | A0 + `OPS-001/A1` |
| MVP-B | Runtime Optimizer and safe learning | `WAITING` | 0/6 | A1 + `CI-ORCH-001` + `OPS-001/B` |
| MVP-C1 | Task Intelligence, JVM Agent, and Linux hermeticity | `WAITING` | 0/9 | B |
| MVP-C4 | PR-only Patch Autopilot | `WAITING` | 0/7 | B; C1 for custom tasks; Test Optimization where applicable |
| MVP-A2 | Self-hosted single-node | `DEFERRED` | 0/1 | A1 |
| MVP-C2 | Edge Cache Node | `DEFERRED` | 0/1 | A1 |
| MVP-C3 | Build Impact Analysis | `DEFERRED` | 0/1 | B + `INT-001` |
| GA-D | Production-ready hardening | `DEFERRED` | 0/1 | Functional beta with demonstrated value |

Design baseline: 51 private-beta decisions are accepted in the RFC; their artifacts and tests remain pending as recorded in this tracker.

### 2.2 Critical path

```text
Preparation
  → Phase 0 core
  → Walking skeleton
  → MVP-A0
  → MVP-A1
  → MVP-B
      ├─→ MVP-C1 ─→ C4 custom-task recipe
      └─→ C4 infrastructure + archive recipe
                   └─→ complete MVP-C4
```

The beta target is not complete until A1, B, C1, and C4 close. A2, C2, C3, and GA-D do not block that release.

### 2.3 Next executable items

| Order | ID | Next outcome | State | Owner |
|---:|---|---|---|---|
| 1 | `F0-001` | Implementation workspace/repository and modules defined | `DONE` | Codex |
| 2 | `F0-002` | Golden lane pinned through an ADR, wrapper, image, and runner class | `DONE` | Codex |
| 3 | `ENV-001` | Toolchain and checksum manifest created | `DONE` | Codex |
| 4 | `ENV-002` | `dev/doctor` reproduces the host inventory | `DONE` | Codex |
| 5 | `ENV-003` | JDK 21 available alongside and isolated from global JDK 25 | `DONE` | Codex |
| 6 | `ENV-004` | Plugin/agent compilation verified with `--release 17` | `DONE` | Codex |
| 7 | `ENV-005` | Locked Go toolchain validated for the core | `DONE` | Codex |
| 8 | `ENV-008` | Docker and the golden image verified by digest | `DONE` | Codex |
| 9 | `ENV-009` | Rust toolchain pinned for the optional hermetic helper | `DONE` | Codex |
| 10 | `ENV-010` | ShellCheck and actionlint provisioned from the lock | `DONE` | Codex |
| 11 | `F0-010` | `contracts/`, `specs/`, `benchmarks/`, and `adr/` structure created | `DONE` | Codex |
| 12 | `F0-011` | First normative schema: `BUILD_SESSION v1` | `DONE` | Codex |
| 13 | `WS-001` | `buildopt run` → `BUILD_SESSION` vertical slice started | `DONE` | Codex |
| 14 | `WS-002` | Process group and signal forwarding | `DONE` | Codex |
| 15 | `F0-040A` | First Gradle correlation fixture on the golden lane | `DONE` | Codex |
| 16 | `SPK-001` | Task → cache key → PUT spike on the golden lane | `UNAVAILABLE` | Codex |
| 17 | `F0-019` | Encode the tested correlation fallback in the local task-event channel | `DONE` | Codex |
| 18 | `ENV-006` | Provision exact `protoc` and adopted Buf through repository-local tooling | `TODO` | — |

---

## 3. Owners and workstreams

| Workstream | Scope | First artifact | Owner |
|---|---|---|---|
| Contracts | JSON Schema, OpenAPI, Protobuf, canonicalization, and compatibility | `contracts/` + codegen | — |
| Go core | `buildopt`, gateway, and `buildopt-server` | CLI passthrough + ingest | — |
| Gradle | Plugin, adapters, TestKit, and capability matrix | Handshake plugin | — |
| JVM Agent | JVM instrumentation and coverage | `SPIKE-AGENT-001` | — |
| Hermetic helper | Rust helper and task-specific producer | `SPIKE-HERMETIC-001` | — |
| CI/orchestration | GitHub Action, validation workflow, and lifecycle | `ci-orchestration-v1.md` | — |
| Cache/storage | L1/L2, pending/commit, SLRU, and recovery | Atomicity ADR | — |
| Experiments | Metrics, A/A, resource profiles, and bandit | `resource-profile.v1` | — |
| Patch Engine | Bundle, patcher, recipes, and draft PR | `PatchBundle v1` | — |
| Test Optimization | Grants and `FULL_RELEVANT_VALIDATION` | Integration fixtures | — |
| Operations | Benchmark, supply chain, runbooks, and pilot | `benchmark-beta-v1.md` | — |

---

## 4. Cross-cutting readiness gates

These states represent implementation and evidence, not the `Accepted` state of RFC decisions.

| Gate | Required outcome | Blocks | State | Owner | Evidence |
|---|---|---|---|---|---|
| `CONTRACTS-001` | Schemas/IDLs, Go/Java clients, N/N-1, and golden vectors | A0+ integration | `TODO` | — | — |
| `GOLDEN-LANE-001` | Gradle 9.6.1, JDK 21, Linux x86-64, Kotlin DSL, pinned 4 vCPU/16 GiB development runner and image | Walking skeleton | `DONE` | Codex | `E-008`: strict host and container checks passed |
| `CI-ORCH-001` | Authoritative normal job, validation queue, isolation, lifecycle, budget, and recovery | B, C1, and C4 | `TODO` | — | — |
| `GRADLE-CORR-001` | Exact `taskExecutionId → cacheKey → PUT` correlation or all-attempt fallback | Selective C1 publication | `DONE` | Codex | `E-023`: non-task stores select tested all-attempt fallback |
| `HERMETIC-SCOPE-001` | Task-specific producer and coverage/fault fixtures | C1 hermetic path | `TODO` | — | — |
| `BANDIT-001` | Arms/features/reward, replay, A/A, delayed outcomes, drift, and rollback | B bandit | `TODO` | — | — |
| `PATCH-BUNDLE-001` | Parser/applier, golden/negative vectors, and idempotency | C4 materialization | `TODO` | — | — |
| `TESTOPT-API-001` | Producer/consumer fixtures, trust, grants, artifacts, retries, and N/N-1 | Test grants and patches with tests | `TODO` | — | — |
| `DEPLOY-001` | Installable artifacts, upgrade/uninstall, and end-to-end fixture | A1 pilot | `TODO` | — | — |
| `OPS-001/A1` | Benchmark/fault/soak, readiness, revocation, bypass, and runbooks | A1 pilot | `TODO` | — | — |
| `OPS-001/B` | GitHub adapter with an `EXACT` queue | B | `TODO` | — | — |

---

## 5. Phase 0 — contracts, fixtures, and walking skeleton

**State:** `TODO`<br>
**Objective:** eliminate implementation ambiguity and prove the first end-to-end flow without active optimizations.

### 5.1 Development environment bootstrap

The environment must be reproducible and must not depend on unversioned global tools:

- `dev/toolchains.lock.yaml` pins each toolchain's version, platform, URL/provider, and SHA-256.
- `dev/doctor` is read-only and verifies the host, versions, paths, Docker, capabilities, and available space.
- `dev/bootstrap` is idempotent and installs verified tools under `.tools/` by default; it does not use `sudo` or silently replace global toolchains.
- `dev/run` sets target-specific variables and `PATH` only for the project process.
- CI and workstations consume the same lock. The golden lane also uses a container image pinned by digest.
- Do not install Gradle globally: every fixture and module uses a pinned Gradle Wrapper.
- Rust remains optional for the core, but must be pinned for `SPK-003` and C1.

#### Initial workstation inventory

Verified snapshot from the initial 4-vCPU workstation on 2026-07-29. Manually installed tools prepared that workstation but do not replace the repository lock or close later `ENV-*` items on their own. Other workstations may have different global tools and paths; `ENV-002` will report those differences against the same lock.

| Dependency | Needed for | Initial target | Detected | State | Action |
|---|---|---|---|---|---|
| Linux x86-64 | Entire MVP | Recorded kernel/capabilities | Linux 6.8 x86-64 | Available | Re-run `dev/doctor` on each workstation; C1 retains strict capability qualification |
| JDK/Javac | Golden lane, plugin, and agent | JDK 21; `--release 17` bytecode | Global OpenJDK/Javac 25.0.3 + isolated Temurin 21.0.12+8 | Available on workstation | Closed by `ENV-003`; use the verified repository-local JDK through `dev/run` |
| Go | Launcher, gateway, and server | Exact version in lock | Global Go 1.26.5 + isolated Go 1.26.5 | Available on workstation | Closed by `ENV-005`; use the verified repository-local Go through `dev/run --toolchain go` |
| Rust/Cargo | C1 hermetic helper | Exact version in `rust-toolchain.toml` | Global stable 1.96.0 + repository Rust/Cargo 1.93.0 | Available, optional | Closed by `ENV-009`; the repository override leaves the global Rustup default unchanged |
| `protoc` | Local-events Protobuf | Exact project-local version | Isolated 35.1; checksum and descriptor smoke test verified | Available on workstation | Provision from the lock in `ENV-006` |
| Buf | Protobuf lint/breaking/codegen | Exact project-local version | Isolated 1.72.0; `buf lint` passed | Adopted by `F0-019`; repository provisioning pending | Provision the adopted locked tool in `ENV-006` |
| Gradle | Plugin/fixtures | Gradle Wrapper 9.6.1 first; pinned 8.14.x for `SPK-001` | No global installation; Wrapper 9.6.1 + isolated 8.14.3 | Available | Keep 9.6.1 as the golden Wrapper; provision 8.14.3 only through the checksum-verified spike harness |
| Docker | Golden image and fixture services | Functional daemon + image by digest | Client/server 24.0.2, `overlay2` | Available | Closed by `ENV-008`; use the digest-verifying golden container runner |
| Git | Workspace and patch workflows | Available | Git 2.54.0 | Available | Doctor records the active path/version; minimum support remains a later policy decision |
| SQLite CLI | Diagnostics/fault fixtures | Available | SQLite 3.45.1 | Available | Doctor reports it when present; do not use it as a runtime API |
| `jq`, `curl`, `tar`, `xz`, `unzip` | Bootstrap and fixtures | Available | Installed | Available | Required host-command probes implemented in `dev/doctor` |
| C/C++ toolchain | Rust/native fault fixtures | GCC/Clang available | GCC 13.3, Clang 18.1 | Available | Doctor inventories active versions; strict verification remains in C1 |
| `shellcheck` | Bootstrap/CI scripts | Project-local or pinned package | Repository-local 0.11.0; all executable `dev/` scripts passed | Available through `dev/bootstrap` and `dev/run` | Closed by `ENV-010`; use `dev/check-lint-toolchains` |
| `actionlint` | GitHub workflows | Project-local and pinned | Repository-local 1.7.12; integrated workflow smoke passed | Available through `dev/bootstrap` and `dev/run` | Closed by `ENV-010`; authoritative workflows remain in `F0-004` |
| `cosign`/`syft` | Signatures, SBOM, and provenance | Project-local and pinned | Isolated cosign 3.1.2 + Syft 1.50.0; version/SBOM smoke tests passed | Available on workstation | Provision from the lock and complete sign/verify in `F0-038` |

Initial 4-vCPU workstation installation:

- Root: `/home/tonyredondo/.local/share/buildopt/toolchains` (646 MB).
- Provenance and SHA-256 manifest: `/home/tonyredondo/.local/share/buildopt/toolchains/manifest.json`.
- Verified artifacts retained on the portable drive: `/media/portable/.cache/buildopt-toolchains/2026-07-29` (420 MB).
- `buildopt-with-jdk21 <command>` set `JAVA_HOME` and `PATH` only for the child process; `ENV-003` supersedes it for repository work with `./dev/run -- <command>`.

#### Bootstrap workboard

| ID | Deliverable | Depends on | State | Owner | Expected evidence |
|---|---|---|---|---|---|
| `ENV-001` | Create `dev/toolchains.lock.yaml` and update policy | `F0-001`, `F0-002` | `DONE` | Codex | `E-010`: portable lock, validator, and update policy |
| `ENV-002` | Implement read-only `dev/doctor` | `ENV-001` | `DONE` | Codex | `E-011`: JSON/human reports, live inventory, and deterministic exit-code fixtures |
| `ENV-003` | Provision JDK 21 alongside JDK 25 without global replacement | `ENV-001` | `DONE` | Codex | `E-012`: verified, idempotent provisioning and isolated `dev/run` |
| `ENV-004` | Verify plugin/agent compilation with `--release 17` | `ENV-003` | `DONE` | Codex | `E-013`: separate JARs, major 61, agent load, and golden-image build |
| `ENV-005` | Validate and pin Go for the core | `ENV-001` | `DONE` | Codex | `E-014`: exact module/toolchain, isolated execution, passing doctor and reproducible build |
| `ENV-006` | Provision `protoc` and adopted Buf | `F0-019`, `ENV-001` | `TODO` | — | Reproducible lint/generate/round trip |
| `ENV-007` | Pin Gradle Wrapper 9.6.1 and verify checksum | `F0-002` | `DONE` | Codex | `E-007`: Wrapper 9.6.1 and checksums verified |
| `ENV-008` | Verify Docker and golden image by digest | `F0-002`, `ENV-002` | `DONE` | Codex | `E-015`: immutable index/platform identity, strict cgroups, and container smoke |
| `ENV-009` | Pin Rust 1.93.0 or an approved version | `ENV-001` | `DONE` | Codex | `E-016`: exact Rustup override, locked manifest, doctor match, and offline Cargo check |
| `ENV-010` | Provision `shellcheck` and `actionlint` | `ENV-001` | `DONE` | Codex | `E-017`: exact archives, isolated execution, and passing integrated lint smoke |
| `ENV-011` | Provision `cosign`/`syft` for the supply chain | `ENV-001`, `F0-038` | `WAITING` | — | SBOM/sign/verify fixture |
| `ENV-012` | Implement `dev/bootstrap`, `dev/run`, and documented cleanup | `ENV-001..011` | `WAITING` | — | Two idempotent runs + uninstall |

Do not mark `ENV-003`, `ENV-006`, `ENV-010`, or `ENV-011` complete because a “similar” tool exists on the host: it must match the lock and pass its smoke test.

### 5.2 Foundation

| ID | Deliverable | Depends on | State | Owner | Expected evidence |
|---|---|---|---|---|---|
| `F0-001` | Create workspace/repository, modules, and build conventions | — | `DONE` | Codex | `E-006`: initial commit + [README](./README.md) + passing `./dev/check-layout` |
| `F0-002` | Pin golden lane and ADR `0001-golden-lane` | `F0-001` | `DONE` | Codex | `E-007`: passing ADR, contract, Wrapper, fixture, and host/container smoke tests |
| `F0-003` | Define ownership by workstream and review boundaries | `F0-001` | `TODO` | — | Documented CODEOWNERS/owners |
| `F0-004` | Configure base CI for Go, Java 17, and optional Rust | `F0-001` | `TODO` | — | Passing reproducible checks |
| `F0-005` | Add generated-code policy and drift detection | `F0-004` | `WAITING` | — | Check fails on stale codegen |

### 5.3 Normative contracts

| ID | Deliverable | Related decision | State | Owner | Expected evidence |
|---|---|---|---|---|---|
| `F0-010` | Create `contracts/`, `specs/`, `benchmarks/`, and `adr/` structure | `CONTRACTS-001` | `DONE` | Codex | `E-018`: checked RFC §29.2 namespaces and artifact indexes |
| `F0-011` | `build-session.v1.schema.json` | `OBS-002`, `METRICS-001` | `DONE` | Codex | `E-019`: Draft 2020-12 schema + 4 valid/7 invalid fixtures |
| `F0-012` | `experiment-result.v1` and `action-record.v1` | `OBS-002`, `MEASURE-001` | `WAITING` | — | Schemas + lifecycle fixtures |
| `F0-013` | Evidence, policy, and resource-profile schemas | `TASK-001`, `BANDIT-001` | `WAITING` | — | Schemas + golden records |
| `F0-014` | Attempt, validation-request, and `CommitDecision` schemas | `CACHE-008`, `CI-ORCH-001` | `WAITING` | — | Schemas + state fixtures |
| `F0-015` | Test grant/result schemas | `TESTOPT-API-001` | `WAITING` | — | Signed schemas + negative fixtures |
| `F0-016` | `PatchBundle v1` schema | `PATCH-BUNDLE-001` | `WAITING` | — | Schema + bundle vectors |
| `F0-017` | OpenAPI BuildOpt control/cache | `CONTRACTS-001`, `CACHE-008` | `WAITING` | — | Validated OpenAPI + mock server |
| `F0-018` | OpenAPI Test Optimization | `TESTOPT-API-001` | `WAITING` | — | Producer/consumer mocks |
| `F0-019` | Protobuf local task-event channel | `GRADLE-CORR-001` | `DONE` | Codex | `E-024`: normative IDL, ADR, descriptor parity, and Go/Java Unix round trips |
| `F0-020` | JCS, SHA-256, timestamp, and signature vectors | `CONTRACTS-001` | `WAITING` | — | Identical Go/Java golden vectors |
| `F0-021` | Error, deadline, retry, and idempotency contract | `CONTRACTS-001` | `WAITING` | — | Conformance fault cases |
| `F0-022` | N/N-1 compatibility and generated Go/Java clients | `F0-011..021` | `WAITING` | — | Passing compatibility suite |
| `F0-023` | Task/action/attempt state machines | `STATE-001`, `CI-ORCH-001` | `WAITING` | — | Transition vectors + recovery |
| `F0-024` | Normative `METRICS-001` catalog | `METRICS-001`, `MEASURE-001` | `WAITING` | — | Definitions, units, and signs |

`F0-019` intentionally follows `SPK-001`: its normative event vocabulary must encode the demonstrated exact task-owned observations plus the tested `UNATTRIBUTED` all-attempt fallback. The spike uses fixture-only instrumentation and does not predeclare the final inter-process contract.

### 5.4 Executable specifications and fixtures

| ID | Deliverable | Related decision | State | Owner | Expected evidence |
|---|---|---|---|---|---|
| `F0-030` | `ci-orchestration-v1.md` | `CI-ORCH-001` | `TODO` | — | Scheduling, isolation, lifecycle, and recovery |
| `F0-031` | `CommitDecision + COMMITTED` atomicity ADR | `CACHE-008`, `STORAGE-001` | `TODO` | — | ADR + transaction test plan |
| `F0-032` | `benchmark-beta-v1.md` and `beta-v1.yaml` | `OPS-001` | `TODO` | — | Seed, workload, and fault matrix |
| `F0-033` | `test-optimization-integration-v1.md` | `TESTOPT-API-001` | `TODO` | — | Contract + fixtures for both products |
| `F0-034` | `patch-bundle-v1.md` | `PATCH-BUNDLE-001` | `TODO` | — | Format, path safety, and recovery |
| `F0-035` | Bandit policy/replay specification | `BANDIT-001` | `TODO` | — | Arms, buckets, reward, and reset |
| `F0-036` | Capability matrix v1 | `COMPAT-001` | `TODO` | — | Exact/Approximate/Unavailable by combination |
| `F0-037` | Data lifecycle, redaction, and JSON/JSONL fixtures | `PRIVACY-001`, `EXPORT-001` | `TODO` | — | Golden export + deletion cases |
| `F0-038` | Packaging, checksums, signatures, SBOM, and provenance | `DEPLOY-001` | `TODO` | — | Verifiable artifacts |
| `F0-039` | Base runbooks: bypass, kill switch, rollback, and uninstall | `OPS-001` | `TODO` | — | Recorded exercises |
| `F0-040A` | Golden-lane correlation fixture | `COMPAT-001` | `DONE` | Codex | `E-022`: parallel equivalent tasks, shared key, miss/hit, and Configuration Cache reuse |
| `F0-040` | Tier 1 fixtures and test repositories | `COMPAT-001` | `TODO` | — | Real Wrapper + TestKit matrix |

### 5.5 Walking skeleton

```text
GitHub Action
  → buildopt run -- ./gradlew build
  → Gradle Optimization Plugin
  → Local Verifying Cache Gateway
  → buildopt-server
  → BUILD_SESSION v1 JSON
```

| ID | Deliverable | Depends on | State | Owner | Expected evidence |
|---|---|---|---|---|---|
| `WS-001` | `buildopt run -- <argv>` preserves argv and exit code | `F0-001`, `F0-002` | `DONE` | Codex | `E-020`: real-binary CLI integration suite |
| `WS-002` | Process group and signal forwarding | `WS-001` | `DONE` | Codex | `E-021`: Linux process-tree signal and cancellation fixtures |
| `WS-003` | Gradle plugin handshakes without optimizing | `F0-019`, `WS-001` | `TODO` | — | Real Wrapper on golden lane |
| `WS-004` | Loopback gateway with authenticated rendezvous | `F0-019`, `WS-001` | `TODO` | — | Restart/concurrency fixture |
| `WS-005` | `buildopt-server` receives a session | `F0-011`, `WS-004` | `WAITING` | — | Ingest integration test |
| `WS-006` | Export valid `BUILD_SESSION v1` JSON | `WS-003`, `WS-005` | `WAITING` | — | Schema validation |
| `WS-007` | GitHub Action pinned by SHA/checksum | `WS-001..006`, `F0-038` | `WAITING` | — | Passing workflow fixture |
| `WS-008` | Bypass, failure/cancel, and attempt/lease cleanup | `WS-001..007`, `F0-039` | `WAITING` | — | Fault suite |
| `WS-009` | Measure overhead from the neutral envelope | `WS-001..008`, `F0-024` | `WAITING` | — | Baseline-vs-wrapper report |

### 5.6 Bounded spikes

| ID | Question to answer | Depends on | State | Owner | Outcome/evidence |
|---|---|---|---|---|---|
| `SPK-001` | Can we correlate task → native key → PUT exactly with parallel execution, Worker API, and child processes? | `F0-002`, `F0-040A` | `UNAVAILABLE` | Codex | `E-023`: task stores exact; non-task Kotlin DSL stores force whole-attempt abort |
| `SPK-002` | What coverage and overhead does the JVM Agent achieve with a real daemon and Configuration Cache? | `F0-002`, `F0-040`, `WS-003` | `WAITING` | — | — |
| `SPK-003` | Can the Rust helper enforce a task-specific producer with complete coverage on the supported runner? | `F0-002`, `F0-040` | `WAITING` | — | — |
| `SPK-004` | Can the patcher safely apply, repeat, reject, and recover the first two bundles? | `F0-016`, `F0-034` | `WAITING` | — | — |

Allowed spike outcomes: `DONE` with a supported capability, or `UNAVAILABLE` with a tested fallback. An ambiguous result remains `BLOCKED` or `DOING`.

### 5.7 Phase 0 exit gate

| ID | Criterion | State | Evidence |
|---|---|---|---|
| `F0-G01` | Private-beta decisions closed in RFC §28 | `DONE` | [RFC §28](./gradle-build-optimization-platform.md#28-product-decisions) |
| `F0-G02` | Applicable contracts, catalog, and golden vectors validated | `TODO` | — |
| `F0-G03` | Complete, passing walking skeleton on the golden lane | `TODO` | — |
| `F0-G04` | Executable conformance/fixtures for the next module | `TODO` | — |
| `F0-G05` | Bypass, kill switch, and `UNAVAILABLE` exist before optimization | `TODO` | — |
| `F0-G06` | Pinned toolchains and passing `dev/doctor` on host/golden image | `TODO` | — |

---

## 6. MVP-A0 — foundation and internal pilot

**State:** `WAITING`<br>
**Entry gate:** Phase 0 closed for A0.

### 6.1 Workboard

| ID | Deliverable | State | Owner | Evidence |
|---|---|---|---|---|
| `A0-001` | Internal launcher/gateway production path | `WAITING` | — | — |
| `A0-002` | Tier 1 plugin and default-deny allowlist | `WAITING` | — | — |
| `A0-003` | Managed L1 `DirectoryBuildCache` | `WAITING` | — | — |
| `A0-004` | Shared single-node: blobs + `cache.sqlite`/`control.sqlite` | `WAITING` | — | — |
| `A0-005` | Pending/abort/`CommitDecision`/CAS/reconciliation | `WAITING` | — | — |
| `A0-006` | Locally authenticated policy and revocation generations | `WAITING` | — | — |
| `A0-007` | Dependency cache and wrapper distributions | `WAITING` | — | — |
| `A0-008` | Complete/partial JSON/JSONL export | `WAITING` | — | — |
| `A0-009` | Neutral measurement envelope and causal pilot harness | `WAITING` | — | — |

### 6.2 Exit gates

| ID | Summarized criterion | State | Evidence |
|---|---|---|---|
| `A0-G01` | Tier 1 conformance: hit/miss/PUT/413/redirect/timeout/corruption/unknown | `WAITING` | — |
| `A0-G02` | L2 → L1 → revocation → miss/rotation; abort leaves no hit | `WAITING` | — |
| `A0-G03` | Gateway restart/rotation compatible with Configuration Cache | `WAITING` | — |
| `A0-G04` | Spool verifies before `200` and survives disk/cancel/checksum/crash faults | `WAITING` | — |
| `A0-G05` | CAS and `CommitDecision + COMMITTED` atomicity; safe recovery | `WAITING` | — |
| `A0-G06` | Sessions ≥5 s: p95 overhead ≤500 ms and ≤2%; <5 s: ≤100 ms or skip L2 | `WAITING` | — |
| `A0-G07` | Schema distinguishes complete/partial records without future effects | `WAITING` | — |
| `A0-G08` | Without a grant, no root/composite `Test` task uses cache | `WAITING` | — |
| `A0-G09` | Internal pilot demonstrates net causal savings | `WAITING` | — |

---

## 7. MVP-A1 — Autonomous Cache private beta

**State:** `WAITING`<br>
**Entry gate:** stable A0 + `OPS-001/A1`.

### 7.1 Workboard

| ID | Deliverable | State | Owner | Evidence |
|---|---|---|---|---|
| `A1-001` | `PRIVATE_BETA_ISOLATED` deployment per pilot | `WAITING` | — | — |
| `A1-002` | Scoped, hashed, revocable read/read-write tokens | `WAITING` | — | — |
| `A1-003` | Quotas, TTL, watermarks, and byte-based SLRU | `WAITING` | — | — |
| `A1-004` | Data lifecycle, redaction, deletion, and diagnostic opt-in | `WAITING` | — | — |
| `A1-005` | Exercised health/readiness, circuit breaker, and runbooks | `WAITING` | — | — |
| `A1-006` | First design-partner onboarding and operation | `WAITING` | — | — |

### 7.2 Exit gates

| ID | Summarized criterion | State | Evidence |
|---|---|---|---|
| `A1-G01` | Scope/namespace isolation and zero tokens in forks | `WAITING` | — |
| `A1-G02` | Fault/soak meets acceptance targets and preserves Gradle | `WAITING` | — |
| `A1-G03` | SLRU/watermarks/pools behave according to contract | `WAITING` | — |
| `A1-G04` | Fail-closed restart/recovery and correct rotations | `WAITING` | — |
| `A1-G05` | Export/redaction/deletion cover managed copies | `WAITING` | — |
| `A1-G06` | External pilot demonstrates causal benefit and safe p95 | `WAITING` | — |

---

## 8. MVP-B — Runtime Optimizer

**State:** `WAITING`<br>
**Entry gate:** stable A1 + valid A/A + `CI-ORCH-001` + `OPS-001/B`.

### 8.1 Workboard

| ID | Deliverable | State | Owner | Evidence |
|---|---|---|---|---|
| `B-001` | Isolated candidate/control/stable paths and validation scheduler | `WAITING` | — | — |
| `B-002` | Local Configuration Cache policy; never distributed entries | `WAITING` | — | — |
| `B-003` | Autotuning with four initial resource profiles | `WAITING` | — | — |
| `B-004` | Contractual removal of `clean` | `WAITING` | — | — |
| `B-005` | Allowlisted invocation merging and policy prefetch | `WAITING` | — | — |
| `B-006` | A/A and fixed cohorts with propensities | `WAITING` | — | — |
| `B-007` | Contextual epsilon-greedy and replay simulator | `WAITING` | — | — |
| `B-008` | Budget, canary, fallback, rollback, and kill switch | `WAITING` | — | — |

### 8.2 Exit gates

| ID | Summarized criterion | State | Evidence |
|---|---|---|---|
| `B-G01` | Valid A/A, sample ratio, delayed reward, and propensities | `WAITING` | — |
| `B-G02` | Intentional candidate/control contamination does not reach stable | `WAITING` | — |
| `B-G03` | Autotuning reduces build time without breaking p95/p99/queue/OOM | `WAITING` | — |
| `B-G04` | Merge/clean pass failure/finalizer/side-effect fixtures | `WAITING` | — |
| `B-G05` | Bandit selects only safe runtime arms | `WAITING` | — |
| `B-G06` | Permanent control/revalidation and suspension on drift | `WAITING` | — |

---

## 9. MVP-C1 — Task Intelligence

**State:** `WAITING`<br>
**Entry gate:** demonstrated B and control pipeline.

### 9.1 Workboard

| ID | Deliverable | State | Owner | Evidence |
|---|---|---|---|---|
| `C1-001` | Opt-in JVM Agent with coverage and budgets | `WAITING` | — | — |
| `C1-002` | Evidence pipeline and `traceComplete/traceCoverage` | `WAITING` | — | — |
| `C1-003` | Task-qualification state machine | `WAITING` | — | — |
| `C1-004` | Exact task/key/PUT correlation on supported combinations | `WAITING` | — | — |
| `C1-005` | Selective pending/commit and suspension/revocation | `WAITING` | — | — |
| `C1-006` | Adapter/source-patch path for custom tasks | `WAITING` | — | — |
| `C1-007` | Rust helper and task-specific producer | `WAITING` | — | — |
| `C1-008` | Repeatability, relocatability, and artifact validation | `WAITING` | — | — |
| `C1-009` | First real custom task from the pilot | `WAITING` | — | — |

### 9.2 Exit gates

| ID | Summarized criterion | State | Evidence |
|---|---|---|---|
| `C1-G01` | Mutating each input changes the key or invalidates the task | `WAITING` | — |
| `C1-G02` | History alone never activates a task; tests remain excluded | `WAITING` | — |
| `C1-G03` | A discrepancy suspends/aborts without failing the baseline | `WAITING` | — |
| `C1-G04` | Incomplete coverage produces `INCONCLUSIVE` | `WAITING` | — |
| `C1-G05` | Hermetic-only uses a continuous task-specific producer | `WAITING` | — |
| `C1-G06` | Exact `GRADLE-CORR-001`; `UNATTRIBUTED` aborts | `WAITING` | — |
| `C1-G07` | Artifact validation and Test Optimization where applicable | `WAITING` | — |
| `C1-G08` | Agent/helper crash preserves baseline and does not contaminate | `WAITING` | — |
| `C1-G09` | A real custom task reaches ACTIVE and saves time causally | `WAITING` | — |

---

## 10. MVP-C4 — Patch Autopilot

**State:** `WAITING`<br>
**Entry gate:** B + `PATCH-BUNDLE-001`; C1 for custom-task patches; `TESTOPT-API-001` when tests are required.

### 10.1 Workboard

| ID | Deliverable | State | Owner | Evidence |
|---|---|---|---|---|
| `C4-001` | `PatchBundle v1` signer/verifier | `WAITING` | — | — |
| `C4-002` | Exact, idempotent, path-safe Java patcher | `WAITING` | — | — |
| `C4-003` | Recipe `ARCHIVE_REPRODUCIBILITY_KOTLIN_DSL_V1` | `WAITING` | — | — |
| `C4-004` | Recipe `CUSTOM_TASK_CONTRACT_JAVA_V1` | `WAITING` | — | — |
| `C4-005` | Candidate/control and artifact validation | `WAITING` | — | — |
| `C4-006` | `FULL_RELEVANT_VALIDATION` integration | `WAITING` | — | — |
| `C4-007` | Customer-side branch + draft PR workflow | `WAITING` | — | — |
| `C4-008` | Branch-without-PR recovery and idempotent retries | `WAITING` | — | — |
| `C4-009` | Post-merge measurement and revert-PR path | `WAITING` | — | — |

### 10.2 Exit gates

| ID | Summarized criterion | State | Evidence |
|---|---|---|---|
| `C4-G01` | No automatic rebase, default-branch write, or automatic merge | `WAITING` | — |
| `C4-G02` | Candidate/control pass correctness checks and validation | `WAITING` | — |
| `C4-G03` | PR explains the change, evidence, impact, and rollback | `WAITING` | — |
| `C4-G04` | `PRELIMINARY` is not counted as confirmed impact | `WAITING` | — |
| `C4-G05` | Post-merge does not fake rollout and creates a revert PR on regression | `WAITING` | — |
| `C4-G06` | First accepted patch saves time causally | `WAITING` | — |
| `C4-G07` | Patcher passes golden/negative/idempotency/recovery vectors | `WAITING` | — |

---

## 11. Optional tracks and hardening

| Track | Minimum scope for reactivation | State | Trigger |
|---|---|---|---|
| MVP-A2 | Self-hosted installer, migration, and recovery | `DEFERRED` | Stable A1 protocol + pilot demand |
| MVP-C2 | Edge proxy, SLRU, and offline committed reads | `DEFERRED` | Latency/volume justify Edge |
| MVP-C3 | Conservative BIA and `BIA-002` gate | `DEFERRED` | Stable B + `INT-001` + demand |
| GA-D Identity | OIDC/workload identity, SSO/RBAC, KMS/HSM | `DEFERRED` | Beta demonstrates value |
| GA-D Storage | Object store, HA metadata, backups, and RPO/RTO | `DEFERRED` | Production-ready requirements |
| GA-D Privacy | Residency, legal hold, and backup deletion | `DEFERRED` | Contractual requirements |
| GA-D Export | OTLP, Prometheus, Parquet, SIEM, and webhooks | `DEFERRED` | Prioritized demand |
| Platform expansion | macOS/Windows, Android, KMP, and native | `DEFERRED` | Stable core matrix |

---

## 12. Continuous validation

This table points to the latest valid result. It does not replace reports or allow gates to be marked without evidence.

| Suite | Scope | Latest result | Date | Evidence |
|---|---|---|---|---|
| Repository layout | Git root, modules, and `F0-001` conventions | `dev/check-layout` and ShellCheck passed; structure included in the initial commit | 2026-07-29 | `E-006` |
| Toolchain lock | Portable versions, platforms, providers, immutable URLs, SHA-256 values, and tracker references | `dev/check-toolchains-lock` passed for ten artifacts on `linux-amd64`; update policy recorded | 2026-07-29 | `E-010` |
| Host inventory | Platform/resources, active Java/Go/Rust/Protobuf/base-tool paths and versions, Docker, cgroups, namespaces, and available space | `dev/doctor --json` passed on the 12-CPU workstation and reported active-path drift without treating deferred provisioning as success; deterministic fixtures passed codes `0/1/64/70` and preserved the working tree | 2026-07-29 | `E-011` |
| Go toolchain | Exact compiler, module baseline, local-only selection and caches, offline module state, and deterministic smoke build | Official Go 1.26.5 archive matched the lock; real and synthetic isolation checks passed; doctor reported the project-local binary as `MATCH`; repeated smoke binaries matched | 2026-07-29 | `E-014` |
| Rust toolchain | Exact optional compiler/Cargo pair, repository override, locked channel manifest, isolated offline state, and Cargo smoke | Rust/Cargo 1.93.0 and the Linux AMD64 host matched; official manifest SHA-256, doctor `MATCH`, temporary-state Cargo check, global-default isolation, and deterministic negative fixtures passed | 2026-07-29 | `E-016` |
| Lint toolchains | Exact ShellCheck/actionlint artifacts, isolated execution, repository scripts, and workflow parsing | Official ShellCheck 0.11.0 and actionlint 1.7.12 archives matched the lock; real and synthetic provisioning, doctor matching, all executable `dev/` scripts, and the integrated workflow smoke passed | 2026-07-29 | `E-017` |
| Normative package layout | RFC §29.2 namespaces, indexes, planned artifact parents, and non-empty materialized artifacts | `dev/check-normative-layout` passed for 14 namespaces and 26 planned artifacts; ShellCheck and repository layout passed | 2026-07-29 | `E-018` |
| Contract schemas | `BUILD_SESSION v1` Draft 2020-12; later JSON Schemas and OpenAPI remain pending | Pinned Go validator compiled the schema with format assertions; 4 valid fixtures passed and 7 focused invalid fixtures were rejected for their intended diagnostics | 2026-07-29 | `E-019` |
| Local task-event contract | Protobuf v1 payload, varint framing, exact/unattributed correlation, and attempt-wide abort | Exact protoc/Buf descriptors matched; Java 17 and Go peers passed both real Unix-socket directions, semantic negatives, frame bounds, and the Go race detector | 2026-07-29 | `E-024` |
| Golden vectors | Go ↔ Java canonicalization/signatures | Not run | — | — |
| Go unit/integration | Launcher CLI; gateway/server remain pending | Real `buildopt` binary preserved argv/process context and child statuses; Linux fixtures proved a distinct child group, exact SIGINT/SIGTERM delivery to a descendant, delayed cancellation cleanup, and conventional unhandled-signal status | 2026-07-29 | `E-021` |
| Gradle correlation fixture | Gradle 9.6.1/JDK 21 golden-lane slice; Tier 1/TestKit expansion remains pending | Independent multi-project fixture produced one shared native key from two parallel equivalent tasks, then restored both from an isolated local cache while reusing Configuration Cache on host and strict container | 2026-07-29 | `E-022` |
| Gradle correlation spike | Gradle 9.6.1 and 8.14.3; parallel tasks, Worker API, child JVM, remote cache, failure/cancellation, and Configuration Cache | Task-owned remote stores correlated exactly by operation ancestry and matched HTTP PUTs; cold Kotlin DSL/accessor stores had no task ancestor, selected `UNAVAILABLE`, and exercised the whole-attempt fallback; complete host matrix and strict 9.6.1 container gate passed | 2026-07-29 | `E-023` |
| Gradle TestKit | Plugin/adapters | Not run | — | — |
| Real Gradle Wrapper | Wrapper 9.6.1, Kotlin DSL fixture, and Configuration Cache | Strict gate passed on the nominal 4 vCPU/16 GiB host and a container with 4 CPU/16 GiB cgroups; bytecode major 61; Configuration Cache reused; `compileJava FROM-CACHE`; negative 2-CPU fixture rejected | 2026-07-29 | `E-008` |
| Golden container runtime | Docker daemon, immutable image index/platform identity, local image provenance, exact JDK patch, and cgroup limits | Docker 29.6.2 resolved the pinned index to the unique Linux AMD64 digest; strict image inspection and a 4-CPU/16-GiB container build passed; deterministic negative fixtures passed | 2026-07-29 | `E-015` |
| JVM release compatibility | Gradle plugin and JVM agent | Locked JDK 21 compilation with `--release 17`; every packaged class verified as major 61; reproducible JARs; agent loaded on Java 17 and 21; complete build passed in the pinned golden image | 2026-07-29 | `E-013` |
| Cache conformance | HttpBuildCache + internal control | Not run | — | — |
| Fault injection | Cache/orchestration/recovery | Not run | — | — |
| Hermetic harness | Rust helper/process tree/coverage | Not run | — | — |
| Patch vectors | Sign/apply/idempotency/path safety | Not run | — | — |
| Test Optimization | Producer/consumer compatibility | Not run | — | — |
| Beta benchmark | Load/fault/soak | Not run | — | — |
| Pilot metrics | Causal build-time impact | Not run | — | — |

---

## 13. Blocker register

| ID | Since | Affects | Blocker | Owner | Next action | State |
|---|---|---|---|---|---|---|
| — | — | — | No active blockers; `WAITING` dependencies are normal sequencing | — | — | — |

---

## 14. Evidence register

| ID | Date | Item/gate | Evidence | Outcome |
|---|---|---|---|---|
| `E-001` | 2026-07-29 | Preparation/RFC | Initial [master RFC](./gradle-build-optimization-platform.md) baseline, SHA-256 `77d2e63075456e3b03e5754a4db83d150e2153a6951ca7bbc424379b179d539b` | Superseded by the revision recorded in `E-008` |
| `E-002` | 2026-07-29 | Preparation/tracker | This document | Initial tracker created |
| `E-003` | 2026-07-29 | Bootstrap/inventory | Version commands and Docker daemon verified locally; snapshot in §5.1 | JDK 21, protoc/Buf, and supply-chain tooling pending |
| `E-004` | 2026-07-29 | Bootstrap/toolchains | Local manifest for seven artifacts; official SHA-256 values verified; JDK `--release 17` produced major 61; Protobuf/Buf/lint and SBOM smoke tests passed | Workstation ready to materialize lock/bootstrap |
| `E-005` | 2026-07-29 | Bootstrap/space | Eight temporary items inspected with no open files, including two MP4 files of equal duration; 3.4 GB moved from `/tmp` to `/media/portable/.recovery/buildopt-cleanup-2026-07-29` | `/tmp` ~4.9→1.6 GB; root 94%→92%; recovery available |
| `E-006` | 2026-07-29 | `F0-001` workspace | Git `main` repository, [module map](./README.md), [conventions](./CONTRIBUTING.md), and `dev/check-layout`; Bash parse, ShellCheck, and layout executed | `DONE`: structure validated and versioned in the initial commit |
| `E-007` | 2026-07-29 | `F0-002` / `ENV-007` | Initial ADR, runner spec, Wrapper 9.6.1, and fixture; official checksums; passing static/host/container smoke tests; Configuration Cache reuse and compile-cache hit | `F0-002` and `ENV-007` validated and versioned; the initial 8 vCPU proposal was superseded by `E-008` |
| `E-008` | 2026-07-29 | `GOLDEN-LANE-001` | Pre-translation [master RFC](./gradle-build-optimization-platform.md) baseline, SHA-256 `b8ab7fb89086365b62bef53219bd919b23a0a867191e092bcb56425a8f82a776`, revised [ADR 0001](./adr/0001-golden-lane.md), and runner spec `linux-amd64-4c-16g-v1`; strict host and digest-pinned container exited 0; negative 2-CPU cgroup exited 1 | `DONE`: 4 vCPU/16 GiB development golden lane verified without extrapolating to customer benchmarks; RFC text superseded by `E-009` without changing the decision set |
| `E-009` | 2026-07-29 | Repository language baseline | Repository-owned tracked content audited in English; English-only contribution policy added; translated [master RFC](./gradle-build-optimization-platform.md) SHA-256 `e97b068433128a51cab509f2f799efdf872b6950056bce308b80cbd1470ef81d` | Current English repository baseline; the 51 accepted private-beta decisions and executable golden-lane state are preserved |
| `E-010` | 2026-07-29 | `ENV-001` | [`dev/toolchains.lock.yaml`](./dev/toolchains.lock.yaml), static validator, and update policy; eight GitHub release digests reverified against official release metadata plus official Go archive and Rust channel-manifest metadata; Bash syntax, lock validation, ShellCheck, layout, golden-lane static checks, and the Java 21 smoke build passed on the non-golden 12-CPU host | `DONE`: one host-independent `linux-amd64` lock now unblocks `ENV-002`, `ENV-003`, `ENV-005`, `ENV-009`, and `ENV-010` |
| `E-011` | 2026-07-29 | `ENV-002` | [`dev/doctor`](./dev/doctor), [`dev/test-doctor`](./dev/test-doctor), documented `buildopt.dev/doctor-report/v1`, live human/JSON inventory on the 12-CPU workstation, isolated JDK 21/25 active-path matches, deterministic missing-command and internal-error fixtures, Bash syntax, ShellCheck 0.11.0, read-only Git-state comparison, and exit codes `0/1/64/70` | `DONE`: workstations now share one active-path comparison contract without global-tool assumptions; `ENV-008` is unblocked |
| `E-012` | 2026-07-29 | `ENV-003` | [`dev/bootstrap`](./dev/bootstrap), [`dev/run`](./dev/run), and [`dev/test-jdk-toolchain`](./dev/test-jdk-toolchain); official Temurin 21.0.12+8 bytes matched lock SHA-256 `e4446ff06a276155697597cc0f1b15da004ff083f4964a35271ecee567177370`; first and idempotent second provisioning passed; synthetic global JDK 25 isolation, real global JDK 17 preservation, doctor `MATCH`, and Gradle 9.6.1 smoke with Java 17 bytecode passed on the non-golden 12-CPU host | `DONE`: JDK 21 is reproducibly available through `dev/run` without replacing global Java; `ENV-004` is unblocked |
| `E-013` | 2026-07-29 | `ENV-004` | [`dev/check-jvm-release`](./dev/check-jvm-release), separate Gradle plugin and JVM agent builds, locked Temurin 21.0.12+8 compiler with `--release 17`, `-Xlint:all`, and `-Werror`; all packaged classes verified at major 61; constrained `Premain-Class` loaded on project Java 21 and host Java 17.0.19; clean rebuild hashes matched; root smoke passed locally and in the digest-pinned golden image | `DONE`: Java 17 artifact compatibility is executable without activating the `WS-003` handshake or `SPK-002` instrumentation behavior; `ENV-005` was unblocked |
| `E-014` | 2026-07-29 | `ENV-005` | Root [`go.mod`](./go.mod), [`dev/check-go-toolchain`](./dev/check-go-toolchain), generalized [`dev/bootstrap`](./dev/bootstrap) and [`dev/run`](./dev/run), plus [`dev/test-go-toolchain`](./dev/test-go-toolchain); official Go 1.26.5 bytes matched lock SHA-256 `5c2c3b16caefa1d968a94c1daca04a7ca301a496d9b086e17ad77bb81393f053`; first and idempotent provisioning, exact Linux AMD64 compiler, `GOTOOLCHAIN=local`, disabled user Go environment, project-local caches, offline module checks, identical repeated smoke binaries, synthetic global-Go isolation, unchanged real global selection, and doctor `MATCH` passed on the non-golden 12-CPU host | `DONE`: the Go core now has a reproducible compiler/module baseline without activating product behavior; `ENV-008` became the next executable block |
| `E-015` | 2026-07-29 | `ENV-008` | Strengthened [`dev/check-golden-lane`](./dev/check-golden-lane), [`dev/run-golden-lane-container`](./dev/run-golden-lane-container), and [`dev/golden-lane-build`](./dev/golden-lane-build), plus deterministic [`dev/test-golden-lane-container`](./dev/test-golden-lane-container); Docker 29.6.2 resolved immutable index `sha256:9d8dcf999b0bce2453e913823595a5ff2a4e8e9e5d5241b45280d0ff069818ec` to the unique Linux AMD64 platform digest `sha256:a5418a1fcf440bb273e1db3bce5b0794eb78bfc9d044ba740de76dcbe6075f50`, pulled and inspected that exact image, verified Temurin 21.0.11+10, enforced 4 CPU/16 GiB and observed matching cgroup v2 limits, then passed the offline Gradle 9.6.1 build; negative fixtures covered mutable/spec drift, daemon, resources, index, local image, Java patch, usage, and child failure without starting a real container | `DONE`: Docker and the golden runtime are executable by immutable digest without making the 12-CPU host itself satisfy the 4-CPU runner class; `ENV-009` became the next executable block |
| `E-016` | 2026-07-29 | `ENV-009` | Root [`rust-toolchain.toml`](./rust-toolchain.toml), [`dev/check-rust-toolchain`](./dev/check-rust-toolchain), deterministic [`dev/test-rust-toolchain`](./dev/test-rust-toolchain), and a read-only Rustup-aware [`dev/doctor`](./dev/doctor); official Rust 1.93.0 channel-manifest bytes matched lock SHA-256 `beb6ba4e41c84e9c11c80e6804a007497d0c8ba0810cd403fabc8f4a9c45b1f8`; the repository selected rustc 1.93.0 commit `254b59607` and Cargo 1.93.0 commit `083ac5135` for `x86_64-unknown-linux-gnu`, doctor reported `MATCH`, and a dependency-free edition-2024 crate passed `cargo check --locked --offline` with temporary Cargo home/target state; outside the repository, stable 1.96.0 remained the unchanged Rustup default; negative fixtures covered missing/mismatched tools, manifest checksum, configuration drift, usage, and Cargo failure without installing a toolchain | `DONE`: the optional helper has a reproducible compiler baseline without implementing or claiming `SPK-003` hermeticity; `ENV-010` is next |
| `E-017` | 2026-07-29 | `ENV-010` | Generalized [`dev/bootstrap`](./dev/bootstrap) and [`dev/run`](./dev/run), [`dev/check-lint-toolchains`](./dev/check-lint-toolchains), and deterministic [`dev/test-lint-toolchains`](./dev/test-lint-toolchains); official ShellCheck 0.11.0 `tar.xz` bytes matched SHA-256 `8c3be12b05d5c177a04c29e3c78ce89ac86f1595681cab149b65b97c4e227198`, and official actionlint 1.7.12 `tar.gz` bytes matched `8aca8db96f1b94770f1b0d72b6dddcb1ebb8123cb3712530b08cc387b349a3d8`; atomic and idempotent provisioning normalized both upstream layouts, exact project-local versions remained isolated from global ShellCheck 0.9.0 and missing global actionlint, doctor reported both as `MATCH`, all executable `dev/` scripts passed ShellCheck, and actionlint parsed an in-memory workflow while using the locked ShellCheck for its embedded Bash; negative fixtures covered checksum and manifest drift, missing tools, usage, and child lint failures | `DONE`: reproducible lint tooling is available without creating or claiming the authoritative `F0-004` workflow; `F0-010` is next |
| `E-018` | 2026-07-29 | `F0-010` | [`dev/check-normative-layout`](./dev/check-normative-layout) and non-empty indexes under [`contracts/`](./contracts/README.md), [`specs/`](./specs/README.md), [`benchmarks/`](./benchmarks/README.md), and [`adr/`](./adr/README.md); all 14 namespaces from RFC §29.2 exist, all 26 planned artifact paths have an indexed owner and materialized parent, existing golden-lane artifacts and any materialized planned path must be non-empty, and repository layout plus locked ShellCheck passed | `DONE`: the normative package structure is executable without prematurely implementing schemas, APIs, IDLs, vectors, specs, benchmarks, or ADR decisions; `F0-011` is unblocked and next |
| `E-019` | 2026-07-29 | `F0-011` | Normative [`BUILD_SESSION v1`](./contracts/jsonschema/build-session.v1.schema.json), documented [fixture set](./contracts/jsonschema/testdata/build-session.v1/README.md), and [`dev/check-build-session-schema`](./dev/check-build-session-schema); exact `jsonschema/v6` 6.0.2 is isolated in the test module and authenticated by `go.sum`; Draft 2020-12 compilation with asserted RFC 3339 formats accepted four complete/partial, CI/local, success/failure fixtures and rejected seven focused cases for lifecycle mismatch, future aggregate effects, impossible timestamps, negative duration, missing recovery ranges, inconsistent success exit, and invented unavailable values; module tidy/verify/vet, the unchanged offline core Go toolchain check, JSON/Bash syntax, repository/normative layouts, locked ShellCheck/actionlint, and the locked-JDK Gradle build passed | `DONE`: `BUILD_SESSION` now fixes neutral-envelope boundaries, explicit metric availability/methods, pre-outcome assignment, complete/partial recovery, strict unknown fields, and OBS-002 lifecycle separation without activating generated clients or later metric catalogs; `WS-001` is next |
| `E-020` | 2026-07-29 | `WS-001` | Dependency-free [`cmd/buildopt`](./cmd/buildopt/main.go), [`internal/launcher`](./internal/launcher/run.go), real-binary [integration tests](./cmd/buildopt/main_test.go), and [`dev/check-buildopt-cli`](./dev/check-buildopt-cli); the locked Go 1.26.5 toolchain built the executable with offline module resolution, then direct execution without a shell preserved empty, whitespace, quoted, wildcard, variable-like, literal-delimiter, Unicode, and newline arguments plus `argv[0]`, cwd, environment, stdin, stdout, and stderr; child exits `0` and `37`, usage `64`, cannot-execute `126`, and not-found `127` passed; Go formatting/vet, the unchanged offline module/toolchain contract, repository layout, Bash syntax, and locked ShellCheck/actionlint passed | `DONE`: the first walking-skeleton behavior now executes the original command and returns ordinary child statuses without activating policy, plugin, gateway, telemetry, optimization, or untested signal semantics; `WS-002` is unblocked and next |
| `E-021` | 2026-07-29 | `WS-002` | Linux process-group and signal handling in [`internal/launcher`](./internal/launcher/run.go), real-binary [signal integration tests](./cmd/buildopt/main_signal_linux_test.go), and a nested [signal helper](./cmd/buildopt/testdata/signal-helper/main.go); the direct child became a group leader distinct from `buildopt`, its descendant inherited the group, and signals sent only to the launcher reached both processes as exact `SIGINT` and `SIGTERM`; handled cancellation waited through a 150 ms cleanup and preserved child exits `41`/`42`, while unhandled `SIGTERM` returned `143` without launcher stderr; the signal suite passed 10 consecutive runs and the race detector, full CLI tests, Go vet including the helper, the unchanged offline module/toolchain contract, repository layout, Bash syntax, and locked ShellCheck/actionlint passed | `DONE`: the launcher now preserves the Linux process-tree cancellation contract without imposing or shortening the CI provider's grace period; platform expansion and observability/lifecycle behavior remain deferred, and `SPK-001` remains dependency-gated |
| `E-022` | 2026-07-29 | `F0-040A` | Independent [`fixtures/gradle-correlation`](./fixtures/gradle-correlation/README.md) repository and executable [`dev/check-gradle-correlation-fixture`](./dev/check-gradle-correlation-fixture), integrated into [`dev/golden-lane-build`](./dev/golden-lane-build); the real Gradle 9.6.1 Wrapper and JDK 21 compiled the Java 17 fixture plugin with all warnings as errors, then an isolated empty-cache run forced `:alpha` and `:beta` task actions to overlap through a barrier, produced byte-identical outputs, and reported the same native cache key `c0111bcb4ba8ba492a6cb273f724a55b`; the next clean invocation reused Configuration Cache and restored both tasks `FROM-CACHE`; the check passed on the 12-CPU host and in the digest-pinned 4-CPU/16-GiB container, with repository layout, Bash syntax, and locked ShellCheck/actionlint also passing | `DONE`: the first golden-lane correlation repository is reproducible without claiming task-to-PUT association; `SPK-001` now owns instrumentation plus Worker API, child-process, failure/cancellation, and Gradle 8.14.x expansion, and `F0-019` follows its exact-or-fallback result |
| `E-023` | 2026-07-29 | `SPK-001` / `GRADLE-CORR-001` | Executable [`dev/check-gradle-correlation-spike`](./dev/check-gradle-correlation-spike), loopback [HTTP cache fixture](./fixtures/gradle-correlation/cache-server/src/main/java/dev/buildopt/fixtures/correlation/CorrelationCacheServer.java), structural [trace analyzer](./fixtures/gradle-correlation/buildSrc/src/main/java/dev/buildopt/fixtures/correlation/CorrelationTraceAnalyzer.java), and [decision specification](./specs/gradle-correlation-v1.md); Gradle 9.6.1 and checksum-pinned 8.14.3 each ran with a cold isolated Gradle home across parallel equivalent tasks, Worker API no/process isolation, a real child JVM, remote miss/hit, failure, cancellation, and Configuration Cache reuse; every task-owned remote store had exactly one `ExecuteTask` ancestor and matched an HTTP PUT, while cold Kotlin DSL/accessor compilation produced ten non-task stores on 9.6.1 and nine on 8.14.3; missing and multiple-ancestor self-tests also emitted `UNATTRIBUTED`; the complete host matrix and the 9.6.1 half inside the digest-pinned 4-CPU/16-GiB container passed | `DONE` gate with `UNAVAILABLE` capability: selective task publication is disabled for both tested combinations; any non-task, missing, or ambiguous PUT sets `attemptAborted=true` for the whole pending attempt, and `F0-019` is unblocked to encode that result |
| `E-024` | 2026-07-29 | `F0-019` | Normative [`task_events.proto`](./contracts/proto/local-events/v1/task_events.proto), accepted [ADR 0003](./adr/0003-local-task-event-channel.md), adopted root [`buf.yaml`](./buf.yaml), and executable [`dev/check-task-events-proto`](./dev/check-task-events-proto); exact `protoc` 35.1 and Buf 1.72.0 linted the IDL and produced the identical descriptor SHA-256 `a8168a4b91824a6f195145c4fba25f9d15fe17c5faa5cd5757bc42269acf1f9e`; locked Go 1.26.5 and Temurin 21/Java 17 peers exchanged conventional length-delimited frames in both directions over real Unix sockets and validated exact task-owned PUTs, non-task `UNATTRIBUTED`, final attempt-wide `UNAVAILABLE`, matching acknowledgements, immediate whole-attempt abort, invalid cross-field combinations, unknown required enums, and the 1 MiB frame limit; the Go race detector, full Go tests/vet, CLI integration, schema suite, layouts, toolchain lock, ShellCheck/actionlint, JVM release check, golden static check, and host golden-lane build passed | `DONE`: the channel never promotes exact task-only observations when the same attempt has an unattributed PUT; Buf is adopted, generated clients remain owned by `F0-022`, and `ENV-006`, `WS-003`, and `WS-004` are unblocked |

---

## 15. Tracker changelog

| Date | Change | Author |
|---|---|---|
| 2026-07-29 | Closed `F0-019`: materialized the versioned local Protobuf channel and ADR, adopted Buf, and proved exact plus fail-closed `UNATTRIBUTED` semantics with Go/Java Unix-socket round trips | Codex |
| 2026-07-29 | Closed `SPK-001` as `UNAVAILABLE`: task-owned stores correlate exactly, but cold Kotlin DSL/accessor stores have no task ancestor; added the pinned 9.6.1/8.14.3 matrix and tested whole-attempt fallback, closing `GRADLE-CORR-001` and unblocking `F0-019` | Codex |
| 2026-07-29 | Closed `F0-040A`: removed the `SPK-001`/`F0-019` dependency cycle and added the first parallel Gradle correlation fixture with cache and Configuration Cache evidence on host and strict golden container | Codex |
| 2026-07-29 | Closed `WS-002`: isolated the Linux child process group, forwarded `SIGINT`/`SIGTERM` through the process tree, preserved cleanup and exit semantics, and added repeated cancellation fixtures | Codex |
| 2026-07-29 | Closed `WS-001`: added the dependency-free `buildopt run --` passthrough, real-binary argv/stdio/cwd/environment and exit-status integration tests, and kept signal semantics isolated for `WS-002` | Codex |
| 2026-07-29 | Closed `F0-011`: added the strict `BUILD_SESSION v1` Draft 2020-12 contract, explicit unavailable/partial metric semantics, lifecycle isolation, positive/negative fixtures, and a pinned validator isolated from the product module | Codex |
| 2026-07-29 | Closed `F0-010`: materialized and checked the RFC §29.2 normative namespaces and artifact indexes without creating empty future contracts | Codex |
| 2026-07-29 | Closed `ENV-010`: provisioned exact repository-local ShellCheck and actionlint archives, integrated their lint smoke, preserved global selections, and added deterministic failure fixtures | Codex |
| 2026-07-29 | Closed `ENV-009`: pinned the optional Linux AMD64 Rust 1.93.0 toolchain, verified its official channel manifest, isolated Cargo smoke state, and preserved the global Rustup default | Codex |
| 2026-07-29 | Closed `ENV-008`: bound the golden index to its Linux AMD64 digest, verified the pulled image and exact Java patch, enforced strict cgroups, and added deterministic Docker failure fixtures | Codex |
| 2026-07-29 | Closed `ENV-005`: added the exact root Go module/toolchain contract, checksum-verified project-local provisioning, isolated execution and caches, deterministic fixtures, doctor matching, and a reproducible offline smoke build | Codex |
| 2026-07-29 | Closed `ENV-004`: added neutral Gradle plugin/JVM agent artifacts, enforced `--release 17`, verified major 61 and agent loading, and passed local plus pinned-image builds | Codex |
| 2026-07-29 | Closed `ENV-003`: added checksum-verified, atomic JDK 21 provisioning and project-local execution without replacing global Java; unblocked `ENV-004` | Codex |
| 2026-07-29 | Closed `ENV-002`: added read-only human/JSON host inventory, stable exit codes, active-path toolchain drift reporting, Docker/capability/resource probes, and deterministic fixtures | Codex |
| 2026-07-29 | Closed `ENV-001`: added the portable toolchain lock, static validation, update policy, and dependency evidence without adopting candidate or optional tools | Codex |
| 2026-07-29 | Established English as the repository-wide language and translated all repository-owned documentation while preserving contracts, decisions, and validated implementation state | Codex |
| 2026-07-29 | Adjusted golden lane to the 4 vCPU/16 GiB development workstation; strict host/container gate passed and `GOLDEN-LANE-001` closed | Codex |
| 2026-07-29 | Closed `F0-002` and `ENV-007`: ADR, runner spec, verified Wrapper, and host/container fixture; unblocked `ENV-001`; strict gate pending | Codex |
| 2026-07-29 | Closed `F0-001`: structure validated and versioned; `F0-002..004` unblocked | Codex |
| 2026-07-29 | Started `F0-001`: monorepo, module boundaries, conventions, and verified layout check | Codex |
| 2026-07-29 | Installed and verified local toolchains; added manifest, smoke evidence, and space preflight | Codex |
| 2026-07-29 | Added reproducible bootstrap and actual workstation dependency inventory | Codex |
| 2026-07-29 | Initial tracker derived from the RFC; all implementation items start without an owner or evidence | Codex |

---

## 16. Periodic update template

Copy this section into a follow-up note or summarize it when updating the tracker:

```text
Date:
Current milestone:

Completed:
- ID — evidence

In progress:
- ID — owner — next checkpoint

Blocked:
- ID — cause — required decision/action

Validation:
- suite — outcome — evidence

Measured impact:
- customerVisibleBuildMs:
- overhead:
- p95/p99:
- queue:
- additional compute:

Scope or RFC changes:
- none | description + decision ID

Next 3 items:
1.
2.
3.
```
