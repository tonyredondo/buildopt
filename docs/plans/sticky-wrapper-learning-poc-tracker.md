# Sticky Wrapper Learning POC Tracker

## Status

**Overall:** `IN_PROGRESS`<br>
**Progress:** `18/22` blocks complete<br>
**Current block:** `SWL-014C` — public opportunity and activation pre-gate<br>
**Predecessor:** the adaptive-fragment experiment closed as
`STOP_ADAPTIVE_FRAGMENT_POC` with zero activations and zero attributable
saving. Its evidence remains immutable context, not authority for this POC.

## Objective

Prove or reject one customer-facing hypothesis:

> A repository-committed, checksum-pinned BuildOpt Wrapper, backed by an
> optional centralized Gradle HTTP cache and a separate typed decision store,
> can learn from ordinary builds, activate only verified profitable actions,
> and produce positive cumulative wall-time value against optimized native
> Gradle with negligible native-retention overhead.

The intended customer change is deliberately small:

```text
./gradlew build
        becomes
./buildoptw build
```

The repository keeps the wrapper and portable non-secret configuration. The
wrapper downloads one immutable BuildOpt distribution, discovers the existing
Gradle Wrapper, preserves Gradle arguments and process behavior, and chooses
among native execution, observation, bounded trial, qualified action and
fallback. A customer does not install BuildOpt globally, hand-author a profile,
or copy BuildOpt internals into CI.

## Why this is a new experiment

The stopped adaptive-fragment POC performed recurring discovery and decision
work but activated no mechanism in 71 eligible builds. This successor does not
weaken that result or revive its profiles. It changes the delivery and economic
model:

1. one repository-committed wrapper is the stable integration point;
2. an exact locally cached decision makes the common native path independent
   of a network round trip;
3. ordinary builds contribute bounded evidence without a manual duplicate
   workflow;
4. expensive trials run only within a declared CI learning budget;
5. the central service shares evidence and Gradle outputs across machines so a
   repository does not relearn the same facts per checkout;
6. actions may be runtime profiles or durable reviewed Gradle changes;
7. an accepted durable change leaves plain Gradle faster without recurring
   BuildOpt action overhead; and
8. terminal value includes wrapper, observation, trial, cache, fallback and
   service costs across chronological commits.

The wrapper and service are enabling infrastructure, not acceleration claims.
The experiment passes only if the complete installed path creates cumulative
customer-visible value beyond the same native Gradle cache opportunity.

## Route correction after the first diagnostic sample

The first bounded `SWL-015 v1` sample exposed a structural problem in the work
sequence rather than a product-value result. Its control arm added
`--build-cache`, while the candidate used the frozen workflow unchanged; the
candidate configuration also had no server identity and a zero trial budget,
so it selected the native no-op/light-observation path. The sample therefore
proved wrapper compatibility and exact outputs, but it did not exercise the
learning lifecycle frozen above.

The codebase also contained independently checked observation, trial, active
execution and durable-catalog packages without one customer-path composition
that could move an ordinary `./buildoptw` invocation through those states. A
long campaign on that route could collect enough timing rows to appear ready
for a terminal decision while being structurally unable to satisfy activation,
payback, runtime-value or durable-value criteria.

The correction preserves every terminal threshold and all historical evidence.
It inserts four prerequisites before the expensive public campaign:

1. identical native-cache opportunity and a versioned `v2` measurement
   protocol;
2. one real end-to-end learning/action lifecycle behind `./buildoptw`;
3. a generic public-repository opportunity pre-gate that must find independently
   testable actions in at least three families; and
4. an installed active-path value gate proving those actions before longitudinal
   credit is possible.

The retained `v1` sample is `DIAGNOSTIC_ONLY`. It cannot contribute a passing
observation to `SWL-015 v2` or `SWL-016`.

## Scope

### In scope

- `buildoptw` and `buildoptw.bat`, generated and committed to the customer
  repository;
- `.buildopt/wrapper.properties` with immutable distribution URL, version and
  SHA-256;
- `.buildopt/config.toml` with portable endpoint, project scope, mode and
  learning-budget settings;
- exact passthrough to the repository's existing `gradlew` or `gradlew.bat`;
- `BUILDOPT_BYPASS=1` before any decision, plugin, gateway or state access;
- an optional HTTPS BuildOpt Server with a Gradle-compatible remote-cache data
  plane and a separate typed decision/evidence control plane;
- locally cached, signed, expiry-bound decisions for a negligible no-op path;
- `OBSERVE`, `SHADOW`, `TRIAL`, `QUALIFIED`, `ACTIVE`, `SUSPENDED` and
  `RETIRED` action states;
- ordinary-build timing and exact-output evidence;
- bounded CI trials and periodic native counterfactuals;
- runtime profiles whose exact bindings remain valid;
- review-required durable Gradle patches that continue to work through native
  Gradle after acceptance;
- `status` and `explain` output with cumulative gross saving, all BuildOpt cost,
  net saving and fallback reason; and
- installed-package, two-machine and chronological public-repository evidence.

### Out of scope

- production SLOs, high availability, multi-tenancy, billing, RBAC, KMS/HSM or
  disaster recovery;
- an eight-hour soak or external design partner;
- automatic merge, direct modification of a customer's long-lived branch or
  production rollout authority;
- replacing the Gradle Wrapper or implementing the Bazel remote-execution API;
- making the central service mandatory for a correct build;
- Test Optimization, including test selection, sharding, retries or flake
  management;
- reviving Runtime Tuning, exact Hot State, standard Copy or the stopped
  adaptive-fragment implementation; and
- claiming value from cache hits, graph reduction, avoided tasks or safe
  fallback without positive end-to-end wall time.

## Customer contract

### Generated repository files

| Path | Committed | Contains | Must not contain |
| --- | --- | --- | --- |
| `buildoptw` | Yes | Portable POSIX bootstrap and invocation shim | Binary payload, token or machine path |
| `buildoptw.bat` | Yes | Windows bootstrap and invocation shim | Binary payload, token or machine path |
| `.buildopt/wrapper.properties` | Yes | Version, HTTPS distribution URL and SHA-256 | Floating `latest`, redirect authority or credential |
| `.buildopt/config.toml` | Yes | Server URL, project scope, mode and budgets | Token, CA private key or checkout path |

Runtime binary downloads belong in the operating-system user cache. Generated
evidence and decision snapshots remain private state. Tokens come from an
environment variable, CI secret/OIDC exchange or a private token file and are
never passed to Gradle.

### Commands

```text
buildopt wrapper init --server https://buildopt.example.com
./buildoptw <gradle args...>
./buildoptw status
./buildoptw explain
BUILDOPT_BYPASS=1 ./buildoptw <gradle args...>
```

`wrapper init` is a maintainer command used once to generate the committed
surface. A developer or CI runner subsequently needs only `./buildoptw`.

### Bootstrap invariants

- The distribution URL is HTTPS and immutable for the selected version.
- The complete archive is verified against the committed SHA-256 before use.
- Extraction is atomic and rejects traversal, links and unexpected files.
- A verified cached distribution works offline.
- A changed version or checksum uses a distinct cache generation.
- Bootstrap failure never executes an unverified binary.
- Gradle arguments, streams, working directory, signals and exit status remain
  identical to direct Wrapper execution.

## Decision lifecycle

```text
UNSEEN -> OBSERVE -> SHADOW -> TRIAL -> QUALIFIED -> ACTIVE
              |         |        |          |          |
              +---------+--------+----------+------> SUSPENDED
                                                     |
                                                     v
                                                  RETIRED
```

Every invocation receives one explicit execution decision:

| Decision | Execution | Evidence effect |
| --- | --- | --- |
| `NATIVE_NOOP` | Original Gradle workflow | May update bounded native timing only |
| `OBSERVE` | Original Gradle workflow | Adds exact ordinary-build evidence |
| `SHADOW` | Original Gradle workflow | Evaluates compatibility/prediction without changing execution |
| `TRIAL` | Isolated candidate plus authoritative native result under budget | Adds paired correctness and value evidence |
| `ACTIVE_RUNTIME_PROFILE` | Exact qualified candidate | Measures candidate and retains immediate native fallback |
| `ACTIVE_DURABLE_PATCH` | Native Gradle with an accepted repository patch | Measures persistent native value; BuildOpt does not own task execution |
| `SUSPENDED` | Original Gradle workflow | Records drift, regression, expiry or revoked authority |
| `RETIRED` | Original Gradle workflow | Stops further exploration for that action generation |

`ACTIVE` never means unconditional execution. Repository scope, Gradle
Wrapper, workflow, options, output contract, action generation, compatibility,
expiry and revocation must still match. Missing or ambiguous state uses native
Gradle before task execution.

## Cache and state architecture

One owner-operated service may host both planes, but their protocols,
authorization and retention remain independent:

```text
./buildoptw
    |
    +-- local verified binary and decision snapshot
    |
    +-- Gradle HTTP cache plane --------> opaque evictable Gradle objects
    |
    +-- BuildOpt state plane -----------> typed evidence, decisions and ledger
```

- The data plane is compatible with Gradle's HTTP Build Cache contract.
- Cache objects are content-verified and repository/namespace scoped.
- Blob presence or a cache hit never grants optimization authority.
- The state plane uses canonical immutable documents and generation CAS.
- A service outage is a cache miss plus a native BuildOpt decision unless an
  exact unexpired signed local snapshot is available.
- A read-only client cannot publish Gradle objects, evidence or decisions.
- The same remote-cache opportunity is available to the native control in
  every value comparison.

The existing central cache and state implementation is reused. This tracker
adds the sticky wrapper and decision lifecycle; it does not create another
cache server.

## Frozen POC scorecard

The terminal decision uses current evidence only. Historical AF results explain
the design but contribute no passing observation.

