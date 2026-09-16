# Build Optimization research execution plan

Date: 2026-09-14. Program: `BUILDOPT-VIABILITY-V1`.
Status: governing plan in execution; BO-01 through BO-05 verified; BO-06 partial. A preparation-only observation found heavy disk pressure that subsided within the existing wait limit. A fresh control and conditional development sequence remain to be measured.

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
| BO-05 | Select a candidate through short comparisons | BO-02 as applicable, BO-03, early BO-06 correctness, and BO-04 for Build Impact | verified; [both fresh sequences passed](../../benchmarks/results/buildopt-product-viability-v1/bo-05-checkstyle-observer-replay/README.md), saving 34.14% and 28.17%; the interrupted attempt remains separate |
| BO-06 | Qualify correctness and freeze the implementation and protocol | Admitted opportunity; screen integration proof before BO-05, final freeze afterward | partial; the [preparation-only observation](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-preparation-observation/README.md) recovered a quiet window after 47 seconds; no project build ran; prepare a separate control and conditional development sequence with the unchanged method |
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

At that closeout, the narrow observer repair had passed focused local tests
without Gradle or JVM starts, but had not run the build comparison. Preserve
all sixteen scheduled outcomes and the interrupted allocation. The next measured attempt needs a
separate allocation and frozen observation sources; the current protocol
permits no automatic replacement trial. Do not advance to BO-07 or close the
candidate through BO-12 on the basis of this interruption.

**Separate attempt, authorized and completed on 2026-09-14:** the complete
comparison was repeated from fresh state with the repaired observer. Its
allocation allowed 16 owner builds, at most 16 comparator JVMs, three hours
and 64 GiB of new state, with
at least 40 GiB free. The candidate, four changes, command, CPU profile and
decision rule are unchanged. All 58 frozen input bindings matched before
launch, the manifest passed validation, and 36 focused checks passed without
Gradle or comparator starts. None of the interrupted attempt's timings
contributes to the new decision. The current state is
`.tools/state/buildopt-product-viability-v1/bo-05-checkstyle-observer-replay-2026-09-14/task-state.json`.
No protected validation history is admitted.

**Completed result:** [both fresh sequences passed](../../benchmarks/results/buildopt-product-viability-v1/bo-05-checkstyle-observer-replay/README.md)
the unchanged short-screen rule: `EXPLORATORY_MATERIAL_SIGNAL`. Whole-request
savings over changes 18–20 were 34.14% and 28.17%, with mean savings of 22.254 and
28.455 seconds per measured change. The first sequence had two positive pairs;
the second had three. The inactive change and every regression remain included.

All 16 builds, eight live output comparisons and eight independent reconstructions
passed. Source inventories verified the declared five-file correction on all
four revisions in both sequences. The 58 frozen bindings matched at closeout;
four owned sessions closed; the attempt finished in about 93 minutes using
25.67 GiB. No observation failure or permission denial was recorded in this run.

Both versions of change 20 were slower in the second sequence. Eight host-pressure
flags remain in the data without exclusions or an asserted cause. Sampled CPU
assignments matched; G0/G3, continuous isolation and long-history value remain
unqualified. The next step is the remaining BO-06 readiness and freeze. BO-07
and protected changes 21–100 remain deferred.

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

**BO-06 readiness review, 2026-09-14:** eight fresh fixture builds preserve the
complete native XML reports through cache restoration, subsequent edits, changed
configuration, disabling and reactivation. A fresh compiler run reproduces all
eight runtime classes exactly. The earlier scoped correctness and supported
failure/cancellation proofs remain separate and bound. All seven histories
retain 101 revisions and their original endpoints; replication order is unchanged.
Only Git metadata was used for history recovery. No protected validation build
or candidate-selection source inspection occurred.

The [readiness review](../../benchmarks/results/buildopt-product-viability-v1/bo-06-readiness/README.md)
does not close BO-06. Actual-owner measurement qualification is missing; the
closed 100-ms precision attempt and process-observation failures stay negative.
The current short-screen manifest includes diagnostic Java instrumentation and
cannot become primary confirmation data by relabeling it. The executable
rejects the incomplete confirmation manifest. G3 remains the future BO-07
outcome, not a BO-06 prerequisite.

The user approved the measurement proposal. The separate
[lean recording mode](../../benchmarks/results/buildopt-product-viability-v1/bo-06-lean-measurement/approved-method.md)
removes the Java phase agent and keeps the output recorder on both arms.
Process samples are diagnostic. The old precision and sampling failures retain
their original outcomes. Any saving in this mode describes the recorded
workflow; BO-09 must still measure the installed experience against ordinary
Gradle before a product-performance claim.

