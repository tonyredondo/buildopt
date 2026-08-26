# Sticky Wrapper Learning POC Tracker

## Status

**Overall:** `IN_PROGRESS`<br>
**Progress:** `6/17` blocks complete<br>
**Current block:** `SWL-006` — Gradle HTTP cache integration<br>
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
| 6 | `SWL-006` Gradle HTTP cache integration | Existing verifying gateway and central cache operate automatically through the wrapper | `TODO` | SWL-005 |
| 7 | `SWL-007` Decision contract and store | Typed state machine, signed decisions, generation CAS, expiry, revocation and plane separation | `WAITING` | SWL-005 |
| 8 | `SWL-008` Native no-op fast path | Exact local snapshot makes native retention independent of a blocking remote lookup | `WAITING` | SWL-007 |
| 9 | `SWL-009` Ordinary-build observation | Bounded exact timings, outputs, task/graph facts and ledger updates from requested builds | `WAITING` | SWL-008 |
| 10 | `SWL-010` Budgeted trial orchestration | Isolated paired candidate/native trials run only within the frozen CI compute budget | `WAITING` | SWL-009 |
| 11 | `SWL-011` Active execution and suspension | Qualified runtime action, revalidation, counterfactual sampling, regression suspension and native fallback | `WAITING` | SWL-010 |
| 12 | `SWL-012` Durable native optimization catalog | Generic detectors and reviewable patches for task contracts and graph breadth | `WAITING` | SWL-009 |
| 13 | `SWL-013` Customer status and explanation | Human/JSON decision, cumulative economics, cache metrics and exact fallback explanation | `WAITING` | SWL-011, SWL-012 |
| 14 | `SWL-014` Two-machine installed proof | Clean producer/consumer checkouts share cache/state through HTTPS and survive outage | `WAITING` | SWL-013 |
| 15 | `SWL-015` Frozen public longitudinal campaign | Current installed wrapper over preregistered chronological windows in five families | `WAITING` | SWL-014 |
| 16 | `SWL-016` Terminal decision | Recompute the immutable scorecard and continue or stop without threshold movement | `WAITING` | SWL-015 |

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

### SWL-007 — Decision contract and store

Define canonical action, observation, trial, decision and economic-ledger
documents. Decisions bind repository, workflow, Gradle, Wrapper, options,
outputs, action generation, expiry and revocation. Gradle object keys are never
accepted as action authority.

Acceptance: lifecycle vectors cover every valid transition plus replay,
conflict, stale generation, expiry, corruption, cross-plane and cross-scope
negative cases in local and central storage.

### SWL-008 — Native no-op fast path

Cache only signed, verified and expiry-bound decisions locally. A current
`NATIVE_NOOP` or compatible `ACTIVE` decision is read without blocking on the
server; synchronization occurs outside the critical decision path. Unknown or
expired state returns native and may schedule observation asynchronously.

Acceptance: the target benchmark class measures at most 100 ms p50 and 250 ms
p95 pre-Gradle overhead; service outage remains at most 500 ms p95; corrupt or
incompatible snapshots fail closed without executing an action.

### SWL-009 — Ordinary-build observation

Capture only facts needed by registered hypothesis classes. Attribute wrapper,
bootstrap, decision, network, cache, observation and Gradle time separately.
Never transform unavailable evidence into zero. Ordinary user builds remain the
only source of natural outcomes.

Acceptance: bounded observation preserves Configuration Cache, reconciles wall
time, emits exact provenance and can be disabled independently when its p95
budget is exceeded.

### SWL-010 — Budgeted trial orchestration

Schedule isolated candidate/native trials in trusted CI only when projected
compatible lifetime can repay them. The customer's requested result remains
authoritative. Trials use separate checkout, Gradle home, daemon, cache and
BuildOpt state and cannot exceed 5% of natural runner-minutes.

Acceptance: deterministic fixtures prove scheduling, budget exhaustion,
concurrency, cancellation, exact outputs, order balance, no-lookahead evidence
and zero hidden local duplicate builds.

### SWL-011 — Active execution and suspension

An exact qualified runtime profile can become `ACTIVE`; every invocation still
revalidates bindings. A bounded native counterfactual measures ongoing value.
One correctness failure, revoked authority, incompatible drift or decisive
negative value suspends before another candidate execution.