| Criterion | Pass requirement |
| --- | --- |
| Wrapper onboarding | A clean checkout needs only the four generated committed files, one token source and `./buildoptw <args>`; no global BuildOpt installation or hand-authored profile |
| Bootstrap integrity | 100% of tested distributions are immutable, checksum verified and atomically installed; tamper, non-HTTPS/excessive redirect and traversal cases reject |
| Gradle equivalence | Arguments, environment boundary, streams, process tree, signals and exit status match direct Wrapper behavior on Linux, macOS and Windows |
| Correctness | 100% of comparable candidate/control outputs satisfy their frozen exact or reviewed equivalence contract; zero product-attributable failures |
| Plane separation | Cache objects cannot authorize decisions; state objects cannot be served as Gradle cache hits; wrong scopes reject |
| Native-retention overhead | Compatible local no-op decision path is at most 100 ms p50 and 250 ms p95 before Gradle; offline/server-unavailable fallback is at most 500 ms p95 |
| Learning budget | Additional trial/control compute is at most 5% of natural runner-minutes with one concurrent trial per repository |
| Generic implementation | No repository-name, task-name, path-extension or manually authored evaluated-profile rules |
| Activation breadth | At least three of five frozen public repository families activate or install at least one independently qualified action |
| Longitudinal confidence | At least three of five families have a positive paired/block-bootstrap 95% lower bound over the complete chronological window |
| Portfolio value | Complete signed cumulative net value is positive after bootstrap, observation, trial, cache, state, fallback and execution costs |
| Time to value | At least three families repay all learning/publication cost within 30 compatible customer-requested builds |
| Durable value | At least one generic reviewed transformation remains net positive over 15 or more later compatible commits with no BuildOpt runtime action dependency |
| Runtime value | Any runtime profile credited with value has a measured active saving and periodic native counterfactual; inactive/shadow actions receive zero saving |
| Explainability | Every invocation reports decision, binding generation, action, fallback reason and recomputable gross/cost/net values without secrets |

The terminal outcomes are deliberately binary:

- `CONTINUE_STICKY_WRAPPER_LEARNING_POC` only when every scorecard criterion
  passes; or
- `STOP_STICKY_WRAPPER_LEARNING_POC` when any correctness, breadth,
  confidence, cumulative-value, payback or overhead requirement fails.

An incomplete campaign is `INCOMPLETE`, not a reason to move thresholds.

## Ordered work

| Order | Block | Deliverable | State | Dependency |
| ---: | --- | --- | --- | --- |
| 0 | `SWL-000` Hypothesis and documentation alignment | Frozen tracker, machine-readable contract and repository-wide POC direction | `DONE` | AF-015 |
| 1 | `SWL-001` Repository wrapper contract | Exact committed files, properties/config grammar, CLI, bootstrap and authority boundaries | `DONE` | SWL-000 |
| 2 | `SWL-002` Wrapper generator | `buildopt wrapper init` creates, updates and verifies deterministic portable files | `DONE` | SWL-001 |
| 3 | `SWL-003` Verified bootstrap | Cross-platform download/cache/install with SHA-256, offline reuse and negative fixtures | `DONE` | SWL-002 |
| 4 | `SWL-004` Gradle passthrough and bypass | Wrapper discovers the repository Gradle Wrapper and preserves complete process behavior | `DONE` | SWL-003 |
| 5 | `SWL-005` Portable connection and project identity | Non-secret committed endpoint/scope plus private credential discovery | `DONE` | SWL-004 |
| 6 | `SWL-006` Gradle HTTP cache integration | Existing verifying gateway and central cache operate automatically through the wrapper | `DONE` | SWL-005 |
| 7 | `SWL-007` Decision contract and store | Typed state machine, signed decisions, generation CAS, expiry, revocation and plane separation | `DONE` | SWL-006 |
| 8 | `SWL-008` Native no-op fast path | Exact local snapshot makes native retention independent of a blocking remote lookup | `DONE` | SWL-007 |
| 8A | `SWL-008A` Sticky-wrapper native retention integration | Native no-op execution and lazy light observation avoid unnecessary launcher infrastructure | `DONE` | SWL-008 |
| 9 | `SWL-009` Ordinary-build observation | Bounded exact timings, outputs, task/graph facts and ledger updates from requested builds | `DONE` | SWL-008A |
| 10 | `SWL-010` Budgeted trial orchestration | Isolated paired candidate/native trials run only within the frozen CI compute budget | `DONE` | SWL-009 |
| 11 | `SWL-011` Active execution and suspension | Qualified runtime action, revalidation, counterfactual sampling, regression suspension and native fallback | `DONE` | SWL-010 |
| 12 | `SWL-012` Durable native optimization catalog | Generic detectors and reviewable patches for task contracts and graph breadth | `DONE` | SWL-009 |
| 13 | `SWL-013` Customer status and explanation | Human/JSON decision, cumulative economics, cache metrics and exact fallback explanation | `DONE` | SWL-011, SWL-012 |
| 14 | `SWL-014` Two-machine installed proof | Clean producer/consumer checkouts share cache/state through HTTPS and survive outage | `DONE` | SWL-013 |
| 14A | `SWL-014A` Protocol and comparison-fairness correction | Preserve v1 as diagnostic-only; freeze cache-symmetric v2 arms, required lifecycle evidence and campaign preflight | `DONE` | SWL-014 |
| 14B | `SWL-014B` End-to-end learning/action composition | Connect observation, proposal, shadow, trial, qualification, signed decision, active execution, suspension and economics to the real wrapper | `DONE` | SWL-014A |
| 14C | `SWL-014C` Public opportunity and activation pre-gate | Apply only generic detectors to all five frozen families and prove at least three independently testable actions before longitudinal spending | `TODO` | SWL-014B |
| 14D | `SWL-014D` Installed active-path value gate | Run balanced candidate/native trials through `./buildoptw`; require exact outputs, positive conservative value and complete cost attribution | `WAITING` | SWL-014C |
| 15 | `SWL-015` Frozen public longitudinal campaign v2 | Exercise real no-op/observe/trial/active transitions over preregistered chronological windows in five families | `WAITING` | SWL-014D |
| 16 | `SWL-016` Terminal decision | Recompute the immutable scorecard from v2 evidence or an earlier immutable stop gate and continue or stop without threshold movement | `WAITING` | SWL-015, or stop evidence from SWL-014C/014D |

## Agent execution contract — decisions already made

This section is normative for the remaining work. Another implementation agent
must execute it; it must not redesign it. In particular, the agent must not
rename a frozen artifact, add a detector, change a threshold or statistics
method, share writable cache state across arms, convert a skipped campaign into
an incomplete success, or treat sample count as readiness. If repository
evidence makes one of these decisions impossible, the block is `BLOCKED` and
the conflict is reported; the agent does not substitute a new design.

### Common completion protocol

For each completed block:

1. start from a clean `main` matching `origin/main` and inspect the exact diff;
2. create/modify only the paths in that block's manifest plus the documentation
   rows explicitly owned by that block;
3. run the focused checks in the manifest, then
   `./dev/check-sticky-wrapper-learning-plan`,
   `./dev/check-tracker-consistency`, `./dev/check-layout`,
   `./dev/check-normative-layout`, `./dev/check-documentation`, exact
   ShellCheck/actionlint where applicable, `./dev/check-base-ci --static` and
   `git diff --check`;
4. update this tracker, `implementation-tracker.md`, the evidence row and every
   document required by the documentation update contract;
5. create one English commit for that block, push `main`, then verify a clean
   worktree and equality of `HEAD`, `origin/main` and remote `main`; and
6. if credentials or the remote are unavailable, report `BLOCKED` with the
   validated local SHA. Do not report the block complete.

No soak, design-partner run, automatic merge, production gate or Test
Optimization work is part of this route.

Every new evidence document uses RFC 8785 JCS, rejects unknown fields, writes
canonical UTC RFC 3339 nanosecond timestamps, lowercase hexadecimal SHA-256
digests, signed 64-bit nanosecond durations and repository-relative paths with
forward slashes. Unavailable values are explicit typed outcomes and are never
represented as zero.

### Measurement state topology

Control and candidate always use the same explicit Gradle argument vector and
the same cache policy. They do **not** use one writable namespace. Each family
and arm receives:

- one separate checkout advanced chronologically through the frozen commits;
- one separate persistent Gradle user home, daemon registry, Configuration
  Cache, local Build Cache and workspace outputs;
- an identically seeded read-only dependency/distribution state prepared
  outside timing, with no task outputs; and
- one separate initially empty remote namespace before the first measured pair,
  through the same service/TLS/network limits.

Remote writes persist within an arm but are never visible to the other arm.
The control gets an empty inert BuildOpt root. Candidate decision/evidence
state is private and persists across compatible commits. No task output,
Configuration Cache, post-start local cache entry or BuildOpt state may be
copied between arms. Both local Build Caches also start empty. The preflight
evidence records all root/empty-seed/namespace digests and rejects equality of
writable paths or namespaces.

Wall time uses an external Go monotonic clock from immediately before direct
command start until the direct command has been waited and its required process
tree is clean. Wrapper, decision, state/network, action, Gradle and required
output verification before the next arm all count. Checkout advance,
dependency/distribution preparation, empty namespace creation and result
serialization after both arms do not. The runner records OS, architecture,
logical CPUs, CPU quota, memory limit, Java and Gradle; this POC does not force
a 4-CPU runner, but a fingerprint change within a pair excludes that pair as
`EXCLUDED_RUNNER_RESOURCE_DRIFT`.

### Composition and trigger

`internal/launcher/sticky_learning.go` is the only new composition root;
`internal/launcher/run.go` delegates to it. It adapts the existing
`stickyobservation`, `stickytrial`, `stickydecision`, `stickyactive` and
`durablecatalog` packages. Shared qualification statistics live in the new
`internal/stickyvalue` package and are consumed by active execution and the
independent value checker. Private test seams are named
`stickyLearningClock`, `stickyLearningDetector`,
`stickyLearningTrialRunner` and `stickyLearningDecisionPublisher`; they exist
only to inject deterministic fixture behavior, never repository rules.

