# BuildOpt Product Viability v1 Implementation Plan

> **For agentic workers:** Execute this research plan task by task, using the
> `superpowers:executing-plans` guidance where applicable. Follow the user's
> current authorization and repository procedures. Do not delegate, commit,
> publish, contact customers, or start experiments merely because they appear
> in this plan. Checkboxes and execution evidence live in the tracker.

**Goal:** Establish whether BuildOpt can autonomously discover, validate,
maintain and replace native build improvements whose measured lifetime benefit
exceeds the full adaptation cost and for which a defined customer will pay.

**Architecture:** Start with a local, reviewable native Gradle correction and
leave Gradle in charge of ordinary builds. Evaluate its behavior through real
Git transitions, then test an adaptive controller that observes, suspends,
searches, validates and replaces corrections within a bounded policy. Keep
central services and the installed wrapper outside this new mechanism's
critical path; this is a proposed product contract, not a claim about the
current implementation.

**Tech stack:** Existing Go diagnostics and experiment tooling, Gradle/JVM
build logic, Git worktrees, existing patch verification infrastructure, and
repository-pinned toolchains through `dev/run`.

**Spec:** The decision and product requirements are below; the measurement
contract is [Historical Replay v1](./buildopt-product-viability-v1-replay.md).

**Updated:** 2026-09-10. **State:** The ForbiddenPatterns seed phase remains negative. Checkstyle correctness and local replay qualification retain their frozen scopes. The corrected short screen and disjoint history are complete: selected CPU cases improve, but the historical candidate loses 4.260 s/build (19.158% slower). BV-006 remains partial for formal readiness; lifecycle and product viability remain unproved.
**Candidate proof:** [Original correctness scope](../../benchmarks/results/buildopt-product-viability-v1/checkstyle-prototype/README.md) plus [corrected task bindings and actual owner results](../../benchmarks/results/buildopt-product-viability-v1/bv006-screen-completion/README.md).
**Scoped decision:** `NO_MATERIAL_SIGNAL_IN_DISJOINT_HISTORY`. All 54 corrected-round native requests and 27 complete live/independent output pairs pass. The historical worker-exclusive-affinity observation stays failed; no older diagnostic gate changes.
**Next proposed work:** [Isolate the ordinal7 Main-checking regression](../../benchmarks/results/buildopt-product-viability-v1/bv006-screen-completion/next-step.md), with an explicitly bounded single-profile diagnostic before further history or adaptive implementation. No new experiment is started by this plan.
**Current measurement record:** The [fixed candidate amendment](./buildopt-screen-completion-fixed-v2.md), [history time amendment](./buildopt-screen-completion-history-time-v2.md), and [negative observation closeout](./buildopt-screen-completion-history-observation-v4.md) preserve their respective inputs and failures. All80 formal validation transitions21–100 remain untouched.
**Execution record:** [Detailed tracker](./buildopt-product-viability-v1-tracker.md).
**Evidence and decisions:** [Evidence ledger](./buildopt-product-viability-v1-evidence.md).
**Repository research disposition:** [Current, retired and retained work](../research-status.md).
**Adaptive requirement:** Incorporated on 2026-09-08. The initial fixed-correction
study proves a mechanism; BV-011 must prove the self-managing product separately.

## 1. Objective and global constraints

The question is whether a customer can adopt BuildOpt, receive a useful change,
keep the benefit as its repository evolves, and recover the cost of discovery,
validation, review, operation, and maintenance. A faster selected task, a safe
fallback, or a successful package build closes only its own proof obligation.

- Compare against the strongest compatible native Gradle baseline we can
  justify before candidate timings, including native cache and incremental
  execution. Charge changes to that baseline symmetrically.
- Preserve requested commands, required outputs, failure behavior, and owner
  tests. No test selection or weaker input/output contract supplies savings.
- Retain all negative observations, unavailable subjects, aborted runs, and
  costs. Do not interpret missing evidence as zero or a successful fallback as
  an optimization.
- Freeze product implementation, controller rules, allowed action classes,
  thresholds, subjects, state policies, metrics, exclusions and limits before
  evaluation. The adaptive controller's learned state and selected corrections
  may evolve during validation using only information already available to it.
  Future commits or experiment-only control results must not supply that state.
