# BuildOpt POC: One-Page Handoff

## The idea

BuildOpt tests whether a generic, one-command layer can reduce wall time for
substantial Gradle workflows beyond an already optimized native Gradle
baseline. Gradle remains the execution engine and safe fallback. BuildOpt
learns the relationship between a Git change, the requested workflow, the task
graph and its outputs; when the evidence is strong enough, it rebuilds only the
necessary graph and restores verified unaffected outputs.

The current onboarding hypothesis uses four generated portable repository
files, but no hand-authored optimization profile:

```text
generate and commit BuildOpt Wrapper once
./buildoptw <the existing Gradle workflow>
```

This is an owner-operated proof of concept. The current question is whether
the idea creates repeatable customer value across ordinary builds and commits,
not whether it is ready for production. Soak, design-partner validation,
production SLOs, autonomous rollout and Test Optimization are outside this
decision.

## Current experiment status

There is deliberately no new performance percentage. The latest complete
change-aware producer has 25/25 conclusive adjacent transitions across all five
public families, but only Spring exposes an action: **1/5** versus the fixed
**3/5** pre-timing gate. Cause reconstruction explains the failure:
**23/24** negative transitions change
no declared input in the fixed requested graph, and the remaining Groovy row
uses a stale unversioned required JAR while the current producer emits a
versioned JAR. The stopped detector paired arbitrary commits with one fixed
leaf workflow; those rows cannot honestly become optimization actions.

The latest experimental route was
`REQUEST_ALIGNED_RECURRENT_CLOSURE_V1`. BuildOpt learned from the exact
ordinary Gradle command a customer repeats, wait for changes relevant to that
requested graph and discover current outputs from producer tasks. Its first
two implementation blocks now pass Gradle 8/9 × Kotlin/Groovy: canonical identity
is independent of checkout path, all eight graph-affecting bindings change the
identity, the versioned Groovy JAR is resolved from `:jar`, and missing,
ambiguous or outside-graph ownership fails typed. The adjacent-transition
classifier adds 32 cases across all five statuses, derives exact relevant
closures, binds current renamed outputs and emits no action for unsafe or
full-graph cases. The fresh 110-transition capture completes Groovy and Spring
but not Kafka, Micronaut or OpenTelemetry. Independent reconstruction now
verifies all 110 reports and confirms **2/5** complete inputs versus required
**5/5**, and **2/5** action families versus required **3/5**, with zero product
failures. Its terminal scorecard passes capture integrity, **110/110** report
reconstruction and exact request preservation, and **10/10** static output
bindings, but it stops the detector at those frozen breadth failures. No
candidate ran and no timing row exists. Installed value, chronological value,
confidence, payback and overhead remain `NOT_MEASURED_NOT_AUTHORIZED`; no
speedup or successor is authorized.

## Mechanisms

| Mechanism | Purpose | Historical diagnostic evidence |
| --- | --- | --- |
| **Structural Build Impact** | Derive the smallest sufficient Gradle graph for the exact change and requested workflow. | Produces large isolated target wins, including Kafka's current **21.43%** qualified target saving, but compatible descendants are much rarer than target calibrations. |
| **Verified output materialization** | Restore byte-exact outputs from unaffected producers so the reduced graph remains correct. | Fail-closed correctness is strong: the current five-repository run verifies 27 exact-output builds with zero product failures. |
| **Structural profile rebinding** | Reuse learned evidence only when Wrapper, workflow, producer lineage, output contract and change family remain compatible. | Safely rejects drift and selected one of six structurally eligible Kafka descendants in the current run. |
| **Ordinary-build learning economics** | Learn only from builds the user requested and stop when the expected compatible lifetime cannot repay discovery. | Four repositories stopped after one requested build and avoided 64 additional learning builds. |
| **Cross-repository hypothesis prior** | Use generic task implementation, plugin, Gradle and graph/output features to decide what to investigate first in a new repository. | Four opaque source scopes rank three hypothesis classes identically for two holdouts and after all repository identities are replaced. Priors never transfer correctness, value or activation. |
| **Durable patch-opportunity learning** | Detect repeated expensive task-contract problems, propose an owner-reviewed reversible source patch and validate it independently. | The strict current rerun accepts the same detector in Kotlin and Groovy: **64.1%** and **74.7%** faster respectively across **16/16** exact pairs. BuildOpt is not needed after acceptance; recipe coverage remains a POC signal, not customer coverage. |
| **Conflict-aware fragment planner** | Compose only qualified fragments whose dependencies, exclusions, authorities and direct joint economics remain valid; otherwise use native Gradle. | Direct timing now shows that the reviewed patch and Build Impact can save 68.56% Groovy and 79.32% Kotlin together, but Kotlin Build Impact reaches only 6/8 positive isolated pairs. The frozen constituent gate therefore retains qualified fragments instead of authorizing the composition. |
| **Local/HTTP cache and central state** | Carry verified task outputs and profiles between builds or machines. | Supporting infrastructure; useful for transport and persistence, but not the primary acceleration claim. |
| **Runtime Tuning, Hot State and standard Copy** | Earlier broad resource and state hypotheses. | Retired after neutral, unstable or regressive end-to-end evidence. |

