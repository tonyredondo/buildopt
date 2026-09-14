# BuildOpt research status

Updated: 2026-09-14. This is the current research disposition register.
The [governing execution plan](./plans/buildopt-research-execution-plan-2026-09-14.md)
owns priorities and sequencing within `BUILDOPT-VIABILITY-V1`. Executable
contracts, frozen inputs and historical results retain their original scope.

## Current work

Follow the governing plan's
[execution tracker](./plans/buildopt-research-execution-plan-2026-09-14.md#execution-tracker):
prove sustained fixed-correction savings, measure a usable manual MVP, then
prove added adaptive value. [BO-01 candidate recovery](../benchmarks/results/buildopt-product-viability-v1/bo-01-candidate-recovery/README.md)
is verified: five candidate records recovered, Checkstyle inputs and manifest
checked, and the old patch-portfolio verifier corrected without altering its
data or criteria. Micronaut/Spring workflow gaps and Build Impact output-scope
limits remain explicit. No new native build ran during BO-01.

[BO-02 identical-code control](../benchmarks/results/buildopt-product-viability-v1/bo-02-measurement-control/README.md)
is verified: 84.035s versus 81.294s per whole request, a 2.742s difference or
3.37% of the faster request. This is below the registered 5% threshold:
`NO_SPURIOUS_MATERIAL_SIGNAL_IN_THIS_CONTROL`. All four builds and both live
and independently reconstructed output pairs passed. All four requests retain
their host-pressure flags; one pair does not establish precision or qualify
G0/G3. Four owner starts and four comparator JVMs were used, with no retries.
The allocation and owned service scopes are closed. The task record and raw
evidence remain under `bo-02-measurement-control-2026-09-14` in program state.

[BO-03 opportunity assessment](../benchmarks/results/buildopt-product-viability-v1/bo-03-opportunity-assessment/README.md)
is verified, with no new builds or protected validation reads. The Checkstyle
dependency model allows 5.774% across the complete native development commands,
leaving about 0.200 seconds per build for residual work and costs. That is a
model with fixed task durations, not a measured V2 saving or an absolute bound.
The planned short screen remains eligible after its readiness checks.
Micronaut needs evidence of recurring useful cache restores before timing.
The selected Spring result does not justify another timing allocation under
the current one-second floor. Their task sources match the original preimages
through development ordinal 20; buildability and execution frequency remain
unmeasured. All five dispositions and cost assumptions are recorded in the
assessment. No long replay is admitted.

Next is BO-04: review the remaining Build Impact question for Ktor and Beam.
This does not launch the 17–20 Checkstyle screen, other repository builds or
the protected validation history.

Build Impact has a bounded admission review at BO-04 for the known Ktor and
Beam cases. A new trial requires a concrete question not already answered by
the retired studies and a prospective protocol. Generic plan selection/reuse,
adaptive fragments, runtime sweeps and another general cache remain retired.

The earlier BV plan/tracker preserve execution evidence; their conflicting
next-step and commercial instructions are superseded. Their old references to
the "current tracker" do not create another work queue. Use the new tracker
for future steps and retain the old replay contract's technical acceptance
criteria. Product viability remains **NOT_ESTABLISHED**.

## Research evidence through 2026-09-10

The sequence below preserves the completed experiments and their original
qualifications. Its historical "next" and "current" wording is subordinate to
the governing plan above. No previous negative or incomplete result is promoted.

**Latest completed experiment:** [supported Checkstyle finalization](../benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/README.md)
passes affected correctness and all eight comparison builds, including four live
and four independently reconstructed complete-output pairs. Whole customer time
is 124.537→83.160s in one replication and 83.003→97.339s in the other. Mean saving
is 13.029%, but the predeclared requirement to improve both pairs fails:
`FINALIZATION_VALUE_NOT_ESTABLISHED`. Product viability remains **NOT_ESTABLISHED**.

Actual checking/reuse is equivalent. Both first measured requests take about 83s;
both second requests are slower, with reversed implementation order and longer
daemon idle ages. CPU pressure and sampled paging/I/O accompany the slow runs;
the exact cause and process remain unverified. No sample was excluded. All owned
scopes and this allocation are closed; no experiment watcher remains active.

**Successor, now completed:** the [four-build identical-code A/A control](../benchmarks/results/buildopt-product-viability-v1/bo-02-measurement-control/README.md)
did not reproduce a material difference under its registered rule. It does not
explain or overturn this earlier mixed result. Native 17–20 remains a possible
later screen after BO-03 and its applicable readiness checks; source preflight
already passes. G0/G3, held-out 21–100, other repository builds and adaptive
product implementation remain unqualified or deferred.

The frozen [V1 plan](./plans/buildopt-checkstyle-finalization-v1.md) and its
deprecated callback are **rejected/superseded**. Preserve the plan bytes because
they bind the failed trial; pending rows inside it do not reopen that route.
The [V2 plan](./plans/buildopt-checkstyle-finalization-v2.md) is a completed protocol;
its agent-proof pointer is superseded by the verified reader receipt in the result.
Recover `engineeringPrefix.checkstyleFinalization` for final evidence and accounting.

**Owner disposition, 2026-09-10:** the workstation also runs other agents' Rust,
.NET and Rust LLVM coverage workloads. The owner accepts the regression
investigation as sufficient and the transient spike as compatible with external
contention; exact historical attribution remains unmeasured and is no longer a
blocker. See the [shared-host addendum](./findings/buildopt-checkstyle-shared-host-disposition-2026-09-10.md).
Preserve the timing evidence and product qualification limits. The safe
finalization comparison above used prospective host-contention observations;
no additional retrospective reproduction is required.

**Latest diagnostic:** the [Checkstyle regression and recovery investigation](../benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-regression/README.md)
completed seven actual native starts, including one retained incomplete-observation
warmup, and verified all three corrected live/independent C5 pairs. At original 7,
Main reuses 4,954 inputs; its adapter takes 1.651s and its worker is idle for 40.140s
before finalization. Actual Gradle lock-retention services and the pinned bytecode
verify the reacquisition mechanism. The fresh customer request takes 98.185s native
versus 83.416s candidate; the following request returns to roughly 7s with Checkstyle
UP-TO-DATE. These are diagnostic observations, not a value confirmation.
The original 47s collect-task episode is not reproduced; its exact internal cause
remains unresolved. Missing target daemon-JFR coverage and a post-run export-size
breach remain explicit. All allocation/process scopes are closed. Next: [prove
safe finalization and measure whole-request impact](../benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-regression/next-step.md);
that diagnostic allocation is closed. Its successor is the completed finalization
comparison above; its historical next-step instruction is superseded. Recover the predecessor at `engineeringPrefix.checkstyleRegression`.

**Prior completed experiment:** the [corrected Checkstyle screen and history](../benchmarks/results/buildopt-product-viability-v1/bv006-screen-completion/README.md)
retain 54 successful native builds and 27 live/independent output pairs. The two
selected commits save 41.287% /40.114% /24.115% at 8/4/2 CPUs. Disjoint history 2–14
loses 4.260s per build (19.158% slower): `NO_MATERIAL_SIGNAL_IN_DISJOINT_HISTORY`.
Its original long Main task span is clarified by the new diagnostic; neither its
raw results nor its failed worker-isolation qualification are overwritten.
No G0, G3 or product gate is promoted. Formal validation 21–100 and all six other
repositories remain unconsumed. Older candidate, reader and resource-audit
failures stay retained. Its locator remains `engineeringPrefix.screenCompletionFixed`.
Earlier “next” statements below are historical when superseded by these closeouts.

**Prior execution of `BUILDOPT-VIABILITY-V1`:** the
[earlier plan](./plans/buildopt-product-viability-v1.md) and
[execution record](./plans/buildopt-product-viability-v1-tracker.md)
retain work authorized on 2026-09-08 and its completed planning proof.
BV-001 verified the actual source tree, required builds and 100 history edges.
The [20-transition native audit](../benchmarks/results/buildopt-product-viability-v1/bv002/opportunity.md)
rejects the registered ForbiddenPatterns intervention at G1. Its generous
whole-task ceiling is 0.787 s per transition and 3.134%, below the unchanged
1 s / 5% floors. Two observed input-repair signatures admit no H2 cause.
Technical closeout is verified; the
[scoped decision](../benchmarks/results/buildopt-product-viability-v1/viability-decision.md)
leaves product viability and other repositories unproved. The
[Checkstyle follow-up](../benchmarks/results/buildopt-product-viability-v1/next-investigation.md)
was approved by the owner. Its [C1-C4 admission](../benchmarks/results/buildopt-product-viability-v1/checkstyle-admission/README.md)
is verified: raw cache enablement is rejected, while a content-aware prototype
has a narrowly positive optimistic G1 model. The
[prototype and correctness proof](../benchmarks/results/buildopt-product-viability-v1/checkstyle-prototype/README.md)
now verify BV-003/BV-004 and G2 for the frozen Linux owner: complete outputs,
failure/recovery, native fallbacks and exact removal. The
[replay instrument](../benchmarks/results/buildopt-product-viability-v1/bv005/README.md)
now verifies BV-005 on local fixtures, including actual native Gradle, crash
recovery, cost reconstruction and rejection cases. [BV-006 is partial](../benchmarks/results/buildopt-product-viability-v1/bv006/prefix-report.md):
the current C5 output reader passes 20 adversarial cases and complete owner
output/reuse proof. A delayed completion stamp was corrected and affected
instrument qualification passes. All 20 actual owner overhead requests succeed,
but capture imbalance is 588.682498 ms against the unchanged 100-ms limit.
The result is `INSUFFICIENT_READINESS`; wrapper p95 passes at 1.352567 ms.
The [collector/daemon attribution continuation](../benchmarks/results/buildopt-product-viability-v1/bv006-attribution/README.md)
has verified all 20 diagnostic requests and selected a
[two-stage precision design](../benchmarks/results/buildopt-product-viability-v1/bv006-attribution/precision-design.md).
Direct collector CPU is similar across arms; the diagnostic native imbalance
is 2011.844440 ms. There is no qualified owner timing or causal JIT/GC finding.
The [fixed 80-request pilot](../benchmarks/results/buildopt-product-viability-v1/bv006-attribution/pilot-result.md)
has now completed and independently passed all native/source/outcome checks.
Its 39.245578-ms point imbalance fits the original limit, but the variance-only
sizing rule requires 25,656 pairs per arm, above the registered maximum of 512.
Outcome: `INSUFFICIENT_PRECISION_NO_CONFIRMATION`. No confirmation starts.
The [daemon/supervisor diagnostic](../benchmarks/results/buildopt-product-viability-v1/bv006-daemon-diagnostic/README.md)
has completed all 32 fresh requests after a reproduced diagnostic sampler
failure. Whole-tree counting consumes approximately one supervisor CPU core;
two static scans reproduce the cost with daemons closed. Earlier timing
variation remains causally unresolved. The [disk-observer correction and control](../benchmarks/results/buildopt-product-viability-v1/bv006-disk-observer/README.md)
now pass behavior proof and all 32 owner builds. Individual scans use about
24% less CPU, but paired native-wall and supervisor-CPU intervals both cross
zero; full polling still consumes approximately one core. This is
`SUPERVISOR_CPU_REDUCTION_NOT_ESTABLISHED`, not owner timing qualification.
The [CPU isolation control](../benchmarks/results/buildopt-product-viability-v1/bv006-cpu-isolation/README.md)
completed 40 successful Gradle starts and verifies native/observer CPU masks,
but rejects observation quality: request 28 has a 2,104-ms sampling gap against
the unchanged 500-ms bound. Outcome: `FAILED_OBSERVATION_QUALITY`. The
[focused next diagnostic](../benchmarks/results/buildopt-product-viability-v1/bv006-cpu-isolation/next-step.md)
must distinguish the unmeasured write/wait phases before further owner work.
The [observer-only reproduction](../benchmarks/results/buildopt-product-viability-v1/bv006-observer-pause/README.md)
now completes four fixed cases with zero Gradle/JVM starts. Nominal file and
memory-sink cases do not reproduce the pause; controlled writer and process
pauses are correctly distinguished. `ORIGINAL_PAUSE_NOT_REPRODUCED` leaves the
historical cause unresolved. [Next: observe the actual sampler phases](../benchmarks/results/buildopt-product-viability-v1/bv006-observer-pause/next-step.md)
before any supported repair or separately bounded owner check. Production
source is unchanged; G0 and product timing remain unqualified.
The [actual sampler diagnostic](../benchmarks/results/buildopt-product-viability-v1/bv006-live-observer/README.md)
now verifies nine non-Gradle cases and two cold owner builds. Real synchronous
writes of 323/341 ms account for over 98% of two 327/344 ms sampling gaps, while both
heartbeats continue. Original 2,104 ms cause remains unresolved. Cold daemon
endpoint gaps of 504/603 ms exceed the original 500 ms criterion; no S1/G0 pass.
The subsequent [bounded writer separation](../benchmarks/results/buildopt-product-viability-v1/bv006-buffered-observer/README.md)
verifies R2/R3 with 15 final non-Gradle cases and one retained fixture-reader
failure. With a controlled 2,104-ms file write, the largest sampling gap is
100.774 ms and all 40 records persist; saturation, cancellation, incomplete
output and helper termination behave explicitly. Six altered artifact sets are
rejected; normal/race compilation and vet pass. Zero additional Gradle/JVM starts.
The [actual replay-consumer integration](../benchmarks/results/buildopt-product-viability-v1/bv006-observer-integration/README.md)
now verifies R4.1–R4.3: 29 final fixture workflows, C5 output/reuse checks,
release-checker corruption rejection, legacy compatibility and race coverage.
The CLI now owns bounded observation and verifies it before completion/resume;
v2 remains compatible. A controlled 2,104-ms writer pause leaves a 100.609-ms
sampling gap with all nine pending records persisted. Initial failures and one
corrected affinity test assumption remain retained. Zero new Gradle/JVM starts;
81 fixture reservations charged, 76 launches observed and five early launches
unresolved. This does not qualify owner timing or explain the historical pause.
Next is [actual-owner cold coverage and symmetric cost qualification](../benchmarks/results/buildopt-product-viability-v1/bv006-observer-integration/next-step.md).
No native allocation is active; S1/G0 remains unqualified.
The subsequent [CPU profile screen](./plans/buildopt-product-viability-v1-cpu-screen.md)
is capped at 12 builds each for 8/4/2 CPU, 36 total including warmups and failed
starts. It is exploratory, with no automatic extension or 80-start pilot per
profile; timing and lifecycle gates remain unchanged. Execution is blocked by
the observation prerequisite, with zero profile reservations or actual starts.
The owner subsequently approved the [three-step walltime decision](./plans/buildopt-walltime-decision-v1.md).
It closes instrumentation within two cold starts and admits the fixed 36-start
screen as exploratory walltime evidence, with diagnostic coverage failures kept
separate and formal readiness unchanged. A material signal alone may trigger an
independent engineering-history check; it cannot establish product viability.
This amendment supersedes the preceding screen-execution block for that narrow
exploratory purpose. No previous failed gate is promoted.
Measured lifecycle value remains unproved; all 80
validation transitions are untouched. All six other repository histories are
available, with their builds and opportunities still unmeasured.

| Route | Disposition | Required outcome |
|---|---|---|
| H1: incremental work inside a native Gradle task | CURRENT; scoped Checkstyle G2, BO-02 control and BO-03 assessment verified | Original ForbiddenPatterns case stays negative; Checkstyle warrants the planned short screen after applicable readiness checks, with no sustained saving established |
| H2: repair observed unnecessary native invalidation | CONDITIONAL | Admit a concrete native cause first; at most two causes, with the same correctness and lifecycle gates |
| H3: adaptive management of native corrections | REQUIRED LATER; implementation deferred | BO-11 follows fixed-correction value, manual MVP and sufficient lifetime evidence; prove added value over both native and fixed correction |
| Selected Build Impact results in Ktor and Beam | CONDITIONAL; BO-04 admission review only | Establish an unanswered, discriminating question before any new timing; generic plan reuse and fragment variants remain retired |

The [replay contract](./plans/buildopt-product-viability-v1-replay.md) starts
with 100 consecutive first-parent transitions, split into 20 engineering and
80 validation transitions. It separates the initial fixed-correction proof
from the required native/fixed/adaptive product comparison. A longer history,
another repository, or an adaptive controller does not revive a retired
optimization mechanism by itself.

## Meaning of the labels

| Label | Meaning for the next agent or researcher |
|---|---|
| CURRENT | Governs the next research work, subject to the user's actual authorization and the tracker prerequisites; not a claim that it is implemented |
| CONDITIONAL | May enter the current program only through its explicit admission gate |
| RETIRED | Discarded as a research direction under the tested mechanism; do not schedule another iteration under a new name |
| HISTORICAL | Closed or superseded plan/evidence; its original pass, stop, failure and unrun phases remain as recorded |
| RETAINED | Useful implementation, contract or evidence to maintain or reuse after checking that its inputs still apply |
| DEFERRED | No investment in this program; reopen only for a demonstrated dependency or a separately scoped problem |
| OUT OF SCOPE | Not part of this product viability study; no negative feasibility conclusion is implied |

These labels separate a research decision from a correctness result. A
qualified native patch remains qualified within its measured scope even when
its campaign is historical. A failed admission gate is not proof that every
possible mechanism in the area is impossible.

## Retired and deferred research routes

The [evidence ledger](./plans/buildopt-product-viability-v1-evidence.md)
contains the source receipts, limits and arithmetic behind these decisions.
Evidence IDs below refer to that ledger; the plan's investment decisions remain
unchanged.

| Route | Disposition | Why we stop or defer | Evidence |
|---|---|---|---|
| ForbiddenPatterns incremental correction in the frozen Elasticsearch seed workflow/prefix | RETIRED for this cohort | Native skips 17/20 requests; even full task deletion cannot meet the unchanged admission floors. Other tasks/repositories remain unproved | [BV-002 native audit](../benchmarks/results/buildopt-product-viability-v1/bv002/opportunity.md) |
| Unguarded Checkstyle `cacheFile` enablement in the pinned Elasticsearch contract | RETIRED as a compatible correction | Reproduced missing XML entries and hidden violations after same-mtime source changes or custom rule changes. A content-aware full-contract intervention requires its own proof | [C1-C4 admission](../benchmarks/results/buildopt-product-viability-v1/checkstyle-admission/README.md) |
| Whole structural profiles and task/graph omission as the product thesis | RETIRED | Selected wins failed to produce sufficient activation and net value across ordinary descendants | E01, E02 |
| Adaptive fragments, producer closures and request/history/graph recurrence variants | RETIRED | Chronological execution activated no fragments; related breadth and input-completeness gates failed. More source matches do not establish runtime opportunity | E01, E03 |
| Generic Safe Cache, another L1 or Edge cache as the default accelerator | RETIRED | No material general advantage over strong native-cache baselines; locality v3 qualified 0/3 eligible families | E04 |
| Worker/heap tuning and hot-state composition | RETIRED | The tested changes did not establish value and the bounded candidates regressed | E05 |
| Annotation-only cacheability searches and accumulating exact recipes as a growth strategy | RETIRED | Correctness, economic or prospective breadth gates stopped the successive routes. Selected native patch wins remain usable evidence | E06, E07 |
| Standalone source recurrence, broad explicit-opt-out scans and the same diagnostic cohorts | RETIRED | These searches did not establish sufficient material, source-bound opportunity; H2 requires a new observed cause | E03, E08 |
| Reopening CNC or the installed Elasticsearch campaign to finish their old downstream value phases | HISTORICAL, closed | CNC stopped at native admission; EIC stopped at persistent delivery. Later phases were unrun and remain so | Closed CNC and EIC trackers below; E09 |
| Repair the 100-ms persistent-upload path, expand central services or add shared state to unlock viability | DEFERRED | Infrastructure repair alone is not a new value mechanism; first establish useful native work and a needed delivery dependency | E09 |
| Test Optimization, other ecosystems and distributed execution | OUT OF SCOPE | Separate products or programs; no new feasibility claim from this study | Current plan scope |

For the 100-pair profile/fragment campaign, -368.623 s is the signed total
delta, not all optimizer overhead: 179.029 s is attributed BuildOpt cost and
-189.593 s is the runner/Gradle residual. Likewise, the EIC delivery stop does
not erase its verified correctness proofs or invent results for V/L/H/O.

## Retained foundations and evidence

Keep native diagnostics and graph capture, exact output comparison,
source-bound patch verification and inverse patches, provenance, pinned
toolchains, and the useful chronological harness concepts. Existing wrapper,
cache and transport code may remain necessary for maintained contracts or
baselines. Their presence is not an investment decision or activation proof.

| Location | How to use it now |
|---|---|
| [Implementation tracker](../implementation-tracker.md) and [master RFC](../gradle-build-optimization-platform.md) | Retained implementation/decision baseline. Historical phase and roadmap language does not schedule new research; use the governing plan's BO tracker |
| [Specifications](../specs/README.md), [contracts](../contracts/README.md) and [ADRs](../adr/README.md) | Preserve executable invariants. A closed POC contract can still be needed to validate old tooling; it grants no fresh experiment budget |
| [Benchmark results](../benchmarks/README.md) and host-local evidence | Preserve raw data, manifests, pins, negative outcomes, costs and failed attempts. Never overwrite them with a new run |
| [Findings](./README.md#findings-and-recommendations) and old handoffs | Historical observations; old “next”, “active” and “authorized” wording is scoped to the original campaign |
| `cmd/`, `internal/`, `jvm/`, `rust/`, `dev/`, `fixtures/`, `packaging/` | Retained implementation and verification tooling. This cleanup deletes or disables none of it; remove code only after a separate dependency and behavior review |
| [User guides](./README.md#user-and-operator-guides) and [runbooks](../runbooks/README.md) | Retained instructions for implemented capabilities; not evidence of a viable or self-managing product |

The master RFC and all executable specs, raw results and source files retain
their existing bytes in this cleanup. Historical plan/finding bodies retain
their original contents below a disposition notice. Index and overview prose
is updated where it misleadingly names a closed route as current.

## Complete plan inventory

Every Markdown file directly under `docs/plans/` is classified here. All
HISTORICAL entries are closed as next-work instructions, including substudies
that passed. Their notices link back here rather than rewriting old decisions.

| Plan or tracker | Disposition |
|---|---|
| [Research execution plan and BO tracker, 2026-09-14](./plans/buildopt-research-execution-plan-2026-09-14.md) | CURRENT; governing priorities and next-work tracker |
| [Product viability plan](./plans/buildopt-product-viability-v1.md) | RETAINED technical background; conflicting priorities and commercial phases superseded |
| [Product viability tracker](./plans/buildopt-product-viability-v1-tracker.md) | RETAINED prior execution record; future work follows the BO tracker |
| [Product viability replay contract](./plans/buildopt-product-viability-v1-replay.md) | CURRENT technical measurement/correctness contract; commercial phases out of scope |
| [Product viability evidence ledger](./plans/buildopt-product-viability-v1-evidence.md) | RETAINED evidence; not a work queue |
| [Checkstyle identical-code measurement control](./plans/buildopt-checkstyle-measurement-control-v1.md) | COMPLETED at BO-02 on 2026-09-14; retained protocol, no false material signal in this one control; no new allocation |
| [CPU profile screening budget](./plans/buildopt-product-viability-v1-cpu-screen.md) | HISTORICAL; preserve the original limits and outcomes, no new CPU-profile sweep |
| [Adaptive fragments](./plans/adaptive-fragment-generalization-tracker.md) | HISTORICAL |
| [Centralized cache and state](./plans/centralized-cache-and-state-roadmap.md) | DEFERRED; implemented foundation retained |
| [Change-aware producer closure](./plans/change-aware-producer-closure-poc-tracker.md) | HISTORICAL |
| [Change-scoped candidate correctness](./plans/change-scoped-candidate-correctness-v1.md) | HISTORICAL |
| [Change-scoped critical-path discovery](./plans/change-scoped-critical-path-discovery-v1.md) | HISTORICAL |
| [Change-scoped native capture](./plans/change-scoped-critical-path-native-capture-v1.md) | HISTORICAL |
| [Chronological failure successor selection](./plans/chronological-failure-successor-selection-v1.md) | HISTORICAL |
| [Complete native correction tracker](./plans/complete-native-correction-poc-tracker.md) | HISTORICAL |
| [Complete native correction plan](./plans/complete-native-correction-poc.md) | HISTORICAL |
| [Configuration-input native corrections](./plans/configuration-input-native-corrections-poc.md) | HISTORICAL |
| [Critical-path build-logic correction](./plans/critical-path-build-logic-correction-v1.md) | HISTORICAL |
| [Critical-path-first native patch](./plans/critical-path-first-reviewed-native-patch-v1.md) | HISTORICAL |
| [Critical-path successor selection](./plans/critical-path-successor-selection-v1.md) | HISTORICAL |
| [Durable native optimization](./plans/durable-native-optimization-poc-tracker.md) | HISTORICAL |
| [Economic opportunity first](./plans/economic-opportunity-first-poc-tracker.md) | HISTORICAL |
| [Economics-gated native patch v1](./plans/economics-gated-reviewed-native-patch-v1.md) | HISTORICAL |
| [Economics-gated native patch v2](./plans/economics-gated-reviewed-native-patch-v2.md) | HISTORICAL |
| [Fresh generic optimization](./plans/fresh-generic-optimization-poc-tracker.md) | HISTORICAL |
| [Fresh graph recurrent-group confirmation](./plans/fresh-graph-recurrent-group-confirmation-v1.md) | HISTORICAL |
| [Graph-aware breadth discovery](./plans/graph-aware-breadth-discovery-v1.md) | HISTORICAL |
| [Graph-aware history admission](./plans/graph-aware-history-admission-v1.md) | HISTORICAL |
| [Graph-owner recurrence inventory](./plans/graph-owner-recurrence-inventory-v1.md) | HISTORICAL |
| [History-admitted breadth](./plans/history-admitted-breadth-v1.md) | HISTORICAL |
| [History-admitted correctness](./plans/history-admitted-candidate-correctness-v1.md) | HISTORICAL |
| [History-admitted paired value](./plans/history-admitted-paired-value-v1.md) | HISTORICAL |
| [Installed Elasticsearch tracker](./plans/installed-elasticsearch-native-correction-v1-tracker.md) | HISTORICAL |
| [Installed Elasticsearch plan](./plans/installed-elasticsearch-native-correction-v1.md) | HISTORICAL |
| [Normalization-aware cacheability](./plans/normalization-aware-cacheability-poc-tracker.md) | HISTORICAL |
| [Observed request portfolio](./plans/observed-request-portfolio-poc-tracker.md) | HISTORICAL |
| [One-command onboarding](./plans/one-command-onboarding-roadmap.md) | RETAINED foundation; historical roadmap |
| [Product-window graph recurrence](./plans/product-window-graph-recurrence-v1.md) | HISTORICAL |
| [Prospective reviewed native patch](./plans/prospective-reviewed-native-patch-controlled-trial-v1.md) | HISTORICAL |
| [Recurrent-root correctness](./plans/recurrent-root-candidate-correctness-v1.md) | HISTORICAL |
| [Recurrent-root paired value](./plans/recurrent-root-paired-value-v1.md) | HISTORICAL |
| [Remote cache locality v2](./plans/remote-cache-locality-value-poc-tracker.md) | HISTORICAL |
| [Remote cache locality v3](./plans/remote-cache-locality-value-v3-poc-tracker.md) | HISTORICAL |
| [Request-aligned learning](./plans/request-aligned-learning-poc-tracker.md) | HISTORICAL |
| [Native patch customer economics](./plans/reviewed-native-patch-customer-economics-v1.md) | HISTORICAL |
| [Native patch delivery](./plans/reviewed-native-patch-delivery-v1.md) | HISTORICAL |
| [Native patch owner acceptance](./plans/reviewed-native-patch-owner-acceptance-v1.md) | HISTORICAL |
| [Native patch portfolio](./plans/reviewed-native-patch-portfolio-v1.md) | HISTORICAL |
| [Source-bound configuration-input corrections](./plans/source-bound-configuration-input-corrections-poc.md) | HISTORICAL |
| [Spring JMS graph confirmation](./plans/spring-jms-graph-aware-confirmation-v1.md) | HISTORICAL |
| [Spring Messaging correctness](./plans/spring-messaging-candidate-correctness-v1.md) | HISTORICAL |
| [Spring Messaging fresh-control correctness](./plans/spring-messaging-fresh-control-correctness-v1.md) | HISTORICAL |
| [Spring Messaging fresh-graph confirmation](./plans/spring-messaging-fresh-graph-confirmation-v1.md) | HISTORICAL |
| [Spring Messaging paired value](./plans/spring-messaging-paired-value-v1.md) | HISTORICAL |
| [Sticky-wrapper learning](./plans/sticky-wrapper-learning-poc-tracker.md) | HISTORICAL |
| [Strict diagnostic capture reliability](./plans/strict-diagnostic-capture-reliability-poc.md) | HISTORICAL |
| [Third-family fresh-graph confirmation](./plans/third-family-fresh-graph-confirmation-v1.md) | HISTORICAL |
| [Third-family graph recurrence](./plans/third-family-graph-recurrence-v1.md) | HISTORICAL |
| [Three-class chronological value](./plans/three-class-chronological-value-v1.md) | HISTORICAL |
| [Verified request hit](./plans/verified-request-hit-poc-tracker.md) | HISTORICAL |
| [Wrapper-coordinated native corrections](./plans/wrapper-coordinated-native-corrections-poc.md) | HISTORICAL |

## Avoid repeating a closed route

Before scheduling work, read this register and the governing plan's BO tracker.
Do not follow an unchecked box, a stale handoff, a retained executable or an old
successor authorization as a new work order.

If new evidence warrants reconsideration, record the exact retired route,
its failed gate, the materially changed causal mechanism or workload, the new
evidence, and the proposed discriminating test and cost. Resolve any new scope
decision with the user. Reuse existing authorization when it actually covers
the work; this rule does not add approval gates to routine current-plan steps.
Keep the old result closed and give a successor its own identity and inputs.

When research status changes, update this register, the governing BO tracker and
the documentation entry points together. Add every new plan to the inventory.
Never move thresholds, omit negative rows or rewrite an old result to make
the successor appear qualified.