- Existing STOP decisions and immutable evidence remain unchanged. This plan
  selects future investment; it neither reruns nor reclassifies those studies.
- The owner requested execution on 2026-09-08. Local work now follows this
  plan's dependencies and declared allocations without asking again at every
  checkbox. External actions retain their own authorization boundaries.
- No new daemon, hosted service, dashboard, general cache, or language expansion
  is a prerequisite for the first technical result.

## 2. Investment decisions

These are scope decisions for this program, not claims that a mechanism can
never work in any environment. Evidence IDs resolve in the linked ledger.

| Route | Decision | Reason and consequence |
|---|---|---|
| Whole structural profiles, graph omission, adaptive fragments, producer-closure and recurrence variants | Stop research under the tested architecture | E01/E02/E03: favorable selected cases did not become sustained installed activation and payback. No additional calibration or recurrence search under a new name. |
| Generic Safe Cache/L1 or Edge as the default accelerator | Stop as the product thesis | E04: native-cache parity and failed locality value gate. Preserve existing code and safety contracts; do not invest in a new cache backend. |
| Worker/heap tuning and hot state | Keep retired | E05: regressive evidence. No parameter sweep or wrapper-wide runtime tuning. |
| More annotation-only cacheability searches and exact-recipe accumulation | Stop as the growth strategy | E06/E07: real isolated wins, failed replication, semantic failures, and low material opportunity. Existing corrections may strengthen a baseline or a customer fix. |
| More source-only recurrence or explicit-opt-out inventories | Stop as standalone experiments | E03/E08: matching source patterns did not establish actionable, material ordinary-build savings. |
| Repair the EIC 100-ms durable upload path to unlock viability | Defer infrastructure work | E09: a real installed admission failure, but solving it would not establish ordinary-build value or prevalence. The failed EIC result remains closed. |
| Avoid repeated work inside a native task after small input changes | Primary hypothesis H1 | E10 identifies a concrete full-file rescan. Prove incremental semantics and whole-workflow value on evolving inputs. |
| Repair an observed, unnecessary native invalidation at its source | Conditional hypothesis H2 | Explicit-opt-out scans did not test all causes of invalidation. Admit only a measured source-bound cause; no generic search for another annotation. |
| Automatically observe, suspend, search, validate and replace corrections | Required adaptive product hypothesis H3 | Must demonstrate net value beyond a fixed correction, controlled no-change overhead, safe replacement and bounded discovery costs. Retiring old adaptive-fragment mechanisms does not retire adaptive management of newly qualified native corrections. |
| Test Optimization, compiler replacement, distributed execution, or another build ecosystem | Outside this program | Different contracts and economics. Do not use them as an automatic escape from negative results. |

Keep diagnostics, output-equivalence checking, native task graphs, exact patch
application/reversal, artifact provenance, and the useful chronological runner
concepts. Reuse does not imply that an existing implementation already supports
this plan. In particular, EIC value/chronology adapters remain partial.

## 3. Product hypothesis and intended customer

The initial customer hypothesis is a team maintaining a substantial Gradle
build, with recurring developer or CI workflows and enough build ownership to
review changes. Persistent runners and local development are the first target
for H1. Ephemeral CI is a separate segment until its state model passes.

The smallest product promise is: identify expensive avoidable work, propose a
small native correction, prove its effect on the actual workflow, monitor its
continued applicability and benefit, and search for a replacement when justified.
The final product must perform this cycle within its configured action policy;
a manually maintained correction is an intermediate mechanism result.

Current tools already provide build analysis, input comparison, and AI-assisted
diagnosis. Differentiation must therefore be tested around a correct delivered
change, lower engineering effort, and maintained savings. It is an inference
to test, not a claim that competitors cannot do this. See E11.

There are two possible businesses to distinguish:

1. A recurring product, if new opportunities or maintenance justify continued
   use and customers actually renew or continue paying.
2. A paid optimization engagement, if useful changes are rare, bespoke, and
   durable without continuing assistance. This may be viable, but it would
   falsify the recurring-product hypothesis.

## 4. H1: Native incremental work

The first subject is Elasticsearch `ForbiddenPatternsTask`. Its inspected
implementation scans every input file when the task runs. C007/C008 show
seconds inside that action following a benign single-file mutation; they are
diagnostics, not paired whole-workflow savings.