Every customer-triggered native, candidate and counterfactual process remains
owned by `internal/launcher.executeChildWithReserved` so the existing WS-002
process-group, signal and exit-code contract is unchanged. `SWL-014B` adds
executor-injection adapters named `stickyactive.Runner.RunWithExecutor` and
`stickytrial.RunPairedWithExecutor`; their existing direct-run methods remain
compatibility wrappers. Calling `os/exec` directly from the composed customer
path is forbidden. Unexported helper signatures, local error wrapping and
fixture identifier values are mechanical implementation freedom; they may not
change any observable contract or evidence field.

The shared value owner is `internal/stickyvalue`. Its only public calculation
entry point for this route is
`Evaluate(pairs []Pair, costs Costs) (Evaluation, error)`. The exact `Pair`,
`Costs` and `Evaluation` fields are frozen in the v2 machine contract. It uses
checked signed 64-bit integer arithmetic; any unknown or missing cost makes the
evaluation unavailable and not qualified.

An ordinary call loads the exact binding, selects a verified local decision
without blocking on a network lookup, executes an exact active runtime action
or native Gradle, appends bounded evidence and records ledger value only when
a counterfactual exists. Refresh is asynchronous. Trusted trials require all
of: mode `auto`, non-zero committed trial budget,
`BUILDOPT_STICKY_LEARNING=1`, `STATE_WRITE`, `CACHE_READ` and `STATE_READ` in
the owner token. The variable and token are scrubbed before Gradle. `observe`
never trials or activates; `off` and bypass stay native before state access.
The scheduler reserves the exact committed non-zero `trial_budget_percent`,
never more than five percent; it does not silently round a smaller owner budget
up to five percent.

Durable proposals remain review-only. Apply/revert happens in an isolated
transaction; an owner applies any accepted patch. The wrapper only verifies
the postimage binding and executes plain native Gradle. There is no automatic
merge or mutation of the long-lived checkout.

### Frozen generic detectors

`SWL-014C` runs exactly two detector adapters, in this order:

1. `TASK_CONTRACT_JAVA_V1` -> `internal/durablecatalog.TaskProposal` ->
   `CUSTOM_TASK_CONTRACT_JAVA_V1`. The current repository has no generic public
   producer for its `PatchOpportunityInput`; every public family therefore
   records `INPUT_UNAVAILABLE_NO_GENERIC_SOURCE_PRODUCER`. This route may not
   invent a source parser after seeing results;
2. `DECLARED_GRAPH_SCOPE_V1` ->
   `internal/durablecatalog.DetectGraphBreadthOpportunity` ->
   `DECLARED_GRAPH_SCOPE_V1`, fed only by the installed `profile outputs`,
   isolated `profile input --confirm`, `profile propose` and
   `gradlecriticalpath.Analyze` pipeline using the frozen repository,
   workflow, required outputs and changed paths.

Each needs three exact recurring observations. Task projection is the minimum
eligible task duration in those three recurrences. Graph projection is the
minimum omitted critical-path contribution in those recurrences. Repayment is
`ceil((trial + validation + publication cost) / projected saving)` and must be
at most 30 compatible builds. A non-positive projection rejects. Repository
name, task name, path extension and manual evaluated-profile rules are banned.

Graph screening runs the full and proposed candidate with operation/task-graph
traces for three recurring identical action identities, requires the frozen
outputs, and projects the minimum omitted critical-path contribution. An
unsupported/incomplete proposal is visible `INPUT_UNAVAILABLE`, not an action.
Action identity is the SHA-256 formula frozen in the machine contract over
detector, repository scope, workflow arguments and candidate plan. No other
detector or fallback heuristic may be added in this route.

### Frozen statistics

`SWL-014D` uses eight balanced alternating pairs per action. The signed effect
is `native wall ns - candidate wall ns`. Confidence uses the existing
4,096-replicate deterministic 32-bit LCG paired bootstrap: initial state
`2654435761 * (replicate + 1) mod 2^32`, step
`1664525 * state + 1013904223 mod 2^32`, sorted bounds at zero-based indices
102 and 3993. Passing requires positive mean, positive lower bound, a strict
majority of positive pairs, exact/reviewed outputs, zero product failures and
nearest-rank candidate p95 no greater than native p95.

`SWL-015` uses signed **net** saving after every BuildOpt cost. Per-family
confidence uses 10,000 deterministic circular moving-block bootstrap
replicates over at least 15 chronological values. Block length is
`ceil(sqrt(n))`; block order is preserved with circular wrap. Xorshift64
steps `(13,7,17)` use the first eight big-endian bytes of
`SHA-256("buildopt-sticky-wrapper-longitudinal-v2\\0" + familyKey)`; zero uses
`0x9e3779b97f4a7c15`. Sorted index 499 is the one-sided fifth-percentile lower
bound and must be positive.

### Exact remaining file manifests

The complete machine-readable lists and focused commands are in
[`poc-sticky-wrapper-longitudinal-v2.json`](../../specs/poc-sticky-wrapper-longitudinal-v2.json).
The owning paths are fixed as follows:

| Block | New implementation/evidence paths |
| --- | --- |
| `SWL-014A` | `dev/run-sticky-wrapper-longitudinal-v2`, `dev/check-sticky-wrapper-longitudinal-v2`, `dev/test-sticky-wrapper-longitudinal-v2`, `fixtures/sticky-wrapper-longitudinal-v2/*`, `benchmarks/results/sticky-wrapper-longitudinal-v2-preflight.json` |
| `SWL-014B` | `internal/launcher/sticky_learning.go`, its test, `internal/stickyvalue/*`, `fixtures/sticky-wrapper-learning/README.md`, lifecycle spec/check/result |
| `SWL-014C` | `internal/durablecatalog/public_screen.go`, `cmd/sticky-opportunity-screen/main.go`, opportunity runner/check/spec/result |
| `SWL-014D` | `cmd/sticky-active-value/main.go`, installed-value runner/check/spec/result |
| `SWL-015` | final-campaign modifications to the v2 runner/checker/test from 014A and `benchmarks/results/poc-sticky-wrapper-longitudinal-v2/` |
| `SWL-016` | `cmd/sticky-wrapper-terminal-decision/main.go`, terminal checker/spec/result |

Each manifest also lists the exact documentation paths, evidence schema
version, top-level/record keys and accepted outcome strings. Those JSON arrays
are normative. A later agent may populate their values and advance the block
state, but it may not add a field, substitute a path or reinterpret an outcome
without first reporting the contract as blocked.

### Early-stop state machine

- If fewer than three families pass `SWL-014C`, close it as
  `DONE_WITH_STOP_EVIDENCE`, mark `SWL-014D` and `SWL-015`
  `SKIPPED_BY_SWL_014C`, and make `SWL-016` the next block.
- If fewer than three families pass `SWL-014D`, close it as
  `DONE_WITH_STOP_EVIDENCE`, mark `SWL-015` `SKIPPED_BY_SWL_014D`, and make
  `SWL-016` the next block.
- The opportunity/value result JSON is the stop evidence. Only `SWL-016` emits
  `STOP_STICKY_WRAPPER_LEARNING_POC`; skipped work is never reported as
  `INCOMPLETE` or silently rerun.

## Block contracts

### SWL-000 — Hypothesis and documentation alignment

Deliverables:

- this detailed tracker;
- [`poc-sticky-wrapper-learning-v1.md`](../../specs/poc-sticky-wrapper-learning-v1.md)
  and its machine-readable contract;
- RFC alignment without changing the stopped AF result;
- implementation-tracker status and evidence registration; and
- one-pager, architecture, onboarding, workflows, CLI/configuration and
  documentation-index alignment.

Acceptance: repository documentation names the same wrapper surface, state
machine, cache/state separation, scorecard, POC boundary and next block.

### SWL-001 — Repository wrapper contract

Closed by the
[`sticky-wrapper-contract-v1`](../../specs/poc-sticky-wrapper-contract-v1.md)
protocol, its machine contract and executable fixture matrix. The contract
freezes UTF-8/ASCII encoding, LF bytes, Git modes, limits, ordered strict
properties and flat-TOML grammars, immutable per-platform HTTPS URLs and
SHA-256 values, fixed download timeouts, environment-only proxy discovery and
at most five HTTPS-only redirects with the pinned archive digest as final
authority.

Default arguments remain Gradle arguments. Only the first-argument
`--buildopt` prefix selects `status`, `explain` or `version`; `--gradle`
escapes that prefix. `BUILDOPT_BYPASS=1` is evaluated before configuration or
bootstrap. `init`, read-only `check`, update-only distribution identity,
idempotence, explicit downgrade and atomic publication semantics are fixed.
Credentials and machine paths are absent from generated files.

Acceptance evidence: independent POSIX- and Windows-shaped parsers agree on
the canonical portable fixture; 13 unknown, duplicate, security-sensitive,
path, URL, redirect, budget and partial-identity cases reject on both parsers;
argument and update vectors pass. The generator and bootstrap remain false in
the contract and belong to SWL-002/003.

### SWL-002 — Wrapper generator

Implemented `buildopt wrapper init`, offline/read-only `check` and explicit
`update --version`. `init` resolves stable public GitHub release metadata into
immutable asset URLs and GitHub-provided SHA-256 digests without downloading
an archive. Generation is deterministic; existing targets reject before
metadata access. Update validates the complete current state, preserves both
scripts and owner-controlled configuration, performs no same-version write and
requires explicit downgrade authority.

Acceptance: two generations are byte-identical; drift fails `--check`; update
changes only pinned distribution identity; partial failure leaves all prior
files intact.

Closure: 15 test functions pass under the race detector. They cover every
existing target, deterministic bytes, a read-only tree snapshot, script /
properties / configuration drift, same-version idempotence, owner-config
preservation, downgrade authority, pre-resolution rejection, concurrent
writers and an injected mid-publication rollback with no transaction residue.
The package passes `go vet` and compiles for Windows AMD64 and macOS ARM64. A
real Linux smoke resolved public `v0.6.1`, generated modes `0755/0644`, checked
all bytes, performed an idempotent update and rejected a second init with code
65. No archive was downloaded or executed.

