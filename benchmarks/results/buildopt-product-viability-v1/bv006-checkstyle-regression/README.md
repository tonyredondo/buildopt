# Checkstyle regression and recovery investigation

Date: 2026-09-10. Program: BUILDOPT-VIABILITY-V1. **Execution verified; exact
historical spike attribution partial.** [Machine-readable outcome](summary.json).

The candidate's cache works in the fresh reproduction. The long Checkstyle task
span includes substantial waiting after its worker has completed. The actual
Gradle 9.7.1 lock-retention mechanism is verified by a controlled service test and
by the pinned task-execution bytecode. The original whole-build slowdown does
not reproduce in the new three-commit replay. Its exact historical filesystem,
page-cache, GC or scheduling cause cannot be recovered from the original data.
This investigation does not promote product viability or change the original
negative history result.

## What was executed

Same corrected private-task-property candidate and native baseline; 8 CPUs and
8 Gradle workers; pinned Java21 launcher and Java25 compiler; persistent N/I
arms over original 6,7,8. The candidate and workflow remain the frozen ones from
the preceding screen. Original 7 is `3a00a9167b54dc298e015cf92ef692d6ddacab77`.

Six builds pass both live and independent complete C5 output comparisons, with
all three success histories verified. A preliminary native warmup also passed,
but lacked worker instrumentation and was closed without a pair. Total: seven
actual Gradle starts, within the maximum eight. The corrected replay takes
44m57s including research preparation/capture; native command execution totals
about 10m30s. No additional Elasticsearch build was used for the service proof.

The table uses the customer envelope plus customer costs outside it, including
candidate application/reversal. Cache preparation, artifact capture and JFR
analysis remain separately recorded research costs. All points are instrumented
exploratory observations, not a precision estimate or formal value confirmation.

| Original ordinal | Native | Candidate | Saving |
|---|---:|---:|---:|
| 6 (fresh warmup) | 213.061 s | 222.097 s | -4.24% |
| 7 (problem commit) | 98.185 s | 83.416 s | +15.04% |
| 8 (following commit) | 7.224 s | 7.159 s | +0.90% |

[Native validation](native-validation.json) includes exact command/envelope
values, state/output checks and observed CPU affinity. Original 6 starts fresh;
it does **not** recreate the daemon and filesystem-cache age of original 0–6.
The three-commit sample is selected around an already observed failure.

## The cache did not miss in the reproduction

At original 7, Main reuses 4,954 of 4,959 supplied inputs. The engine receives five:
four changed Java files plus `module-info.java`, which the original owner policy
excludes before checking. Its report contains4,958 files. Test reuses 2,819 of 2,825
inputs and checks six. Cold original 6 correctly reuses zero files. Hashing,
prepare/commit, emitted reports and success-state publication are observed.

Main's native engine takes 60.053s. The candidate's native engine takes 1.273s,
and its complete adapter takes 1.651s. Prepare takes 0.031s and Commit0.044s.
The other40.185s between adapter return and Commit are outside those operations.
The complete Main task takes 51.904s, not1.651s.

The worker's dispatch thread enters its idle queue45ms after the adapter returns
and stays there throughout40.140s of that gap. The Test worker does the same for
42.867s. Their JFR windows cover those intervals completely. These are observed
idle-worker periods, not inferred from an absent CPU sample.
[Phase measurements](phase-evidence.json), [coverage and worker stacks](coverage-and-idle.json).

In the original history, success records and XML reports survive original 6→7
unchanged, Main/Test context identities match, and the candidate's checking
workers use27.31 sampled CPU seconds versus 195.52 natively. Actual original
processed-file counts were not logged. These facts support reuse; the new run
provides its direct observation. All 15 original N/I requests have identical
executed-task identities. There is no exclusive additional Gradle task execution
explaining the candidate's original regression.
[Original trace](original-evidence.json).

## Why a completed checker can remain a running task

The candidate appends a `Commit` action with `doLast`. In the exact pinned Gradle
bytecode, an asynchronous action with more actions to execute uses
`RELEASE_AND_REACQUIRE_PROJECT_LOCKS`; a final action with no applicable
scoped action/execution listeners uses `RELEASE_PROJECT_LOCKS`. Those listeners
can also require reacquisition. A [separate actual-API check](listener-proof.txt)
shows that the runner's exact `taskGraph.beforeTask/afterTask` hooks do not set
these scoped ListenerManager flags in ProjectBuilder; it is not a dump of every
owner plugin listener. The extra action therefore requires recovering the
project lock before continuing. Other work in the same project can hold it.
[Verified bytecode branch](gradle-lock-branch.json).

The [controlled proof](LockRetentionProof.java) invokes the actual
`DefaultAsyncWorkTracker`, `DefaultWorkerLeaseService` and resource-lock
coordination service from Gradle 9.7.1. Another thread acquires the project lock
while the checker work completes. Only the retention mode changes:

| Actual Gradle mode | Returns while another thread still holds the project lock? |
|---|---|
| Release project locks | Yes; 15.2 microseconds after work completion |
| Release and reacquire | No; waits 257.2ms until the controlled holder releases it |

The blocked thread's captured stack passes through `acquireLocks`,
`withoutLocks`, `runAsIsolatedTask` and `DefaultAsyncWorkTracker.waitForCompletion`.
This verifies the mechanism, not a synthetic build-performance estimate.
[Complete output](lock-proof.txt), [exit receipt](lock-proof-receipt.json).