The candidate should use Gradle's native change information to process only
eligible changed files after a successful prior execution, while doing a full
scan when history or semantics require it. The existing source recreates its
file collection in the getter, so a correct implementation needs a stable
input property as well as an incremental task action. The applicability audit
must check all callers and all properties; adding an annotation is insufficient.

For the seed case, the control should include the previously reviewed
cacheability correction wherever its source and semantic contract match. The
incremental candidate builds on that same control. Record the control's delta
from upstream: do not attribute the old cacheability benefit to H1. If the
old correction is not applicable in the selected history, freeze an explicit
compatible baseline before timing rather than silently dropping it.

Correctness must cover additions, modifications, deletions, renames, empty
sources, rule/exclusion/root changes, malformed text, native validation errors,
failed builds followed by fixes, output/history loss, cache restoration,
cross-root execution, and supported Gradle/JDK changes. Successful markers alone
cannot prove preserved diagnostics or all owner workflow outputs.

H1 is falsified for a subject when native Gradle already avoids the relevant
work, incremental processing cannot preserve semantics, or the complete
workflow does not clear the economic gate over the frozen sequence.

### 4.1. Approved Checkstyle subject amendment, 2026-09-08

The owner requested continuation of the [Checkstyle proposal](../../benchmarks/results/buildopt-product-viability-v1/next-investigation.md).
The completed ForbiddenPatterns decision remains immutable and negative. The
new subject is Elasticsearch's existing Checkstyle execution in the same
`:server:precommit` workflow. Observed overlapping task spans motivate an
admission audit; they are not an estimate of realizable savings.

Execute C1 (source and output contract), C2 (native capability qualification),
C3 (residual materiality), and C4 (complete six-repository selection denominator)
as tracked below. Existing native engine caching or incremental capabilities
must become part of N only if they preserve the full contract. A native setting
alone is not a new BuildOpt mechanism. If none can preserve the contract,
record the concrete counterexample before considering a residual correction.

The 20 engineering transitions remain discovery data, including unchanged
requests; the 80 validation transitions stay unconsumed. C4 may complete history
prerequisites before seed value is proved, but opportunity execution and G4
replication retain their existing dependencies. Keep all six frozen endpoints
and unavailable outcomes. Do not select only favorable source-changing commits.

The new admission allocation is six hours, 20 total outer/nested Gradle starts,
80 direct engine JVM starts, at most 120 GiB of new state and at least 40 GiB
free. Record all setup/failure charges separately from the closed phase's
31 reservations and 30 actual starts. Native discovery stops on a resolved
negative gate or a declared limit. Only an admitted residual intervention may
advance to C5 and the unchanged BV-003 through BV-013 proof gates, including
the 5% / 1-second economic floors and fixed/adaptive/customer requirements.

**Admission result:** [C1-C4 evidence](../../benchmarks/results/buildopt-product-viability-v1/checkstyle-admission/README.md)
rejects raw engine-cache enablement on three reproduced correctness/output
counterexamples. The residual fixed-dependency model reaches 1.49245 s per
transition and 5.9452%, which admits a bounded content-aware prototype; it is
not measured saving. All six replication histories are verified, with their
buildability and opportunities still unmeasured. The
[candidate boundary](../../benchmarks/results/buildopt-product-viability-v1/checkstyle-admission/candidate-boundary.md)
is the next implementation input. The old ForbiddenPatterns G1 result stays negative.

## 5. H2 and H3 admission

H2 receives at most one bounded investigation after the H1 seed decision, or
when H1 discovery exposes its causal signature. Admission requires two
consecutive native requests with complete task/input/output provenance and an
identified build-logic cause of unnecessary work. Examples to investigate only
when observed are needless regeneration of stable inputs or an over-broad
declared input that can be corrected without omitting a real dependency.

`No history available`, an ordinary cache miss on genuinely changed content,
or an expensive standard compiler task is not such a cause. Before coding,
state the source change, semantic proof, and recoverable workflow-time ceiling.
Allow at most two concrete source-cause candidates within the same investigation;
if neither qualifies, close H2 instead of broadening the detector indefinitely.