### SWL-003 — Verified bootstrap

Implemented thin shell/batch bootstrap around a user-cache installation. Exercise
Linux AMD64, macOS ARM64/Intel and Windows AMD64 package selection, archive
checksum, internal manifest verification, atomic publication, concurrent
bootstrap and verified offline reuse.

Acceptance: clean online and warm offline execution pass; altered archive,
checksum, non-HTTPS or excessive redirect, traversal, link, unsupported
platform and interrupted install reject before BuildOpt starts.

Closure: the generated POSIX and Windows scripts select the pinned native
archive, follow at most five HTTPS-only redirects, verify the outer SHA-256
before extraction, reject unsafe paths and links, verify every internal
`SHA256SUMS` entry and publish one version/platform/archive-digest directory
atomically in the user cache. A verified warm entry is rechecked without a
network request; concurrent first use performs one download. Thirteen
synthetic POSIX scenarios, Go race/vet checks, PowerShell parsing and public
`v0.6.1` Linux and Windows-body online/offline smokes pass. Native macOS and
Windows wrapper entrypoints are wired into platform CI. The initial public
GitHub asset smoke exposed that release assets may return an HTTPS redirect;
the frozen policy was corrected from zero redirects to at most five
HTTPS-only redirects while the pinned archive SHA-256 remains authoritative.
This block authorizes only `--buildopt version` and bootstrap behavior. The
current templates also contain the later SWL-004 passthrough, whose independent
evidence supplies process authority; SWL-003 itself makes no passthrough,
build-time or production claim.

### SWL-004 — Gradle passthrough and bypass

The installed binary discovers only the repository's `gradlew`/`gradlew.bat`.
It forwards arbitrary Gradle arguments without a shell and owns signals and
cleanup through existing launcher primitives. `BUILDOPT_BYPASS=1` skips server,
state, cache gateway, plugin and observation before any of them start.

Acceptance: direct Gradle and wrapper executions match arguments, cwd,
standard streams, environment boundary, descendants and ordinary/signal exit
codes across the supported native CI matrix.

Closure: ordinary arguments now invoke only the sibling `gradlew`/`gradlew.bat`
through the existing `buildopt run --` process launcher. The POSIX executable
fixture preserves empty/space/glob/variable/quote/Unicode/newline arguments,
cwd, stdin/stdout/stderr, ordinary environment and exit 37; removes private
BuildOpt environment; proves `--gradle --buildopt` routing, unknown management,
pre-bootstrap bypass with missing configuration, bootstrap-failure fallback,
missing/non-executable wrapper exits 127/126 and `SIGTERM` delivery to a child plus descendant with
Gradle-owned exit 42. Native platform CI exercises the real macOS and Windows
wrapper entrypoints; macOS also routes its cancellation tree through
`buildoptw`, while the existing Windows Job Object gate retains descendant
cleanup. No cache, observation, learning or optimization is activated.

### SWL-005 — Portable connection and project identity

Bind committed endpoint/project scope to a private token source. Forks and
untrusted CI have no credential. Project identity is stable across checkout
paths but cannot collide across server namespaces. Plain HTTP outside loopback,
embedded tokens and redirects reject.

Acceptance: two clean machines derive the same authorized project scope;
another repository, missing/wrong scopes and revoked token retain native and
cannot read state or cache objects.

Closure: the generated wrapper declares its canonical repository root only to
the verified BuildOpt process. The binary loads the strict committed
configuration, reads the exact owner-issued `buildopt.central/access-token/v1`
document from the named private environment variable, derives project identity
without the checkout path and additionally binds server, tenant, trust domain,
cache namespace and namespace generation. It requires `CACHE_READ` plus
`STATE_READ`, checks expiry locally and probes both capabilities without
mutating either plane; redirects are disabled and live revocation is
authoritative. Two independent checkout roots produce one project/connection
identity, while another namespace produces a distinct connection. Another
repository, absent credential and incomplete capabilities make no request;
revoked credentials receive `401` for both planes. The dynamic credential and
wrapper root are removed before Gradle while ordinary environment remains
unchanged. The race-enabled HTTPS fixture, redirect/root negatives, Go vet and
macOS ARM64/Windows AMD64 compilation pass. No Gradle cache, typed state,
decision, learning or optimization is activated.

### SWL-006 — Gradle HTTP cache integration

Reuse the existing local verifying gateway and central Gradle-compatible
object plane. The wrapper configures read-only access automatically when the
connection permits it. Trusted writers remain explicit; no developer or pull
request becomes a publisher by default.

Acceptance: one trusted build publishes, a clean read-only build obtains exact
`FROM-CACHE` outputs, wrong/corrupt objects miss, outage rebuilds, and a native
control receives the same remote-cache opportunity in timing experiments.

Closure: a valid sticky-wrapper connection now creates an invocation-local
verifying gateway bound to the committed project scope, namespace, generation
and attempt. The launcher adds Gradle's native `--build-cache` flag unless the
caller explicitly uses `--no-build-cache`, while preserving every other
argument. The central path is read-only for ordinary consumers and uses
Gradle's native cache policy for arbitrary cacheable tasks; the private managed
L1 remains a separate BuildOpt policy. An explicit write-only producer publishes
the existing eight-task fixture, a clean sticky-wrapper consumer restores all
required outputs with `FROM-CACHE`, a corrupt object becomes a byte-free miss,
and a stopped service performs a successful native rebuild with identical
outputs. The race-enabled integration also proves that the gateway transport
cannot race with Gradle cache requests. No decision/state document is consumed,
no automatic writer is granted, and no wall-time advantage is claimed by this
functional block.

### SWL-007 — Decision contract and store

Implemented the canonical action, observation, trial, decision and
economic-ledger documents plus the mutable state-head and revocation inputs.
The JSON Schema union is closed and JCS-addressed; the Go implementation adds
the cross-record invariants that a schema cannot express:

- qualification and rollout state machines are independent, with every valid
  transition covered and direct activation, invalid rollback, repeated
  suspension and premature retirement rejected;
- action sequences carry the exact prior state, active decisions require
  earlier resolvable observation/trial evidence, and ledgers resolve their
  action and observation references;
- decisions bind repository, workflow, Gradle, Wrapper, options, outputs,
  action generation, policy/cache-contract digests, expiry and revocation and
  are verifiable with the owner Ed25519 key;
- immutable records are published before a single generation-CAS head, with
  exact idempotent replay and changed-request conflict behavior;
- local files and the existing central `EVIDENCE` state adapter share the same
  canonical validation, while opaque Gradle cache objects remain in a separate
  namespace and cannot authorize an action; and
- signed ledger arithmetic rejects negative/overflowing or non-reconciled
  values instead of turning a regression into a false saving.

Acceptance: the focused checker passes every valid transition, replay,
conflict, stale generation, expiry, revocation, corruption, cross-plane,
cross-scope, unknown-evidence, unknown-observation and arithmetic-overflow
negative case in both local and central storage. Seven valid and six invalid
Draft 2020-12 fixtures are validated with the isolated compiler. This block
defines storage and authority only; no selector, automatic activation,
learning loop or wall-time claim is enabled. `SWL-008` consumes the store
through the read-only native-retention selector; ordinary observation is next.

### SWL-008 — Native no-op fast path

Cache only signed, verified and expiry-bound decisions locally. A current
`NATIVE_NOOP` or compatible `ACTIVE` decision is read without blocking on the
server; synchronization occurs outside the critical decision path. Unknown or
expired state returns native and may schedule observation asynchronously.

Acceptance: the target benchmark class measures at most 100 ms p50 and 250 ms
p95 pre-Gradle overhead; service outage remains at most 500 ms p95; corrupt or
incompatible snapshots fail closed without executing an action.

Closed evidence: the deterministic selector and benchmark in
[`poc-sticky-wrapper-noop-v1`](../../specs/poc-sticky-wrapper-noop-v1.md) read
only private local state, accept a verified `NATIVE_NOOP`, defer compatible
active actions, and return native Gradle for missing, expired, revoked,
corrupt, busy or incompatible state. On this Linux AMD64 host, 200 selections
recorded verified-local p50/p95 of 0.492/1.369 ms; missing-state fallback was
0.0025/0.0025 ms and the no-synchronous-refresh case was 0.0025/0.0026 ms.
All values are retention-cost measurements, not acceleration evidence, and
the checked-in JSON remains the source of truth. `SWL-009` is complete and
`SWL-008A` and the later blocks extend this retention boundary without changing
the selector's fail-closed authority.

### SWL-008A — Sticky-wrapper native retention integration

Connect the native-retention idea to the customer-facing `buildoptw` execution
path. When no server credential and no explicit BuildOpt integration are
configured, the wrapper must validate its boundary and let the repository's
native Gradle Wrapper run without starting a gateway, plugin handshake,
managed L1, central-cache probe or bootstrap state. BuildOpt-only environment
variables are removal-only and never become Gradle inputs. A configured
credential or explicit integration keeps the established instrumented path;
malformed committed configuration retains that path so its diagnostics are not
hidden.

The ordinary observer is deliberately lazy and has explicit modes:

| Mode | Behavior |
| --- | --- |
| unset, `1`, `light` | Bounded light observation; no pre-build Git lookup, executable digest computed concurrently when possible, and recorder creation only after the child exits |
| `full` | Explicit diagnostic mode that also attempts source-revision evidence |
| `0` | No observation state; use the lightweight process supervisor so process groups and descendant signals remain equivalent |
| unknown | Retain the established full launcher path |

Acceptance requires exact passthrough and child-environment scrubbing, no
gateway/handshake/cache setup on the implicit native path, lazy recorder
creation, and an interleaved local overhead result with at least 20 samples.
This block does not claim a Gradle speedup and does not authorize an
optimization action. It only prevents wrapper plumbing from erasing the value
of later cache or Build Impact experiments.