## Discarded historical public-repository result

The previous installed package was run chronologically across 20 comparable
commits in each of Spring Framework, OpenTelemetry Java Instrumentation, Apache
Kafka, Micronaut Core and Apache Groovy. Each control/candidate pair used
separate persistent state, alternating order and byte-exact required-output
verification. Dependency preparation was outside pair wall time and copied no
task outputs or BuildOpt policy state.

| Repository | Positive pairs | Signed net | Historical outcome |
| --- | ---: | ---: | --- |
| Spring Framework | 2/20 | **-130.406 s** | `NET_NEGATIVE` |
| OpenTelemetry | 11/20 | **-26.044 s** | `INCONCLUSIVE` |
| Apache Kafka | 2/20 | **-42.309 s** | `NET_NEGATIVE` |
| Micronaut Core | 4/20 | **-144.656 s** | `NET_NEGATIVE` |
| Apache Groovy | 6/20 | **-25.207 s** | `NET_NEGATIVE` |

The current campaign closes at 100 exact-output pairs, 25 positive and 75
negative, zero product failures and **-368.623 seconds** cumulative signed
value. Every candidate retains native Gradle; no whole profile or adaptive
fragment activates. This proves current safety and measurement breadth, but it
does **not** prove current acceleration.

The current attribution separates **179.029 seconds** of recorded BuildOpt
path cost from **-189.593 seconds** of residual Gradle/runner variation; those
components reconcile the full -368.623-second signed result. Discovery and
learning account for 98.385 seconds, output verification 28.890 seconds and
state access 18.154 seconds. Native retention costs 0.531 seconds p50 and
8.656 seconds p95. Since no profile or fragment activated, attributable
mechanism saving is **zero** and the outcome is
`CURRENT_VALUE_NOT_ATTRIBUTABLE`.

The terminal AF-015 scorecard now stops this successor hypothesis. It passes
9/15 frozen criteria, including correctness, generic implementation, native
substrate, cohort integrity, ordinary learning and attribution. It fails all
six value/economics gates: 0/71 eligible builds activate, 0/5 families are
positive, 0/5 have a positive lower confidence bound, signed portfolio value is
-368.623 seconds, no family repays and native retention is 0.531 seconds p50 /
8.656 seconds p95. With zero activated saving, there is no evidence-backed
bounded fragment class to specialize.

## Historical sticky-wrapper measurements

The successor POC has now completed its first implementation block. `SWL-008`
reads a signed local decision snapshot before Gradle and returns the native
workflow for every missing, expired, revoked, corrupt, busy or incompatible
state. A compatible active decision is recognized but deliberately not
executed yet. On this Linux AMD64 host, 200 synthetic selections measured
verified-local lookup at **0.492 ms p50 / 1.369 ms p95**; missing-state
fallback measured **0.0025 ms p50 / 0.0025 ms p95**, and the no-synchronous-
refresh path measured **0.0025 ms p50 / 0.0026 ms p95**. All three budgets pass.

This is a retention-cost result, not a build-time saving: no optimization was
activated and no Gradle wall-time claim is made.

The follow-up `SWL-008A` measurement removes avoidable cost from the committed
wrapper itself. When no server credential or explicit BuildOpt integration is
present, the wrapper validates its boundary and lets native Gradle run without
starting a gateway, plugin handshake, managed L1 or central-cache probe. The
default observer is `light`: it skips the pre-build Git lookup, computes the
executable digest concurrently with the child when possible and creates its
private recorder only after the child exits. On this Linux host, 20 interleaved
samples measured **+9 ms p95** for the native no-op path and **+38 ms p95** for
light observation over direct execution; light pre-child decision time was
**0.093 ms p95**. The result passes the POC guardrails, but it is wrapper-cost
evidence rather than a Gradle speedup claim. `full` observation remains an
explicit diagnostic mode and `0` disables recording. See the
[overhead specification](../../specs/poc-sticky-wrapper-noop-overhead-v1.md)
and [checked result](../../benchmarks/results/sticky-wrapper-noop-overhead-v1.json).

