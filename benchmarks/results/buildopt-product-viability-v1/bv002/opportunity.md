# Why checking fewer Elasticsearch files offered too little saving

Elasticsearch's ForbiddenPatterns task scans files for text the project
disallows. This study asked whether it should check only changed files instead
of scanning everything again. Through the first twenty code changes, Gradle
already skipped the task on seventeen builds. There was too little remaining
work to meet the required average saving, so the additional correction was
not implemented or timed.

The decision on 8 September 2026 was `NO_MATERIAL_NATIVE_OPPORTUNITY`. It applies
to this additional file-checking idea in the selected Elasticsearch workflow.
The earlier correction that lets Gradle cache the whole task was already part
of the reference build.

<a id="source-native-baseline-and-proof"></a>

## The history and reference build

The study ran `:server:precommit`, the server checks required before accepting
a change, through twenty consecutive Git changes. It kept the same workspace,
build cache and background Gradle process between changes. The sequence covered
9.068 hours of development. It was the development portion of a 100-change,
45.959-hour window; the eighty validation changes were not run.

The reference used Gradle 9.7.1, the pinned Java toolchains, eight CPUs and
workers, parallel task execution and Gradle's build cache. The existing
ForbiddenPatterns caching patch was included on purpose: any further
optimization had to improve on it.

Gradle's Configuration Cache, a separate feature that reuses build preparation,
was disabled for this workflow. Saving it worked, but reusing it in a fresh
Gradle process failed in Spotless, the source-formatting plugin. The recorded
error was `Spotless JVM-local cache is stale`. The
[compatibility decision](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv002/configuration-cache-decision.json)
retains both attempts. This finding is limited to the tested setup.

The starting build and all twenty changes completed successfully. All twenty
project tests passed without failures, errors or skips. Checks confirmed the
source files, executed tasks and expected completion marker; none of the builds
changed tracked source. The [baseline record](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv002/baseline.json)
and [project-test results](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv002/owner-proof.json)
identify the exact inputs and checks.

<a id="causal-finding-and-frequency"></a>

## How often the task ran

Only changes 7, 19 and 20 changed inputs that required the file scan. Gradle
reported the other seventeen as `UP-TO-DATE`: their existing result was already
valid, so there was no scan to shorten.

| Change | Changed inputs | Files scanned | Time scanning | Complete Gradle build |
| --- | ---: | ---: | ---: | ---: |
| 7 | 11 | 8,984 | 5.557 s | 130.750 s |
| 19 | 1 | 8,984 | 3.801 s | 74.464 s |
| 20 | 6 | 8,985 | 5.298 s | 88.801 s |

On each of those three builds, the scan finished well before the last required
check or compilation step. Shortening it would not necessarily make the build
finish sooner, because other work was still running.

<a id="admission-arithmetic"></a>

## The most generous estimate still fell short

The twenty Gradle build spans totalled 502.069 seconds. The complete commands
took 516.979 seconds. The calculation used the smaller total to give the idea
the more favorable percentage, and assumed no time spent finding, checking or
maintaining the correction.

| Assumption | Time removed across all twenty builds | Average per build | Reduction against the 502.069-second total |
| --- | ---: | ---: | ---: |
| Remove all scanning, including checks that changed files still need | 14.656 s | 0.733 s | 2.919% |
| Remove the entire task, including Gradle's input checks | 15.735 s | 0.787 s | 3.134% |
| Required saving | At least 25.103 s for the percentage requirement | At least 1.000 s | At least 5.000% |

Even deleting the whole task fell below both requirements. These are optimistic
estimates from ordinary Gradle observations, not measured optimization results.
No candidate ran, and there is no paired statistical saving to report.

A model that keeps other tasks and their dependencies unchanged predicted no
direct reduction in build completion time. It does not capture effects such
as reducing competition for CPU or disk access. No evidence established that
those effects would make this idea large enough to pass.

<a id="conditional-h2-and-the-next-observation"></a>

## What this left to investigate

The study also checked two possible causes of unnecessary rebuilding. Neither
was established. A private implementation change did not trigger downstream
module compilation, and Checkstyle ran after genuine source changes. The
[input-change assessment](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv009/h2-admission.md)
records those findings; they do not prove that every build step was minimal.