Closed evidence: [`poc-sticky-wrapper-noop-overhead-v1`](../../specs/poc-sticky-wrapper-noop-overhead-v1.md),
the native fast-path implementation and focused tests, plus the executable
[`check-sticky-wrapper-noop-overhead`](../../dev/check-sticky-wrapper-noop-overhead)
validate the checked-in 20-sample Linux result. Direct execution is 2 ms p50 /
3 ms p95;
the native no-op path adds **9 ms p95**, light observation adds **38 ms p95**,
and light pre-child decision work is **0.093 ms p95**. All values pass the
100/250/100 ms POC guardrails. The result is retention-cost evidence only;
`SWL-009` remains the ordinary-build observation contract and `SWL-015` the
current longitudinal value campaign.

### SWL-009 — Ordinary-build observation

Capture only facts needed by registered hypothesis classes. Attribute wrapper,
bootstrap, decision, network, cache, observation and Gradle time separately.
Never transform unavailable evidence into zero. Ordinary user builds remain the
only source of natural outcomes.

Acceptance: bounded observation preserves Configuration Cache, reconciles wall
time, emits exact provenance and can be disabled independently when its p95
budget is exceeded.

Closed evidence: [`poc-sticky-wrapper-observation-v1`](../../specs/poc-sticky-wrapper-observation-v1.md),
the Draft 2020-12 schema, private append-only recorder and executable
[`check-sticky-wrapper-observation`](../../dev/check-sticky-wrapper-observation)
run two real Gradle 9.6.1 Wrapper invocations. Both succeed with
Configuration Cache present; the cold invocation records **19.876 s** and the
reuse invocation **3.732 s**. Exact decision work is **53.8/57.4 ms** and
exact child-Gradle work is **19.821/3.673 s**. Every record reconciles its
timing object, binds the Wrapper/BuildOpt/argument digests, rejects tampering,
and keeps network, bootstrap and post-build observation unavailable rather
than inventing zeros. The default path now uses light observation, with the
executable digest computed concurrently when possible; `full` is
explicit for diagnostic source-revision evidence and `0` disables recording.
The output path and mode are scrubbed from the child; observation failures
remain diagnostic and never change the requested exit code. This is
cost-accounting evidence, not a speedup claim.

### SWL-010 — Budgeted trial orchestration

Schedule isolated candidate/native trials in trusted CI only when projected
compatible lifetime can repay them. The customer's requested result remains
authoritative. Trials use separate checkout, Gradle home, daemon, cache and
BuildOpt state and cannot exceed 5% of natural runner-minutes.

Closed evidence: [`poc-sticky-wrapper-trial-v1`](../../specs/poc-sticky-wrapper-trial-v1.md),
its Draft 2020-12 schema and executable [`check-sticky-wrapper-trial`](../../dev/check-sticky-wrapper-trial)
run four alternating pairs (eight direct invocations) against a 256-class
Gradle 8.14.3 fixture. Every pair uses eight distinct private roots, balances
execution order, assigns before running either arm, and hashes required output
trees exactly. The observed trial cost is **58.050 s** against a **180 s**
ceiling (5% of a declared 3,600 s natural runner window); all eight commands
succeed and all four output pairs match exactly.

The value result is intentionally negative: candidate mean wall time is
**7.534 s**, optimized native Gradle is **6.979 s**, mean saving is
**-0.555 s**, and **0/4** pairs are positive. The scheduler and evidence
accounting pass, but this candidate is not authorized to activate. The result
is retained as overhead evidence and the next block must reduce that cost
before active execution is attempted.

### SWL-011 — Active execution and suspension

An exact qualified runtime profile can become `ACTIVE`; every invocation still
revalidates bindings. A bounded native counterfactual measures ongoing value.
One correctness failure, revoked authority, incompatible drift or decisive
negative value suspends before another candidate execution.

Acceptance: exact active, drift, expiry, server outage, cache miss, regression,
cancellation and bypass cases preserve outputs and original exit behavior; only
measured active executions receive saving attribution.

Implementation and evidence: `internal/stickyactive` provides the generic
direct-command runner, signed-decision revalidation, required-output hashing,
native counterfactuals and permanent suspension until a new decision
generation. `cmd/sticky-active-benchmark` runs eight repository-independent
scenarios and rejects the checked-in SWL-010 report as
`TRIAL_NOT_PROFITABLE` (7.534 s candidate versus 6.979 s optimized native,
0/4 positive). One synthetic active scenario saves about 24.6 ms with exact
outputs; three scenarios suspend and four retain native. This is a control-flow
proof only, not customer-repository performance evidence or activation
authority.

Status: `DONE`; see `E-414` and `SWL-E012`. SWL-012 is closed and SWL-013 is next.

### SWL-012 — Durable native optimization catalog

Start with two evidence-backed generic opportunity classes:

1. expensive repeatable tasks whose missing/incorrect input-output contract
   prevents native cache or up-to-date reuse; and
2. over-broad task/project dependency edges for a declared output workflow.

Detection is generic; each transformation recipe remains exact, reviewable,
reversible and separately validated in an isolated worktree. The POC may
propose but never merge automatically.

Acceptance: at least two repository families expose one candidate from the
same structural detector; accepted patches preserve complete native workflows
and outputs, show positive paired wall time, revert exactly, and require no
BuildOpt runtime action after merge.

Implementation and evidence: `internal/durablecatalog` and
`cmd/sticky-durable-catalog-benchmark` expose the two detector classes with
strict typed inputs, digest-bound recipes and an apply/revert transaction that
never mutates a checkout. The checked-in
[`sticky-wrapper-durable-catalog-v1.json`](../../benchmarks/results/sticky-wrapper-durable-catalog-v1.json)
was regenerated from the current strict `linux-amd64-4c-16g-v1` campaign at
revision `1d93570c02147eda8671253663d50605bff9f25a`:

- the same task-contract detector accepts Kotlin and Groovy families;
- the reviewed task patch is byte-exact, reversible and needs no BuildOpt
  runtime after acceptance;
- Kotlin averages **2.438 s -> 0.875 s**, saving **1.563 s / 64.1%** with
  **8/8** positive pairs and exact outputs;
- Groovy averages **3.574 s -> 0.903 s**, saving **2.671 s / 74.7%** with
  **8/8** positive pairs and exact outputs; and
- the graph detector proposes a generic **3 -> 2 project** transformation for
  both DSLs, but durable graph timing remains explicitly **unmeasured**.

This is promising synthetic POC evidence, not customer coverage or automatic
patch authority. The task detector passes the current value gate in both
families; the graph class remains proposal-only until a separate native timing
experiment proves that the committed dependency change preserves outputs and
has positive wall-time value.

Status: `DONE`; see `E-415` and `SWL-E013`. SWL-013 is now the next block.

### SWL-013 — Customer status and explanation

`./buildoptw status` reports current decision, active action, observations,
trials, native fallbacks, cache usage, gross saving, every cost and signed net
value. `./buildoptw explain` reports exact bindings and the reason an action
did or did not execute. JSON recomputes the human output.

Acceptance: no tracker vocabulary or secret is required to understand the
result; missing evidence stays unavailable; percentages from different actions
are never added; tampering or arithmetic mismatch rejects.

Implementation and evidence: `internal/stickywrapper/status.go` derives one
validated report model for both `status` and `explain`. The generated POSIX and
Windows wrappers route management invocations without ambiguity, while an
ordinary Gradle task named `status` or `explain` remains an ordinary task.
Human output and `--json` are rendered from the same model. The report exposes
wrapper version/mode, decision state, observation counts and phase totals,
trial/cache/economic values, fallback reason and exact latest bindings. Missing
values remain `UNAVAILABLE` rather than zero; credentials and checkout paths
are never emitted. A tampered observation log, invalid arithmetic or an
unverified decision fails closed and retains native Gradle. The checker
[`check-sticky-wrapper-status`](../../dev/check-sticky-wrapper-status) covers
empty state, JSON/human parity, read-only behavior, Gradle-task ambiguity and
tamper rejection.

Status: `DONE`; see `E-416` and `SWL-E014`. SWL-014 is now the next block.

### SWL-014 — Two-machine installed proof

Generate and commit the wrapper in an external fixture repository. A trusted
producer and clean consumer use the verified package, central HTTPS service
and separate credentials. Exercise restart, pending-publication visibility,
cache reuse and service outage through the same committed `./buildoptw`
command. The wrapper archive is preloaded only into the user cache; the
launcher still verifies its internal manifest and outer SHA-256 before use.

Closure: [`poc-sticky-wrapper-two-machine-v1.md`](../../specs/poc-sticky-wrapper-two-machine-v1.md),
its machine contract and [`check-sticky-wrapper-two-machine`](../../dev/check-sticky-wrapper-two-machine)
prove the installed path in isolated 4-CPU/8-GiB containers. The producer
publishes two Gradle cache objects, a same-generation read records
`OWNER_COMMIT_REQUIRED`, the owner commits and restarts the HTTPS service, and
the clean read-only consumer restores **2 tasks from cache**. Producer,
consumer and outage runs produce the same required output SHA-256
(`170ffd3f...077baeb` in the checked result). With the service offline, the
wrapper performs a native Gradle rebuild with zero central hits and the same
output. The distribution archive is verified in both machines, credentials
remain outside Gradle/logs, and the configuration/decision planes stay
separate. The final clean-SHA run records producer **11.027 s**, consumer
**7.938 s** and outage **7.435 s** for a later fair timing experiment; this block
makes no wall-time or profile-qualification claim.

The checked result is
[`sticky-wrapper-two-machine-v1.json`](../../benchmarks/results/sticky-wrapper-two-machine-v1.json).
Soak, design-partner validation, production authority and Test Optimization
remain out of scope. The first longitudinal diagnostic then exposed the route
gap described above, so `SWL-014A` is now the next block.

### SWL-014A — Protocol and comparison-fairness correction