`SWL-009` now observes ordinary requested Wrapper builds without changing
their arguments or result. The default path uses light observation; the first
checked-in sample contains two successful
Gradle 9.6.1 invocations with Configuration Cache present. The first run took
**19.876 s** of observed wall time and the second took **3.732 s**; the second
reused Configuration Cache and the records reconcile every measured phase plus
the residual. Decision work was **53.8/57.4 ms**, while Gradle itself was
**19.821/3.673 s**. Network, bootstrap and post-build observation are marked
`UNAVAILABLE` rather than silently counted as zero. This is instrumentation
evidence, not a speedup claim; it gives the next trial block an honest cost
ledger.

`SWL-010` now runs those trials only in trusted CI. Four alternating pairs
(eight direct invocations) use separate checkouts, Gradle homes, daemon/cache
roots and BuildOpt state roots, and every required output is hashed. The first
checked-in result is exact but negative: BuildOpt averages **7.534 s** versus
**6.979 s** for optimized native Gradle, for **-0.555 s** mean saving and
**0/4** positive pairs. All **4/4** output trees match and all **8/8**
invocations succeed. The observed extra compute is **58.050 s**, below the
declared **180 s** (5%) trial ceiling. The mechanism is safe and the evidence
accounting works, but this path does not yet provide customer value on this
fixture; no action is activated.

`SWL-011` now supplies that revalidation and suspension boundary. Its generic
runner rechecks signed bindings on every invocation, executes candidate and
authoritative native commands without a shell, compares required output hashes
and retains native after any failure or regression. The focused result has one
synthetic active execution (about 24.6 ms saved), three suspensions and four
native retentions. The real SWL-010 report remains unauthorized because it is
negative (7.534 s candidate versus 6.979 s optimized native, 0/4 positive), so
this is safety/control-flow evidence rather than customer value. A runtime or
cache action still needs a positive repository-level paired result before the
wrapper may execute it for a customer.

`SWL-012` now tests whether a learned opportunity can become a durable native
Gradle change instead of paying BuildOpt runtime cost forever. Its catalog uses
typed task inputs/outputs and dependency relationships, never repository names,
and every proposal is review-only with an exact apply/revert transaction.

The fresh strict-runner result is
[`sticky-wrapper-durable-catalog-v1.json`](../../benchmarks/results/sticky-wrapper-durable-catalog-v1.json)
from `linux-amd64-4c-16g-v1` at BuildOpt revision
`1d93570c02147eda8671253663d50605bff9f25a`. The same task-contract detector
accepted two DSL families:

| Family | Native control | Reviewed patch | Mean saving | Positive pairs | Outputs |
| --- | ---: | ---: | ---: | ---: | --- |
| Kotlin DSL | 2.438 s | 0.875 s | **1.563 s / 64.1%** | **8/8** | Exact |
| Groovy DSL | 3.574 s | 0.903 s | **2.671 s / 74.7%** | **8/8** | Exact |

The task patch adds the missing native cache/input/output contract, so plain
Gradle remains responsible for future builds after owner review. BuildOpt is
not required after acceptance, and the report records zero product failures.
The graph detector also proposes a generic 3 -> 2 project scope for both DSLs,
but durable graph timing is deliberately **not measured yet**. This is
promising synthetic POC evidence, not customer coverage or automatic patch
authority.

`SWL-013` is complete: the wrapper exposes customer-readable `status` and
`explain` output with recomputable economics, cache facts and exact fallback
reasons. These commands can display stored state, but the normal wrapper path
does not yet compose the observer, trial, signed decision, active runner and
durable catalog into one automatic lifecycle.

## Historical installed two-machine proof

`SWL-014` proves the transport and lifecycle, not acceleration. Two isolated
4-CPU/8-GiB containers use the committed `./buildoptw` command and the same
SHA-verified archive. A trusted producer publishes two Gradle cache objects;
the consumer uses a separate read-only credential after the owner commits and
restarts the HTTPS service. The consumer restores **2 tasks from cache** and
the producer, consumer and outage rebuilds all emit the same required output
SHA-256. When the service is offline, the wrapper records **0 central hits**
and falls back to native Gradle successfully. The final clean-SHA run observed
**11.027 s producer**, **7.938 s consumer** and **7.435 s outage**. Credentials are not
visible to Gradle or logs, and pending objects remain invisible before owner
commit.

This is a functional installed-path result; it sets `wallTimeClaim=false` and
does not qualify a profile.

## Historical longitudinal diagnostic

The bounded `SWL-015 v1` sample ran one frozen current revision in each of five
public repository families. Both arms used isolated worktrees and every pair
required successful execution and byte-identical outputs. A later route audit
found that the runner injected `--build-cache` only into control, while
candidate had no server identity and a zero trial budget and therefore used
no-op/light observation. The numbers below measure compatibility and wrapper
cost under that asymmetric protocol, not learning or optimization value.

