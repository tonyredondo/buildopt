# Gradle Build Optimization — Implementation Tracker

**Overall status:** `DOING` — drift-checked Go and Java 17 clients now expose all 13 control-plane operations and pass the same N/N-1 fail-closed compatibility corpus; `F0-023` is next for lifecycle state machines<br>
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
| Phase 0 | Toolchains, executable contracts, fixtures, spikes, and walking skeleton | `TODO` | 3/6 | Preparation |
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
| 18 | `ENV-006` | Provision exact `protoc` and adopted Buf through repository-local tooling | `DONE` | Codex |
| 19 | `WS-003` | Gradle plugin handshakes without optimizing | `DONE` | Codex |
| 20 | `WS-004` | Loopback gateway with authenticated rendezvous | `DONE` | Codex |
| 21 | `WS-005` | `buildopt-server` receives a session | `DONE` | Codex |
| 22 | `WS-006` | Export valid `BUILD_SESSION v1` JSON | `DONE` | Codex |
| 23 | `F0-038` | Packaging, checksums, signatures, SBOM, and provenance | `DONE` | Codex |
| 24 | `WS-007` | GitHub Action pinned by SHA/checksum | `DONE` | Codex |
| 25 | `F0-039` | Base runbooks for bypass, kill switch, rollback, and uninstall | `DONE` | Codex |
| 26 | `WS-008` | Bypass, failure/cancel, and attempt/lease cleanup fault suite | `DONE` | Codex |
| 27 | `F0-024` | Normative metric definitions, units, methods, and signs | `DONE` | Codex |
| 28 | `WS-009` | Measure overhead from the neutral envelope | `DONE` | Codex |
| 29 | `ENV-012` | Complete bootstrap cleanup and uninstall with two idempotent runs | `DONE` | Codex |
| 30 | `F0-003` | Define ownership by workstream and review boundaries | `DONE` | Codex |
| 31 | `F0-004` | Configure base CI for Go, Java 17, and optional Rust | `DONE` | Codex |
| 32 | `F0-005` | Add generated-code policy and drift detection | `DONE` | Codex |
| 33 | `F0-012` | Materialize `experiment-result.v1` and `action-record.v1` lifecycle contracts | `DONE` | Codex |
| 34 | `F0-013` | Materialize evidence, policy, and resource-profile schemas | `DONE` | Codex |
| 35 | `F0-014` | Materialize attempt, validation-request, and `CommitDecision` schemas | `DONE` | Codex |
| 36 | `F0-015` | Materialize signed Test Optimization grant/result schemas | `DONE` | Codex |
| 37 | `F0-016` | Materialize the signed declarative `PatchBundle v1` schema | `DONE` | Codex |
| 38 | `F0-017` | Materialize the BuildOpt control/cache OpenAPI contracts | `DONE` | Codex |
| 39 | `F0-018` | Materialize the Test Optimization OpenAPI contract | `DONE` | Codex |
| 40 | `F0-020` | Materialize common JCS, digest, timestamp, and signature vectors | `DONE` | Codex |
| 41 | `F0-021` | Materialize common error, deadline, retry, and idempotency conformance | `DONE` | Codex |
| 42 | `F0-022` | Generate Go/Java clients and prove N/N-1 compatibility | `DONE` | Codex |
| 43 | `F0-023` | Materialize task, action, and attempt state-machine vectors | `TODO` | — |

---

## 3. Owners and workstreams

| Workstream | Scope | First artifact | Owner |
|---|---|---|---|
| Contracts | JSON Schema, OpenAPI, Protobuf, canonicalization, and compatibility | `contracts/` + codegen | @tonyredondo |
| Go core | `buildopt`, gateway, and `buildopt-server` | CLI passthrough + ingest | @tonyredondo |
| Gradle | Plugin, adapters, TestKit, and capability matrix | Handshake plugin | @tonyredondo |
| JVM Agent | JVM instrumentation and coverage | `SPIKE-AGENT-001` | @tonyredondo |
| Hermetic helper | Rust helper and task-specific producer | `SPIKE-HERMETIC-001` | @tonyredondo |
| CI/orchestration | GitHub Action, validation workflow, and lifecycle | `ci-orchestration-v1.md` | @tonyredondo |
| Cache/storage | L1/L2, pending/commit, SLRU, and recovery | Atomicity ADR | @tonyredondo |
| Experiments | Metrics, A/A, resource profiles, and bandit | `resource-profile.v1` | @tonyredondo |
| Patch Engine | Bundle, patcher, recipes, and draft PR | `PatchBundle v1` | @tonyredondo |
| Test Optimization | Grants and `FULL_RELEVANT_VALIDATION` | Integration fixtures | @tonyredondo |
| Operations | Benchmark, supply chain, runbooks, and pilot | `benchmark-beta-v1.md` | @tonyredondo |

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
| `protoc` | Local-events Protobuf | Exact project-local version | Global 3.21.12 + project-local 35.1 | Available and provisioned | Closed by `ENV-006`; use the verified repository-local distribution through `dev/run --toolchain protoc` |
| Buf | Protobuf lint/breaking/codegen | Exact project-local version | Project-local 1.72.0; no global installation required | Available and provisioned | Closed by `ENV-006`; use the verified repository-local binary through `dev/run --toolchain buf` |
| Gradle | Plugin/fixtures | Gradle Wrapper 9.6.1 first; pinned 8.14.x for `SPK-001` | No global installation; Wrapper 9.6.1 + isolated 8.14.3 | Available | Keep 9.6.1 as the golden Wrapper; provision 8.14.3 only through the checksum-verified spike harness |
| Docker | Golden image and fixture services | Functional daemon + image by digest | Client/server 24.0.2, `overlay2` | Available | Closed by `ENV-008`; use the digest-verifying golden container runner |
| Git | Workspace and patch workflows | Available | Git 2.54.0 | Available | Doctor records the active path/version; minimum support remains a later policy decision |
| SQLite CLI | Diagnostics/fault fixtures | Available | SQLite 3.45.1 | Available | Doctor reports it when present; do not use it as a runtime API |
| `jq`, `curl`, `tar`, `xz`, `unzip` | Bootstrap and fixtures | Available | Installed | Available | Required host-command probes implemented in `dev/doctor` |
| C/C++ toolchain | Rust/native fault fixtures | GCC/Clang available | GCC 13.3, Clang 18.1 | Available | Doctor inventories active versions; strict verification remains in C1 |
| `shellcheck` | Bootstrap/CI scripts | Project-local or pinned package | Repository-local 0.11.0; all executable `dev/` scripts passed | Available through `dev/bootstrap` and `dev/run` | Closed by `ENV-010`; use `dev/check-lint-toolchains` |
| `actionlint` | GitHub workflows | Project-local and pinned | Repository-local 1.7.12; integrated workflow smoke passed | Available through `dev/bootstrap` and `dev/run` | Closed by `ENV-010`; authoritative workflows remain in `F0-004` |
| `cosign`/`syft` | Signatures, SBOM, and provenance | Project-local and pinned | Project-local cosign 3.1.2 + Syft 1.50.0; real local-sign/verify and SPDX plus synthetic provisioning tests passed | Available through `dev/bootstrap` and `dev/run` | Closed by `ENV-011`/`F0-038`; use `dev/check-supply-chain-toolchains` |

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
| `ENV-006` | Provision `protoc` and adopted Buf | `F0-019`, `ENV-001` | `DONE` | Codex | `E-025`: reproducible lint/generate/round trip |
| `ENV-007` | Pin Gradle Wrapper 9.6.1 and verify checksum | `F0-002` | `DONE` | Codex | `E-007`: Wrapper 9.6.1 and checksums verified |
| `ENV-008` | Verify Docker and golden image by digest | `F0-002`, `ENV-002` | `DONE` | Codex | `E-015`: immutable index/platform identity, strict cgroups, and container smoke |
| `ENV-009` | Pin Rust 1.93.0 or an approved version | `ENV-001` | `DONE` | Codex | `E-016`: exact Rustup override, locked manifest, doctor match, and offline Cargo check |
| `ENV-010` | Provision `shellcheck` and `actionlint` | `ENV-001` | `DONE` | Codex | `E-017`: exact archives, isolated execution, and passing integrated lint smoke |
| `ENV-011` | Provision `cosign`/`syft` for the supply chain | `ENV-001`, `F0-038` | `DONE` | Codex | `E-030`: exact tools, isolated sign/verify, and SPDX fixture |
| `ENV-012` | Implement `dev/bootstrap`, `dev/run`, and documented cleanup | `ENV-001..011` | `DONE` | Codex | `E-036`: two-run bootstrap, guarded uninstall, and explicit cache/state purge |

Do not mark `ENV-003`, `ENV-006`, `ENV-010`, or `ENV-011` complete because a “similar” tool exists on the host: it must match the lock and pass its smoke test.

### 5.2 Foundation

| ID | Deliverable | Depends on | State | Owner | Expected evidence |
|---|---|---|---|---|---|
| `F0-001` | Create workspace/repository, modules, and build conventions | — | `DONE` | Codex | `E-006`: initial commit + [README](./README.md) + passing `./dev/check-layout` |
| `F0-002` | Pin golden lane and ADR `0001-golden-lane` | `F0-001` | `DONE` | Codex | `E-007`: passing ADR, contract, Wrapper, fixture, and host/container smoke tests |
| `F0-003` | Define ownership by workstream and review boundaries | `F0-001` | `DONE` | Codex | `E-037`: validated CODEOWNERS, accountable owners, and review map |
| `F0-004` | Configure base CI for Go, Java 17, and optional Rust | `F0-001` | `DONE` | Codex | `E-038`: passing immutable read-only base workflow |
| `F0-005` | Add generated-code policy and drift detection | `F0-004` | `DONE` | Codex | `E-039`: source-first inventory, descriptor generation, and mandatory CI drift |