Preserve the existing `SWL-015 v1` sample and protocol as immutable diagnostic
evidence. Freeze a `v2` protocol in which control and candidate receive the
same Gradle arguments, local-cache policy, optional remote-cache opportunity,
dependency preparation and state isolation. The candidate must identify the
selected lifecycle action; an implicit no-op cannot be credited as an active
optimization.

Deliverables:

- [x] [`poc-sticky-wrapper-longitudinal-v2`](../../specs/poc-sticky-wrapper-longitudinal-v2.md)
  and its machine contract are preregistered;
- [x] the retained v1 sample is explicitly `DIAGNOSTIC_ONLY` and its checker
  rejects terminal readiness;
- [x] RFC, trackers, one-pager, onboarding, workflows and findings describe the
  corrected route;
- [x] cache topology, lifecycle composition, detector catalog, statistics,
  early-stop transitions and exact file manifests are locked;
- [x] create `dev/run-sticky-wrapper-longitudinal-v2` with
  `--preflight-only OUTPUT`;
- [x] create `dev/check-sticky-wrapper-longitudinal-v2` with
  `--preflight RESULT`;
- [x] create `dev/test-sticky-wrapper-longitudinal-v2` plus the exact five JSON
  fixtures named in the machine contract; and
- [x] check in `benchmarks/results/sticky-wrapper-longitudinal-v2-preflight.json`.

The final campaign cannot consume public opportunity/value evidence until
`SWL-014C/D` create and prove it. Therefore `SWL-015`, not this block, owns
the exact manifest entries that modify these three scripts with final campaign
and result-directory modes. This dependency correction prevents a preflight
implementation from inventing an unowned future evidence interface.

Acceptance: a zero-pair fixture proves both arms receive the same native cache
opportunity and the checker cannot emit `READY_FOR_SWL_016` from wall-time rows
alone. The four negatives independently reject asymmetric cache policy,
missing lifecycle evidence, zero-budget learning and a shared/unbound state
root. Run the three `SWL-014A` focused commands in the machine contract, then
the common completion protocol. No new performance claim is made in this
block.

Closed evidence: the generated preflight is canonical JCS, binds the frozen
contract and cohort digests, records two distinct sets of writable-root
identities and remote namespaces with identical workflow/cache/service/empty
seed identities, and declares an authorized non-zero learning configuration.
It contains zero pairs, remains `VALIDATED_NOT_READY`, and cannot authorize
`SWL-016`. Each of the four named negative fixtures fails for its exact reason.

### SWL-014B — End-to-end learning/action composition

Connect the already implemented packages to the customer command. An ordinary
wrapper build records bounded evidence; a generic detector may create a typed
proposal; trusted CI may schedule an isolated trial within budget; a qualified
result may publish a signed decision; and the next compatible wrapper
invocation must actually execute, counterfactually revalidate or suspend that
action. Missing or invalid state remains native.

Implementation is confined to the exact `SWL-014B` manifest. The launcher
composition and `internal/stickyvalue` are new; the existing domain packages
remain authoritative. The repository-independent fixture supplies its proposal
through `stickyLearningDetector`; it must not hard-code a repository/task/path
rule. The fixture token has explicit state-write authority and the learning
environment switch; negative cases omit each prerequisite independently.

Acceptance: one repository-independent fixture traverses
`UNSEEN -> OBSERVE -> SHADOW -> TRIAL -> QUALIFIED -> ACTIVE`, plus expiry,
drift, regression, revocation and suspension paths, through `./buildoptw` rather
than benchmark-only commands. Every transition and gross/cost/net value is
visible through `status --json` and recomputes from immutable evidence. The
same checker must prove mode `observe`, mode `off`, bypass, missing write
authority and exhausted budget retain native Gradle without an action.

Closed by `SWL-E020`. The launcher now owns one composition root and explicit
active/trial executor adapters; conservative value uses checked integer
arithmetic and complete costs. The committed fixture traverses all seven
required transitions, reports paired value and a signed-ledger-shaped status,
then suspends and retires a regression. All ten negative cases retain native
execution. This is deterministic lifecycle evidence only, not a public-build
speedup claim.

### SWL-014C — Public opportunity and activation pre-gate

Run the composed generic detectors over the five frozen public families without
repository-name, task-name, path-extension or hand-authored profile rules. This
is an opportunity screen, not the longitudinal value campaign. Runtime actions
must be executable through the wrapper; durable actions remain review-only and
must have an isolated apply/revert transaction.

The runner executes only `TASK_CONTRACT_JAVA_V1` and
`DECLARED_GRAPH_SCOPE_V1` in the declared order and writes
`benchmarks/results/sticky-wrapper-opportunity-gate-v1.json`. The independent
checker recomputes recurrence, projected saving, all costs and repayment from
raw evidence. An unsupported family is retained with an explicit reason; it is
not replaced after results are seen.

Acceptance: at least three of five families expose one independently testable
action with a complete output contract and enough projected compatible builds
to repay its bounded trial. If fewer than three qualify, stop the experiment
without spending the 75-or-more builds required by `SWL-015 v2`, using the
early-stop state machine above.

### SWL-014D — Installed active-path value gate

For every action admitted by `SWL-014C`, run balanced candidate/native pairs
through the installed `./buildoptw` surface with identical native-cache
opportunity. The candidate must consume the actual signed decision or reviewed
durable transaction that would be used later; direct benchmark-only runners do
not qualify.

Every action receives exactly eight alternating pairs and is evaluated by
`internal/stickyvalue` using the frozen LCG interval and nearest-rank arm p95.
The runner writes `benchmarks/results/sticky-wrapper-active-value-v1.json`; the
checker recomputes the interval, p95, ledger reconciliation and family count
from raw pairs. No failed action may be substituted after timing.

Acceptance: exact/reviewed outputs, zero product-attributable failures,
positive mean saving, positive conservative lower bound, no p95 regression,
complete learning/execution cost attribution and proven native fallback. At
least three families must pass before `SWL-015 v2` opens. A failed action is
suspended or retired and receives zero value credit. Fewer than three uses the
early-stop state machine instead of opening the campaign.

### SWL-015 — Frozen public longitudinal campaign v2

Before timing, freeze one current package, five substantial repository
families, first-parent primary/reserve commits, workflows, required outputs,
JDKs, Wrapper distributions, network/cache opportunity and trial budget. Use at
least 20 comparable requested builds per family. Preserve all positive and
negative observations, including builds with no action.

Acceptance: every row has at least 15 valid comparable builds, exact/reviewed
outputs, complete phase attribution, deterministic checkpoints and a signed
economic ledger. Candidate rows identify the selected lifecycle decision and
whether a runtime or durable action actually executed. No repository-specific
product rule or post-result threshold change is allowed.

The per-family confidence calculation is the frozen deterministic circular
moving-block bootstrap above. The result directory contains raw rows,
exclusions, immutable checkpoints, signed ledgers, a report and their digests;
`dev/check-sticky-wrapper-longitudinal-v2` independently rebuilds every
aggregate before it can emit `READY_FOR_SWL_016`.

Historical diagnostic: [`poc-sticky-wrapper-longitudinal-sample-v1`](../../benchmarks/results/poc-sticky-wrapper-longitudinal-sample-v1/README.md)
completed one `CONTROL_FIRST` pair in each frozen family. All five pairs
completed successfully with exact required outputs and zero product failures;
two were positive and three were negative, for a signed portfolio delta of
**-22.149 seconds**. The control added `--build-cache`, the candidate did not,
and the candidate selected no-op/light observation with no trial or action.
This is useful wrapper-compatibility evidence, but it is `DIAGNOSTIC_ONLY`, is
not part of v2 and can never feed `SWL-016`.

The first bounded execution also exposed a harness-only final-aggregation
failure after all five subject records had been written. The raw subject
records were retained and the checked `raw.json`/`report.json` were regenerated
from those records without changing any observation. The runner now passes its
multi-line jq programs as explicit arguments, and that finalization path has
been validated against the retained records. No subsequent v1 run is required;
the v2 preflight supersedes that harness path before new timing is accepted.

### SWL-016 — Terminal decision

Recompute the frozen scorecard from SWL-015 v2 only when the campaign ran. If
SWL-014C or SWL-014D produced stop evidence, consume that evidence and the
explicit skipped-block states instead. Historical mechanism results are
context, not passing inputs. Report wrapper usability separately from
acceleration and name every failed criterion.

Acceptance: an independent checker reproduces the result and rejects edited
thresholds, omitted builds, altered outputs or arithmetic. The decision is
exactly `CONTINUE_STICKY_WRAPPER_LEARNING_POC` or
`STOP_STICKY_WRAPPER_LEARNING_POC`.

## Documentation update contract

Every block updates the owning documentation in the same commit when its
customer behavior, architecture, value evidence or direction changes.