Acceptance: exact active, drift, expiry, server outage, cache miss, regression,
cancellation and bypass cases preserve outputs and original exit behavior; only
measured active executions receive saving attribution.

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

### SWL-013 — Customer status and explanation

`./buildoptw status` reports current decision, active action, observations,
trials, native fallbacks, cache usage, gross saving, every cost and signed net
value. `./buildoptw explain` reports exact bindings and the reason an action
did or did not execute. JSON recomputes the human output.

Acceptance: no tracker vocabulary or secret is required to understand the
result; missing evidence stays unavailable; percentages from different actions
are never added; tampering or arithmetic mismatch rejects.

### SWL-014 — Two-machine installed proof

Generate and commit the wrapper in an external fixture repository. A trusted
producer and clean consumer use the public package, central HTTPS service and
separate credentials. Exercise online, offline snapshot, outage, revocation,
cache corruption and decision drift.

Acceptance: the customer command is only `./buildoptw <args>`; exact outputs,
private credentials, cache/state separation and native fallback pass on two
isolated machines or containers and the native OS CI matrix validates wrapper
bootstrap behavior.

### SWL-015 — Frozen public longitudinal campaign

Before timing, freeze one current package, five substantial repository
families, first-parent primary/reserve commits, workflows, required outputs,
JDKs, Wrapper distributions, network/cache opportunity and trial budget. Use at
least 20 comparable requested builds per family. Preserve all positive and
negative observations, including builds with no action.

Acceptance: every row has at least 15 valid comparable builds, exact/reviewed
outputs, complete phase attribution, deterministic checkpoints and a signed
economic ledger. No repository-specific product rule or post-result threshold
change is allowed.

### SWL-016 — Terminal decision

Recompute the frozen scorecard from SWL-015 only. Historical mechanism results
are context, not passing inputs. Report wrapper usability separately from
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
| [Master RFC](../../gradle-build-optimization-platform.md) | Stable customer promise, authority and architecture decisions | SWL-000, SWL-001, SWL-007, SWL-016 |
| [Implementation tracker](../../implementation-tracker.md) | Active phase, progress, evidence and pointer here | Every completed block |
| [Specifications index](../../specs/README.md) | New contract, checker, authority and POC boundary | SWL-000, SWL-001, SWL-007, SWL-015, SWL-016 |
| [POC one-pager](../findings/buildopt-poc-handoff.md) | Current idea, latest data, conclusion and next step only | SWL-000, SWL-014, SWL-015, SWL-016 |
| [Performance findings](../findings/build-optimization-performance.md) | Attributable action and complete-path timing | SWL-008, SWL-011..016 |
| [Generalization audit](../findings/buildopt-generalization-audit.md) | Generic detector coverage, activation breadth and lifetime | SWL-012, SWL-015, SWL-016 |
| [Architecture](../architecture/overview.md) | Wrapper, bootstrap, cache/state planes and decision lifecycle | SWL-000, SWL-003, SWL-006..008 |
| [Repository map](../architecture/repository-map.md) | Owning source paths and executable checks | SWL-002..013 |
| [Product onboarding](../getting-started/product-onboarding.md) | Generated files, one command, token boundary, bypass and first result | SWL-000, SWL-002..005, SWL-013, SWL-014 |
| [Product workflows](../guides/product-workflows.md) | Observe/shadow/trial/active/suspend and durable patch flow | SWL-000, SWL-009..013 |
| [CLI reference](../reference/cli.md) | Exact wrapper/init/status/explain syntax and behavior | SWL-000..004, SWL-013 |
| [Configuration reference](../reference/configuration.md) | Committed non-secret config, private credentials, defaults and scopes | SWL-000, SWL-001, SWL-005..008 |
| [Central cache/state roadmap](./centralized-cache-and-state-roadmap.md) | Reused infrastructure and wrapper integration, never merged planes | SWL-000, SWL-005..007, SWL-014 |
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
| `SWL-E007` | SWL-006 | Central Gradle HTTP cache producer/consumer/outage proof | `WAITING` |
| `SWL-E008` | SWL-007 | Decision lifecycle, signed storage and cross-plane negatives | `WAITING` |
| `SWL-E009` | SWL-008 | No-op and unavailable-service overhead evidence | `WAITING` |
| `SWL-E010` | SWL-009 | Ordinary-build observation and phase attribution | `WAITING` |
| `SWL-E011` | SWL-010 | Bounded trial scheduling and isolation evidence | `WAITING` |
| `SWL-E012` | SWL-011 | Active, counterfactual, suspension and fallback proof | `WAITING` |
| `SWL-E013` | SWL-012 | Durable detector/patch breadth and value evidence | `WAITING` |
| `SWL-E014` | SWL-013 | Recomputable customer status/explanation contract | `WAITING` |
| `SWL-E015` | SWL-014 | Two-machine installed wrapper evidence | `WAITING` |
| `SWL-E016` | SWL-015 | Frozen five-family longitudinal campaign | `WAITING` |
| `SWL-E017` | SWL-016 | Immutable terminal scorecard and continue/stop decision | `WAITING` |

