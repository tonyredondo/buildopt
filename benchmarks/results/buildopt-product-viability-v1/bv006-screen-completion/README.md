# Checkstyle on selected changes and a longer sequence

Checkstyle checks Java source against coding rules and writes reports of any
violations. This Elasticsearch experiment changed it to reuse checks for
unchanged files while preserving the complete reports. Two selected code
changes became faster with eight, four and two CPUs. A separate sequence of
thirteen changes was 19.16% slower overall.

The result shows a benefit on the selected changes, but it does not establish
a saving through ordinary development. The recorded decision on 10 September
2026 was `NO_MATERIAL_SIGNAL_IN_DISJOINT_HISTORY`.

<a id="measured-walltime"></a>

## What ran and how long it took

The experiment ran the Elasticsearch server's precommit checks, the work
required before accepting a change. Both versions kept ordinary Gradle build
state between commits. Reported times include the complete build request and
recorded extra work.

The reference build already included the earlier ForbiddenPatterns caching
correction. Gradle's Configuration Cache was disabled in both versions because
this workflow had failed its compatibility check.

| Changes tested | CPU and worker limit | Measured comparisons | Average with Gradle | Average with the correction | Result |
| --- | --- | ---: | ---: | ---: | --- |
| Selected changes 19–20 | Eight | 2 | 85.697 s | 50.315 s | 35.381 s faster / 41.287% |
| Selected changes 19–20 | Four | 2 | 106.609 s | 63.844 s | 42.765 s faster / 40.114% |
| Selected changes 19–20 | Two | 2 | 233.740 s | 177.374 s | 56.366 s faster / 24.115% |
| Consecutive changes 2–14 | Eight | 13 | 22.235 s | 26.495 s | 4.260 s slower / 19.158% |

Each CPU setting used four warmup builds on changes 17–18 and four measured
builds on 19–20: 24 builds altogether. Eight CPUs was the first setting to meet
the selection rule, following the order fixed before measurement.

The thirteen-change study started separately, warmed up at changes 0–1 and
kept every measured result from 2–14. It used another thirty builds. These were
development changes already available for investigating the correction. The
80 changes reserved for validation, 21–100, were not used.

Reducing CPUs increased the absolute saving on the two selected changes, but
reduced the percentage saving. Both the CPU limit and Gradle worker count
changed. Two comparisons per setting cannot show a general scaling pattern.
The separately initialized studies cannot be combined into a single history.

## What the history exposed

Across the thirteen changes, Gradle took 289.056 seconds and the corrected
version took 344.433 seconds: a loss of 55.377 seconds. Checkstyle ran on only
change 7. The other twelve builds still count; together they added 14.283
seconds to the observed difference. That difference alone does not identify
how much delay came from the correction or the machine.

Change 7 took 110.724 seconds with Gradle and 151.818 seconds with the
correction, a loss of 41.094 seconds. The task timings help locate the problem:

| Work | Original duration | Duration with the correction |
| --- | ---: | ---: |
| Checkstyle on production source | 72.718 s | 117.340 s |
| Checkstyle on test source | 60.568 s | 9.284 s |
| Collecting transport-version references | 11.268 s | 47.120 s |

These tasks overlap, so their differences cannot be added to explain the
whole-build loss. The middle paired saving across the thirteen changes was
almost zero, −0.003 seconds; a few slow cases strongly affected the average.

Saved Checkstyle records and reports survived unchanged between changes 6
and 7. The production task had 4,954 unchanged files, four changed files and
one removal; the test task had 2,819 unchanged files and six changes. At this
stage, those counts showed what could be reused. They did not yet prove what
the checking engine actually skipped or explain where it spent its time.
The [later investigation](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-regression/README.md)
examined that question. The [latest comparison](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/README.md)
records the subsequent change and its mixed result.

## Correctness and repairs

All 54 builds in the corrected round succeeded. All 27 pairs passed the full
output comparison, both during the run and when independently checked afterward.
Every corrected build retained the three required Checkstyle histories.

An earlier installer defect had left two Checkstyle tasks using ordinary
Gradle behavior. The repair gave each task its own settings, and the corrected
round verified that all three used the intended configuration. An earlier
43.814% result does not prove full use of the correction and remains marked
as superseded. The associated four-CPU attempt had incomplete recording and
also remains outside these results.

The measurement tool was repaired to handle a process ending while it was
being inspected, while still failing on ownership or permission errors.
Recorded output copies were kept independent of files that a later build
could change. The [installer repair](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-screen-completion/corrected/candidate-fix.patch)
and [measurement-tool checks](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-screen-completion/harness/runtime-proof.log)
retain the implementation details and proof.

<a id="observation-qualification-remains-limited"></a>

## Limits of the measurements

The study did not keep every monitoring thread within its assigned CPU limit.
The eight-CPU test recorded one startup exception. The history study recorded
a monitoring thread briefly sharing the build's CPUs; its exact duration was
not measured. The four- and two-CPU samples passed their CPU-placement checks.

That failed separation check remains a failure. It limits confidence in the
performance setup without changing the successful output comparisons. The
[detailed CPU observations](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-screen-completion-history-observation-v4.md)
retain the exception and the checks used to assess it.

<a id="costs-scope-and-preserved-failures"></a>

## Recorded work and earlier attempts

The corrected round recorded 54 Gradle starts and 55 helper Java processes.
The earlier round recorded eleven Gradle starts and eleven helpers. The two
rounds together therefore observed 65 Gradle starts against 74 reserved starts.
A reservation could be consumed by an attempt that failed before Gradle began.
At this point, the research program recorded 615 observed starts against 734
charged starts, with older incomplete observations retained in the accounting.

Preparing, running and independently checking the four corrected profiles took
22,236.111 seconds. That is experiment time, not time saved by an installed
optimization. The storage audit estimated at most 82,489,585,664 allocated bytes,
plus a 16-MiB allowance for final records, within the 160-GiB limit. Shared files
can make that estimate larger than the space used exclusively by this study.
The result was obtained on a workstation with a hard disk; it does not establish
the same timings on an SSD.

## Final validation scope

The relevant command-line build, unit checks, test workflows and independent
output comparisons passed. At the time of this experiment, the repository-wide
documentation check reported 1,631 issues: eight missing links in this bundle,
subsequently filled, and 1,623 in older documents and archived copies. That
historical check was not a documentation pass. It is separate from later
publication checks.

The [final audit](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-screen-completion/final-audit.json)
records those limits. The [summary data](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-screen-completion/summary.json),
[task timings](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-screen-completion/corrected/history-task-diagnostic.json)
and [saved-state checks](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-screen-completion/corrected/history-state-continuity.json)
provide the evidence behind this report.

<a id="recovery"></a>

## Research records

The [artifact list](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-screen-completion/completion-artifacts.json)
and [current tracker](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-product-viability-v1-tracker.md)
locate the remaining records. Local recovery uses
`engineeringPrefix.screenCompletionFixed` in
`.tools/state/buildopt-product-viability-v1/task-state.json`; the raw files are
under `.tools/state/buildopt-product-viability-v1/bv006-screen-completion-fixed`.
Earlier failures remain in the records. The eighty reserved changes have not
been run.