| Document | Required content | Blocks |
| --- | --- | --- |
| [Master RFC](../../gradle-build-optimization-platform.md) | Stable customer promise, authority, current implementation truth and architecture decisions | SWL-000, SWL-001, SWL-007, SWL-014A, SWL-016 |
| [Implementation tracker](../../implementation-tracker.md) | Active phase, progress, evidence and pointer here | Every completed block |
| [Specifications index](../../specs/README.md) | New contract, checker, authority and POC boundary | SWL-000, SWL-001, SWL-007, SWL-014A..016 |
| [POC one-pager](../findings/buildopt-poc-handoff.md) | Current idea, latest data, conclusion and next step only | SWL-000, SWL-014, SWL-014A..016 |
| [Performance findings](../findings/build-optimization-performance.md) | Attributable action and complete-path timing | SWL-008, SWL-011..016 |
| [Generalization audit](../findings/buildopt-generalization-audit.md) | Generic detector coverage, activation breadth and lifetime | SWL-012, SWL-014C..016 |
| [Architecture](../architecture/overview.md) | Wrapper, bootstrap, cache/state planes and the implemented decision lifecycle | SWL-000, SWL-003, SWL-006..008, SWL-014B |
| [Repository map](../architecture/repository-map.md) | Owning source paths and executable checks | SWL-002..014B |
| [Product onboarding](../getting-started/product-onboarding.md) | Generated files, one command, token boundary, bypass and implemented behavior | SWL-000, SWL-002..005, SWL-013, SWL-014, SWL-014B |
| [Product workflows](../guides/product-workflows.md) | Implemented and planned observe/shadow/trial/active/suspend and durable patch flow | SWL-000, SWL-009..013, SWL-014B..015 |
| [CLI reference](../reference/cli.md) | Exact wrapper/init/status/explain syntax and behavior | SWL-000..004, SWL-013 |
| [Configuration reference](../reference/configuration.md) | Committed non-secret config, private credentials, defaults and scopes | SWL-000, SWL-001, SWL-005..008 |
| [Central cache/state roadmap](./centralized-cache-and-state-roadmap.md) | Reused infrastructure and wrapper integration, never merged planes | SWL-000, SWL-005..007, SWL-014, SWL-014B |
| [Root README](../../README.md) | Short current direction, customer command and evidence link | SWL-000, SWL-014..016 |

## Evidence registry

| Evidence | Block | Description | State |
| --- | --- | --- | --- |
| `SWL-E001` | SWL-000 | Frozen hypothesis, machine contract, detailed tracker and aligned repository documentation | `DONE` |
| `SWL-E002` | SWL-001 | [Repository wrapper protocol](../../specs/poc-sticky-wrapper-contract-v1.md), exact [machine contract](../../specs/poc-sticky-wrapper-contract-v1.json), portable [fixtures](../../fixtures/sticky-wrapper-contract/README.md) and executable [`check-sticky-wrapper-contract`](../../dev/check-sticky-wrapper-contract): four paths, strict properties/config grammars, POSIX/Windows agreement, 13 negative cases, argument routing, bypass and downgrade semantics | `DONE` |
| `SWL-E003` | SWL-002 | [Generator contract](../../specs/poc-sticky-wrapper-generator-v1.md), `internal/stickywrapper`, executable [`check-sticky-wrapper-generator`](../../dev/check-sticky-wrapper-generator), deterministic/read-only/drift/update/downgrade/concurrency/rollback tests, portable compilation and real public-metadata smoke | `DONE` |
| `SWL-E004` | SWL-003 | [Bootstrap contract](../../specs/poc-sticky-wrapper-bootstrap-v1.md), embedded POSIX/Windows templates and executable [`check-sticky-wrapper-bootstrap`](../../dev/check-sticky-wrapper-bootstrap): checksum-pinned download, safe extraction, internal manifest, atomic concurrent publication, verified offline reuse, public package smokes and native platform CI gates | `DONE` |
| `SWL-E005` | SWL-004 | [Neutral passthrough contract](../../specs/poc-sticky-wrapper-passthrough-v1.md), machine-readable [boundary](../../specs/poc-sticky-wrapper-passthrough-v1.json), executable [`check-sticky-wrapper-passthrough`](../../dev/check-sticky-wrapper-passthrough) and native platform gates: difficult argv/cwd/streams/environment, ordinary and signal exits, descendants, pre-bootstrap bypass and fail-open Gradle fallback | `DONE` |
| `SWL-E006` | SWL-005 | [Portable connection contract](../../specs/poc-sticky-wrapper-connection-v1.md), machine-readable [scope boundary](../../specs/poc-sticky-wrapper-connection-v1.json) and executable [`check-sticky-wrapper-connection`](../../dev/check-sticky-wrapper-connection): two clean checkouts share one path-independent project/connection identity; namespace changes separate it; absent/cross-project/incomplete credentials avoid the server; live revocation blocks both planes; redirects and foreign wrapper roots reject; the dynamic secret is scrubbed before Gradle | `DONE` |
| `SWL-E007` | SWL-006 | [Sticky-wrapper Gradle cache contract](../../specs/poc-sticky-wrapper-cache-v1.md), machine contract, automatic native cache flag, read-only central policy and race-enabled producer/consumer/corruption/outage integration | `DONE` |
| `SWL-E008` | SWL-007 | [Decision-store specification](../../specs/poc-sticky-wrapper-decision-store-v1.md), JSON Schema union and fixtures, `internal/stickydecision`, and executable [`check-sticky-wrapper-decision-store`](../../dev/check-sticky-wrapper-decision-store): all valid state transitions, signed decisions, evidence/ledger references, local and central generation CAS, replay/conflict, expiry, revocation, corruption and cache/state plane separation | `DONE` |
| `SWL-E009` | SWL-008 | [`poc-sticky-wrapper-noop-v1`](../../specs/poc-sticky-wrapper-noop-v1.md), read-only selector, fail-closed state matrix and [`sticky-wrapper-noop-v1.json`](../../benchmarks/results/sticky-wrapper-noop-v1.json): 200 verified local, missing and no-synchronous-refresh selections stay below the 100 ms p50, 250 ms p95 and 500 ms fallback budgets | `DONE` |
| `SWL-E018` | SWL-008A | [`poc-sticky-wrapper-noop-overhead-v1`](../../specs/poc-sticky-wrapper-noop-overhead-v1.md), native sticky-wrapper fast path, lightweight process supervisor, lazy light observer, focused launcher tests and [`sticky-wrapper-noop-overhead-v1.json`](../../benchmarks/results/sticky-wrapper-noop-overhead-v1.json): 20 interleaved Linux samples record 9 ms p95 native no-op overhead, 38 ms p95 light-observation overhead and 0.093 ms p95 pre-child decision time; the light executable digest runs concurrently when possible; process-group signal forwarding remains covered; all 100/250/100 ms POC guardrails pass | `DONE` |
| `SWL-E010` | SWL-009 | [`poc-sticky-wrapper-observation-v1`](../../specs/poc-sticky-wrapper-observation-v1.md), private append-only recorder, real Wrapper checker and [`sticky-wrapper-observation-v1.json`](../../benchmarks/results/sticky-wrapper-observation-v1.json): two successful Gradle 9.6.1 observations with Configuration Cache present, exact phase reconciliation, provenance hashes, child-environment scrubbing and tamper rejection; 19.876 s cold and 3.732 s reuse wall time | `DONE` |
| `SWL-E011` | SWL-010 | [`poc-sticky-wrapper-trial-v1`](../../specs/poc-sticky-wrapper-trial-v1.md), bounded scheduler, direct isolated runner and [`sticky-wrapper-trial-v1.json`](../../benchmarks/results/sticky-wrapper-trial-v1.json): four alternating candidate/native pairs, eight invocations, eight private roots, exact output hashes, 58.050 s used of a 180 s ceiling; candidate 7.534 s versus native 6.979 s, 0/4 positive | `DONE` |
| `SWL-E012` | SWL-011 | [`poc-sticky-wrapper-active-v1`](../../specs/poc-sticky-wrapper-active-v1.md), generic active runner and [`sticky-wrapper-active-v1.json`](../../benchmarks/results/sticky-wrapper-active-v1.json): current negative trial rejected, one exact synthetic active execution, three suspensions, four native retentions, signed binding revalidation and direct-command fallback | `DONE` |
| `SWL-E013` | SWL-012 | Durable detector/patch breadth and value evidence | `DONE` |
| `SWL-E014` | SWL-013 | Recomputable customer status/explanation contract | `DONE` |
| `SWL-E015` | SWL-014 | [Two-machine installed wrapper evidence](../../benchmarks/results/sticky-wrapper-two-machine-v1.json), verified bootstrap, owner-commit visibility, read-only central cache hit, output equality and native outage fallback | `DONE` |
| `SWL-E016` | SWL-015 v1 | [Bounded five-family diagnostic sample](../../benchmarks/results/poc-sticky-wrapper-longitudinal-sample-v1/README.md): 5/5 compatible exact-output pairs, 2/5 positive, 3/5 negative and -22.149 s signed total; cache-asymmetric no-op/light-observation evidence only | `DONE — DIAGNOSTIC_ONLY` |
| `SWL-E017` | SWL-016 | Immutable terminal scorecard and continue/stop decision | `WAITING` |
| `SWL-E019` | SWL-014A | Cache-symmetric v2 protocol, historical-v1 classification, locked autonomous execution contract, canonical zero-pair preflight, lifecycle-aware checker and four exact negative fixtures | `DONE` |
| `SWL-E020` | SWL-014B | Real wrapper-driven lifecycle composition and recomputable transition/economic evidence | `DONE` — `./dev/check-sticky-wrapper-learning-lifecycle`; seven transitions, four exact profitable fixture pairs, ten native-fallback negatives and reconciled signed economics |
| `SWL-E021` | SWL-014C | Five-family generic opportunity screen and at-least-three-family activation pre-gate | `TODO` |
| `SWL-E022` | SWL-014D | Installed active-path candidate/native value evidence for every admitted action | `WAITING` |
| `SWL-E023` | SWL-015 v2 | Complete chronological lifecycle, action and cumulative-value campaign evidence | `WAITING` |

## Risks and stop conditions

| Risk | Required response |
| --- | --- |
| Wrapper hides or changes Gradle behavior | Stop before cache integration; passthrough equivalence is mandatory |
| Network access becomes part of every no-op decision | Redesign the local snapshot path; do not raise the overhead gate |
| Cache objects become action authority | Fail the experiment; separate protocols and credentials are mandatory |
| Observation breaks Configuration Cache or exceeds budget | Disable that observation class and retain native Gradle |
| Trials consume ordinary developer time | Move them to isolated CI or stop the candidate |
| A campaign arm receives a different native-cache opportunity | Reject the protocol before timing; never explain it after the result |
| Benchmark-only packages are mistaken for customer-path composition | Keep the block open until `./buildoptw` traverses the lifecycle end to end |
| Fewer than three families expose an independently testable action | Stop at SWL-014C; do not spend the longitudinal campaign budget |
| Wall-time rows omit lifecycle, action or ledger evidence | Keep the campaign incomplete regardless of sample count |
| A patch detector needs repository-name rules | Reject that detector class |
| A runtime profile is safe but not cumulatively positive | Suspend it; safety is not value |
| Only cache-off controls show value | Report cache enablement, not BuildOpt acceleration |
| Fewer than three families are net positive | Terminal outcome is stop, regardless of isolated wins |
| Correctness failure or additional product failure | Stop the affected action immediately and fail the terminal criterion |

