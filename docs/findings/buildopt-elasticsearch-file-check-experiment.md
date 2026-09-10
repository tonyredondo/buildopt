# Reusing Elasticsearch's file checks

Elasticsearch's ForbiddenPatterns task scans source files for text that the
project disallows. A Patch Autopilot experiment changed the build so Gradle
could reuse that task's completed result from its cache. The selected
comparison saved 15.54% and produced the required result.

## The conditions behind the saving

Before each run of `:server:forbiddenPatterns`, the experiment removed the
marker showing that the task had completed. The original task had to do its
work again; the corrected version could restore the saved result. This gave
the correction an opportunity to help on every measured run.

Eight comparisons alternated which version ran first. A warmup in each build
directory was excluded before measurement. Timings from an earlier invalid
attempt were not reused.

| Measurement | Original task | Corrected task |
| --- | ---: | ---: |
| Average elapsed time | 46,139 ms | 38,967.25 ms |
| p95, describing the slower builds | 53,909 ms | 46,305 ms |

All eight comparisons were faster. The average saving was 7,171.75 ms, or
15.54%, with a 95% interval of 5,524.625–9,833.125 ms. Required results matched.

## Why ordinary development needed a separate test

At that measured rate, recovering the recorded machine preparation would take
an estimated 231 applicable builds. Including the measured 20-second patch
review raised the estimate to 233. The experiment did not observe 233 later
builds benefiting from the correction.

The later history study kept this caching correction in its reference build
and asked a different question: when ForbiddenPatterns does run, could it
check only modified files? Gradle already skipped the task on seventeen of
the first twenty changes. Even removing its entire recorded duration would
fall below the required average saving, so that additional idea was dropped.

The original caching result remains valid under its deliberately repeated-task
conditions. Neither experiment establishes its total saving through normal
Elasticsearch development.

The [original measurements](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/economics-gated-reviewed-native-patch-v2/raw.json)
and [evidence notes](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-product-viability-v1-evidence.md#e06-native-patches-can-help-within-a-selected-scope)
preserve the source records. The [history calculation](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv002/opportunity.md)
explains why checking only changed files offered too little additional time.