The eight-build identical-code control passed: 127.738 versus 129.505 seconds
over the three measured changes, a 1.767-second difference or 1.38%. All four
live and four independently reconstructed output comparisons passed. This
does not measure an optimization saving. The local checks used 94 fixture
requests and covered the command paths, but missed a prerequisite in the real
Elasticsearch development manifest.

That manifest was refused before any prefix build: its output policy has no
qualification for the current comparator. The older qualification binds a
different comparator. Adding the missing proof changes the frozen policy
identity, so this control cannot silently admit that changed input. The
[attempt and diagnosis](../../benchmarks/results/buildopt-product-viability-v1/bo-06-lean-measurement/README.md)
are retained. The allocation is closed as incomplete, with eight owner builds,
eight comparison JVMs, no owner retries and no protected builds.

**Qualification completed, 2026-09-15:** all 33 comparator tests, six comparisons
of retained Elasticsearch outputs and 20 full admission cases passed. One
focused Go check also verified reader binding. This used eight comparator JVMs
and no owner or fixture builds. The [qualification record](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-comparator-qualification/README.md)
includes a new qualified policy and consistent control/development inputs.
Positive admission tests use synthetic trial records strictly as test data;
the live development proposal still refuses to proceed without a genuine
control. The old control correctly fails admission for the new policy.

Follow the [next measurement plan](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-comparator-qualification/next-measurement.md):
eight control builds, then all 42 development builds if the control passes,
within one proposed ten-hour allocation. No retries or additional profiles.
The candidate and savings criteria are unchanged. Confirmation will also
need evidenced preparation costs for each repetition. BO-06 remains partial;
BO-07 and protected changes 21–100 remain deferred.

The user authorized this measurement on 2026-09-15. The new allocation allows
50 owner builds and 50 comparator JVMs within ten hours and 80 GiB, with no
retries. Control admission passed with the frozen qualified policy.

**Measurement completed, 2026-09-15:** the three measured control pairs totaled
177.233 seconds in N and 133.974 seconds in I. The 43.259-second difference,
32.29% of the faster side, exceeds the unchanged stopping rule. The result is
`CONTROL_MATERIAL_DIFFERENCE`. All eight builds, four live comparisons and
four independent reconstructions passed. The 42-build development sequence
did not start, and the allocation and both owned sessions are closed. Eight
owner and eight comparator JVM starts were used; there were no retries or
protected builds.

Change 20 accounts for nearly all the timing difference. Its recorded task
outcomes match on both sides, but several compilation tasks slowed down during
higher disk-wait pressure. Checkstyle's own checks were slightly faster in the
slower build. These observations do not establish the exact cause. The
[complete control result](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-qualified-measurement/README.md)
retains every pair and the diagnostic limitations. Address the measurement
conditions before another control; no result is discarded and the correction
and stopping rule remain unchanged. A fresh passing control, complete
development replay and final freeze are still required before BO-07.

**Recording-pressure diagnosis, 2026-09-15:** bulk result copying follows native
execution, with no overlap in the eight recorded builds. Before the slow side
at change 20, the preceding 30 seconds averaged 53.40% disk-wait and 32.59%
memory-wait pressure. The supervisor also used roughly one CPU core during most
builds; its work includes repeatedly walking the entire experiment directory.
These observations identify measurement problems but do not prove the cause
of the full timing difference. Missing outer-recorder and per-thread observations
prevent complete attribution. The original negative control is retained.

The standalone utility from that diagnosis passed nine tests and one live
observation. Its integration conditions are now covered by the
[quiet-start runner verification](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-quiet-start-integration/README.md).

**Quiet-start integration, 2026-09-15:** the actual runner waits after preparation,
binds the observation to the request, and checks freshness before native launch.
It records waiting as research cost and measures the outer recorder's CPU and
I/O. The final real fixture pair, four refusal paths, independent missing-evidence
checks, 34 unit tests and five selected compatibility groups passed. All 30
owned worker sessions closed. Across development and verification, 23 native
fixture processes started, including retained failed attempts; no Elasticsearch
build, comparator JVM or protected source read occurred.

The new runner, policy and qualified comparator are frozen together. Complete
control admission passes; development correctly remains refused without a
fresh passing control. The prior negative control is unchanged. The next block
activates a separate allocation for the eight-build identical-code control;
only a pass can admit the complete 42-build development sequence. The existing
50-build, 50-comparator, ten-hour and 80 GiB ceilings include quiet waiting and
preparation. No owner allocation was activated by this integration block.
The full-tree disk guard remains unchanged; its contribution to the earlier
supervisor cost is still unproven. BO-06 remains partial. Current state:
`.tools/state/buildopt-product-viability-v1/bo-06-quiet-start-integration-2026-09-15/task-state.json`.

