# Checkstyle CPU screen and disjoint historical replay

Experiment and scoped closeout verified, 2026-09-10.
Decision: **`NO_MATERIAL_SIGNAL_IN_DISJOINT_HISTORY`**.

The corrected candidate saves substantial walltime on the selected opportunity
commits. It does not sustain a net saving in the disjoint engineering history.
The next investigation is the observed `checkstyleMain` regression at ordinal 7.
There is no product, lifecycle, precision or formal readiness pass.

## Measured walltime

| Cohort | Measured pairs | Native mean | Candidate mean | Mean net saving | Aggregate saving |
| --- | --- | --- | --- | --- | --- |
| 8 CPUs / 8 workers, ordinals 19–20 | 2 | 85.697 s | 50.315 s | 35.381 s | 41.287% |
| 4 CPUs / 4 workers, ordinals 19–20 | 2 | 106.609 s | 63.844 s | 42.765 s | 40.114% |
| 2 CPUs / 2 workers, ordinals 19–20 | 2 | 233.740 s | 177.374 s | 56.366 s | 24.115% |
| 8 CPUs / 8 workers, history 2–14 | 13 | 22.235 s | 26.495 s | **-4.260 s** | **-19.158%** |

Positive saving means faster. These are exploratory customer-request timings,
including recorded customer-side costs. The [machine-readable summary](./summary.json)
binds the source reports and keeps every measured delta and cost component.

Each CPU profile has two warmup pairs at 17/18 and two measured pairs at 19/20:
eight builds per profile, 24 total. Eight CPUs was the first qualifying profile
in the frozen 8/4/2 order. Its separate history starts fresh at 0, warms at 0/1,
and measures **all 13 consecutive transitions 2–14**: 30 builds, with ordinary
native state retained between commits. The history was already available for
engineering; it is disjoint from the CPU screen, not blinded validation.
Formal validation 21–100 remains untouched.

Fewer CPUs increased absolute saving in these selected cases, but did not
increase percentage saving. Both CPU affinity and Gradle worker count changed.
Two selected pairs per profile cannot establish general scaling or lifetime value.
Do not pool the separate, freshly initialized CPU and historical segments into
an invented full chronological cohort.

## What the history exposed

Across the 13 measured transitions, native takes 289.056 s and the candidate
344.433 s: **55.377 s of accumulated regression**. The paired median is almost
zero (-0.003 s saving), so the mean is strongly affected by individual cases.
Only ordinal 7 executes the selected Checkstyle actions; the 12 no-action
transitions remain in the denominator and add 14.283 s of observed net loss.
Those no-action differences are not proof of candidate overhead or its cause.

At ordinal 7, the whole request regresses from 110.724 to 151.818 s, losing 41.094 s.
The [task-span diagnostic](./corrected/history-task-diagnostic.json) finds:

| Task | Native span | Candidate span |
| --- | --- | --- |
| `:server:checkstyleMain` | 72.718 s | 117.340 s |
| `:server:checkstyleTest` | 60.568 s | 9.284 s |
| `:server:collectTransportVersionReferences` | 11.268 s | 47.120 s |

Task spans overlap; their differences cannot be added as causal contributions.
The [state-continuity proof](./corrected/history-state-continuity.json) verifies
that all three success records and XML reports are unchanged between candidate 6
completion and candidate 7 start. Main retains the same context, with 4 changed
files, 1 removal and 4,954 same-content files; Test has 6 changes and 2,819 same-content
files. These are **potential reuse counts**, not directly observed engine skip
counts. The retained data does not identify the actual Prepare reuse decision,
hashing/engine/commit durations, worker wait, GC, JIT or I/O cause.

The concrete next step is a [bounded ordinal 7 investigation](./next-step.md),
not admission of the current candidate to automatic deployment or adaptive value.

## Correctness and repairs

All 54 corrected-round native builds succeed. All 27 pairs pass both live and
independent complete C5 output comparison. All 27 candidate builds retain the
three task histories. The [profile reports](./completion-artifacts.json) retain
the exact execution inputs, raw results and receipts.

The replay harness handles ENOENT/ESRCH process departure during cgroup enumeration
while preserving ownership/permission errors. Retained outputs reuse verified
research objects and remain independent of writable native files.
[Patch](./harness/repair.patch), [unit results](./harness/unit.log),
[ten fixture workflows and independent reconstruction](./harness/runtime-proof.log),
and [current-checkout CLI build](./harness/root-build.json) retain the proof.
This is proof for the relevant runner, not a new claim that every repository
component was rebuilt in this continuation.

