# Automatic Breadth Transfer Result

This retained POC evidence runs the unchanged customer-shaped command across
five frozen public Gradle repositories:

```text
buildopt optimize <the repository's existing Gradle workflow>
```

The installed `0.3.0-dev` package at revision
`21c913db7ad744fbce46f1b72a650c7f63b45940` receives no BuildOpt manifest,
graph, output contract, profile or repository-specific hint. Every repository
uses a fresh checkout and private Gradle home on the same 12-CPU Linux host.

## Terminal observations

| Repository / workflow | Automatic graph | Native mean | Candidate mean | Saving | Pairs / interval | Learning cost / payback | Decision |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Spring Framework `testClasses` | 27 -> 10 | 12.340 s | 9.029 s | **3.311 s / 26.83%** | 7/8; +0.426..+8.114 s | 339.603 s / 103 builds | Native retained: value not proven. |
| OpenTelemetry Spring family `testClasses` | 1,024 -> 34 | 76.087 s | 60.681 s | **15.407 s / 20.25%** | 8/8; +10.364..+24.323 s | 1,555.444 s / 101 builds | Native retained: payback exceeds 30. |
| Apache Kafka `testClasses` | 64 -> 36 | 14.766 s | 12.785 s | **1.982 s / 13.42%** | 3/8; -0.091..+5.183 s | 374.762 s / 190 builds | Native retained: value not proven. |
| Micronaut Core `assemble` | no timed candidate | n/a | n/a | n/a | n/a | no calibration | Native retained: 73 candidate task entrypoints are too broad. |
| Apache Groovy `classes` | 37 -> 30 | 71.480 s | 69.472 s | **2.008 s / 2.81%** | 7/8; +0.498..+3.634 s | 1,423.987 s / 710 builds | Native retained: value not proven. |

All five executions returned zero product-attributable failures and required
zero BuildOpt files in the target repository. All four timed candidates
preserved exact required outputs and passed full-graph fallback. Percentages
are repository-specific and are neither averaged nor added to another
mechanism.

## What the result means

The generic implementation can discover real structural reductions. The
OpenTelemetry result is particularly strong technically: 990 projects are
omitted, all eight pairs improve and the lower confidence bound is positive.
The current synchronous eight-pair learning model is nevertheless too
expensive: none of the five subjects can repay within the POC's maximum 30
matching builds.

The result also explains why older reviewed profiles cannot be presented as
zero-configuration onboarding results. Kafka and Groovy require much broader
output contracts when BuildOpt must preserve the complete declared workflow,
and Micronaut's aggregate `assemble` surface is too large to calibrate safely.
The successor [incremental-learning block](../poc-incremental-learning-v1/README.md)
removes measurement-only workflow runs while preserving the same decision
gates. The later [verified-output materialization block](../poc-verified-output-materialization-v1/README.md)
closes the clean-workspace output gap without adding a timing claim. Aggregate
graph precision and the unchanged five-repository rerun remain next; neither
may add repository-name rules or weaken output/fallback checks.

## Recompute the evidence

```bash
./dev/check-automatic-breadth-transfer
```

The checker verifies the five frozen revisions, one BuildOpt binary, raw and
state-tree hashes, exact terminal decisions, alternating observations, output
identity, full fallback, means, reduction ratios and break-even calculations.

This is POC evidence only. It grants no production authority, soak or
design-partner requirement, and it does not modify Test Optimization.