**Quiet-start measurement closed, 2026-09-15:** the authorized control passed:
128.494 versus 128.363 seconds across the three measured changes. Individual
differences were 0.745, 0.434 and 0.443 seconds. All eight builds, four live
comparisons and four independent reconstructions passed. This qualifies the
registered control; it does not establish general precision or a correction's
saving.

The genuine control admitted the 42-build development sequence. Seventeen builds
completed successfully. Eight pairs passed both live and independent comparison;
the native side at change 8 also completed but has no candidate comparison.
The candidate side was refused before launch after the full three-minute wait:
storage-wait pressure exceeded 10% in all 179 observed intervals. The remaining
25 builds did not run. Nine completed development builds also retain their
registered host-pressure flags; a quiet start does not guarantee a quiet build.

The [control and interrupted development result](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-quiet-measurement/README.md)
retains every scheduled outcome, the unpaired build and the refusal. The runner's
`INCOMPLETE_EVIDENCE` result is unchanged. Its conservative counter keeps the
unlaunched reservation as unknown; the timeout receipt and pre-launch return path
explain that reservation without changing the raw result. No complete-sequence
saving is claimed. The allocation and all four worker sessions are closed, with
25 owner builds and 24 comparison JVMs, no retries and no protected builds.

**Storage diagnosis complete, 2026-09-15:** the frozen code reproduces all
26 quiet-start decisions, including the refusal. During the failed wait the
same 17 sampled processes used 1.18 seconds of CPU and recorded little I/O.
The available observations cannot distinguish delayed earlier writes from
unrelated storage activity. The exact historical cause remains unproven.

A separate measurement problem is confirmed. Across all 25 completed builds,
supervisor CPU time was 95.82%–98.57% of native elapsed time. Its full-directory
size scan, scheduled every 100 ms, took 2.500 seconds for one pass over the
retained development tree. The profile places 98.81% of CPU samples beneath
that scanner. This establishes substantial supervision work, not its effect
on build wall time or the cause of the later storage wait. The
[diagnosis and reproducible analysis](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-storage-pressure/README.md)
retain all rows and their limits. No owner build or comparator JVM ran here.

Next, follow the [measurement conditions](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-storage-pressure/next-step.md):
reduce repeated disk-accounting work, preserve every resource and output
contract, and qualify the change with bounded fixtures. Add inexpensive
observations for the remaining attribution gaps. Freeze one final identity
before a separate eight-build control and, only if it passes, a new complete
42-build development sequence. The next implementation block stops at fixture
proof and frozen inputs; it does not start Elasticsearch timing.

Do not raise thresholds, discard the interruption, resume the closed allocation
or move to BO-07. BO-06 and its final freeze remain partial; protected changes
21–100 remain deferred. Current diagnosis record:
`.tools/state/buildopt-product-viability-v1/bo-06-storage-pressure-2026-09-15/task-state.json`.

**Disk-accounting correction qualified, 2026-09-15:** the
[fixture result and frozen inputs](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-disk-accounting/README.md) complete conditions 1–6 of the
storage diagnosis follow-up. Twenty checks over 32,768 retained files used
about 74% less CPU in both repetitions, including initial watch setup and
closure. The small case added about seven milliseconds. Limits and complete
boundary counts remain unchanged; unsupported observations return to full scans.
All 24 native fixture starts and affected compatibility checks passed, including
actual cancellation, output refusal, missing evidence and driver loss. New
storage counters preserve unavailable observations and record their own cost.

The control proposal passes full validation with the new runner and sampler;
development is refused until a fresh genuine control passes for the same
identity. The comparator and quiet policy remain unchanged. No Elasticsearch
build, comparator JVM or protected source read ran in this block. Conditions
7–8 are next: a separate allocation for eight control builds, then all 42
development builds only if the control passes. The existing ceilings and
stopping rules still apply. This closes the implementation block, not BO-06.
Current task record:
`.tools/state/buildopt-product-viability-v1/bo-06-disk-accounting-2026-09-15/task-state.json`.

**New control closed, 2026-09-15:** [the retained attempt](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-disk-measurement/README.md)
completed one build before the second launch exhausted its three-minute wait.
Eleven of 179 intervals exceeded the disk-pressure threshold; the mean was
1.78%, but no 30-second continuous window qualified. All other scheduled
control builds and the conditional development sequence remain unrun. There
are no output pairs or performance comparison. Both worker sessions closed;
one owner build and zero comparison JVMs were used, without retries.