### 5.3 Normative contracts

| ID | Deliverable | Related decision | State | Owner | Expected evidence |
|---|---|---|---|---|---|
| `F0-010` | Create `contracts/`, `specs/`, `benchmarks/`, and `adr/` structure | `CONTRACTS-001` | `DONE` | Codex | `E-018`: checked RFC §29.2 namespaces and artifact indexes |
| `F0-011` | `build-session.v1.schema.json` | `OBS-002`, `METRICS-001` | `DONE` | Codex | `E-019`: Draft 2020-12 schema + 4 valid/7 invalid fixtures |
| `F0-012` | `experiment-result.v1` and `action-record.v1` | `OBS-002`, `MEASURE-001` | `DONE` | Codex | `E-040`: Draft 2020-12 schemas + individual and linked lifecycle fixtures |
| `F0-013` | Evidence, policy, and resource-profile schemas | `TASK-001`, `BANDIT-001` | `DONE` | Codex | `E-041`: strict schemas + four-arm golden catalog |
| `F0-014` | Attempt, validation-request, and `CommitDecision` schemas | `CACHE-008`, `CI-ORCH-001` | `DONE` | Codex | `E-042`: strict schemas + CAS/coverage state vectors |
| `F0-015` | Test grant/result schemas | `TESTOPT-API-001` | `DONE` | Codex | `E-043`: signed strict schemas + policy/request negatives |
| `F0-016` | `PatchBundle v1` schema | `PATCH-BUNDLE-001` | `DONE` | Codex | `E-044`: strict envelope + exact blobs/digest mutation vectors |
| `F0-017` | OpenAPI BuildOpt control/cache | `CONTRACTS-001`, `CACHE-008` | `DONE` | Codex | `E-045`: validated OpenAPI 3.1 + nine-operation mock |
| `F0-018` | OpenAPI Test Optimization | `TESTOPT-API-001` | `DONE` | Codex | `E-046`: validated OpenAPI 3.1 + four-operation mock |
| `F0-019` | Protobuf local task-event channel | `GRADLE-CORR-001` | `DONE` | Codex | `E-024`: normative IDL, ADR, descriptor parity, and Go/Java Unix round trips |
| `F0-020` | JCS, SHA-256, timestamp, and signature vectors | `CONTRACTS-001` | `DONE` | Codex | `E-047`: identical Go/Java JCS, digest, time, and Ed25519 vectors |
| `F0-021` | Error, deadline, retry, and idempotency contract | `CONTRACTS-001` | `DONE` | Codex | `E-048`: common catalog + nine fault cases + all-OpenAPI audit |
| `F0-022` | N/N-1 compatibility and generated Go/Java clients | `F0-011..021` | `DONE` | Codex | `E-049`: drift-checked clients + shared nine-case compatibility suite |
| `F0-023` | Task/action/attempt state machines | `STATE-001`, `CI-ORCH-001` | `TODO` | — | Transition vectors + recovery |
| `F0-024` | Normative `METRICS-001` catalog | `METRICS-001`, `MEASURE-001` | `DONE` | Codex | `E-034`: 35-definition catalog, validator, and policy negatives |

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
| `F0-038` | Packaging, checksums, signatures, SBOM, and provenance | `DEPLOY-001` | `DONE` | Codex | `E-030`: reproducible signed release bundle and fail-closed verifier |
| `F0-039` | Base runbooks: bypass, kill switch, rollback, and uninstall | `OPS-001` | `DONE` | Codex | `E-032`: real-launcher bypass and recorded recovery exercises |
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
| `WS-003` | Gradle plugin handshakes without optimizing | `F0-019`, `WS-001` | `DONE` | Codex | `E-026`: Real Wrapper on golden lane |
| `WS-004` | Loopback gateway with authenticated rendezvous | `F0-019`, `WS-001` | `DONE` | Codex | `E-027`: restart/concurrency fixture |
| `WS-005` | `buildopt-server` receives a session | `F0-011`, `WS-004` | `DONE` | Codex | `E-028`: ingest integration test |
| `WS-006` | Export valid `BUILD_SESSION v1` JSON | `WS-003`, `WS-005` | `DONE` | Codex | `E-029`: real producer schema validation |
| `WS-007` | GitHub Action pinned by SHA/checksum | `WS-001..006`, `F0-038` | `DONE` | Codex | `E-031`: immutable Action/archive pins and passing hosted fixture |
| `WS-008` | Bypass, failure/cancel, and attempt/lease cleanup | `WS-001..007`, `F0-039` | `DONE` | Codex | `E-033`: host and strict-container fault suite |
| `WS-009` | Measure overhead from the neutral envelope | `WS-001..008`, `F0-024` | `DONE` | Codex | `E-035`: strict four-pair baseline-vs-wrapper report |

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
| `F0-G03` | Complete, passing walking skeleton on the golden lane | `DONE` | `E-020`, `E-021`, `E-026..035` |
| `F0-G04` | Executable conformance/fixtures for the next module | `TODO` | — |
| `F0-G05` | Bypass, kill switch, and `UNAVAILABLE` exist before optimization | `DONE` | `E-023`, `E-032` |
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
| Ownership boundaries | Path routing, accountable workstreams, cross-stack review, and authority limits | `dev/check-ownership` validated 16 CODEOWNERS paths, all 11 tracker workstreams, one verified repository principal, producer/consumer review, Test Optimization ownership, and the absence of an independent-approval claim | 2026-07-29 | `E-037` |
| Base CI | Read-only push/PR validation for Go, Java 17 compatibility, and optional Rust | Immutable checkout/setup Actions, exact Ubuntu/Java/Rust identities, locked project tools, race-enabled Go tests, vet, nested module checks, Java 17 bytecode/runtime loading, Rust manifest/smoke, ShellCheck, and actionlint passed locally and in hosted run [30481886001](https://github.com/tonyredondo/buildopt/actions/runs/30481886001) | 2026-07-29 | `E-038` |
| Generated artifacts | Source-first inventory, locked generation, reviewable output, and stale-code rejection | The local-events descriptor regenerated byte-for-byte with protoc 35.1, a temporary valid source mutation produced divergent output, unsafe/unknown generator requests failed, conformance passed, and hosted base-CI run [30482598040](https://github.com/tonyredondo/buildopt/actions/runs/30482598040) enforced the gate on a clean checkout | 2026-07-29 | `E-039` |
| Experiment/action lifecycles | Immutable aggregate versions, action transition preconditions, and exact promotion/invalidation linkage | Draft 2020-12 accepted 4 aggregate and 6 transition positives, rejected 4 negatives for each schema, and classified 3 valid/3 invalid linked vectors; local semantic checks and hosted base-CI run [30484082786](https://github.com/tonyredondo/buildopt/actions/runs/30484082786) passed on the exact implementation SHA | 2026-07-29 | `E-040` |
| Evidence/policy/resource contracts | Task authority gates, fail-closed invocation policy, and finite prevalidated resource arms | Draft 2020-12 accepted the bound golden records, rejected incomplete trace, active kill switch, and failed-memory eligibility, and enforced exact policy/evidence/profile linkage plus a four-arm catalog; local core and hosted base-CI run [30485226736](https://github.com/tonyredondo/buildopt/actions/runs/30485226736) passed on the exact implementation SHA | 2026-07-29 | `E-041` |
| Attempt/commit contracts | Durable CAS lifecycle, isolated validation requests, and exact atomic publication authority | Draft 2020-12 accepted one positive and rejected one negative per schema, while 2 valid/3 invalid linked vectors enforced state order, one owner, positive validation, task boundary, and complete pending-object coverage; local core and hosted base-CI run [30485995394](https://github.com/tonyredondo/buildopt/actions/runs/30485995394) passed on the exact implementation SHA | 2026-07-29 | `E-042` |
| Test Optimization signed contracts | Explicit cache authority and final validation verdicts owned by Test Optimization | Draft 2020-12 accepted 2 signed positives, rejected 5 structural negatives, and rejected 2 schema-valid semantic attacks for reversed grant time and candidate-artifact rebinding; policy/request linkage, local core, and hosted base-CI run [30486510411](https://github.com/tonyredondo/buildopt/actions/runs/30486510411) passed on the exact implementation SHA | 2026-07-29 | `E-043` |
| Toolchain lock | Portable versions, platforms, providers, immutable URLs, SHA-256 values, and tracker references | `dev/check-toolchains-lock` passed for ten artifacts on `linux-amd64`; update policy recorded | 2026-07-29 | `E-010` |
| Toolchain lifecycle | Lock-owned bootstrap, local execution, uninstall, downloads, and build state | A temporary marked tools root passed two-run bootstrap, idempotent uninstall, cached reinstall, active-lock refusal, all-tool removal, exact download purge, explicit state purge, and unrelated/provider-managed path preservation | 2026-07-29 | `E-036` |
| Host inventory | Platform/resources, active Java/Go/Rust/Protobuf/base-tool paths and versions, Docker, cgroups, namespaces, and available space | `dev/doctor --json` passed on the 12-CPU workstation and reported active-path drift without treating deferred provisioning as success; deterministic fixtures passed codes `0/1/64/70` and preserved the working tree | 2026-07-29 | `E-011` |
| Go toolchain | Exact compiler, module baseline, local-only selection and caches, offline module state, and deterministic smoke build | Official Go 1.26.5 archive matched the lock; real and synthetic isolation checks passed; doctor reported the project-local binary as `MATCH`; repeated smoke binaries matched | 2026-07-29 | `E-014` |
| Rust toolchain | Exact optional compiler/Cargo pair, repository override, locked channel manifest, isolated offline state, and Cargo smoke | Rust/Cargo 1.93.0 and the Linux AMD64 host matched; official manifest SHA-256, doctor `MATCH`, temporary-state Cargo check, global-default isolation, and deterministic negative fixtures passed | 2026-07-29 | `E-016` |
| Lint toolchains | Exact ShellCheck/actionlint artifacts, isolated execution, repository scripts, and workflow parsing | Official ShellCheck 0.11.0 and actionlint 1.7.12 archives matched the lock; real and synthetic provisioning, doctor matching, all executable `dev/` scripts, and the integrated workflow smoke passed | 2026-07-29 | `E-017` |
| Release supply chain | Exact Cosign/Syft tooling, deterministic Linux AMD64 payload, SPDX, SLSA provenance, checksums, local signatures, and fail-closed verification | Locked Cosign 3.1.2 and Syft 1.50.0 passed real and synthetic provisioning; repeated clean builds produced byte-identical archive, SPDX, provenance, release manifest, and checksum manifest, while both ECDSA bundles signed the same digest without a transparency-log entry or timestamp; archive/signature tampering, extra files, wrong trust roots, dirty sources, permissive keys, and invalid invocations were rejected | 2026-07-29 | `E-030` |
| GitHub Action fixture | Linux x64 composite setup Action, full commit pins, version/HTTPS/SHA-256 archive identity, safe extraction, outputs/PATH, and neutral wrapper behavior | Local offline fixtures proved checksum-before-extract, exact entries/types/modes, atomic reuse, output consistency, argv/exit preservation, and negative cases; actionlint accepted the manual read-only workflow; hosted run [30471969171](https://github.com/tonyredondo/buildopt/actions/runs/30471969171) passed all steps on `ubuntu-24.04` at `d49c56c9f4bdf1d98ee0450f1ed06008779510c9` | 2026-07-29 | `E-031` |
| Base recovery | Control-plane-independent bypass, CI kill switch, immutable rollback, uninstall, state preservation/purge, and partial patch recovery | The real launcher consumed `BUILDOPT_BYPASS=1` before invalid server configuration or local rendezvous setup, preserved argv/stdio/cwd/exit and process-group cancellation, restored the normal path when cleared, rejected a bad release digest before running the pinned known-good fixture, and exercised guarded preserve/purge uninstalls without changing the working tree; full host and strict 4-CPU/16-GiB container regression passed | 2026-07-29 | `E-032` |
| Normative package layout | RFC §29.2 namespaces, indexes, planned artifact parents, and non-empty materialized artifacts | `dev/check-normative-layout` passed for 15 namespaces and 27 planned artifacts, including the versioned metrics namespace; ShellCheck and repository layout passed | 2026-07-29 | `E-018`, `E-034` |
| Contract schemas | `BUILD_SESSION v1` Draft 2020-12; later JSON Schemas and OpenAPI remain pending | Pinned Go validator compiled the schema with format assertions; 4 valid fixtures and 2 real producer exports passed, while 7 focused invalid fixtures were rejected for their intended diagnostics | 2026-07-29 | `E-019`, `E-029` |
| Metric catalog | `build-impact-v1` definitions and `beta-measurement-v1` promotion policy | Dependency-free strict validation accepted all 35 required core metrics with complete governance fields, bounded dimensions, explicit availability/methods, and fixed saved/delta/overhead signs; semantic, missing/duplicate, null, dimension, unit, sign, sample, threshold, and correctness-policy drift fixtures were rejected on host and the strict golden container | 2026-07-29 | `E-034` |
| Walking-skeleton overhead | External monotonic native-versus-complete-wrapper timing with required-output parity and no optimization | Four balanced alternating pairs ran in the pinned 4-CPU/16-GiB container; the first pair was retained, every wrapper arm completed one authenticated session/export, all deliverable digests matched, signed negative observations remained raw, and the report stayed explicitly non-promotional; first `+236.020 ms`, nearest-rank p50 `-798.384 ms`, mean `-35.033 ms`, and p95 `+1338.013 ms` are descriptive only | 2026-07-29 | `E-035` |
| Protobuf toolchains | Exact project-local compiler/linter, immutable artifacts, normalized layouts, and isolated execution | Official protoc 35.1 ZIP and Buf 1.72.0 binary matched the lock; real and synthetic provisioning, idempotency, isolation, failure fixtures, doctor matching, lint, descriptor parity, and round trips passed | 2026-07-29 | `E-025` |
| Local task-event contract | Protobuf v1 payload, varint framing, exact/unattributed correlation, and attempt-wide abort | Exact protoc/Buf descriptors matched; Java 17 and Go peers passed both real Unix-socket directions, semantic negatives, frame bounds, and the Go race detector | 2026-07-29 | `E-024` |
| Golden vectors | Go ↔ Java canonicalization/signatures | Not run | — | — |
| Go unit/integration | Launcher CLI, neutral local gateway, server ingest, cancellation, metrics, neutral envelope, and BUILD_SESSION producer/exporter | Real binaries preserved process context and child statuses; gateway tests proved authenticated restart/concurrency; server tests proved strict idempotent ingest plus cancellation classification; catalog and envelope tests proved governance, paired order/output reconciliation, first/negative observation retention, exact input binding, non-qualifying host labeling, and report drift rejection; producer tests proved observed/unavailable/cancelled semantics, the shared metric version, and private atomic immutable export | 2026-07-29 | `E-021`, `E-027`, `E-028`, `E-029`, `E-033`, `E-034`, `E-035` |
| Local gateway rendezvous | Loopback binding, local Basic credential, connection generation, event-token preface, and peer identity | Four concurrent gateways used distinct endpoints, credentials, and generations across 64 raced requests; cross-slot credentials returned `401`; restart preserved identity; real CLI and Gradle plugin authenticated both local hops and removed them after exit | 2026-07-29 | `E-027` |
| Server session ingest | Provisional internal record, loopback HTTP, Bearer auth, idempotency, fail-open delivery, and no child credential exposure | Real `buildopt` and `buildopt-server` binaries delivered success, exit `37`, and handled cancellation exit `42` through the active gateway; identical concurrent records deduplicated, conflicting content returned `409`, unavailable delivery preserved the child, and absent/bypass configuration made no contact | 2026-07-29 | `E-028`, `E-033` |
| BUILD_SESSION export | Complete local passthrough producer, strict pre-outcome context, immutable atomic JSON, schema conformance, and explicit unavailable metrics | Real successful and failed Gradle 9.6.1 sessions produced distinct private mode-`0600` documents; both passed the pinned Draft 2020-12 schema with exact envelope and approximated launcher-observed process facts, partial failure timing, tokenized identities, the normative `build-impact-v1` catalog version, and no ingest credential or temporary residue | 2026-07-29 | `E-029`, `E-034` |
| Gradle correlation fixture | Gradle 9.6.1/JDK 21 golden-lane slice; Tier 1/TestKit expansion remains pending | Independent multi-project fixture produced one shared native key from two parallel equivalent tasks, then restored both from an isolated local cache while reusing Configuration Cache on host and strict container | 2026-07-29 | `E-022` |
| Gradle correlation spike | Gradle 9.6.1 and 8.14.3; parallel tasks, Worker API, child JVM, remote cache, failure/cancellation, and Configuration Cache | Task-owned remote stores correlated exactly by operation ancestry and matched HTTP PUTs; cold Kotlin DSL/accessor stores had no task ancestor, selected `UNAVAILABLE`, and exercised the whole-attempt fallback; complete host matrix and strict 9.6.1 container gate passed | 2026-07-29 | `E-023` |
| Gradle plugin handshake | Packaged `dev.buildopt` plugin, authenticated v1 local channel, baseline neutrality, and Configuration Cache | Two real Wrapper invocations each authenticated gateway readiness plus the event preface before one accepted `ProducerHello`; the second reused Configuration Cache with the task up-to-date; outputs matched baseline; missing rendezvous and intentional failure preserved results; host and strict-container gates passed | 2026-07-29 | `E-026`, `E-027` |
| Gradle TestKit | Plugin/adapters | Not run | — | — |
| Real Gradle Wrapper | Wrapper 9.6.1, Kotlin DSL fixtures, Configuration Cache, and BUILD_SESSION export | Strict gate passed on the nominal 4 vCPU/16 GiB host and a container with 4 CPU/16 GiB cgroups; bytecode major 61; correlation and neutral plugin-handshake fixtures reused Configuration Cache; real success/failure exports validated; negative 2-CPU fixture rejected | 2026-07-29 | `E-008`, `E-026`, `E-029` |
| Golden container runtime | Docker daemon, immutable image index/platform identity, local image provenance, exact JDK patch, cgroup limits, and read-only Go-binary injection | Docker 29.6.2 resolved the pinned index to the unique Linux AMD64 digest; strict 4-CPU/16-GiB build passed; locked static launcher/server/signal-helper/schema-validator/metrics-validator/neutral-envelope binaries exercised authenticated rendezvous, ingest, cancellation cleanup, schema-valid export, catalog validation, and the four-pair overhead report inside the JDK-only image; deterministic negative fixtures passed | 2026-07-29 | `E-015`, `E-026`, `E-027`, `E-028`, `E-029`, `E-033`, `E-034`, `E-035` |
| JVM release compatibility | Gradle plugin and JVM agent | Locked JDK 21 compilation with `--release 17`; every packaged class verified as major 61; reproducible JARs; agent loaded on Java 17 and 21; complete build passed in the pinned golden image | 2026-07-29 | `E-013` |
| Cache conformance | HttpBuildCache + internal control | Not run | — | — |
| Fault injection | Walking-skeleton bypass/failure/cancellation and invocation cleanup; cache/orchestration faults remain pending | Ordinary failure and handled `SIGTERM` retained exits `37`/`42` and distinct `BUILD_FAILURE`/`CANCELLED` records; bypass made no contact; the child tree, plugin attempt directory/socket, and gateway were absent after cleanup on the host and strict golden container; the cache-disabled slice created no lease | 2026-07-29 | `E-033` |
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
| `E-025` | 2026-07-29 | `ENV-006` | Generalized [`dev/bootstrap`](./dev/bootstrap) and [`dev/run`](./dev/run), integrated [`dev/check-protobuf-toolchains`](./dev/check-protobuf-toolchains), and deterministic [`dev/test-protobuf-toolchains`](./dev/test-protobuf-toolchains); official protoc 35.1 ZIP bytes matched SHA-256 `6930ebf62bd4ea607b98fff052596c6ee564b9835b4ce172c75a3f53ae9d91b7`, and official Buf 1.72.0 binary bytes matched `8720830e26a733da55bb89bcd3cb44849c0965fc0c44fb5d691cccdc64dca5af`; atomic and idempotent provisioning normalized `bin/protoc` plus standard includes and `bin/buf`, exact tools remained isolated from global protoc 3.21.12 and missing global Buf, nested project-local execution made doctor report both `MATCH`, and the real `F0-019` lint/descriptor/Go-Java Unix round-trip suite retained descriptor SHA-256 `a8168a4b91824a6f195145c4fba25f9d15fe17c5faa5cd5757bc42269acf1f9e`; negative fixtures covered checksum, standard-include layout, version and manifest drift, missing tools, usage, and child failure; shared JDK, Go, lint, and doctor fixtures, full Go race/vet, CLI, schema, layouts, toolchain lock, ShellCheck/actionlint, JVM release, golden static, and the host golden-lane build all passed | `DONE`: Protobuf lint and compilation no longer depend on workstation-global tools; generated clients remain deferred to `F0-022`, and `WS-003` is next |
| `E-026` | 2026-07-29 | `WS-003` | Packaged `dev.buildopt` plugin marker and [`BuildOptHandshakeService`](./jvm/gradle-plugin/src/main/java/dev/buildopt/gradle/BuildOptHandshakeService.java), private invocation-scoped Unix receiver in [`internal/launcher`](./internal/launcher/plugin_handshake.go), independent [`fixtures/gradle-handshake`](./fixtures/gradle-handshake/README.md), and executable [`dev/check-gradle-plugin-handshake`](./dev/check-gradle-plugin-handshake); the launcher replaced inherited rendezvous values, validated the exact v1 Gradle `ProducerHello`, returned a matching acknowledgement, and removed its socket; two real Gradle 9.6.1 Wrapper invocations each handshook once, including Configuration Cache reuse with the task up-to-date; direct and wrapped output hashes matched, a missing receiver remained fail-open, and an intentional Gradle failure retained exit `1`; full Go race/vet, CLI, Protobuf, schema, layout, lock, Bash, ShellCheck/actionlint, JVM release, host golden-lane, and strict 4-CPU/16-GiB container gates passed | `DONE`: the plugin and launcher now prove one neutral handshake per invocation without gateway authentication, retained task-event streaming, or optimization; `WS-004` is next |
| `E-027` | 2026-07-29 | `WS-004` | Neutral [`localGateway`](./internal/launcher/local_gateway.go), authenticated event preface in [`plugin_handshake.go`](./internal/launcher/plugin_handshake.go), Java [`BuildOptRendezvousContext`](./jvm/gradle-plugin/src/main/java/dev/buildopt/gradle/BuildOptRendezvousContext.java), race-enabled [`dev/check-local-gateway`](./dev/check-local-gateway), and the extended real Wrapper fixture; the gateway bound only an ephemeral `127.0.0.1` endpoint, exposed authenticated readiness with a fresh local Basic credential and `gatewayConnectionGeneration`, and left cache data routes disabled; event connections required a separate 256-bit token, constant-time comparison, and matching Unix peer UID before Protobuf; restart retained endpoint/credential/generation, four concurrent slots kept distinct identities across 64 raced requests, and cross-slot credentials returned `401`; the real CLI replaced hostile parent values, observed `204/401`, and proved socket/gateway cleanup; two Gradle 9.6.1 invocations authenticated both hops before `ProducerHello`, including an up-to-date Configuration Cache reuse, while missing rendezvous and intentional failure retained baseline behavior; full Go race/vet, CLI, Protobuf, schema, layout, lock, Bash, ShellCheck/actionlint, JVM release, host golden-lane, deterministic container tests, and strict 4-CPU/16-GiB container gates passed | `DONE`: the walking skeleton now has an authenticated local rendezvous without cache payload routing, upstream credentials, retained event streaming, or optimization; `WS-005` is unblocked and next |
| `E-028` | 2026-07-29 | `WS-005` | Real [`buildopt-server`](./cmd/buildopt-server/main.go), provisional internal [`sessioningest`](./internal/sessioningest/record.go) boundary, launcher-owned [delivery](./internal/launcher/session_ingest.go), and executable [`dev/check-session-ingest`](./dev/check-session-ingest); the gateway emitted one strict record bound to its connection generation after the child outcome, kept the server URL/Bearer token outside Gradle, disabled proxies/redirects, and treated delivery as fail-open; the loopback server required authentication, a matching idempotency key, strict single-value JSON, and a 64 KiB bound, returned `202` for first acceptance, `204` for identical replay, and `409` for conflicting content, while its concurrent in-memory store admitted one copy; real launcher/server binaries delivered `SUCCESS/0` and `BUILD_FAILURE/37`, an unavailable server retained exit `23`, absent configuration contacted no server, diagnostics exposed no token, and `SIGTERM` shut down cleanly; full Go race/vet, CLI, Protobuf, schema, layout, lock, Bash, ShellCheck/actionlint, JVM release, host golden-lane, deterministic container tests, and strict 4-CPU/16-GiB container gates passed | `DONE`: `buildopt-server` now receives the walking-skeleton session without claiming durable state or the normative export; `WS-006` is unblocked and next |
| `E-029` | 2026-07-29 | `WS-006` | Dependency-free normative [`buildsession`](./internal/buildsession/document.go) producer and atomic [exporter](./internal/buildsession/exporter.go), strict reserved `BUILDOPT_BUILD_SESSION_CONTEXT`, packaged [`build-session-validator`](./dev/schema-validator/cmd/build-session-validator/main.go), server `--export-dir`, and executable [`dev/check-build-session-export`](./dev/check-build-session-export); the launcher accepted at most 32 KiB of pre-outcome repository/revision/task plus tokenized HMAC context, removed it from the child, and attached it only to one authenticated Gradle producer/process interval; the server derived requested-work, empty-deliverables, and versioned baseline SHA-256 values, emitted the exact neutral-envelope duration plus an explicitly approximated launcher-observed Gradle interval, marked CI timing/critical path/task/cache/resource/overhead/cost families unavailable without values, represented failed-build first-failure timing as a conservative partial approximation, and atomically published newline-terminated mode-`0600` immutable files without raw source content or the ingest token; real Gradle 9.6.1 success and intentional failure preserved exits `0/1`, produced two distinct documents accepted by the pinned Draft 2020-12 schema, and left no temporary residue; full Go race/vet, schema module tests/vet, CLI, gateway, ingest, Protobuf, layouts, lock, Bash, ShellCheck/actionlint, JVM release, deterministic container tests, host golden-lane, and strict 4-CPU/16-GiB container gates passed | `DONE`: the walking skeleton now reaches valid complete local `BUILD_SESSION v1` JSON; durable state, partial recovery, JSONL, retries/spooling, CI metadata, and optimization remain deferred, while `F0-038` is next to unblock `WS-007` |
| `E-030` | 2026-07-29 | `F0-038`, `ENV-011` | Normative [`release-bundle-v1`](./specs/release-bundle-v1.md), executable [`dev/package-release`](./dev/package-release) and fail-closed [`dev/verify-release`](./dev/verify-release), integrated [`dev/check-release-package`](./dev/check-release-package), and exact supply-chain provisioning through [`dev/bootstrap`](./dev/bootstrap)/[`dev/run`](./dev/run); locked Cosign 3.1.2 and Syft 1.50.0 matched their immutable artifacts and passed real local-key signature verification plus SPDX generation, while synthetic fixtures covered atomic idempotent installation, isolation, checksum/version/manifest drift, usage, and child failures; two clean-source builds packaged static Linux AMD64 `buildopt`/`buildopt-server` binaries and versioned Java-17-compatible plugin/agent JARs into the exact safe-mode layout, then produced byte-identical deterministic TAR/Gzip, normalized SPDX 2.3, SLSA provenance v1, release manifest, and canonical SHA-256 manifest; separate ECDSA bundles bound the same checksum digest under the pinned public key with no transparency-log entry or timestamp, and the private key never entered the bundle; modified archive/signature, extra entries, wrong key, dirty source, existing output, permissive key, absent password, and invalid usage/version all failed; full Go race/vet, schema, CLI, gateway, ingest/export, Protobuf, toolchain, layout, Bash/ShellCheck/actionlint, JVM release, Gradle 9.6.1/8.14.3 correlation, host golden lane, and digest-pinned 4-CPU/16-GiB container gates passed | `DONE`: release consumers can verify an exact source/tool/artifact chain without executing payloads; public CI publication, install/upgrade/uninstall and revocation lifecycle, managed-key signing, Rust/helper and patcher artifacts remain deferred, while `WS-007` is unblocked and next |
| `E-031` | 2026-07-29 | `WS-007` | Repository-root composite [`action.yml`](./action.yml), pinned installer/setup/init-script boundary under [`.github/actions`](./.github/actions), immutable [`fixture-lock.json`](./fixtures/github-actions/fixture-lock.json), offline positive/negative [`dev/test-github-action-install`](./dev/test-github-action-install), integrated [`dev/check-github-action`](./dev/check-github-action), and manual hosted [`ws-007-fixture.yml`](./.github/workflows/ws-007-fixture.yml); the consumer pins BuildOpt commit `3fe068790878420a2a9e1d84b6ae5fc83f5752c3` and `actions/checkout` commit `3d3c42e5aac5ba805825da76410c181273ba90b1`, while version `0.0.0-ws007`, its commit-bound HTTPS URL, and archive SHA-256 `d15319d6a9c08bb02587a68a93ea49803678a57d137fb812676f7f614682dcc9` independently bind the synthetic Linux AMD64 release fixture; the installer required HTTPS for redirects, verified the complete digest before extraction, rejected extra/linked/wrong-mode content, published atomically under runner temp, revalidated matching reused files, rejected checksum/platform/input/mutated-install failures, and exposed consistent launcher/server/plugin/agent/init-script outputs plus `PATH`; literal empty, whitespace, wildcard-like, and variable-like argv and child exit `37` survived both local execution and hosted run [30471969171](https://github.com/tonyredondo/buildopt/actions/runs/30471969171), whose read-only `ubuntu-24.04` job passed at source SHA `d49c56c9f4bdf1d98ee0450f1ed06008779510c9`; full Go race/vet, schema, CLI, gateway, ingest/export, Protobuf, toolchain fixtures, release packaging, ShellCheck/actionlint, JVM release, Gradle 9.6.1/8.14.3 correlation, host golden lane, and strict 4-CPU/16-GiB container gates also passed | `DONE`: the first GitHub consumer path is immutable and executable without secrets or write permission; authoritative CI, protected validation orchestration, fork/token policy, real release publication, and full install/upgrade/uninstall/revocation remain with `F0-004`, `CI-ORCH-001`, and `DEPLOY-001`, while `F0-039` is next to unblock `WS-008` |
| `E-032` | 2026-07-29 | `F0-039`, `F0-G05` | Early launcher-owned `BUILDOPT_BYPASS=1` in [`internal/launcher`](./internal/launcher/run.go), real-binary argv/environment and [process-tree signal tests](./cmd/buildopt/main_signal_linux_test.go), operator [`base-recovery`](./runbooks/base-recovery.md) procedures, and executable [`dev/check-base-runbooks`](./dev/check-base-runbooks); bypass consumed itself plus every launcher-only credential/rendezvous value, skipped server/export parsing and plugin/gateway startup, preserved stdin/stdout/stderr, cwd, literal argv, child exit `38`, and delayed process-group cancellation; the CI-variable kill switch restored the normal rendezvous when cleared; a bad candidate checksum failed before extraction, the fully pinned known-good Action fixture installed and ran, and guarded uninstall exercises separately preserved and explicitly purged export state before the baseline command ran directly; the runbook also records immutable tuple rollback, step/init-script removal, and non-destructive partial C4 branch/PR recovery; full Go race/vet, schema, CLI, gateway, ingest/export, Protobuf, layout/lock, ShellCheck/actionlint, JVM release, release/Action checks, real Gradle handshake and correlation, host golden lane, and digest-pinned 4-CPU/16-GiB container gates passed | `DONE`: local bypass, the CI-owned base kill switch, and the existing tested `UNAVAILABLE` fallback now precede every optimization, closing `F0-G05`; online revocation, already-running invocation cancellation, durable attempt/lease cleanup, authoritative deployment lifecycle, and full fault coverage remain with `OPS-001/A1`, `WS-008`, and `DEPLOY-001`, while `WS-008` is unblocked and next |
| `E-033` | 2026-07-29 | `WS-008` | Normative [`walking-skeleton-faults-v1`](./specs/walking-skeleton-faults-v1.md), `CANCELLED` ingest/producer support in [`internal/sessioningest`](./internal/sessioningest/record.go) and [`internal/buildsession`](./internal/buildsession/document.go), signal-aware launcher classification plus real [process-tree integration](./cmd/buildopt/main_signal_linux_test.go), and executable [`dev/check-walking-skeleton-faults`](./dev/check-walking-skeleton-faults); an ordinary child exit `37` remained `BUILD_FAILURE/37`, while `SIGTERM` reached a leader and descendant, waited for explicit cleanup, preserved cleanup exit `42`, and emitted `CANCELLED/42`; the live plugin attempt socket and gateway were observed before cancellation, then the complete child tree, private attempt directory/socket, and endpoint were absent afterward; the optimization-off gateway exposed no cache data route and created no cache lease, and `BUILDOPT_BYPASS=1` returned exit `38` without server contact; cancellation diagnostics exposed no token, cancelled `BUILD_SESSION` production retained `UNKNOWN` deliverables and no invented first-failure time, the race-enabled fault packages passed three consecutive runs, and full Go tests/vet, schema, layout, locked ShellCheck/actionlint, deterministic container fixtures, the JDK-21 host golden lane, and digest-pinned strict 4-CPU/16-GiB container all passed | `DONE`: the first walking skeleton now preserves failure versus cancellation classification and leaves no active invocation resource; durable managed-cache attempt/lease recovery remains correctly owned by A0, while `F0-024` is next and will unblock `WS-009` |
| `E-034` | 2026-07-29 | `F0-024` | Machine-readable [`build-impact-v1`](./contracts/metrics/build-impact-v1.json) and indexed [metrics namespace](./contracts/metrics/README.md), dependency-free strict [`metricscatalog`](./internal/metricscatalog/catalog.go) package/CLI, and executable [`dev/check-metrics-catalog`](./dev/check-metrics-catalog); 35 required definitions cover neutral-envelope/session durations, non-overlapping components, causal and estimated effects, PRODUCT_TOTAL and ACTION_INCREMENTAL overhead, p95/p99 build/feedback/queue guardrails, economic value, cache drivers, correctness, and measurement coverage, with every RFC §22.9 owner/purpose/formula/unit/grain/population/denominator/source/boundary/dimension/null/quality/retention/caveat/sign/method field present; comparison rules retain positive saved/reduction, negative-improving deltas, signed overhead regressions, outcome isolation, fixed strata and paired requirements, while `beta-measurement-v1` fixes 7-day/100 and 14-day/200 gates, 500-ms/2% benefit, 500-ms/3% p95, 1-s/5% p99 after 1,000 per arm, zero correctness targets, 5% control, and 28-day compute limit; strict parsing and semantic negatives rejected unknown/trailing fields, version/required-set/duplicate drift, reversed signs, zero-filled unavailable values, unbounded dimensions, units, policy sample, and nonzero correctness; the `BUILD_SESSION` producer now imports the same version constant; full Go race/vet, schema, 15-namespace/27-artifact layout, locked ShellCheck/actionlint, deterministic container tests, the JDK-21 host golden lane, and digest-pinned strict 4-CPU/16-GiB container all passed | `DONE`: the first exportable metric vocabulary and beta decision thresholds are executable without creating `EXPERIMENT_RESULT` early or claiming causal data; `WS-009` is unblocked to record the first non-promotional overhead sample |
| `E-035` | 2026-07-29 | `WS-009`, `F0-G03` | Executable [`walking-skeleton-overhead-v1`](./specs/walking-skeleton-overhead-v1.md), dependency-free strict [`neutralenvelope`](./internal/neutralenvelope/report.go) package/CLI, integrated [`dev/check-walking-skeleton-overhead`](./dev/check-walking-skeleton-overhead), and immutable strict [four-pair report](./benchmarks/results/ws-009-golden-lane.json) with SHA-256 `68c3d2f5fe23757d76a14298356e15a524674f57acf7539bb24434290fb172e8`; one external monotonic envelope timed native Gradle and the complete optimization-off launcher/plugin/gateway/server/export wrapper with the same Wrapper 9.6.1, fixture, shared cache context, removed output, and required deliverable; two pairs ran each order, the first pair remained in the sample, every arm exited `0`, each wrapper arm completed exactly one authenticated handshake, ingest, and export, and all eight deliverable SHA-256 values matched; strict validation bound the runner/catalog/envelope/launcher/server/plugin bytes, rejected unknown/trailing/tampered fields, invalid pair order, mismatched output, misleading runner qualification, workload or metric drift, and retained signed negative differences; the qualified 4-CPU/16-GiB sample recorded first `+236.020 ms`, nearest-rank p50 `-798.384 ms`, mean `-35.033 ms`, and p95 `+1338.013 ms`, explicitly as order-sensitive descriptive evidence with `promotionGateActive=false`; race-enabled Go tests/vet, host smoke, deterministic container tests, and the complete digest-pinned strict golden lane passed | `DONE`: RFC §29.4 criterion 6 and the complete `WS-001..009` optimization-off path are now measured and passing, closing `F0-G03` without claiming causality, savings, or a promotion decision; `ENV-012` is next |
| `E-036` | 2026-07-29 | `ENV-012` | Guarded [`dev/uninstall-toolchains`](./dev/uninstall-toolchains), marked-root support in [`dev/bootstrap`](./dev/bootstrap), documented preserve/purge choices, and isolated [`dev/test-toolchain-lifecycle`](./dev/test-toolchain-lifecycle); two bootstrap invocations downloaded once, a default uninstall removed only the selected lock-derived installation, a second uninstall succeeded without mutation, reinstall reused the verified cache, and an active bootstrap lock rejected cleanup before deletion; all-tool cleanup ignored provider-managed Rust and unrelated files, exact downloads disappeared only with `--purge-downloads`, project-local Go/Gradle state disappeared only with `--all --purge-state`, and unsafe/unmarked roots plus invalid combinations failed closed; Bash syntax, locked ShellCheck/actionlint, layout, lifecycle fixtures, and working-tree preservation passed | `DONE`: every directly provisioned development tool now has one bounded, idempotent uninstall path without global replacement or implicit data loss; `F0-003` is next |
| `E-037` | 2026-07-29 | `F0-003` | GitHub-native [CODEOWNERS](./.github/CODEOWNERS), explicit [ownership and review map](./.github/OWNERS.md), contribution boundary, and executable [`dev/check-ownership`](./dev/check-ownership); the verified repository owner is accountable for all 11 current workstreams without fabricating teams, while path-specific routing preserves contracts, Go, Gradle, agent, Rust, CI, patcher, fixtures, evidence, and operations boundaries; review rules require producer/consumer analysis, least-privilege and rollback review for delivery, tracker gates beyond passing tests, external Test Optimization authority, source-bound generated changes, and no unsupported independent-approval claim; locked ShellCheck/actionlint, ownership validation, layout, and working-tree checks passed | `DONE`: ownership is explicit enough to route parallel work without conflating repository accountability, product authority, or independent approval; `F0-004` is next |
| `E-038` | 2026-07-29 | `F0-004` | Authoritative read-only [`base-ci.yml`](./.github/workflows/base-ci.yml), immutable [`base-ci.lock.json`](./.github/base-ci.lock.json), executable [`dev/check-base-ci`](./dev/check-base-ci), and Java 17 runtime extension in [`dev/check-jvm-release`](./dev/check-jvm-release); push-to-main, pull-request, and manual triggers share `contents: read`, cancellation concurrency, exact `ubuntu-24.04`, full-commit `actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1` v7.0.1 and `actions/setup-java@03ad4de0992f5dab5e18fcb136590ce7c4a0ac95` v5.6.0, Temurin 17.0.19+10, and the optional Rust 1.93.0 target; the core lane provisioned locked JDK 21/Go/ShellCheck/actionlint, verified the Wrapper checksum, ran layout/ownership/lock/workflow checks, all root and nested-schema Go tests plus race/vet, compiled major-61 plugin/agent bytes, and loaded the agent on Java 21 and 17, while the separate Rust lane verified deterministic negatives, the official locked channel manifest, exact compiler/Cargo, and offline smoke; local lanes passed and hosted push run [30481886001](https://github.com/tonyredondo/buildopt/actions/runs/30481886001) passed both jobs on source SHA `f0b57253827b8f542170c87bb40afa093cce4f07` | `DONE`: reproducible base checks now run automatically without secrets or write permission; protected validation queues, broader isolation/lifecycle, budgets, and recovery remain with `CI-ORCH-001`, while `F0-005` is unblocked and next |
| `E-039` | 2026-07-29 | `F0-005` | Source-first [`GENERATED_CODE.md`](./GENERATED_CODE.md), strict [`generated-artifacts.json`](./dev/generated-artifacts.json) inventory, atomic locked [`generate-code`](./dev/generate-code), drift/mutation [`check-generated-code`](./dev/check-generated-code), and reviewable [`task_events.descriptor.textproto`](./contracts/proto/local-events/v1/task_events.descriptor.textproto) with SHA-256 `af2c1f9bd4f76eebfd45b3765f7c5982b1aab4b839dff24594083e326c33c928`; the descriptor records its normative `.proto`, repository command, exact protoc 35.1 checksum, and complete `FileDescriptorSet` semantics without generating the Go/Java clients owned by `F0-022`; clean regeneration matched exact bytes, while a valid temporary source mutation necessarily diverged and unknown/unsafe generator requests failed with stable codes; local generator/drift, Protobuf lint/descriptor parity, Go/Java Unix-socket conformance, base core, layout, and locked ShellCheck/actionlint passed, then hosted read-only base-CI run [30482598040](https://github.com/tonyredondo/buildopt/actions/runs/30482598040) passed both jobs on source SHA `2d9e60b13d8c895ffb9cba9318c6c179c6f6191f` | `DONE`: every checked-in generated artifact now has one declared source/tool/command/output chain and CI fails rather than rewriting stale output; `F0-012` is next |
| `E-040` | 2026-07-29 | `F0-012` | Normative [`experiment-result.v1`](./contracts/jsonschema/experiment-result.v1.schema.json) and [`action-record.v1`](./contracts/jsonschema/action-record.v1.schema.json), isolated validator entrypoints, executable [`dev/check-experiment-action-schemas`](./dev/check-experiment-action-schemas), and synthetic fixture indexes for both schemas plus their [linked lifecycle](./contracts/jsonschema/testdata/experiment-action-lifecycle.v1/README.md); `EXPERIMENT_RESULT` versions retain explicit population, assigned outcomes, exclusions, methods, intervals, effect scope, metric/policy versions, overhead/economics, beta gates, and distinct `PRELIMINARY | FINAL | INVALIDATED` shapes, while `ACTION_RECORD` retains one preconditioned append-only transition, source/policy/evidence binding, and policy/result/safety authorization without embedding causal effects or becoming an executable command; 4 valid/4 invalid aggregate fixtures, 6 schema-valid/4 invalid transition fixtures, and 3 valid/3 invalid cross-record vectors exercised policy-only shadow entry, final promotion, invalidation rollback, preliminary relabeling, inconclusive activation, and stale result versions; semantic checks additionally enforced ordered time/intervals, immediate ancestry, outcome/exclusion/economic reconciliation, state/sequence, authorization time, action membership, scope, and exact reference identity; local schema/vet/layout/lint checks passed, then hosted read-only base-CI run [30484082786](https://github.com/tonyredondo/buildopt/actions/runs/30484082786) passed Go/Java 17 in 1m49s and optional Rust in 13s on source SHA `40ddf9ed75d78afa350e71a862d27ffcbfb40725` | `DONE`: aggregate evidence cannot retroactively enter `BUILD_SESSION`, and only an exact final `PROMOTE` result can authorize activation; `F0-G02` remains open for the rest of the applicable contracts/vectors, while `F0-013` is next |
| `E-041` | 2026-07-29 | `F0-013` | Normative [`evidence-record.v1`](./contracts/jsonschema/evidence-record.v1.schema.json), [`optimization-policy.v1`](./contracts/jsonschema/optimization-policy.v1.schema.json), and [`resource-profile.v1`](./contracts/jsonschema/resource-profile.v1.schema.json), isolated validator entrypoints, executable [`dev/check-foundation-contract-schemas`](./dev/check-foundation-contract-schemas), and synthetic [foundation fixtures](./contracts/jsonschema/testdata/foundation-contracts.v1/README.md); strict Draft 2020-12 shapes bind source/task/output evidence to policy and cache contracts, separate complete trace coverage from repeatability and relocatability authority, make discrepancies suspend qualification, and disable actions/cache/build enablement under bypass or kill switch; the exact golden-runner catalog contains `STABLE_CONTROL`, `W2_H3G`, `W3_H4G`, and `W4_H6G`, varies only workers/Gradle heap outside identity/evidence, and requires startup, memory, rollback, and cgroup headroom; local schema negatives, cross-record semantics, layout, locked lint, nested test/vet, and base core passed, then hosted read-only base-CI run [30485226736](https://github.com/tonyredondo/buildopt/actions/runs/30485226736) passed Go/Java 17 in 2m27s and optional Rust in 15s on source SHA `08d7b628b1880f647ee0b80fabf6007a0cf8b8d7` | `DONE`: observation alone cannot qualify a task and arbitrary resource values cannot enter the finite catalog; `BANDIT-001` and `F0-G02` remain open for simulator/A-A/propensity/signature vectors, while `F0-014` is next |
| `E-042` | 2026-07-29 | `F0-014` | Normative [`attempt-state.v1`](./contracts/jsonschema/attempt-state.v1.schema.json), [`ci-validation-request.v1`](./contracts/jsonschema/ci-validation-request.v1.schema.json), and [`commit-decision.v1`](./contracts/jsonschema/commit-decision.v1.schema.json), isolated validator entrypoints, executable [`dev/check-attempt-commit-schemas`](./dev/check-attempt-commit-schemas), and synthetic [attempt/commit fixtures](./contracts/jsonschema/testdata/attempt-commit.v1/README.md); strict Draft 2020-12 fixes the durable `CREATED → POLICY_BOUND → GRADLE_STARTED → TASK_ACTION_STARTED → VALIDATED → COMMITTED | ABORTED` transitions, task-action replay boundary, owner/lease and budget binding, explicit candidate/control state isolation, complete policy/grant/epoch/verdict authorization, and a closed Ed25519 authentication envelope; one positive/one negative record per schema plus 2 valid/3 invalid lifecycle vectors exercised happy commit, abort before task action, skipped state, dual owner, shared L1, inconclusive verdict, and incomplete decision coverage; semantic checks enforced contiguous CAS/state versions, unique command idempotency, ordered time, immutable source/policy/owner, decision lifetime, positive validation, unique object identities, and exact pending-object tuples; local schema/vet/layout/lint and base core passed, then hosted read-only base-CI run [30485995394](https://github.com/tonyredondo/buildopt/actions/runs/30485995394) passed Go/Java 17 in 2m14s and optional Rust in 14s on source SHA `6692cbbff86e5df7b2f00f355a1ab73f5af869f1` | `DONE`: skipped, ambiguous-boundary, shared-state, inconclusive, or incompletely covering authorization cannot commit; `CACHE-008`, `CI-ORCH-001`, and `F0-G02` remain open for queue/storage/crash/reconciliation and common signature vectors, while `F0-015` is next |
| `E-043` | 2026-07-29 | `F0-015` | Normative [`test-cache-grant.v1`](./contracts/jsonschema/test-cache-grant.v1.schema.json) and [`test-validation-result.v1`](./contracts/jsonschema/test-validation-result.v1.schema.json), isolated validator entrypoints, executable [`dev/check-test-optimization-schemas`](./dev/check-test-optimization-schemas), and synthetic [Test Optimization fixtures](./contracts/jsonschema/testdata/test-optimization-contracts.v1/README.md); both strict Draft 2020-12 documents require closed Ed25519/JCS envelopes with key ID, canonical payload digest, and 64-byte unpadded signature encoding; grants bind epoch/digest, repository/trust domain, exact revision or policy range, explicit task-type/adapter selectors, namespace, read/write authority, policy, and expiration, while results bind request/action, repository, revision/source state, `FULL_RELEVANT_VALIDATION`, content-addressed artifacts, policy, evidence, disjoint final status, and expiration; 2 valid records, 5 schema negatives, and 2 schema-valid semantic negatives exercised missing signatures, wildcard/no-capability grants, contradictory `PASSED`, reversed validity, and artifact rebinding; cross-record checks linked the grant digest/expiration to the F0-013 policy and the result identity/artifacts/time window to the F0-014 request; local schema/vet/layout/lint and base core passed, then hosted read-only base-CI run [30486510411](https://github.com/tonyredondo/buildopt/actions/runs/30486510411) passed Go/Java 17 in 2m9s and optional Rust in 14s on source SHA `71fe575c4d2abe16b76720d6d622af23c566a6af` | `DONE`: absence, wildcard authority, contradictory status, invalid time, or artifact rebinding cannot become test-cache or validation authority; `F0-018`, `F0-020`, `TESTOPT-API-001`, and `F0-G02` remain open for API status/revocation/polling/compatibility and real signature vectors, while `F0-016` is next |
| `E-044` | 2026-07-29 | `F0-016` | Normative [`patch-bundle.v1`](./contracts/jsonschema/patch-bundle.v1.schema.json), isolated validator entrypoint, executable [`dev/check-patch-bundle-schema`](./dev/check-patch-bundle-schema), and synthetic [bundle vectors](./contracts/jsonschema/testdata/patch-bundle.v1/README.md) for the only two private-beta recipes; the strict Draft 2020-12 envelope binds repository/action, base revision/tree, source state, recipe/version, validation lifetime, ordered `ADD | MODIFY` operations, exact UTF-8 replacement blobs, constrained `buildopt/` draft-PR delivery, and a closed Ed25519/JCS signature shape, while rejecting deletes, executable modes, commands, hooks, fuzzy patches, unknown fields, and `ADD` preimages; semantic checks verified exact blob size/SHA-256, sorted inventory, unique contiguous operations, exact postimage linkage, validation coverage, the normative bundle digest, and signature rebinding; 2 valid bundles and 12 declarative schema/semantic mutations exercised absolute/traversal/`.git`/NUL paths, delete, mode, command, preimage, blob, digest, signature, and duplicate-operation failures; local schema, full root/nested Go tests and vet, layout, locked lint, and base core passed, then hosted read-only base-CI run [30487111150](https://github.com/tonyredondo/buildopt/actions/runs/30487111150) passed Go/Java 17 in 2m13s and optional Rust in 13s on source SHA `46b3efe3760c060fde2c148ea1cdd58cb287a671` | `DONE`: an unbound or executable bundle cannot enter the format; `PATCH-BUNDLE-001`, `F0-020`, and `F0-G02` remain open for the real signature vectors and Java parser/applier, symlink/submodule, staged-apply, Git idempotency, and branch-without-PR recovery, while `F0-017` is next |
| `E-045` | 2026-07-29 | `F0-017` | Normative OpenAPI 3.1/JSON Schema 2020-12 [`buildopt-control.v1`](./contracts/openapi/buildopt-control.v1.yaml) and [`buildopt-cache-control.v1`](./contracts/openapi/buildopt-cache-control.v1.yaml), documented [HTTP boundary](./contracts/openapi/README.md), executable [`dev/check-buildopt-openapi`](./dev/check-buildopt-openapi), and a request/response-validating in-process mock pinned to `kin-openapi` 0.145.0 inside the isolated contract module; four control operations cover pre-Gradle policy resolution, preconditioned attempt transitions, isolated validation submission, and unknown-response status reads, while five cache-control operations cover bounded pending attempts, status recovery, exact authenticated commit, abort, and cumulative revocation/L1 generations; every operation requires TLS-scoped bearer authority, exact API version, deadline/cancellation semantics, stable errors, bounded retry/backoff, and an explicit unknown-response outcome, while every mutation requires idempotency and stateful mutations require `If-Match`; the cache-control document exposes no opaque payload `PUT` and directly binds the materialized policy, attempt, validation-request, and `CommitDecision` schemas; the mock validated all nine positive request/response pairs, byte-equivalent exact policy/commit replays, and schema-valid `IDEMPOTENCY_CONFLICT` on changed payload reuse; local full OpenAPI/schema/root tests including race, both-module vet, layout, locked lint, and base core passed, then hosted read-only base-CI run [30488109970](https://github.com/tonyredondo/buildopt/actions/runs/30488109970) passed Go/Java 17 in 2m18s and optional Rust in 13s on source SHA `221ae1139e8e804d0bed9b6e43d0082f830cec4b` | `DONE`: control metadata is versioned and executable without conflating it with Gradle cache bytes; `CONTRACTS-001`, `CACHE-008`, `CI-ORCH-001`, and `F0-G02` remain open for Test Optimization, common crypto/error/compatibility clients, durable queues, and atomic cache storage/recovery, while `F0-018` is next |
| `E-046` | 2026-07-29 | `F0-018` | Normative OpenAPI 3.1/JSON Schema 2020-12 [`test-optimization.v1`](./contracts/openapi/test-optimization.v1.yaml), documented [producer/consumer boundary](./contracts/openapi/README.md), executable [`dev/check-test-optimization-openapi`](./dev/check-test-optimization-openapi), and an in-process request/response-validating producer mock; four operations resolve an exact signed grant, read cumulative signed grant status before commit, submit idempotent `FULL_RELEVANT_VALIDATION`, and poll a delayed operation within the original deadline; every operation requires TLS-scoped bearer authority, exact contract version, bounded deadline/cancellation/retry behavior, stable errors, and a fail-closed unknown-response outcome; candidate artifacts require content address, size, media type, and a customer channel or ephemeral HTTPS retrieval kind; the mock validates all positive exchanges against the API and existing signed grant/result schemas, proves byte-equivalent replay, and returns a schema-valid `IDEMPOTENCY_CONFLICT` for changed payload reuse; the focused API/schema checks, both Go modules under the race detector, layout, normative layout, locked lint, and diff hygiene passed locally | `DONE`: the HTTP boundary cannot turn a missing grant, timeout, or incomplete validation into positive authority; `TESTOPT-API-001` remains open for the full integration fixture matrix, while `F0-020` is next |
| `E-047` | 2026-07-29 | `F0-020` | Language-neutral [canonical JSON](./contracts/test-vectors/canonical-json/README.md) and [Ed25519](./contracts/test-vectors/signatures/README.md) corpora consumed by dependency-free Go and Java 17 implementations through [`dev/check-contract-crypto`](./dev/check-contract-crypto); seven JCS rows fix exact UTF-8 canonical bytes, UTF-16 member ordering, preserved Unicode, control escaping, IEEE-754 rendering, lowercase `sha256:` digests, and stable rejection of duplicate keys, malformed UTF-8, unpaired surrogates, and non-finite input; six timestamp rows accept only parseable UTC RFC 3339 with explicit seconds and uppercase `Z`; four real Ed25519 rows use synthetic public test material and cover exact payload verification, changed content, wrong key, and malformed signature without storing a private or deployment key; both consumers passed the exact same 17 rows, root Go race/vet, Java `--release 17 -Xlint:all -Werror`, layout, normative layout, locked lint, and diff hygiene locally | `DONE`: signed bytes, digests, and time parsing now have cross-language golden truth; signed-schema unknown fields remain fail-closed, while `F0-021` is next for common endpoint failure behavior |
| `E-048` | 2026-07-29 | `F0-021` | Machine-readable [HTTP failure-semantics v1](./contracts/test-vectors/http-semantics/http-semantics.v1.json), documented [contract](./contracts/test-vectors/http-semantics/README.md), and executable [`dev/check-http-semantics`](./dev/check-http-semantics); the catalog fixes all 16 stable codes currently exposed by the three OpenAPI documents, their default HTTP status/retry ceiling, the global 5-second maximum backoff, seven legal unknown-response actions, six fail-closed deadline outcomes, seven cancellation actions, and durable/none accepted-mutation states; nine interpreted fault cases cover retryable success, changed-payload key reuse, deadline during backoff, unknown stateful mutation, accepted cancellation, non-idempotent retry refusal, policy timeout preserving baseline, terminal invalid request, and capped server backoff; a full operation audit loads all 13 control-plane paths and rejects uncatalogued errors, recovery, deadlines, cancellation, mutation state, or excess backoff; focused and existing OpenAPI checks, both Go modules under the race detector and vet, layout, normative layout, locked lint, and diff hygiene passed locally | `DONE`: retries retain original identity/deadline, unknown writes require recovery, and failure cannot manufacture policy, grant, validation, or commit authority; `F0-022` is next for generated clients and compatibility |
| `E-049` | 2026-07-29 | `F0-022` | Source-first [generated-code policy](./GENERATED_CODE.md) and three-entry [`generated-artifacts`](./dev/generated-artifacts.json) inventory now cover the Protobuf descriptor plus generated [Go](./internal/generated/openapi/client_v1.go) and [Java 17](./jvm/generated-client/src/main/java/dev/buildopt/generated/BuildOptClientsV1.java) single-attempt HTTPS clients; the generator extracts all 13 operation IDs, methods, paths, mutation flags, and contract versions from the three OpenAPI documents, binds output to their aggregate digest plus the [N/N-1 corpus](./contracts/test-vectors/compatibility/n-n-minus-1.tsv), formats Go with the locked toolchain, and rejects drift from each of nine individual source bindings; both clients expose explicit bearer/version/request/idempotency/precondition/deadline inputs without implicit retries, bounded response bodies, fail-closed TLS, immutable endpoint inventories, and same-major adjacent-minor negotiation; nine shared vectors cover N/N-1 both directions, adjacent future, too-old/new minors, incompatible major preserving baseline, strict signed-command unknown fields, and preserved export extensions; real Go TLS transport/header/path escaping passed under race, Java compiled with `--release 17 -Xlint:all -Werror`, the new Gradle module built reproducibly at class major 61, Java conformance passed, and layout/normative/drift checks passed locally | `DONE`: clients cannot silently widen versions, fields, transport, or retry policy; `CONTRACTS-001` remains open for lifecycle vectors, while `F0-023` is next |

---

## 15. Tracker changelog

| Date | Change | Author |
|---|---|---|
| 2026-07-29 | Closed `F0-022`: generated drift-checked single-attempt Go/Java 17 clients from all three OpenAPI documents, added the Java module and real Go TLS proof, and passed one shared nine-case N/N-1/major/unknown-field corpus; moved `F0-023` next without hiding retry or semantic validation in generated code | Codex |
| 2026-07-29 | Closed `F0-021`: added the shared stable-error and failure-semantics catalog, nine executable deadline/retry/idempotency/unknown/cancellation cases, and an audit of all three OpenAPI documents; moved `F0-022` next without claiming durable service implementations complete | Codex |
| 2026-07-29 | Closed `F0-020`: added shared JCS, SHA-256, UTC timestamp, and real Ed25519 vectors, independent dependency-free Go/Java 17 consumers, strict malformed-input negatives, and passing race/vet/lint validation; moved `F0-021` next without claiming endpoint fault or version compatibility conformance complete | Codex |
| 2026-07-29 | Closed `F0-018`: added the fail-closed Test Optimization OpenAPI contract, signed grant status, idempotent validation polling, content-addressed artifact retrieval, a four-operation request/response mock, and passing focused plus race validation; moved `F0-020` next without claiming full integration or cryptographic conformance complete | Codex |
| 2026-07-29 | Closed `F0-017`: added validated BuildOpt control/cache OpenAPI 3.1 contracts, direct links to existing signed/state schemas, explicit auth/idempotency/precondition/deadline/cancellation/retry/error semantics, a nine-operation request/response mock, payload-boundary negatives, and passing hosted CI; moved `F0-018` next without claiming durable queue/cache transactions or the wider contract gate complete | Codex |
| 2026-07-29 | Closed `F0-016`: added the strict declarative PatchBundle envelope, exact replacement blobs and normative digest binding, two recipe vectors, 12 path/operation/content/authentication negatives, and passing hosted CI; moved `F0-017` next without claiming the Git applier or wider `PATCH-BUNDLE-001` gate complete | Codex |
| 2026-07-29 | Closed `F0-015`: added strict signed Test Optimization grant/result schemas, explicit selector/capability authority, policy/request/artifact linkage, structural and semantic negatives, and passing hosted CI; moved `F0-016` next without claiming API or cryptographic conformance complete | Codex |
| 2026-07-29 | Closed `F0-014`: added strict attempt/request/commit schemas, durable CAS and replay-boundary vectors, explicit candidate/control isolation, exact pending-object coverage, and passing hosted CI; moved `F0-015` next without claiming the queue or atomic SQLite implementation complete | Codex |
| 2026-07-29 | Closed `F0-013`: added strict evidence/policy/resource-profile schemas, independent task gates, fail-closed bypass/kill-switch behavior, the exact four-arm golden catalog, cross-record bindings, and passing hosted CI; moved `F0-014` next without closing `BANDIT-001` or the broader contract gate early | Codex |
| 2026-07-29 | Closed `F0-012`: added independent append-only aggregate and action-transition schemas, local and cross-record lifecycle invariants, exact final-promotion authorization, invalidation rollback, and passing hosted CI; moved `F0-013` next without closing the broader contract gate early | Codex |
| 2026-07-29 | Closed `F0-005`: added the source-first generated-code policy, strict artifact inventory, locked atomic generator, reviewable local-events descriptor, mutation proof, and mandatory hosted CI drift gate; moved `F0-012` next without generating `F0-022` clients early | Codex |
| 2026-07-29 | Closed `F0-004`: added immutable read-only push/PR CI, exact Java 17 compatibility, locked Go/JDK/lint checks, a separately named optional Rust lane, local parity, and a passing hosted run; unblocked `F0-005` | Codex |
| 2026-07-29 | Closed `F0-003`: assigned the verified repository owner across all workstreams, added path-specific CODEOWNERS, documented cross-boundary review and authority limits, and made the map executable; moved `F0-004` to the next item | Codex |
| 2026-07-29 | Closed `ENV-012`: added a marked tools root, lock-derived idempotent uninstall, active-bootstrap exclusion, explicit download/state purge, lifecycle fixtures, and documented preservation boundaries; moved `F0-003` to the next executable item | Codex |
| 2026-07-29 | Closed `WS-009` and `F0-G03`: added the strict neutral timing envelope, four balanced real Gradle pairs, immutable qualified report with input digests and signed differences, and host/strict-container validation; moved `ENV-012` to the next executable item | Codex |
| 2026-07-29 | Closed `F0-024`: added the 35-definition `build-impact-v1` catalog, bounded semantic validator, fixed METRICS-001 signs and MEASURE-001 beta gates, shared producer version, and host/strict-container validation; unblocked `WS-009` | Codex |
| 2026-07-29 | Closed `WS-008`: added explicit cancellation classification, deterministic failure/bypass/cancellation fixtures, complete process/socket/gateway cleanup evidence, and full host/strict-container regression; moved `F0-024` to the next executable item | Codex |
| 2026-07-29 | Closed `F0-039` and `F0-G05`: added the early consumed local bypass, CI kill-switch procedure, immutable rollback and guarded uninstall runbook, real recorded exercises, and full host/strict-container regression; unblocked `WS-008` | Codex |
| 2026-07-29 | Closed `WS-007`: added the full-SHA composite Action, commit/checksum-bound release installer, local failure fixtures, and a passing manual read-only GitHub-hosted workflow; left authoritative CI and lifecycle work with their owning gates | Codex |
| 2026-07-29 | Closed `F0-038` and `ENV-011`: added deterministic release packaging, exact Cosign/Syft provisioning, SPDX and SLSA evidence, local signatures, canonical checksums, and fail-closed verification; unblocked `WS-007` | Codex |
| 2026-07-29 | Closed `WS-006`: added strict pre-outcome context, authenticated Gradle facts, atomic private `BUILD_SESSION v1` JSON, isolated real-producer schema validation, and strict golden-lane execution | Codex |
| 2026-07-29 | Closed `WS-005`: added authenticated idempotent gateway-to-server session ingest, real-binary success/failure and fail-open fixtures, credential isolation, graceful shutdown, and strict golden-lane execution | Codex |
| 2026-07-29 | Closed `WS-004`: added the authenticated neutral loopback gateway and event preface, proved stable restart and concurrent-slot isolation, and retained Configuration Cache plus baseline behavior on the strict golden lane | Codex |
| 2026-07-29 | Closed `WS-003`: packaged the neutral Gradle plugin handshake, added the private launcher receiver and real Wrapper fixture, and proved Configuration Cache reuse, fail-open baseline behavior, and strict golden-lane execution | Codex |
| 2026-07-29 | Closed `ENV-006`: provisioned exact protoc and Buf from immutable artifacts, isolated them from global tools, and retained lint, descriptor parity, and Go/Java round-trip evidence | Codex |
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