## Risks and stop conditions

| Risk | Required response |
| --- | --- |
| Wrapper hides or changes Gradle behavior | Stop before cache integration; passthrough equivalence is mandatory |
| Network access becomes part of every no-op decision | Redesign the local snapshot path; do not raise the overhead gate |
| Cache objects become action authority | Fail the experiment; separate protocols and credentials are mandatory |
| Observation breaks Configuration Cache or exceeds budget | Disable that observation class and retain native Gradle |
| Trials consume ordinary developer time | Move them to isolated CI or stop the candidate |
| A patch detector needs repository-name rules | Reject that detector class |
| A runtime profile is safe but not cumulatively positive | Suspend it; safety is not value |
| Only cache-off controls show value | Report cache enablement, not BuildOpt acceleration |
| Fewer than three families are net positive | Terminal outcome is stop, regardless of isolated wins |
| Correctness failure or additional product failure | Stop the affected action immediately and fail the terminal criterion |

## Changelog

| Date | Change |
| --- | --- |
| 2026-08-26 | Closed SWL-005. Bound canonical committed endpoint/project scope to the private owner-issued token document, derived checkout-independent project identity plus namespace-specific connection identity, required both read capabilities, honored expiry/live revocation and scrubbed the dynamic credential before Gradle. Two-checkout HTTPS/race evidence and cross-project, missing, capability, namespace, redirect, foreign-root and revoked negatives pass without consuming cache/state; opened SWL-006. |
| 2026-08-26 | Closed SWL-004. Ordinary wrapper calls now execute the repository Gradle Wrapper through the verified `buildopt run --` launcher with exact argv/cwd/streams/environment/exit behavior; `--gradle` escapes management, exact bypass runs before configuration/bootstrap, ordinary bootstrap failures warn and run direct Gradle, and management fails closed. Linux difficult-argument and signal-tree fixtures plus native macOS/Windows entrypoint gates pass without activating any optimization; opened SWL-005. |
| 2026-08-26 | Closed SWL-003. Implemented checksum-pinned POSIX/Windows bootstrap for four native packages, safe extraction and internal manifest verification, atomic concurrent user-cache publication and verified zero-network reuse. Synthetic negatives, race/vet, ShellCheck, PowerShell parsing and public `v0.6.1` Linux/Windows-body smokes pass. A real GitHub release-asset redirect corrected the frozen policy to at most five HTTPS-only redirects without weakening the archive checksum authority; opened SWL-004. |
| 2026-08-26 | Closed SWL-002. Implemented deterministic `wrapper init`, offline/read-only `check` and distribution-only `update`; bound real immutable GitHub release digests without archive download, preserved owner files, rejected drift/downgrade/concurrency and proved full rollback. Linux race/vet, Windows/macOS compilation and public `v0.6.1` metadata smoke pass; opened SWL-003. |
| 2026-08-26 | Closed SWL-001. Frozen four portable file formats, strict ordered properties and flat-TOML configuration, four platform distributions, HTTPS/checksum/timeouts/proxy/redirect rules, unambiguous `--buildopt`/`--gradle` routing, pre-bootstrap bypass, update/downgrade semantics and fixed error behavior. Both parser shapes accept the canonical fixture and reject all 13 negative cases; opened SWL-002. |
| 2026-08-26 | Opened the successor POC after AF-015. Frozen the repository-committed wrapper surface, separate Gradle-cache and typed-state planes, lifecycle, budgets, 17-block sequence and terminal value scorecard; completed SWL-000 and opened SWL-001. |