| Repository | Native control | BuildOpt wrapper | Signed delta | Result |
| --- | ---: | ---: | ---: | --- |
| Spring Framework | 283.944 s | 289.226 s | -5.282 s / -1.86% | Negative |
| OpenTelemetry Java Instrumentation | 512.194 s | 517.979 s | -5.786 s / -1.13% | Negative |
| Apache Kafka | 210.889 s | 205.213 s | +5.676 s / +2.69% | Positive |
| Micronaut Core | 494.033 s | 511.159 s | -17.126 s / -3.47% | Negative |
| Apache Groovy | 128.134 s | 127.765 s | +0.370 s / +0.29% | Positive |

The sample contains one `CONTROL_FIRST` pair per family, **2/5 positive pairs,
3/5 negative pairs, -22.149 s signed total, 5/5 exact outputs and zero product
failures**. It is now immutable `DIAGNOSTIC_ONLY` evidence. It cannot contribute
to an activation, confidence, payback or terminal criterion.

The machine-readable sample is
[`poc-sticky-wrapper-longitudinal-sample-v1`](../../benchmarks/results/poc-sticky-wrapper-longitudinal-sample-v1/README.md).
The corrected path was deliberately staged: `SWL-014A` proved cache-symmetric
arms and lifecycle-aware zero-pair readiness, and `SWL-014B` connected the real
wrapper-driven learning/action loop. `SWL-014C` then ran the two preregistered
generic adapters over all five families and found **0/5 complete independently
testable actions**. The task-proposal input is unavailable; Spring (16 graph
rows), Micronaut (8) and Kafka (1) lack exact candidate-plan/critical-path
bindings, while OpenTelemetry and Groovy have no graph rows. These are input
counts, not savings. The frozen stop rule therefore skips `SWL-014D` and
`SWL-015 v2`; only the unchanged `SWL-016` terminal scorecard remains.

## Historical complete-profile result

One exact BuildOpt executable was used across frozen Spring Framework,
OpenTelemetry Java Instrumentation, Apache Kafka, Micronaut Core and Apache
Groovy windows. The run used requested ordinary builds only, kept the robust
eight-pair qualification gate unchanged and stopped learning early when the
five-match lifetime economics could not pay back.

| Repository | Requested builds | Target evidence | Later selection | Signed lifetime net |
| --- | ---: | --- | ---: | ---: |
| Spring Framework | 1 | Early native retention; only four compatible historical matches | 0 / 0 | **-10.113 s** |
| OpenTelemetry | 1 | Early native retention; one compatible match | 0 / 0 | **-9.961 s** |
| Apache Kafka | 17 | **+6.673 s / 21.43%, 8/8 positive** | **1 / 6 eligible** | **+82.527 s** |
| Micronaut Core | 1 | Early native retention; one compatible match | 0 / 0 | **-9.149 s** |
| Apache Groovy | 1 | Early native retention; one compatible match | 0 / 0 | **-2.760 s** |

Kafka's selected descendant improves from 178.566 seconds to 43.439 seconds,
saving **135.127 seconds / 75.67%** with 4,449 exact outputs. Five other
structurally eligible descendants retain native Gradle. After 42.040 seconds
of measured fallback wrapper work and 10.560 seconds of qualification plus
publication cost, Kafka finishes **82.527 seconds net positive**.

Across all five repositories the experiment uses 21 requested builds, zero
measurement-only builds, verifies 27 exact-output observations and records
zero product failures. The signed total is +50.544 seconds, but that number is
descriptive only: repository percentages are not averaged and mechanism
percentages are not added.

## Historical fragment-composition result

AF-011 ran 48 new Gradle 9.6.1 pairs on one controlled workflow, always against
the complete optimized native workflow and with identical required outputs.

| Direct comparison | Groovy | Kotlin | Decision |
| --- | ---: | ---: | --- |
| Build Impact | **46.45% / 2.180 s**, 7/8 | **43.97% / 1.973 s**, 6/8 | Groovy qualifies; Kotlin misses repeatability |
| Reviewed task patch | **37.95% / 1.749 s**, 7/8 | **35.08% / 1.705 s**, 8/8 | Both qualify |
| Direct composition | **68.56% / 2.947 s**, 8/8 | **79.32% / 3.025 s**, 8/8 | Faster in both, but not authorized |
| HTTP cache locality, independent scope | **35.07% / 2.482 s**, 4/4 | Same controlled cache contract | Qualifies independently |

Every lower confidence bound is positive, outputs are byte-identical and
product failures are zero. The composed percentages are measured directly and
are not sums. The outcome is `RETAIN_BEST_SINGLE_FRAGMENT`: a positive
composition cannot hide the failed 7/8 constituent gate. Cache locality is not
claimed as part of that composition because its remote-cache object contract
is different.

## What the historical evidence says

- The latest current-installed campaign is safe but not valuable yet: 100/100
  pairs preserve exact outputs with zero product failures, while 75/100 regress
  and cumulative signed value is -368.623 seconds.