H3 follows a technically positive correction. First measure a fixed correction
to establish its mechanism and survival. BV-011 then tests the full adaptive
product against both native Gradle and that fixed-correction strategy on the
same new chronological cohort. Adaptation is required for a self-managing
product claim, rather than an optional maintenance feature after that claim.

The controller follows `OBSERVE -> SUSPECT -> SEARCH -> VALIDATE -> ACTIVE`,
with `SUSPENDED` and `COOLDOWN` states. Correctness and invalid applicability
bypass performance waiting windows: suspend the affected correction before use
and preserve the safe native workflow. A normal cache miss or full scan after
lost history does not by itself invalidate an otherwise supported correction.

| Signal | Required response |
|---|---|
| Correction remains applicable and economically useful | Keep it, collect bounded observations and avoid unnecessary exploration |
| Source/API/input contract invalidates applicability | Suspend immediately; restore the verified native path and schedule diagnosis |
| Benefit appears to decline | Confirm using comparable windows and charged native probes; a slow build alone is not sufficient evidence |
| Persistent loss of value or a newly observed material opportunity | Start a bounded search for a repair or another admitted correction |
| Alternative passes correctness and net economics | Activate within the configured authorization policy; retain exact inverse and rollback proof |
| No profitable alternative, inconclusive evidence, or exhausted search budget | Keep a still-useful correction or native Gradle, then back off until a declared retry condition occurs |

Production activation and suspension thresholds are distinct and calibrated
from prefix variability; require repeated evidence for economic changes and
use a cooldown to prevent repeated switching. The 5%/1-second campaign gate is
not automatically a per-build drift threshold. Freeze the actual thresholds,
window lengths, probe schedule, maximum candidates, episode costs and retry
conditions before adaptive validation. BV-011 cannot start with these unset.

Autonomy covers detection, search, validation and activation only within the
configured, previously authorized correction classes. A new arbitrary source
rewrite or a change outside that policy remains a proposal for review. Neither
the controller nor the evaluator may secretly ask an engineer to repair every
commit and call the result self-managing. Record human interventions, costs and
waiting time; count such episodes separately from automatic completion.

