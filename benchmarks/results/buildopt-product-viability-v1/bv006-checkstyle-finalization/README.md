# Saving Checkstyle results at the end of a build

Checkstyle checks Java source against coding rules. Our experimental correction
keeps the results for unchanged files so they do not need checking again. This
follow-up changed when those results are saved for the next build.

The revised implementation passed the correctness checks. Its speed result was
mixed: one comparison improved and the other became slower. It did not meet
the requirement that both comparisons improve. The recorded decision on
10 September 2026 was `FINALIZATION_VALUE_NOT_ESTABLISHED`.

Both versions already contained our Checkstyle optimization. This experiment
compared the previous and revised implementations; it did not measure a new
saving against ordinary Gradle.

## Results

| Comparison | Previous optimized version | Revised optimized version | Result |
| --- | ---: | ---: | --- |
| First | 124.537 s | 83.160 s | 41.377 s faster / 33.22% |
| Second | 83.003 s | 97.339 s | 14.337 s slower / 17.27% |
| Average | 103.770 s | 90.250 s | 13.520 s faster / 13.03% |

The rule fixed before measurement required at least one second saved in each
comparison and at least 5% saved overall, with complete output and timing
records. The positive average did not make the second comparison pass.

There were eight successful Elasticsearch builds: four warmups and four
measured builds. The two measured comparisons reversed which version ran first.
Every result was kept; no slow sample was removed or rerun to improve the
numbers. These are complete build-request times, including the work of saving
Checkstyle results. The [individual timings](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/requests.tsv)
also retain the Gradle command time separately.

## What changed and what passed

The new version saves Checkstyle's results through a supported Gradle mechanism
that runs when a shared build service closes. It saves only after the relevant
work has succeeded, and only if the source context and completed report still
match. This replaced a deprecated listener and an extra action attached to the
end of the task. The checked configuration is Gradle 9.7.1 with Configuration
Cache disabled.

All four output comparisons, including warmups, passed both the live check and
an independent reconstruction. The three Checkstyle histories and report
fingerprints matched their expected contents. Smaller tests covered cold and
warm runs, changed or deleted reports, changed checking settings, failures, cancellation,
recovery and builds with no source to check. Strict warning checks, patch
reversal, four core tests and 25 report-reader cases also passed. The
[correctness report](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/correctness-v2.md)
records their exact scope.

Saving results after a task finishes can make that task's reported duration
shorter without making the whole build finish sooner. The
[detailed timing record](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/task-phases.tsv)
therefore keeps checking, task completion and saving the results separate.
The decision uses the complete request time.

## Why the timing result remains mixed

Both implementations did the same amount of Checkstyle work. The production
check processed five files and reused results for 4,954. The test check processed
six and reused 2,819. The internal-cluster test check was already up to date.
That reuse also occurred in the slower revised build: losing the saved
Checkstyle history does not explain its slowdown.

The first measured build in each comparison took about 83 seconds, whichever
implementation ran first. Both second builds were slower, and their background
Gradle processes had been idle longer:

| Run order | Implementation | Build time | Time idle since its warmup ended |
| --- | --- | ---: | ---: |
| First comparison, first run | Revised | 83.160 s | 334.95 s |
| First comparison, second run | Previous | 124.537 s | 1,394.44 s |
| Second comparison, first run | Previous | 83.003 s | 210.26 s |
| Second comparison, second run | Revised | 97.339 s | 652.12 s |

The workstation was busy during part of the experiment. The slower revised
build also showed substantial disk reads and waits to load memory pages from
storage. Those observations support investigating machine state and run order;
they do not identify the exact cause of every delay. One sampling gap exceeded
the agreed limit, and no swap counters were collected. The
[machine observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/host-summary.json)
and [run-order data](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/order-and-idle-age.json)
retain those limits.

All eight builds triggered the recorded machine-pressure flags, which include
pressure from the experiment itself. None was excluded. Running identical code
on both sides is the proposed way to check how much difference this measurement
schedule can produce without an implementation change.

<a id="retained-failures-and-cost"></a>

## Earlier failures and recorded work

The first controller chose the wrong build runner and stopped before Gradle
started. The first patch then failed Elasticsearch's rule that treats warnings
as errors. The revised patch removed the deprecated mechanism and passed the
strict warning check. The earlier failures remain in the evidence.

This phase observed ten Elasticsearch starts, including the two from the
rejected patch comparison; seventeen smaller correctness starts; and two
formatter starts. Of the smaller starts, twelve succeeded, three tested
expected failures and two tested cancellation. There were also twenty separate
Java helper processes and nine compiler commands.

That makes 29 observed Gradle starts against 43 reserved starts for this phase.
At its close, the program recorded 651 observed starts against 791 charged
starts. Reserved work includes attempts that stopped before a build began;
these counts are not counts of independent performance experiments. The
[closeout record](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/proof-v2/analysis/closeout-v2.json)
retains earlier accounting limits and the 42.90-GiB storage estimate.

One analysis initially counted an aggregate task as a Checkstyle worker. The
analysis was corrected to use the three actual checking tasks. It reused the
same successful runs and kept the full output requirements; it did not create
another timing sample.

## What this means for viability

The earlier thirteen-change sequence ran Checkstyle on only one change. It
would need at least 14.453 seconds of net saving across the whole sequence to
meet the existing percentage and per-build targets, before further setup or
maintenance work. The separate twenty-change estimate left only 4.746 seconds
above its 5% target. Both calculations show why overhead and frequency matter;
neither is a measured result for the revised implementation.

The next unresolved check is the proposed
[four-build comparison using identical code](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/next-step.md).
It has not run. A comparison against ordinary Gradle and the eighty reserved
changes would still be required afterward. Other repository histories and
adaptive behavior remain untested by this experiment.

The reviewable changes are the [revised patch](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/candidate-v2.patch),
[reversal](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/candidate-v2.inverse.patch)
and [difference from the previous version](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/current-to-supported.patch).
The [summary](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/summary.json),
[full measurements](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/profile.json)
and [source and evidence list](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization/comparison-evidence.json)
allow the result to be checked. Local recovery uses
`engineeringPrefix.checkstyleFinalization` in
`.tools/state/buildopt-product-viability-v1/task-state.json`.
