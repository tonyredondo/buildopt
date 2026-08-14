# Generic Change-Shape Breadth

This directory contains the terminal evidence for the preregistered
[generic change-breadth POC](../../../specs/poc-generic-change-breadth-v1.md).
It asks whether the same reviewed Structural Build Impact path transfers
across distinct source changes and still retains optimized native Gradle for
build-logic or global-configuration changes.

## Selective result

All six selective cells qualify independently. Percentages are deliberately
not averaged across cells or added to other mechanisms.

| Repository and workflow | Change class | Full -> selected projects | Native mean | BuildOpt mean | Mean saving | Blocks | p95 native -> BuildOpt |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Apache Groovy `jar` | leaf source | 37 -> 2 | 71.227 s | 18.843 s | **52.384 s / 73.54%** | 8/8 positive | 75.044 -> 21.650 s |
| Apache Groovy `jar` | shared source | 37 -> 2 | 79.557 s | 27.207 s | **52.349 s / 65.80%** | 8/8 positive | 84.150 -> 30.375 s |
| Kafka `checkstyleMain` | metadata source | 64 -> 2 | 82.431 s | 59.353 s | **23.078 s / 28.00%** | 8/8 positive | 85.360 -> 62.772 s |
| Kafka `checkstyleMain` | client-utils source | 64 -> 2 | 87.522 s | 61.177 s | **26.345 s / 30.10%** | 8/8 positive | 93.850 -> 63.909 s |
| Kafka `shadowJar` | clients source | 64 -> 2 | 44.828 s | 14.956 s | **29.872 s / 66.64%** | 8/8 positive | 46.699 -> 17.433 s |
| Kafka `shadowJar` | generator source | 64 -> 2 | 37.079 s | 7.587 s | **29.492 s / 79.54%** | 8/8 positive | 39.946 -> 8.636 s |

Each cell contains two independent captures of eight alternating pairs. The
96 raw pairs form 48 reciprocal AB/BA blocks; all 96 pairs and all 48 blocks
improve. Every deterministic bootstrap lower bound is positive, every
candidate p95 is lower, required outputs are equivalent under their reviewed
contracts, task shapes are stable, all 12 selective full-graph fallbacks pass,
and product-attributable failures are zero.

The two source changes in each workflow are not treated as one cached answer.
For example, the Kafka generator change executes one more changed-target task
than the direct clients change. The generic graph still proves that the
selected lifecycle entrypoint covers both the changed source and the declared
output.

## Conservative result

All four fallback cells also pass independently:

| Repository and workflow | Change class | Captures | Decision |
| --- | --- | ---: | --- |
| Apache Groovy `jar` | build logic | 2/2 | `NATIVE_FULL_GRAPH / GLOBAL_CHANGE_REQUIRES_FULL_GRAPH` |
| Apache Groovy `jar` | global configuration | 2/2 | `NATIVE_FULL_GRAPH / GLOBAL_CHANGE_REQUIRES_FULL_GRAPH` |
| Kafka `checkstyleMain` | build logic | 2/2 | `NATIVE_FULL_GRAPH / GLOBAL_CHANGE_REQUIRES_FULL_GRAPH` |
| Kafka `shadowJar` | global configuration | 2/2 | `NATIVE_FULL_GRAPH / GLOBAL_CHANGE_REQUIRES_FULL_GRAPH` |

Those eight captures run and validate the complete owner workflow, emit no
candidate documents, and make no timing claim. A safe fallback is correctness
evidence, not an acceleration result.

## Interpretation and boundary

The terminal evidence uses immutable BuildOpt revision
`706ee5480fd24aade078565925ec853d7ae67de7` and executable SHA-256
`cf2107fbc269af4dd46c5acdd6f8f4dc8d12e3df7f1882f3d9764672d48d89af`.
It demonstrates transfer across the exact ten changes, three workflows, two
public repositories, and reviewed outputs in this matrix without adding a
repository-name product rule.

It does not prove repository-wide or universal activation. Discovery,
distribution preparation, and warm-up are outside stable-state pair timing
and can be expensive. Profiles remain review-required; a new, ambiguous,
global, or drifted input retains optimized native Gradle.

## Layout and validation

- `summary.json` is the terminal ten-cell matrix.
- `<cell>/qualification.json` is recomputed from both captures.
- `<cell>/capture-{1,2}/` contains the proposal, reviewed output contract,
  observations or explicit fallback, logs, and result.

Recompute all selective qualifications and validate every fallback:

```bash
./dev/check-generic-change-breadth-result \
  "$PWD/benchmarks/results/poc-generic-change-breadth-v1"
```

This is bounded POC evidence. It creates no production authority, automatic
activation, soak requirement, design-partner claim, or Test Optimization
scope.