- No current campaign build activated a whole profile or adaptive fragment.
  AF-014D attributes 179.029 seconds to recorded BuildOpt work and -189.593
  seconds to residual Gradle/runner variation; AF-015 still credits zero saving
  because no mechanism activated.
- The core idea can create very large value when a learned structural profile
  remains compatible: Kafka's selected replay saves 75.67%.
- Isolated acceleration is not enough. A profile must recur across real commits
  often enough to repay discovery and publication.
- The earlier whole-profile breadth was insufficient. Only **1/5 repository
  families** is net positive and only **1/6 eligible descendants (16.67%)**
  selects a profile. The preregistered pass gate required at least 3/5 net
  positive families and at least 50% selection coverage.
- Safety and fail-open behavior work: every uncertain case uses optimized
  native Gradle, outputs remain exact and product failures are zero.
- Early economics prevent waste: the four short-lived hypotheses stop after
  one requested build instead of spending 16 additional builds each.
- Frozen shadow decomposition explains part of the coverage loss: all **6/6**
  eligible Kafka descendants retain a compatible structural fragment even
  though only **1/6** retains the complete profile. Five cases are partial, so
  output freshness or economics can invalidate materialization without erasing
  the structural opportunity. This is compatibility evidence, not a timing
  result.
- The current adaptive breadth is also insufficient: **0/5** families are net
  positive, **0/71** eligible builds activate and **0/5** lower confidence
  bounds are positive.

The earlier complete-profile gate yielded **`STOP_GENERIC_POC`**. Five criteria
pass: matrix completeness, exact outputs/zero failures, generic selection,
robust Kafka qualification and bounded Kafka payback. Three fail:

- 1/5 net-positive repository families versus the required 3/5;
- 1/6 selected eligible descendants (16.67%) versus the required 50%; and
- one observed pre-Gradle economic rejection at 4,098 ms, above the 500-ms
  median and 1,000-ms p95 limits.

## Historical conclusion and active direction

The generic structural-profile hypothesis and its adaptive-fragment successor
both stop here. This is not a claim that BuildOpt never works: Kafka pays back
strongly, Spring and other
bounded experiments remain valid mechanism evidence, and exact-output plus
fail-open controls work. It is a rejection of the broad claim that the current
one-command implementation already delivers repeatable net wall-time value to
ordinary Gradle repositories.

The successor decision is `STOP_ADAPTIVE_FRAGMENT_POC`, not specialization.
Its scorecard passes 9/15 safety, integrity and auditability criteria but fails
all six value/economics criteria. There is no further block in this tracker.
Any future work must preregister a materially different mechanism that can
activate generically, repay its own learning/verification cost and beat
optimized native Gradle over real commit sequences; it cannot reopen these
thresholds or inherit authority from inactive fragments.

## Closed fresh-evidence and change-aware predecessors

The new [Sticky Wrapper Learning POC](../plans/sticky-wrapper-learning-poc-tracker.md)
tests a materially different customer and economic model. A maintainer
generates and commits `buildoptw`, `buildoptw.bat`, checksum-pinned wrapper
properties and portable non-secret configuration. Every developer and CI run
then uses one command:

```text
./buildoptw <gradle args...>
```

The target wrapper lifecycle can retain native Gradle, observe, shadow,
schedule a bounded CI trial, execute an exact qualified runtime profile or
report a reviewed durable Gradle patch. Today those mechanisms are separately
checked; only bootstrap, passthrough, cache, light observation, status and
fail-open behavior are composed behind ordinary `./buildoptw`. It reuses the
existing optional HTTPS service, but Gradle cache objects and typed BuildOpt
decisions remain separate planes and a cache hit never grants action authority.

This route was a preregistered fresh-evidence experiment, not a performance
result. Its continuation gates required exact outputs, zero product failures,
at most 100-ms p50
and 250-ms p95 local native-decision overhead, a positive complete portfolio
and independently positive value/payback in at least three of five public
families. Every bootstrap, observation, trial, cache, fallback and action cost
counts. All historical BuildOpt observations, profiles, timing rows, cohorts
and decisions are context only and are forbidden inputs. The earlier
`SWL-014C` screen reported **0/5** actions because its inputs lacked complete
public task and graph producers; it proves an evidence-pipeline gap, not
absence of generic opportunity. It has no terminal authority. The producer
proof is complete without importing historical data. The fresh five-family
capture now binds 100 primary plus 50 reserve commits, five conclusive producer
inputs and byte-exact current outputs to one package digest. It exposes one
Spring task-contract action and no safe declared-graph reduction.
`SWL-FRESH-003` independently confirms **1/5** action breadth against the fixed
**3/5** gate. Because even perfect Spring economics cannot raise that upper
bound, installed timing and the chronological campaign were not authorized.
The terminal scorecard now stops this detector set without presenting
unmeasured value, confidence, payback or overhead as zero. The result is not a
performance regression or speedup result: it says the current two detectors
cannot reach enough public families to justify expensive timing. A successor
must start with a broader generic detector hypothesis, not another campaign on
the sole Spring action.

