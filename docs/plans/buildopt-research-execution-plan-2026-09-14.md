# Build Optimization research execution plan

Date: 2026-09-14. Program: `BUILDOPT-VIABILITY-V1`.
Status: governing plan in execution; BO-01 through BO-04 verified; BO-05 partial after an observer failure.

This plan records the direction agreed after the research review and its
discussion. It owns research priorities, sequencing and the
[execution tracker](#execution-tracker) below. Read it with the
[research status register](../research-status.md) before choosing work.

## Objective and north star

Demonstrate that Build Optimization can consistently shorten real project
builds, preserve their required results, and save more time than it adds.
The eventual product should maintain that saving as the project changes.

The main measure is **net time saved per scheduled validation build across a
complete development history**. Count builds where the optimization does
nothing, along with the work required to discover, validate, apply and maintain
it. Measure the complete requested command. Adding the durations of concurrent
tasks does not measure elapsed build time.

Start by proving a fixed improvement. Then make it usable through an
experimental MVP with manual controls. Finally, test whether automatic
selection and maintenance add further savings. A useful fixed correction is
an intermediate result; it does not prove a self-managing product.

This is a research plan. Customer recruitment, paid pilots, staffing allocations,
commercial launch and expansion to other build systems are outside its scope.

## Authority and preserved evidence

This plan supersedes conflicting research priorities and next-step instructions
in the [earlier plan](./buildopt-product-viability-v1.md), its
[execution record](./buildopt-product-viability-v1-tracker.md), old handoffs,
architecture overviews and the quarterly review. Those documents retain their
original results, inputs and historical decisions.

The [replay contract](./buildopt-product-viability-v1-replay.md) remains the
technical source for measurement, correctness, cost accounting and fixed versus
adaptive comparisons. Its commercial phases are not part of this plan.
The [existing evidence ledger](./buildopt-product-viability-v1-evidence.md)
remains a source of results, not a second work queue. Existing executable
contracts and frozen experiment manifests remain binding within their scope.

Registering this plan does not start a trial, grant a new allocation, or
authorize a commit or publication. When execution is requested, follow the
dependencies and recorded limits without asking again about routine work
already covered by that request. A passed prerequisite permits the next
in-scope step; an old unchecked box does not.

Changes from the earlier research direction are explicit:

- Evaluate an MVP with manual controls before adaptive management.
- Review the unmeasured lifetime of the strongest Build Impact cases through
  the bounded admission step BO-04. This does not reopen generic plan reuse.
- Retain current technical acceptance criteria and all negative results.
- Separate fixed-correction value, usable MVP value and adaptive value.

Do not change thresholds, replace unfavorable commits, reinterpret old failures
as passes, or rename a retired mechanism to restart it. A material change in
scope or method needs a recorded prospective amendment under the user's actual
authorization. Preserve the original decision and identify the new experiment
separately.

## Acceptance criteria

These are continuation thresholds, not a product-wide performance promise.
Apply them per repository and per replication, using the replay contract's
formulas. Charge required work once, including work outside the foreground
command. Keep experiment-only work separate; do not assume that a future
automatic delivery path has zero cost.

| Requirement | Evidence needed |
|---|---|
| Material saving | At least 5% net reduction and at least one second net per scheduled validation build. |
| Repetition | Two complete history replays from independent state; both pass. |
| Correctness | Preserve required artifacts, diagnostics and failure behavior at the same revision. No candidate-attributable correctness failure. |
| Preparation recovered | Accumulated validation savings cover required preparation and maintenance within the 100-change horizon and remain nonnegative through its end. |
| Complete accounting | Classify every scheduled change. Each replication needs at least 76 of 80 comparable validation rows; unavailable rows receive the contract's conservative charges, never invented savings. |
| Statistical support | Positive lower 95% bound under the existing dependence-aware calculation and its predeclared sensitivity check. |
| Slow-build limit | Candidate p95 no greater than native p95 plus the larger of one second or 5% of native p95. Keep individual regressions and the worst case. |
| Replication elsewhere | The same mechanism passes in at least two of three independent repositories. Preserve all selected outcomes. |

The denominator includes builds where the correction declines to act. A
successful fallback is correct behavior, not an optimization win. Different
task savings or unrelated mechanisms cannot be pooled into one claimed result.

## Work to pursue and work to leave closed

| Area | Treatment under this plan |
|---|---|
| Corrections inside build tasks, continuing the useful part of Patch Autopilot | Primary research. Checkstyle is the most developed candidate; the known Micronaut and Spring fixes need whole-workflow opportunity assessment before more timing. |
| Build Impact on Ktor and Apache Beam | BO-04 review complete; no new trial admitted for the reviewed continuations. Preserve the selected wins. A new observed cause and distinct intervention would need admission before implementation or timing. |
| Generic selection/reuse of complete plans, adaptive fragments and recurrence variants | Retired. Manual selection is not automatically a different mechanism and cannot excuse per-commit repairs or weaker output requirements. |
| Unnecessary invalidation inside native tasks | Conditional on a new observed cause and the existing admission rules. The earlier rejected causes stay rejected; no broad source-only search. |
| Build History, graph capture, cache integrations, patch delivery and output comparison | Retained support for the admitted experiments. Working infrastructure alone is not a speedup. |
| Runtime settings, another general cache, Edge Cache as a default accelerator | Closed as broad research directions. No new CPU/memory sweep, cache backend or repeated locality campaign. |
| Automatic management | Required later for the autonomous product claim, after fixed improvements and usable delivery pass. |

## Starting point

The [supported Checkstyle finalization experiment](../../benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/README.md)
preserves its scoped correctness result. Its timing comparison is mixed and
compares two already optimized implementations. It does not establish a saving
against ordinary Gradle. BO-02 has since completed the four-build identical-code
control without a false material signal under its registered rule; that does
not establish savings or general measurement precision.

The earlier Checkstyle result was favorable on selected changes but slower over
the separate development history tested. The shared-host explanation is an
accepted diagnostic disposition, not permission to remove slow observations.
The 80 Elasticsearch validation changes remain unused.

The [Ktor and Beam results](../../benchmarks/results/poc-magic-end-to-end-value-v2/README.md)
show selected savings, without a lifetime replay of those setups. The
[Micronaut and Spring results](../../benchmarks/results/reviewed-native-patch-portfolio-v1/README.md)
show individual task savings, without whole-history value. Their separate
preparation costs must not be combined into a fictional single build.

These are entry facts to verify before execution, not fresh build or environment
qualification. Detailed prior accounting remains in the earlier execution
record and its task-scoped state.

## Execution tracker

This is the current step tracker. Update it when evidence changes. Detailed
receipts belong to the corresponding experiment and task-scoped state; the
earlier BV tracker remains the record of work completed before this amendment.

Use `pending`, `in progress`, `partial`, `verified`, `blocked` or `deferred`.
`verified` means the step's question was resolved with sufficient evidence;
the outcome may be negative. Advance only on the outcome required below.

| Step | Question or deliverable | Dependency | Current status |
|---|---|---|---|
| BO-01 | Recover the small candidate set and valid prerequisites | This plan | verified; [candidate inventory and checks](../../benchmarks/results/buildopt-product-viability-v1/bo-01-candidate-recovery/README.md) |
| BO-02 | Decide whether the timing comparison can proceed | BO-01 recovery for Checkstyle | verified; [one A/A control without a false material signal](../../benchmarks/results/buildopt-product-viability-v1/bo-02-measurement-control/README.md); no general precision or value qualification |
| BO-03 | Estimate realistic whole-build opportunity per candidate | BO-01; measurement readiness before fresh timing | verified; [five candidate assessments](../../benchmarks/results/buildopt-product-viability-v1/bo-03-opportunity-assessment/README.md); only Checkstyle warrants the planned short screen after readiness checks; missing workflow evidence remains unqualified |
| BO-04 | Admit or reject a focused Build Impact hypothesis | BO-01 and existing evidence review | verified; [no new trial admitted](../../benchmarks/results/buildopt-product-viability-v1/bo-04-build-impact-review/README.md); original savings retained |
| BO-05 | Select a candidate through short comparisons | BO-02 as applicable, BO-03, early BO-06 correctness, and BO-04 for Build Impact | partial; [first sequence retained after an observer failure](../../benchmarks/results/buildopt-product-viability-v1/bo-05-checkstyle-screen/README.md); second sequence and independent output reconstruction did not run |
| BO-06 | Qualify correctness and freeze the implementation and protocol | Admitted opportunity; screen integration proof before BO-05, final freeze afterward | partial; early scoped correctness and screen inputs verified; complete screen, confirmation readiness and final freeze remain incomplete |
| BO-07 | Establish sustained saving in the first repository | BO-05 and complete BO-06 | deferred |
| BO-08 | Test the same mechanism on two other repositories | BO-07 positive | deferred |
| BO-09 | Deliver and measure an MVP with manual controls | BO-08 positive | deferred |
| BO-10 | Establish lifetime and supported compatibility limits | BO-09 positive | deferred |
| BO-11 | Demonstrate added value from automatic management | BO-09 and sufficient BO-10 evidence | deferred |
| BO-12 | Decide the supported outcome or close the initiative | Relevant preceding decisions resolved or explicitly unavailable | pending |

BO-06 has an early correctness part and a final freeze. No candidate reaches a
live comparison before its required correctness and integration checks pass.
The final freeze happens after development timing, before protected validation.
BO-03 and BO-04 may advance through read-only evidence review while timing is
unavailable; this does not authorize parallel agent work or competing builds.

BO-01 recovered all five candidate records on 2026-09-14. Checkstyle's frozen
inputs and manifest pass current checks; the other cases retain their stated
workflow and environment gaps. A pre-existing patch-evidence checker defect
was corrected without changing data or criteria. No native build ran during BO-01.

BO-02 then completed its four-build control. Whole-request times were 84.035s
and 81.294s with identical effective code: a 2.742s difference, 3.37% of the
faster request. This is below the registered 5% threshold. All four builds and
both live and independently reconstructed output pairs passed. Host-pressure
flags remain in the evidence. BO-03 then assessed opportunity using retained
observations; G0/G3 and a fresh timing allocation remain separate prerequisites.

### BO-01: Recover candidates and reusable proof

Verify the actual repository, working tree, subject checkouts, toolchain and
candidate identities. Use repository-owned tools through `dev/run`. Reuse
checks whose inputs and purpose still match; rerun affected checks only.

Start with five known cases: Elasticsearch Checkstyle, Micronaut Python
compilation, Spring architecture checks, and Build Impact on Ktor and Beam.
For each, record the exact requested command, source versions, required outputs,
available implementation, consumed history, valid proof and unresolved question.
Do not claim a source inventory proves buildability or opportunity.

**Deliverable:** a bounded candidate inventory and recovered prerequisites.
**Decision:** cases without recoverable evidence or a specific unanswered
question do not enter timing. Missing environmental proof remains explicit.

### BO-02: Check measurement before claiming another improvement

Execute the [four-build A/A protocol](./buildopt-checkstyle-measurement-control-v1.md)
with identical supported V2 code on both sides: two cold and two measured
builds. Preserve its source guards, complete live/independent output checks,
single CPU profile, order and state policies.

An absolute difference of at least one second and 5% of the faster request is
a false material signal for this control. A smaller difference means only that
this control did not reproduce one; it does not qualify G0 or G3 by itself.

Keep the existing limits: four owner starts, at most four comparator JVMs,
90 minutes, 32 GiB new state and at least 40 GiB free disk. Set the actual
deadline before launch. No timing-driven retry or sample removal.

If the sequence needs repair, identify the cause and prove the affected
behavior before registering another bounded control. Do not restart the
historical search for which unrelated process occupied the workstation.
Unresolved measurement quality prevents a value claim; it is not a negative
optimization result. No chain of instrumentation projects without a concrete
remaining decision and a feasible allocation.

**Deliverable:** a documented timing disposition and list of any remaining
measurement prerequisites. **Decision:** proceed, make a specific supported
correction, or retain an inconclusive result.

**Completed, 2026-09-14:**
[`NO_SPURIOUS_MATERIAL_SIGNAL_IN_THIS_CONTROL`](../../benchmarks/results/buildopt-product-viability-v1/bo-02-measurement-control/README.md).
Four owner starts, four comparator JVMs, no retries, and 12.09 GiB of state;
the allocation and both owned service scopes are closed. This single control
does not establish precision or explain the earlier mixed result. Proceed to
BO-03 using existing evidence. Before fresh timing, resolve the applicable
readiness checks and freeze that comparison's inputs; native 17–20 and the
protected 21–100 history have not started.

### BO-03: Estimate the opportunity before a long replay

For each eligible case, determine what work it avoids, how often that work
occurs in the development prefix, whether shortening it advances completion of
the whole command, and what checking/application/maintenance costs it adds.

Use existing observations first. Do not inspect protected validation results
to choose a candidate. Keep native caching, Configuration Cache and incremental
behavior enabled wherever they preserve the workflow's contract.

Estimate a generous whole-workflow ceiling from observed frequency and the
build's actual limiting work. A task-level percentage alone is insufficient.
The Spring saving near one second, for example, still needs enough whole-build
opportunity after overhead; do not assume it reaches the acceptance floor.

**Deliverable:** a per-candidate opportunity and cost assessment.
**Decision:** reject cases whose generous ceiling cannot meet the unchanged
requirements; admit only cases with a defensible reason for a short comparison.

**Completed, 2026-09-14:** the
[whole-build opportunity assessment](../../benchmarks/results/buildopt-product-viability-v1/bo-03-opportunity-assessment/README.md)
reproduces all 20 native development rows and 32 retained timing pairs. No new
build ran and no protected validation source or timing was read. Checkstyle's
fixed-duration model allows 5.774% over the complete command, leaving about
0.200 seconds per build for residual work and costs in that development model.
It supports the planned short screen after readiness checks, not a long replay.
Micronaut timing waits for an evidenced recurring cache-restore workflow.
The selected Spring result does not justify more timing under today's floor;
no absolute full-workflow ceiling is claimed from its task average. Ktor and
Beam remain BO-04 review cases, with output scope and lifetime unresolved.
Source-only checks match the original Micronaut/Spring task preimages at all
21 permitted snapshots each; native buildability and task/cache frequency
remain unmeasured. Proceed to BO-04. No new experiment allocation is opened.

### BO-04: Decide what remains unanswered in Build Impact

Reconstruct the strongest Ktor and Beam setups. Identify which preparation or
execution disappeared, the original requested command and required results,
and the conditions under which reducing the build remained correct.

Compare those conditions with the failed plan-reuse studies. Distinguish
selection failure, later incompatibility and unrecovered preparation cost.
State which limitation a proposed follow-up would test and why the existing
negative evidence has not already answered that question.

A manually configured scope is eligible for review, not presumed viable. It
must be fixed prospectively, preserve the requested results and survive changes
without a researcher repairing the plan at every commit. Document its difference
from retired mechanisms. A smaller plan or another repository name alone is
not a new hypothesis.

**Deliverable:** either a bounded, previously untested question and prospective
protocol, or a rejection with its evidence. **Decision:** only an admitted case
can proceed to BO-03, BO-05 and BO-06. Ktor and Beam remain known exploratory
subjects; old favorable comparisons are not new independent confirmation.
An admitted Build Impact study needs its own protocol because the existing
native-correction replay contract does not automatically cover that mechanism.
It must preserve the same measurement and correctness standards.

**Completed, 2026-09-14:**
[Build Impact admission review](../../benchmarks/results/buildopt-product-viability-v1/bo-04-build-impact-review/README.md).
The measured v0.6.1 source and all sixteen selected pairs reconstruct task
selection, output scope and the per-pair workspace/cache reset policy. The
launcher forwards ordinary Gradle task requests; project counts do not prove
configuration was removed. A fixed module supplies no identified additional
work to avoid beyond its direct native command. Changing selections through
history returns to the retired reuse route; preserving all broad-command
results needs wider proof and a distinct intervention. Neither proposed
continuation is admitted. No source checkout was rebuilt, target build run,
protected history read or new timing allocation opened. The old results remain
qualified only in their original scope. Proceed to BO-05 preparation and its
early BO-06 prerequisites for Checkstyle.

### BO-05: Use short comparisons to choose a candidate

After BO-02 and the required correctness checks, the planned Checkstyle screen
compares native Gradle with the supported correction through changes 17-20.
Its two replications use 16 owner builds, including active and inactive changes.
Verify the existing source preflight still applies before reuse.

For another admitted candidate, freeze at most four comparison pairs for its
first screen, on one CPU profile. Include changes where it can help and changes
where it cannot. Register all preparation, warmup and helper starts as part of
the total allocation before launching; four pairs are not a hidden unlimited
preparation budget. Freeze screening decisions before observing timings.

Measure the complete command and compare its required results. This screen
can reject obvious overhead, lack of opportunity or defects. It cannot prove
sustained savings, however large an individual percentage looks.

**Deliverable:** a ranked candidate decision with all observations and costs.
**Decision:** advance the candidate with the strongest recurring opportunity.
If none warrants long validation, stop this selection without consuming the
protected history. A newly observed invalidation cause still needs admission;
it does not open an unlimited successor search.

**Partial, 2026-09-14:** the [short comparison](../../benchmarks/results/buildopt-product-viability-v1/bo-05-checkstyle-screen/README.md)
stopped after a permissions error in the observation script, before the second
sequence's first build. All eight completed builds and four live output checks
passed. The first sequence's three post-cold requests show 26.00% less elapsed
time with the correction, including the inactive change. The second sequence
and independent output reconstruction are missing, so the recorded decision is
`INCOMPLETE_SCREEN`; it is neither an admission nor a measured rejection.

The narrow observer repair passed focused local tests without Gradle or JVM
starts. It has not run this build comparison. Preserve all sixteen scheduled
outcomes and the interrupted allocation. The next measured attempt needs a
separate allocation and frozen observation sources; the current protocol
permits no automatic replacement trial. Do not advance to BO-07 or close the
candidate through BO-12 on the basis of this interruption.

### BO-06: Qualify correctness and freeze the experiment

Before live candidate timing, prove the relevant required outputs and integration
paths. Reuse existing scoped proof where it still applies. Cover additions,
modifications, deletions and renames; changes to rules, dependencies and build
configuration; failed builds, cancellation and recovery; missing history; and
activation, deactivation and exact removal.

A manual switch does not override applicability checks. An incompatible
correction must leave the normal build available. A newly observed mismatch
cannot be dismissed by inventing a new output-normalization rule.

After the development screen, freeze the implementation, source predicates,
commands, tools, resources, state policies, cost classification and complete
history before confirmation. Define the replication selection before inspecting
candidate validation timings. Gradle's compatible native capabilities remain
the reference on both sides.

**Deliverable:** correctness receipts and a complete frozen protocol.
**Decision:** no protected replay while prerequisites are incomplete. An
implementation repair invalidates affected proof and is recorded as a separate
version; it is not inserted silently into validation.

### BO-07: Measure sustained saving on the first repository

Use 100 consecutive first-parent changes: twenty for development and eighty
for validation. Replay the entire sequence twice from independent state,
including the cold anchor. Each arm preserves only its own declared state
between commits. Freeze and alternate comparison order as the contract requires.

Keep all scheduled outcomes, including native failures, unavailable dependencies,
fallbacks and unrun changes. Never replace an inconvenient commit, reset caches
to manufacture opportunity, or tune against the protected eighty changes.
Compare outputs between arms at the same commit, not between different commits.

Both replications must pass every acceptance criterion above. Report per-build
and cumulative net time, activation frequency, preparation recovery, uncertainty
and all regressions. A future repair needs a differentiated evaluation.

**Deliverable:** positive, negative or inconclusive whole-history value.
**Decision:** only a positive result supports replication elsewhere.
The base schedule is 404 outer starts per repository, plus explicitly allocated
auxiliary work. Reserve it for candidates that pass the short stages; do not
multiply it by CPU profiles. Git replay models committed source evolution under
declared conditions, not historical developer build frequency or machine load.

### BO-08: Test the same mechanism elsewhere

For the current native-task route, preserve the frozen inventory and order:
Apache Groovy, Apache Kafka, Spring Framework, OpenTelemetry Java, Micronaut
Core and Hibernate ORM. Retain all six outcomes. Follow the existing prefix-only
opportunity admission and select the first two qualifying subjects in that order.
Fewer than two qualifying replication subjects fails the breadth prerequisite.

An admitted Build Impact hypothesis requires its own prospectively fixed
selection and history contract. Previously inspected projects are not unseen
repositories merely because their new validation timings are unused.

Apply the same mechanism and unchanged acceptance criteria across the three
subjects. A Checkstyle improvement in one and a different Build Impact change
in another do not establish replication of one mechanism.

**Deliverable:** complete outcomes for all three selected subjects and the
selection denominator. **Decision:** at least two pass to support a broader
MVP. A single useful correction retains its limited result without a general
product claim.

### BO-09: Deliver and measure an MVP with manual controls

Build only the integration needed to enable and disable admitted optimizations,
per module where the mechanism supports it; show application and measured
results; decline incompatible changes; and remove the integration cleanly.
Reuse the existing delivery and observation components where they fit.

Decide delivery from the surviving mechanism. Some corrections may need build
source changes; others may be applied by the integration. Neither mandatory PRs
nor automatic application of every arbitrary patch is a settled requirement.
Do not add a new service, dashboard or synchronous remote decision dependency
without a demonstrated need.

Test installation in a clean environment and measure the installed command.
Separate the value of the correction from the cost of keeping BuildOpt involved
after applying it. A useful permanent patch may need no permanent runtime
component. Substitute actual delivery costs for prototype assumptions and reopen
affected value proof if those costs change the result.

**Deliverable:** an experimental MVP with measured behavior and overhead.
**Decision:** it must retain the demonstrated saving. Simplify or reject delivery
that consumes the benefit; successful installation alone is insufficient.

### BO-10: Establish lifetime and compatibility limits

A hundred commits can cover only days. After a positive result, define an
extension before seeing its timings to cover less frequent rule, dependency,
build-configuration and tool-version changes. Report missing event categories
instead of manufacturing favorable history.

Synthetic cases can prove failure handling and recovery, but cannot replace
natural changes as evidence of recurring performance. Keep persistent workspaces
and fresh CI jobs as separate state modes with their own proof. State the
supported Gradle/JDK scope; the current prototype does not prove portability to
other build systems.

**Deliverable:** correction lifetime, invalidation causes and supported limits.
**Decision:** advance only claims supported by the observed history and tested
compatibility. Lack of event coverage remains unmeasured.

### BO-11: Prove that automatic management adds value

Start only with useful, deliverable corrections. Keep one while it helps,
confirm sustained loss of value before an economic switch, and stop incompatible
reuse immediately. An ordinary cache miss or one slow build does not by itself
justify searching for another optimization.

Use development data to set observation windows, distinct activation/suspension
thresholds, probes, candidate limits, search budgets and retry conditions.
Freeze actual values before adaptive validation. Search must be bounded and
may end without a profitable alternative. Operate within authorized correction
classes; researcher repairs cannot masquerade as autonomous recovery.

On new history compare native Gradle, a fixed correction, and adaptive BuildOpt.
Count observations, failed searches, qualification, switching and background
work. Preserve the replay contract's natural-event, uncertainty and coverage
requirements: complete outcomes for three subjects, at least two passing versus
native, and supported added value versus the fixed strategy in both replications
of the two subjects supporting the adaptive claim. Each of those two subjects
must include successful autonomous recovery or replacement after a natural event.
Management cost must remain below the predeclared 20% of gross saving.

**Deliverable:** measured added adaptive value and its correctness proof, or a
negative/inconclusive result. **Decision:** no natural events means functionality
and stable overhead can be checked but adaptation benefit is unmeasured. Failure
here does not erase a positive fixed-correction result; it closes the autonomous
claim under this approach.

### BO-12: Decide the supported outcome

| Evidence | Decision |
|---|---|
| No sustained net saving from admitted hypotheses | Close the initiative under those hypotheses. Preserve evidence and useful code. |
| One useful isolated case | Retain the correction and describe its limited scope. |
| Same mechanism passes across repositories and installed MVP retains value | Continue with an experimental MVP of defined scope. |
| Adaptation adds saving and maintains correct improvements through natural changes | Continue toward the self-managing product. |
| Fixed corrections help but adaptation does not compensate for its costs | Preserve the narrower MVP result and close the autonomous claim under this approach. |

Do not manufacture a successor after a negative decision. A new candidate needs
observed material opportunity and admission; an unresolved environment or
incomplete experiment remains unavailable evidence rather than a measured
mechanism failure.

## Execution records and limits

For every started step, record the question, status, exact inputs and commands,
dependencies, prior proof reused, complete observations, required outcome,
decision, failed-correction count, spent limits and next action. Use an existing
approved task-scoped store for runtime state and link its portable result here.
Preserve the three-correction stop rule in the repository working agreement.

Every new allocation must include preparation, failures, helper/nested starts,
time and disk limits. Earlier allocations remain closed and charged. No
timing-driven retry, favorable-only report, broad CPU sweep or unrelated process
termination is permitted. Use lean measurements and expand diagnostics only to
resolve an identified prerequisite; do not remove required output checking.

Program evidence remains under `.tools/state/buildopt-product-viability-v1`.
The earlier state key `engineeringPrefix.checkstyleFinalization` locates the
supported candidate and completed comparison; verify it before reuse. Do not
move, overwrite or silently reset old experiment roots.

BO-01 through BO-04 are complete. The control's task record is
`.tools/state/buildopt-product-viability-v1/bo-02-measurement-control-2026-09-14/task-state.json`;
its original inputs and raw results remain there. The BO-03 assessment record is
`.tools/state/buildopt-product-viability-v1/bo-03-opportunity-assessment-2026-09-14/task-state.json`.
The BO-04 admission and publication record is
`.tools/state/buildopt-product-viability-v1/bo-04-build-impact-review-2026-09-14/task-state.json`.
The user has authorized committing and pushing each completed block. BO-01,
BO-02 and BO-03 were published in separate commits before BO-04 began.
BO-05's partial result and observer repair are recorded under
`.tools/state/buildopt-product-viability-v1/bo-05-checkstyle-screen-2026-09-14/task-state.json`.
The unchanged candidate, runner, lifecycle evidence and four source revisions
passed preparation checks. The interrupted allocation retains sixteen build
reservations; eight builds and four live-comparison JVMs ran, with no retries.
All three created services are inactive and the raw evidence remains available.
The proposed next attempt keeps sixteen builds, at most sixteen comparator
JVMs, one CPU profile, three hours, 64 GiB and at least 40 GiB free disk. It
requires its own allocation before launch. Formal G0/G3, protected validation
and Build Impact trials remain unqualified or unadmitted.