The actual candidate also exhibits this gap during the cold warmup: Main's
adapter finishes 54.760s before its 0.092s Commit. The same pattern appears in all
three cold checking tasks and both changed-input tasks.

**Attribution limit:** the circular daemon recording evicted the candidate's
problem interval. The precise native daemon stack at each instant of the 40s gap
is unavailable. The observed phase/idle-worker timing and controlled real-service
proof strongly support project-lock reacquisition as its explanation; the service
fixture does not retroactively recreate that missing stack.

## Where the original whole-build regression occurred

Original 7 takes 110.724s natively and 151.818s with the candidate. Main's reported
task span increases from 72.718s to 117.340s, but both arms still have other work
running when Main finishes. Its extra span must not be added to concurrent tasks.

| Observed task in the original completion sequence | Native | Candidate |
|---|---:|---:|
| collectTransportVersionReferences |11.268s|47.120s|
| compileTestJava |42.831s|48.271s|
| compileInternalClusterTestJava |9.673s|10.878s|
| loggerUsageCheck |3.575s|7.027s|

The candidate reaches that sequence4.749s earlier, then loses 35.852s in the
collect task,5.440s in test compilation,1.204s in integration-test compilation
and 3.451s in the log check. Including the measured gaps and tail reconstructs
41.131s of additional native-command walltime. This is an observed timeline
decomposition; it is not a claim that these tasks always form a dependency chain
or that their internal code exclusively caused each span.
[Exact decomposition](original-tail-decomposition.json).

In the fresh run the collect task takes 1.176s natively and 0.424s with the
candidate. The original 47s episode is not reproduced. The old run lacks the I/O,
GC and detailed daemon traces required to distinguish filesystem/page-cache
misses, GC, input fingerprinting and other scheduling effects. Original receipts
explicitly report unavailable cgroup I/O accounting. Whole-host load and available
memory do not establish an exclusive cause. No specific disk-cache or GC diagnosis
is claimed from those missing measurements.

## Does it return to normal afterwards?

The following commit returns to roughly 7s in both fresh arms, and Checkstyle is
`UP-TO-DATE`. Original 8–14 likewise skip Main. This proves later avoidance; it
must not be described as repeated fast execution of the checker. The fresh
changed-input commit separately demonstrates fast actual cached checking.
Gradle distinguishes skipped up-to-date work from outputs restored from its build
cache: [incremental-build behavior](https://docs.gradle.org/current/userguide/incremental_build.html).

The original peak explains about 74% of the 13-request history loss. Excluding7
still leaves14.283s of observed loss across twelve builds, with no active
Checkstyle work. That remainder is not a causally measured optimizer tax, and
removing an outlier after seeing it is not evidence of positive product economics.
The original history verdict remains `NO_MATERIAL_SIGNAL_IN_DISJOINT_HISTORY`.
[Window arithmetic](original-window-economics.json).

## Integrity, limits and retained failures

- Actual native worker phases, cold/reuse behavior, all three histories and
  three live/independent C5 pairs pass. Native0–7, observer8 and controller9
  affinities pass in the collected observations; finite samples do not prove
  unobserved instants. No formal timing/product gate follows.
- The first warmup lacked worker-agent propagation through JAVA_TOOL_OPTIONS.
  Explicit fork options fixed that integration gap. Its one native start and
  eight original reservations remain retained.
- Failed agent/fixture probes remain in raw state. Timer-frame verification,
  Gradle logger selection, inherited API lookup, fixture code-source isolation
  and bounded fixture shutdown were resolved before the corrected native run.
  One observed standalone helper start has no recovered completion receipt;
  it is counted, and its process is absent.
- V2 captures enclosing Prepare/Commit time; its separate context-method timer
  did not match the descriptor. Context work is included in the enclosing bounds.
- All 26 final JFR containers parse, but container completeness does not imply
  coverage of the desired interval. Candidate daemon coverage for original 7 is
  absent; checking-worker coverage is present. Long pipe reads are not disk-read
  stalls. Partial GC/I/O counts cannot establish absence of those causes.
- The post-run JSON export reached 568,933,606 bytes, exceeding the 512-MiB bound.
  It is now losslessly compressed to 12,529,265 bytes, with the uncompressed hash
  verified before removing only the regenerable uncompressed copy. This resource
  failure is retained; no resource qualification is claimed.
  [Compression receipt](flight-export-compression.json).
- Total new charges: 14 Gradle reservations/7 actual starts,17 standalone helper
  JVMs and 8 compiler commands. The final two helpers test actual lock services and listener registration; both
  add zero Gradle starts. Combined program Gradle totals: 748 reservations/622
  observed starts. Older unresolved accounting remains in its historical scope.

Raw recovery root:
`.tools/state/buildopt-product-viability-v1/bv006-checkstyle-regression`.
Durable locator: `engineeringPrefix.checkstyleRegression` in the existing program
`task-state.json`. The six-start controller, all native scopes and helper commands
are closed. The candidate, old experiments, held-out21–100 and other repositories
remain outside any new implementation or value-confirmation claim.

## Product implication and next step

Use actual reuse/correctness to judge cache validity and whole-request time to
judge value. A long task span alone can trigger an incorrect adaptive decision
while the optimized worker has already finished. This investigation establishes
useful work avoidance in the selected case; recurrence and net product value
remain unproven.

[Next: prove a finalization design and measure its effect on whole requests](next-step.md).

Final checks: [validation receipt](validation.json) and [portable artifact index](artifact-index.json).