The successor was preregistered as
[`CHANGE_AWARE_PRODUCER_CLOSURE_V1`](../plans/change-aware-producer-closure-poc-tracker.md).
It uses real adjacent-revision changes, finalized Gradle inputs and complete
direct/transitive producer lineage to derive the exact work and required-output
closure for each change. The deterministic
[selection result](../../benchmarks/results/generic-opportunity-discovery-v1.json)
rejects a rerun of the terminal detectors, a standard archive-cache adapter and
single-revision reuse as the next generic experiment. This is a selected
hypothesis, not a speed result. Its first fresh evidence block is now complete:

| Family | Conclusive transitions | Testable actions |
| --- | ---: | ---: |
| Spring Framework | 5/5 | 1 |
| OpenTelemetry | 5/5 | 0 |
| Apache Kafka | 5/5 | 0 |
| Micronaut Core | 5/5 | 0 |
| Apache Groovy | 5/5 | 0 |

The producer therefore proves complete evidence and safe retention on
uncertainty, but not broad value. `SWL-CHANGE-002` rebuilt every report from
the raw capture, ignored the aggregate summary and confirmed only **1/5**
action breadth against the unchanged **3/5** threshold. It still has **zero
authorized timings and zero authorized activations**. Installed and
chronological value are not measured because this detector cannot meet the
generic breadth target even if the sole Spring action were infinitely fast.
The terminal scorecard therefore closes this detector without a Spring-only
benchmark. Input completeness, report reconstruction, static output binding
and zero capture/analyzer failures pass; breadth fails. Installed value,
chronological value, confidence, payback and overhead remain explicitly
unmeasured, and no successor is automatically authorized.

The subsequent independent
[cause analysis](../../benchmarks/results/request-aligned-successor-selection-v1.json)
groups those negatives into 23 declared-input-disjoint transitions and one
current-output discovery drift. It rejects unsafe relabelling, more arbitrary
commit sampling, a one-row filename repair as a breadth claim and cache
transport as an equal-opportunity speed claim. The selected and now closed
[request-aligned route](../plans/request-aligned-learning-poc-tracker.md)
preserves the user's command and the unchanged 3/5 gate. Request identity,
current producer-output discovery and adjacent relevance classification are
complete. The fresh capture now contains **110** chronological transitions:
Groovy and Spring each produce five relevant exact actions, while Kafka,
Micronaut and OpenTelemetry exhaust their 30-transition budgets. The aggregate
is therefore **2/5 complete and 2/5 action-broad**. Independent reconstruction
verifies every report, and the terminal decision
`STOP_REQUEST_ALIGNED_RECURRENT_CLOSURE_POC_FOR_CURRENT_DETECTOR` closes this
detector without candidate execution, timing, a speedup claim or automatic
successor. The recommended work is cause analysis of the global, ambiguous and
request-irrelevant public rows before selecting a materially different generic
hypothesis.

The closed predecessor remains implementation-locked as historical evidence:
separate per-arm cache namespaces started empty, only the task-contract and
declared-graph detectors entered its breadth screen, and no timing opened after
the 1/5 result. The exact terminal route remains in the
[fresh tracker](../plans/fresh-generic-optimization-poc-tracker.md).

The old task-contract result was deliberately unavailable in that public
screen. The route added generic public producers and required typed
completeness for both detector families before scoring breadth; that completed
screen is now immutable predecessor evidence.

`SWL-001` freezes the four file formats, immutable per-platform distribution
identities, strict portable configuration, explicit management routing,
pre-bootstrap bypass and update/downgrade behavior. `SWL-002` implements
deterministic `buildopt wrapper init`, offline/read-only `check` and
distribution-only `update`, including transactional rollback. `SWL-003` now
adds the generated POSIX/Windows bootstrap: it selects one of four pinned
native packages, permits at most five HTTPS-only redirects, verifies the outer
SHA-256 and internal manifest, rejects unsafe archive entries and atomically
publishes a user-cache installation. Verified warm reuse performs no network
request and concurrent first use downloads once. Synthetic negatives,
race/vet, ShellCheck, PowerShell parsing and real public `v0.6.1` Linux and
Windows-body online/offline smokes pass. The public smoke exposed and corrected
the original zero-redirect assumption for GitHub release assets without
weakening checksum authority. `SWL-004` completes the neutral customer command:
ordinary arguments run the repository Gradle Wrapper through the existing
process launcher, difficult arguments/cwd/streams/environment and child exit
codes remain equivalent, signals reach descendants, exact bypass occurs before
configuration/bootstrap and bootstrap failure falls back to direct Gradle.
Native macOS/Windows entrypoints exercise the same route. Cache/state,
observation, learning and optimization remain disabled, so there is still no
new performance claim. `SWL-005` now binds the committed endpoint and project
scope to the private owner-issued token document. Two clean checkouts share one
project identity; namespace/generation remain distinct, and missing,
mismatched, incomplete, redirected or revoked credentials retain native Gradle
and never reach the child. `SWL-006` now consumes the existing Gradle-compatible
central cache through an invocation-local verifying gateway when the connection
is valid; the wrapper enables Gradle's native cache unless explicitly disabled,
keeps ordinary consumers read-only, and falls back to native rebuilds on missing,
expired, revoked, corrupt or unavailable data. Typed-state consumption and
decision learning remain later blocks.