The installer previously mutated a shared Gradle convention properties map.
Main/Test consequently used native fallback while only InternalClusterTest kept
successful history. The [repair](./corrected/candidate-fix.patch) gives each task
its own map before adding its state path. The
[actual compiled installer test](./corrected/installer-proof.log) and all current
owner results verify the repaired bindings. The
[full corrected patch](./corrected/candidate.patch) is the successor input;
the original frozen candidate must not be reused as full activation proof.
The engine, state codec and native fallback source remain unchanged. Original
correctness tests retain their original input scope.

## Observation qualification remains limited

The 8-CPU screen retains one pre-exec systemd mask observation before ready/native
boundaries. Its [v3 treatment](./corrected/affinity-startup-observation.json)
keeps all-time affinity false while sampled native/running-worker masks pass.
All sampled masks pass in the 4- and 2-CPU profiles.

History retains a different event: one supervisor launch thread briefly has the
native mask 0..7, while its other seven threads have observer CPU 8, in a sample
29.774–34.451 ms after ordinal 8 N starts. The pinned source changes its locked
launch thread around `exec.Cmd.Start`, then restores and verifies the observer
mask. This is consistent with the event; its exact duration is not measured.
**Worker-exclusive affinity failed and stays failed.** All observed native-thread
masks match 0..7. The [v4 observation closeout](../../../../docs/plans/buildopt-screen-completion-history-observation-v4.md)
and [regression proof](./corrected/history-reader-v4-test.log) preserve the failure,
reject bad native masks/scope/missing coverage, and add zero native/JVM starts.
The original controller/reader failures remain available. None becomes a G0,
diagnostic-quality, precision, G3 or product pass.

## Costs, scope and preserved failures

The corrected round uses 54 reservations/54 observed Gradle starts and 55 known
comparator/API JVM starts. The superseded round retains 20 reservations/11 observed
Gradle starts and 11 helper JVMs. This continuation therefore charges 74 reservations
and observes 65 Gradle starts, with 66 known helpers. Across the program the totals
are 734 charged/615 observed Gradle starts; the older 29 nested starts, possible
metadata JVM, five early R4 non-Gradle launches, two old unknown-reservation flags,
and incomplete superseded CPU 4 capture retain their recorded limits.

The four corrected profiles take 22,236.111 s in total, including preparation,
capture and independent checking. These laboratory costs are separate from
customer-request saving. Warmup differences do not measure marginal adoption,
automatic discovery, maintenance or adaptive lifetime ROI.

The final partitioned storage audit gives an allocated-block upper bound of
82,489,585,664 bytes (plus a 16-MiB closeout reserve), within the shared 160-GiB cap.
Cross-part hardlinks or shared/reflink extents can make this accounting conservative;
it is not exact exclusive Btrfs physical usage. The original sequential 20-minute
`du` timeout remains retained. This host uses rotational Btrfs storage; the result
does not establish behavior on SSD-backed machines or other repositories.

The [superseded 8-CPU result](./superseded/profile-8.json) retains its 43.814%
aggregate and highly uneven deltas; it did not prove full Main/Test activation.
Its subsequent 4-CPU baseline exited 0 before incomplete capture was stopped.
[Owned-process closure](./superseded/profile-4-cancel-complete.json) and all charges
remain. The earlier walltime attempt remains sealed and inconclusive.

## Final validation scope

The [final audit](./final-audit.json) binds the current report, artifact copies,
source checks and allocation closure. The relevant CLI build, unit checks,
fixture workflows, native results and independent comparisons pass. Layout and
focused documentation checks cover the current deliverables.

The repository-wide documentation command failed with 1,631 diagnostics:
eight missing result-file links in this report bundle, now completed with
source-identical copies, and 1,623 diagnostics in older evidence/documents.
The latter remain recorded and untouched; archived source/document copies include
relative links that no longer resolve from their snapshot locations. This is
not a repository-wide documentation pass. The final audit records that limitation.
The legacy aggregate verification-command counter is not a complete command
census; native and helper launch accounting is reconstructed separately.

## Recovery

Use the [current tracker](../../../../docs/plans/buildopt-product-viability-v1-tracker.md)
and `engineeringPrefix.screenCompletionFixed` in
`.tools/state/buildopt-product-viability-v1/task-state.json`.
Raw root: `.tools/state/buildopt-product-viability-v1/bv006-screen-completion-fixed`.
The separate `receipts/controller-closeout-v4.json` closes the analysis; the failed
`controller-v3.json` is immutable. The original round remains at sibling
`bv006-screen-completion`. Copied evidence binds local source paths; it is not an
independently runnable Elasticsearch/toolchain distribution.