Search must not block the native build on an unbounded optimization job. Charge
all observer, probe, discovery, verification, switching and retry work even if
it runs outside the foreground invocation. Only admitted H1/H2 actions enter
the search space; this requirement does not reopen retired optimizers. The
[adaptive replay contract](./buildopt-product-viability-v1-replay.md#10-adaptive-product-evaluation)
defines the states, comparison arms, policy fields, evidence and pass criteria.

## 6. Evidence ladder and decisions

Thresholds below are planning judgments, not market facts. They deliberately
require meaningful complete-workflow benefit. Freeze them in the executable
manifest before evaluation; changing them after results creates a new protocol
version and leaves the original decision intact.

| Gate | Required outcome | If it fails |
|---|---|---|
| G0: Reproducible foundation | Exact working-tree/artifact identity, supported toolchain resolution, verified relevant build/tests, native workflow, complete capture, and a feasible allocation | Mark prerequisite blocked or incomplete; no value claim |
| G1: Causal opportunity | Native execution identifies avoidable work and a source-level intervention whose optimistic whole-workflow ceiling can reach G3; scope and frequency come from observed prefix requests | Close this subject as no material opportunity; do not implement on source shape alone |
| G2: Correctness | Every required semantic and owner integration case passes, no unexplained output/diagnostic difference, exact reversal, and refusal of unsupported source/version drift | Stop candidate execution, preserve failure, fix within the declared correction budget or reject |
| G3: Seed value | Two complete chronological replications; each clears the replay's coverage rules, at least 5% aggregate net workflow reduction and at least 1 s net per scheduled validation transition; positive dependence-aware interval; payback reached and retained within the 100-transition sequence | Negative or inconclusive seed result; no product packaging based on a selected fast task |
| G4: Repeatability across owners | Same mechanism/policy tested on two additional independent repositories; at least two of the three total subjects pass G3, and every selected subject's outcome is retained | At most a specialized correction; no multi-repository product claim |
| G5: Usable adaptive product | BV-010 delivery and BV-011 controller correctness pass; complete N/F/A replay demonstrates net value versus native, added adaptation value versus a fixed correction on natural change histories, and bounded stable/no-opportunity overhead | Useful mechanism or experimental pilot only; no validated self-managing product claim |
| G6: Customer value | Three explicitly authorized pilots from at least two independent organizations; at least two adopt the correction, observe positive conservative net value, and pay for a pilot; recurring claims additionally require an observed paid continuation | Commercially unverified, or evaluate a bounded services offering |

G3 also requires zero candidate-attributable correctness failures and the
latency guardrail in the replay contract. A correct refusal counts as no action
and contributes its overhead. A native failure is a classified unavailable
comparison, not an optimizer win.

The 2/3 technical and 2/3 pilot gates are progression hurdles for a focused
offering, not estimates of market prevalence. Report the entire discovery
denominator and customer recruitment denominator. Do not revive the old generic
claims when a narrowly selected segment passes.

Early G3 uses the actual measured operator/prototype delivery process, with
all customer-required work identified. It cannot assume an inexpensive future
automatic path. BV-010 must substitute measured installed delivery costs and
reopen the economics if they invalidate the result. G5 functional proof can
admit an explicitly experimental pilot even when no natural adaptation event
has occurred; full G5 and H3 added-value evidence remain unverified until a
frozen extension or pilot supplies those events. Such a pilot is not a passed
self-managing product gate, and cannot make G6 or the final product decision
silently bypass BV-011.

## 7. Metrics and economics

Three primary outcomes govern this program:

1. **Net workflow time saved:** compare the full invocation on the same
   revision, include recurring product work and allocated adoption/maintenance
   cost once, and report per-repository and per-state-mode results.
2. **Delivery and maintenance efficiency:** elapsed time to first verified
   correction, machine work, active engineer/reviewer time, successful delivery
   rate, refusal rate, and cost of each maintenance event.
3. **Observed customer value:** retained use, measured workflow frequency,
   conservative benefit after fees and customer effort, paid adoption, and paid
   continuation. No willingness-to-pay number exists yet.

For H3 additionally measure detection delay, time without a useful correction,
false alarms, repeated switches, search/qualification cost, automatic completion
rate, human intervention rate, and net saving versus the fixed strategy.
Correctness and p95 regression are guardrails. Changed-file work, task execution
reasons, action rate, cache outcomes, and patch survival diagnose the result;
none substitutes for the primary outcomes.

Keep three ledgers: recurring customer machine cost, customer active human
time, and research expenditure. The customer's operational payback includes
product discovery, verification, adoption, and maintenance. Research cost is
reported separately and also in a fully loaded sensitivity analysis. Do not
subtract the same internal phase twice or treat an experimental control build
as a permanent customer obligation unless the product actually runs it.

For commercial evaluation use customer-provided runner rates and actual build
frequency. Report developer waiting time separately; do not turn every saved
second into salary savings without an explicit observed blocking fraction.
Measure support and delivery cost before proposing a margin or price. A plan
with unknown customer inputs cannot establish ROI or a market size.

## 8. Scope, duration, and resource controls

Start with **100 consecutive first-parent transitions per subject**: 20 for
engineering and method development, followed by 80 locked validation
transitions. The initial ten transitions qualify the runner inside the prefix.
After freezing the candidate, replay the anchor and the complete sequence from
fresh independent state in each of two replications. Development observations
do not become confirmation rows. Record the actual calendar span.

This first N/I experiment isolates the mechanism. The final H3 experiment uses
N/F/A: optimized native, a fixed correction, and the adaptive product. Its own
20-transition prefix fixes the controller policy; it continues learning and
replacing eligible corrections throughout its 80 validation transitions. Use
an unconsumed cohort frozen before H3 development, as specified in the replay
contract, so controller tuning cannot exploit earlier confirmation outcomes.

This replaces a one-year first attempt with a bounded test. If the sequence is
too short to observe maintenance or toolchain changes, report that limitation.
After G4, a separately frozen extension may cover the next 200 transitions or
30 calendar days, whichever requires more transitions, subject to sizing before
launch. Failure to fit is an incomplete extension, not permission to sample
only favorable commits. No year-long campaign is scheduled now.

| Stage | Initial planning ceiling | Review boundary |
|---|---|---|
| Foundation and causal seed audit | 6 elapsed hours; 30 total Gradle starts | Before candidate implementation |
| Candidate and semantic proof | 12 elapsed hours; 120 total Gradle starts | Before engineering prefix |
| Runner qualification and development prefix | 12 elapsed hours; 120 total Gradle starts | Before candidate/manifest freeze |
| Seed confirmation | 72 elapsed hours; 600 total Gradle starts | Checkpoint every 20 transitions; complete both replications unless safety, validity, or allocation stops execution |
| Two-repository replication | 144 elapsed hours; 1,200 total Gradle starts | Separate complete outcome per subject |
| Conditional H2 investigation | 6 elapsed hours; 20 total Gradle starts | Close or produce one admitted source-cause experiment |
| Local product delivery proof | 12 elapsed hours; 120 total Gradle starts | Before customer installation or a delivery-cost claim |
| Local adaptive-controller qualification | 12 elapsed hours; 120 total Gradle starts | Before N/F/A public replay; freeze controller cases, policy and accounting |
| Adaptive N/F/A replay, per subject | 108 elapsed hours; 900 total Gradle starts | Two replications require 606 outer workflow starts before extra probes/preparation; size those explicitly |

These are upper bounds, not reservations, estimates of fit, or permission to
consume paid services. All preparation, failed starts, and nested TestKit
starts count. Use a 1,200-second default native child timeout, 120 GiB maximum
new campaign footprint and 40 GiB minimum free space. Size actual commands,
trace growth and nested counts before launch. Reconcile allocated and measured
work at each checkpoint; retain separate CPU and elapsed time where available.

Resize an unstarted stage using measured costs and the user's applicable
execution authorization. An exhausted running allocation yields an incomplete
record and a concrete continuation estimate; it does not falsify a mechanism.
Never lower proof requirements to fit a budget, or repeat a failed correction
more than the user's three-correction limit. Leave unrelated processes and all
older evidence roots alone.

The adaptive stage covers the seed and the two G4-selected repositories, each
with its own checked allocation. Its initial combined ceiling is 324 hours and
2,700 total Gradle starts, not a resource reservation or permission to spend it.
Mode E and later calendar extensions require separately frozen sizing. Search
episode limits are nested within these totals; reaching an episode limit causes
controller backoff, whereas reaching a campaign limit yields incomplete evidence.

## 9. File ownership and execution order

| Artifact or existing seam | Responsibility |
|---|---|
| This plan | Product decisions, hypotheses, progression gates and scope |
| [Evidence ledger](./buildopt-product-viability-v1-evidence.md) | Prior findings, limits, retirement rationale and source pointers |
| [Replay contract](./buildopt-product-viability-v1-replay.md) | Cohort, state, timing, statistics, correctness and accounting definitions |
| [Tracker](./buildopt-product-viability-v1-tracker.md) | Step status, actions, expected outcomes, proof and recovery |
| `dev/current-longitudinal-*`, `dev/installed-elasticsearch-runner/` | Inspect reusable capture/verification seams; preserve closed protocol behavior |
| `internal/durablecatalog/`, `jvm/patcher/` | Later diagnostic admission and exact native patch delivery, only after mechanism proof |
| Future `specs/poc-product-viability-v1.*` | Executable manifest/schema selected and implemented in BV-005; absent today |
| Future `benchmarks/results/product-viability-v1/` | Portable receipts, summaries, decisions and provenance; absent today |
| Future `.tools/state/buildopt-product-viability-v1/` | Task-owned worktrees, large raw evidence, checkpoints and cumulative costs; absent today |

Execution follows BV-001 through BV-013 and the dependencies in the tracker.
Customer research material can be prepared after the seed decision; actual
outreach and data access require their specific authorization. Do not wait for
customers to complete the authorized public-repository technical proof.

## 10. Program stop point

The program ends with one evidence-backed decision: proceed with a focused
product pilot, operate a specialized optimization service, or stop this
product thesis. A positive seed alone does not settle that decision.

If H1 fails on the seed and H2 provides no admitted cause, close the current
technical thesis. If improvements exist but cannot be delivered economically,
close the automated-product claim. If fixed corrections work but adaptation
does not add net value, retain the narrower result and close the self-managing
product claim. If technical delivery works but customers
do not pay or continue, close the corresponding commercial claim. Preserve
useful fixes and evidence without manufacturing another successor experiment.

The current task ends earlier: these documents must be complete, linked,
internally consistent and checked. All future execution remains accurately
marked as unstarted.