No generic whole-profile implementation block followed automatically from that
failed gate. The separately preregistered
[Adaptive Fragment Generalization POC](../plans/adaptive-fragment-generalization-tracker.md)
then tested a replacement: independently compatible producer,
subgraph, task, patch and cache-locality fragments; learn their signed economics
from ordinary builds; compose only fragments with current correctness and value
authority; and retain native Gradle through a sub-second no-value path. The new
tracker preserves the same five-repository baseline, exact-output and zero-
failure requirements, and makes cumulative longitudinal value—not isolated
target speedup—the terminal decision. AF-001..AF-009 now provide typed fragment
identity/state, sub-millisecond lookup and a no-lookahead shadow result with
100% fragment retention across the six eligible descendants, plus immutable
signed economics. The retained Kafka composition recomputes to `+82.527 s`
after `135.127 s` gross saving, `42.040 s` synchronous wrapper cost and
`10.560 s` one-time qualification/publication cost; payback occurs on the
second requested descendant. Existing evidence cannot attribute that gross
saving to one fragment. AF-006 now provides immutable ordinary-build
checkpoints: exact restart succeeds, insufficient evidence remains
`OBSERVED`/`SHADOW`, and a synthetic value regression suspends only the affected
fragment and its dependent. AF-007 adds a name-independent prior: replacing all
source/holdout identities leaves the structural ranking unchanged, and source
outcomes can reorder exploration while fresh target correctness and value stay
mandatory. The vectors make no timing claim. AF-008 now proves that a generic
repeated-task detector can lead to one durable reviewed patch: the exact recipe
saves 1.369 seconds/67.28% in Kotlin and 2.349 seconds/68.01% in Groovy across
16 native Gradle pairs, with exact outputs and no BuildOpt runtime dependency.
That does not make the recipe generic or automatic. AF-009 now composes only
compatible fragments whose predicted net value remains positive, without
adding isolated effects: it closes
dependencies, rejects one-way or two-way conflicts, preserves every exact
constituent authority and falls back to native Gradle when the direct joint
prediction is absent, ambiguous or below `100 ms`. The proof is synthetic and
does not claim runtime saving. AF-010 now executes that boundary on real Gradle:
six control/candidate scenarios restore four exact unaffected outputs, rebuild
two changed producers locally and retain the complete original workflow for
global, ambiguous or missing state. Every producer and final bundle is exact
and product failures are zero. AF-011 now measures each mechanism and the
complete compatible composition directly. The composed arms are strongly
positive, but Kotlin Build Impact misses the fixed repeatability gate, so the
planner retains independently qualified fragments instead of activating the
joint path. AF-012 now persists the same portfolio and economic ledger locally
and over the existing HTTPS state plane. Exact bytes restore on a clean second
machine, CAS conflicts remain visible, corrupt state rejects, a verified local
generation survives outage and a clean offline machine retains native Gradle.
Adaptive documents generate zero Gradle cache-plane requests. AF-013 preserves
the frozen historical measurements without rerunning favorable arms:
Spring closes at `+59.550 s`, Kafka at `+88.219 s`, OpenTelemetry at
`-168.751 s` and Groovy at `-37.684 s`. Micronaut remains `INCONCLUSIVE`
because its first descendant failed byte reproducibility before an attributable
timing pair could be accepted. All 14 comparable builds have exact outputs and
zero product failures, but only 2/5 rows are positive versus the frozen 3/5
breadth target. This is an audit of earlier behavior, not the current scorecard:
OpenTelemetry and Groovy have only three later commits, Micronaut has no valid
descendant pair and several rejection paths changed afterward. AF-014A now
proves one current-SHA installed package, separate arm state, chronological
learning, exact selected/native-retained/bypass behavior and reconciled phase
timing. Its controlled fixture is apparatus evidence, not a speedup. AF-014B
froze 20 primary commits plus ten ordered reserves per family before timing.
AF-014C then collected 100 comparable exact-output pairs under one installed
package. AF-014D now attributes the current result: 179.029 seconds of recorded
BuildOpt cost, -189.593 seconds of residual variation, and zero
activated-mechanism saving. The largest actionable recorded slice is
discovery/learning at 98.385 seconds; however, reducing overhead alone cannot
create customer value when selection coverage remains zero. AF-015 has now
applied the frozen breadth, lifetime, fallback and value criteria to this
current campaign only and records `STOP_ADAPTIVE_FRAGMENT_POC`. Nine safety,
integrity and auditability criteria pass; activation coverage, breadth,
confidence, portfolio value, time to value and native-retention cost fail.
Historical favorable observations remain context, not current decision inputs.
Production hardening, soak, design partners and Test Optimization remain
outside this POC.