The first cold build triggered the disk guard's full-scan fallback. Supervisor
CPU was 201.760 seconds during 205.826 seconds of native execution. The fixture
improvement did not persist through this request. Output capture took another
565.115 seconds outside build timing; its effect on later pressure is unproven.
No incremental supervision result is available.

This reaches the follow-up contract's stop for another failed quiet control.
Before any further timing, prepare a separate proposal for the measurement
environment or observation policy, using the retained data and accounting for
recording costs. Do not resume this allocation, weaken its criteria, alter the
candidate or start BO-07. BO-06 remains partial. Current task record:
`.tools/state/buildopt-product-viability-v1/bo-06-disk-measurement-2026-09-15/task-state.json`.

**Reserved-workstation repeat authorized, 2026-09-16:** the owner offered to
reserve the workstation, then confirmed it was free. This supplies the separate
environment decision required by the previous stop. It does not establish the
cause of the earlier disk-pressure bursts or remove work done by the recorder.

One fresh allocation repeats the eight-build control with the same frozen
runner, sampler, comparator, candidate, output capture and admission rules.
Only a passing control admits the complete 42-build development sequence.
The 50-build, 50-comparator, ten-hour and 80-GiB limits include preparation and
waiting; there are no retries or protected changes. Ordinary system activity
remains possible, so the existing quiet-start rule still applies. Another
interruption closes this allocation and must be investigated from its retained
evidence before another timing decision. Previous results remain unchanged.
Task record:
`.tools/state/buildopt-product-viability-v1/bo-06-reserved-host-2026-09-16/task-state.json`.

**Reserved-workstation result, 2026-09-16:** the
[separate attempt](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-reserved-host/README.md)
refused its first launch after the unchanged three-minute wait. Disk pressure
exceeded the threshold in all 179 intervals, averaging 62.38%; CPU pressure
averaged 0.19%. No project build or comparator JVM ran. All eight control builds
and the conditional 42-build sequence remain unrun, with no saving claimed.

A system file indexer remained active during preparation, but its contribution
is unproven. The runner did little work during the refused wait; pending writes
from preparation and other host storage activity remain possible. Native
supervision and post-build capture never ran. The worker session and controllers
closed, and the raw unknown reservation remains preserved. The next step is a
separate host-only observation with the recorder stopped and indexer finished,
then a measurement decision. No automatic control retry, weaker threshold,
BO-07 or protected replay follows. BO-06 remains partial.

**Host-only observation, 2026-09-16:** the
[three-minute observation](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-host-observation/README.md)
found its first qualifying quiet window at 58 seconds. Disk pressure averaged
0.71%; three of 179 intervals exceeded 10%, and the final window did not qualify.
All 180 window decisions agree with the original frozen rule. The earlier
recorder and worker group were closed, and no indexer appeared in 19 process
checks. The observer completed with no project builds, comparison JVMs or retries.

The sustained pressure was no longer present throughout this observation;
its earlier cause remains unproven. Before another timing decision, check
whether preparation brings it back using a separately bounded observation of
preparation without launching Gradle. This host trace is not launch admission
or a control result. BO-06 remains partial and the protected history stays
deferred. Task record:
`.tools/state/buildopt-product-viability-v1/bo-06-host-observation-2026-09-16/task-state.json`.

**Preparation-only observation, 2026-09-16:** both environments and the first
request's preflight completed in the
[single diagnostic pass](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-preparation-observation/README.md).
Disk pressure averaged 0.80% before preparation, 84.56% during it and 0.88%
within the first three minutes afterward. The first qualifying quiet window
appeared after 47 seconds. Copies consumed 445.74 seconds, or 63% of preparation
time. The observer used 2.38 CPU seconds across the complete observation.
No indexer appeared in the 107 process checks. The earlier prolonged wait
was not reproduced; its exact cause remains unproven.

All 1,250 window evaluations agree with the original rule. The recovery
collection closed on the next sample at 181 seconds; the decision uses only
windows within the unchanged 180-second budget. Source bindings, preparation
checks and process closure passed. The diagnostic could not launch a workflow;
no project build, comparison JVM or protected-history read occurred. The timing
runner and its identity remain unchanged.

Next, prepare a separate eight-build identical-code control. Only a passing
result permits the complete 42-build development sequence, under the existing
50-build, 50-comparator, ten-hour and 80-GiB limits, with at least 40 GiB free and
no retries. This diagnostic activates no timing allocation. Reducing copy cost
is a separate question and is not required before that next control. BO-06
remains partial; BO-07 and protected validation remain deferred. Task record:
`.tools/state/buildopt-product-viability-v1/bo-06-preparation-observation-2026-09-16/task-state.json`.

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