Checkstyle, which checks Java coding rules, spent much longer running and was
on the chain of work delaying completion at changes 19 and 20. Its production
check accumulated 184.178 seconds; all three Checkstyle tasks accumulated
316.072 seconds, with overlap. Those totals are not directly recoverable build
time. They justified the separate [Checkstyle investigation](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv006-screen-completion/README.md),
whose later results include both selected savings and a slower history.

<a id="every-engineering-transition"></a>

## Results for all twenty changes

The table retains every measured change. `UP-TO-DATE` means Gradle reused the
existing completed result; `EXECUTED` means the scan ran. The first column is
the change's position in the selected history, not an independent repetition.

| Ordinal | Revision | Changed task inputs | Native task outcome | Native build s | Scanner action s |
|---|---|---|---|---|---|
| 1 | 61470113b4cc | 0 | UP-TO-DATE | 7.304 | 0.000 |
| 2 | 8de19265331e | 0 | UP-TO-DATE | 5.889 | 0.000 |
| 3 | 620db26a7418 | 0 | UP-TO-DATE | 19.723 | 0.000 |
| 4 | 7bbd92d9d98a | 0 | UP-TO-DATE | 16.383 | 0.000 |
| 5 | f35bad5aaed0 | 0 | UP-TO-DATE | 4.598 | 0.000 |
| 6 | b653442570bc | 0 | UP-TO-DATE | 4.653 | 0.000 |
| 7 | 3a00a9167b54 | 11 | EXECUTED | 130.750 | 5.557 |
| 8 | a48778e0c3fc | 0 | UP-TO-DATE | 7.345 | 0.000 |
| 9 | 3b0e446a54e5 | 0 | UP-TO-DATE | 5.394 | 0.000 |
| 10 | 10176f3a95ef | 0 | UP-TO-DATE | 16.325 | 0.000 |
| 11 | 368a07ceed50 | 0 | UP-TO-DATE | 5.621 | 0.000 |
| 12 | 3acb36664e57 | 0 | UP-TO-DATE | 27.997 | 0.000 |
| 13 | 9aec7d25f20e | 0 | UP-TO-DATE | 18.320 | 0.000 |
| 14 | 48a69d226907 | 0 | UP-TO-DATE | 15.941 | 0.000 |
| 15 | d3c85a62f63a | 0 | UP-TO-DATE | 4.271 | 0.000 |
| 16 | 54f83a8180be | 0 | UP-TO-DATE | 16.519 | 0.000 |
| 17 | 1e1b40d2f3dd | 0 | UP-TO-DATE | 5.427 | 0.000 |
| 18 | d47f43f5ff06 | 0 | UP-TO-DATE | 26.344 | 0.000 |
| 19 | 72aa4d4120d0 | 1 | EXECUTED | 74.464 | 3.801 |
| 20 | 22d6425e9a44 | 6 | EXECUTED | 88.801 | 5.298 |

<a id="costs-retained-failures-and-recovery"></a>

## Recorded work and source records

This phase recorded thirty actual Gradle starts against 31 reservations, with
no nested starts. The first reservation ended before Gradle began. An earlier
failed dependency-copy attempt and the Configuration Cache failure remain
recorded.

The full measurement session took 735.712 seconds and used 2,205.309 CPU seconds,
including source checks and trace processing. Its 21 main build commands took
538.748 seconds. These are research measurements, not the overhead of an installed
product. The [resource record](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv002/resource-ledger.json)
retains the counts; the [storage check](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv002/allocation-at-close.json)
records compliance with the 120-GiB footprint and 40-GiB free-space limits.

The [calculation](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv002/opportunity-analysis.json),
[decision](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv002/opportunity-decision.json)
and [evidence list](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bv002/evidence-manifest.json)
retain source versions and file fingerprints. Per-build records are under
`native-requests/NNN/`; larger traces remain in the local research state. The
[current tracker](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-product-viability-v1-tracker.md)
records later work. The diagnostic-tool copies in this result directory describe
this experiment's inputs; they are not a replacement for the current runner.