## Evidence

- [Lifetime breadth V3 result](../../benchmarks/results/poc-lifetime-breadth-v3/README.md)
- [Terminal functional-coverage decision](../../benchmarks/results/poc-functional-coverage-decision-v1/README.md)
- [Adaptive-fragment terminal scorecard](../../benchmarks/results/adaptive-fragment-terminal-decision-v1.json)
- [Adaptive-fragment terminal contract](../../specs/poc-adaptive-fragment-terminal-decision-v1.md)
- [Sticky Wrapper Learning POC Tracker](../plans/sticky-wrapper-learning-poc-tracker.md)
- [Fresh Generic Optimization POC Tracker](../plans/fresh-generic-optimization-poc-tracker.md)
- [Fresh generic optimization contract](../../specs/poc-fresh-generic-optimization-v1.md)
- [Fresh generic producer evidence](../../benchmarks/results/sticky-evidence-producers-v1.json)
- [Sticky wrapper machine contract](../../specs/poc-sticky-wrapper-learning-v1.md)
- [Sticky wrapper generator contract](../../specs/poc-sticky-wrapper-generator-v1.md)
- [Sticky wrapper bootstrap contract](../../specs/poc-sticky-wrapper-bootstrap-v1.md)
- [Sticky wrapper connection contract](../../specs/poc-sticky-wrapper-connection-v1.md)
- [Machine-readable V3 summary](../../benchmarks/results/poc-lifetime-breadth-v3/summary.json)
- [V3 protocol](../../specs/poc-lifetime-breadth-v3.md)
- [Detailed historical findings](./build-optimization-performance.md)
- [Ordinary-build learning economics](../../benchmarks/results/poc-ordinary-learning-economics-v1/README.md)
- [Structural profile rebinding](../../benchmarks/results/poc-structural-profile-rebinding-v1/README.md)
- [Adaptive fragment shadow replay](../../specs/poc-adaptive-fragment-shadow-v1.md)
- [Machine-readable shadow result](../../benchmarks/results/adaptive-fragment-shadow-v1.json)
- [Adaptive fragment economics](../../specs/poc-adaptive-fragment-economics-v1.md)
- [Active Build Impact fragment evidence](../../benchmarks/results/adaptive-fragment-activation-v1.json)
- [Machine-readable economic ledger](../../benchmarks/results/adaptive-fragment-economics-v1.json)
- [Ordinary-build learner contract](../../specs/poc-adaptive-fragment-online-v1.md)
- [Machine-readable learner proof](../../benchmarks/results/adaptive-fragment-online-v1.json)
- [Cross-repository prior contract](../../specs/poc-adaptive-fragment-prior-v1.md)
- [Machine-readable prior proof](../../benchmarks/results/adaptive-fragment-prior-v1.json)
- [Patch-opportunity contract](../../specs/poc-adaptive-fragment-patch-opportunity-v1.md)
- [Machine-readable durable patch proof](../../benchmarks/results/adaptive-fragment-patch-opportunity-v1.json)
- [Conflict-aware planner contract](../../specs/poc-adaptive-fragment-planner-v1.md)
- [Machine-readable planner proof](../../benchmarks/results/adaptive-fragment-planner-v1.json)
- [Direct fragment-composition result](../../benchmarks/results/adaptive-fragment-composition-v1.json)
- [Fresh HTTP cache-locality result](../../benchmarks/results/adaptive-cache-locality-v1.json)
- [Adaptive state portability contract](../../specs/poc-adaptive-state-portability-v1.md)
- [Adaptive longitudinal matrix](../../benchmarks/results/adaptive-fragment-longitudinal-v1.json)
- [Adaptive longitudinal protocol](../../specs/poc-adaptive-fragment-longitudinal-v1.md)
- [Current longitudinal raw evidence](../../benchmarks/results/current-longitudinal-raw-v1.json)
- [Current longitudinal attribution](../../benchmarks/results/current-longitudinal-attribution-v1.json)
- [Current attribution protocol](../../specs/poc-current-longitudinal-attribution-v1.md)