## Changelog

| Date | Change |
| --- | --- |
| 2026-08-27 | Closed SWL-014A. Added the canonical zero-pair v2 preflight runner/checker, exact valid and four negative fixtures, separate planned writable-root and remote-namespace identities, identical frozen workflow/cache/service/empty-seed bindings, explicit authorized learning fields and a checked `VALIDATED_NOT_READY` result. Moved final campaign-mode modification of the same scripts into the exact SWL-015 manifest because lifecycle/action evidence does not exist before SWL-014B..D; opened SWL-014B. |
| 2026-08-27 | Closed SWL-014B. Added the sole launcher composition root, launcher-owned active/trial process adapters, explicit trusted-learning eligibility and scrubbing, checked `stickyvalue` statistics, lifecycle/status evidence and ten fail-closed cases. Opened SWL-014C; no public-family timing or activation breadth is claimed by the deterministic fixture. |
| 2026-08-27 | Hardened the corrected sticky-wrapper route into an autonomous execution contract. Froze separate initially empty writable cache namespaces, per-arm chronological persistence, an external monotonic timing boundary, the launcher composition root and trusted-learning trigger, the two allowed generic detectors, exact evidence schemas, deterministic installed/longitudinal statistics, exact file/check manifests and conditional early-stop transitions into SWL-016. SWL-014A remains open only for its v2 runner, checker, five fixtures and checked zero-pair preflight. |
| 2026-08-27 | Audited the sticky-wrapper route before expanding SWL-015. The retained v1 sample compared control `--build-cache` with a no-op/light-observation candidate, while benchmark-only trial, active and durable packages were not composed into the normal wrapper path. Reclassified that sample as `DIAGNOSTIC_ONLY`, preserved every terminal threshold and inserted SWL-014A..D for comparison fairness, end-to-end composition, five-family opportunity breadth and installed active-path value before a versioned SWL-015 v2 campaign. |
| 2026-08-27 | Closed SWL-008A and hardened its signal boundary. Unconfigured invocations skip gateway, plugin handshake, managed L1, central-cache probes and bootstrap state; ordinary observation defaults to lazy light mode, with explicit full and disabled modes. The light executable digest runs concurrently when possible instead of delaying startup. The no-op path now retains the lightweight process supervisor so descendant signal forwarding and cleanup remain intact. The regenerated 20-sample Linux microbenchmark records +9 ms p95 native no-op overhead, +38 ms p95 light-observation overhead and 0.093 ms p95 pre-child decision time, all within the frozen guardrails. The passthrough fixture also exercises the unconfigured no-op path and bounded signal cleanup. This is wrapper-cost evidence only; no Gradle speedup or action authority is claimed. |
| 2026-08-27 | Closed SWL-014. The committed sticky wrapper bootstrapped one verified archive in isolated producer/consumer containers, published two Gradle cache objects over HTTPS, kept pending reads invisible until owner commit, restored two tasks from a clean read-only machine, and rebuilt the same output during service outage. Durations were recorded without a wall-time or profile-qualification claim. This originally opened SWL-015 v1; the later route audit superseded that sequence with SWL-014A..D. |
| 2026-08-27 | Captured the first SWL-015 v1 diagnostic sample across Spring Framework, OpenTelemetry Java Instrumentation, Apache Kafka, Micronaut Core and Apache Groovy. All five candidate/control pairs produced exact required outputs with zero product failures; two were positive and three negative for -22.149 seconds signed value. The later route audit classified it as cache-asymmetric no-op evidence that cannot feed SWL-016. |
| 2026-08-27 | Closed SWL-013. Added read-only customer `status` and `explain` management commands to both generated wrapper templates, with one recomputable human/JSON report model, explicit unavailable metrics, exact bindings, native fallback reasons and tamper rejection. Ordinary Gradle tasks named `status` or `explain` remain unambiguous. Opened SWL-014 two-machine installed proof. |
| 2026-08-27 | Closed SWL-012. Added the generic durable native catalog for task-contract and graph-breadth opportunities, exact reviewable recipes, isolated apply/revert proofs and a strict 4-CPU/16-GiB evidence report. The task-contract detector is shared by Kotlin and Groovy; the reviewed patch saves 64.1% and 74.7% respectively across 16/16 exact pairs. Graph proposals remain structural-only with durable timing unmeasured. Opened SWL-013 customer status and explanation. |
| 2026-08-27 | Closed SWL-011. Added the generic signed-decision active runner, direct candidate/native execution, exact required-output hashing, native counterfactual sampling, fail-closed suspension and native fallback. The checked-in SWL-010 report remains unauthorized because it is negative; synthetic control-flow evidence records one active execution, three suspensions and four native retentions. Opened SWL-012 durable native optimization catalog. |
| 2026-08-27 | Closed SWL-010. Added trusted-CI-only budgeted paired trials with balanced order, eight-root isolation, direct command execution, exact output hashing, cancellation/concurrency accounting and a 5% compute ceiling. The checked-in four-pair Gradle 8.14.3 result is exact but negative: 7.534 s candidate versus 6.979 s optimized native, 0/4 positive pairs and 58.050 s used of a 180 s ceiling. Opened SWL-011 to diagnose overhead before any active action. |
| 2026-08-27 | Closed SWL-009. Added the private append-only ordinary-build observation plane, phase timing/provenance schema and real Wrapper checker. Two Gradle 9.6.1 invocations with Configuration Cache present record 19.876 s cold and 3.732 s reuse wall time; exact timing reconciles and unavailable phases remain unavailable. Opened SWL-010 budgeted candidate/native trials. |
| 2026-08-27 | Closed SWL-008. Added a read-only local selector for signed, scope- and binding-compatible sticky decisions, coalesced best-effort refresh scheduling, fail-closed native fallback and a deterministic pre-Gradle budget benchmark. Two hundred Linux AMD64 selections record 0.492 ms verified-local p50 / 1.369 ms p95, with sub-0.003 ms missing and no-synchronous-refresh fallback; this is retention-cost evidence only, not acceleration. Opened SWL-009 ordinary-build observation. |
| 2026-08-27 | Closed SWL-007. Added the canonical JCS decision-store union, signed decision verification, independent qualification/rollout transitions, evidence and ledger cross-links, local filesystem and central EVIDENCE adapters with generation CAS, exact replay/conflict semantics, expiry, revocation, corruption and cache/state plane separation. Seven positive and six negative schema fixtures plus race-enabled local/central conformance pass; opened SWL-008 native no-op fast path. |
| 2026-08-26 | Closed SWL-006. Connected the committed wrapper to the existing Gradle-compatible central object plane through an invocation-local verifying gateway. A valid read-capable connection automatically enables Gradle's native HTTP cache unless explicitly disabled, preserves the native cache policy for central tasks, keeps ordinary consumers read-only, and retains private managed L1 separation. Race-enabled producer/consumer/outage integration proves exact `FROM-CACHE` outputs, byte-free corruption misses, identical native rebuild outputs, secret isolation and no central PUT from the consumer; opened SWL-007 decision contract and store. |
| 2026-08-26 | Closed SWL-005. Bound canonical committed endpoint/project scope to the private owner-issued token document, derived checkout-independent project identity plus namespace-specific connection identity, required both read capabilities, honored expiry/live revocation and scrubbed the dynamic credential before Gradle. Two-checkout HTTPS/race evidence and cross-project, missing, capability, namespace, redirect, foreign-root and revoked negatives pass without consuming cache/state; opened SWL-006. |
| 2026-08-26 | Closed SWL-004. Ordinary wrapper calls now execute the repository Gradle Wrapper through the verified `buildopt run --` launcher with exact argv/cwd/streams/environment/exit behavior; `--gradle` escapes management, exact bypass runs before configuration/bootstrap, ordinary bootstrap failures warn and run direct Gradle, and management fails closed. Linux difficult-argument and signal-tree fixtures plus native macOS/Windows entrypoint gates pass without activating any optimization; opened SWL-005. |
| 2026-08-26 | Closed SWL-003. Implemented checksum-pinned POSIX/Windows bootstrap for four native packages, safe extraction and internal manifest verification, atomic concurrent user-cache publication and verified zero-network reuse. Synthetic negatives, race/vet, ShellCheck, PowerShell parsing and public `v0.6.1` Linux/Windows-body smokes pass. A real GitHub release-asset redirect corrected the frozen policy to at most five HTTPS-only redirects without weakening the archive checksum authority; opened SWL-004. |
| 2026-08-26 | Closed SWL-002. Implemented deterministic `wrapper init`, offline/read-only `check` and distribution-only `update`; bound real immutable GitHub release digests without archive download, preserved owner files, rejected drift/downgrade/concurrency and proved full rollback. Linux race/vet, Windows/macOS compilation and public `v0.6.1` metadata smoke pass; opened SWL-003. |
| 2026-08-26 | Closed SWL-001. Frozen four portable file formats, strict ordered properties and flat-TOML configuration, four platform distributions, HTTPS/checksum/timeouts/proxy/redirect rules, unambiguous `--buildopt`/`--gradle` routing, pre-bootstrap bypass, update/downgrade semantics and fixed error behavior. Both parser shapes accept the canonical fixture and reject all 13 negative cases; opened SWL-002. |
| 2026-08-26 | Opened the successor POC after AF-015. Frozen the repository-committed wrapper surface, separate Gradle-cache and typed-state planes, lifecycle, budgets, original sequence and terminal value scorecard; completed SWL-000 and opened SWL-001. |
