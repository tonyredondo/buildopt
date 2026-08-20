# Materialization Economics V2

This bundle is the terminal result for
`POC-MATERIALIZATION-ECONOMICS-V2-001`. It reruns the same five frozen public
repositories and customer workflows as Automatic Breadth Transfer V2 with one
exact BuildOpt binary. The candidate still starts from a clean workspace,
reproduces the complete required-output digest and retains working full-graph
fallback.

The follow-up changes only generic POC mechanics:

- task selection comes from the observed Gradle task graph rather than every
  task in a selected project;
- discovery shares the initial useful full-graph invocation;
- verified outputs are stored in one manifest-bound pack and restored with
  bounded parallel reads;
- learning cost includes only the one-time discovery and capture overhead;
- every timed duration is end-to-end wall time, including Gradle and BuildOpt.

No target repository contains a BuildOpt file, manual profile or
repository-name product rule.

## Result

| Repository / workflow | Graph | Native mean | Candidate mean | Mean saving | 95% saving interval | Positive pairs | Learning / payback |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Spring Framework `testClasses` | 27 -> 10 | 11.062 s | 9.960 s | **1.102 s / 9.97%** | +0.369..+2.166 s | 8/8 | 4.070 s / 4 builds |
| OpenTelemetry Spring family | 1,024 -> 34 | 75.861 s | 64.534 s | **11.328 s / 14.93%** | +8.866..+14.535 s | 8/8 | 7.357 s / 1 build |
| Apache Kafka `testClasses` | 64 -> 3 | 8.246 s | 5.036 s | **3.210 s / 38.93%** | +2.727..+3.738 s | 8/8 | 3.436 s / 2 builds |
| Micronaut Core `assemble` | 75 -> 22 | 23.997 s | 9.710 s | **14.287 s / 59.54%** | +12.724..+15.903 s | 8/8 | 7.298 s / 1 build |
| Apache Groovy `classes` | 37 -> 2 | 61.855 s | 15.398 s | **46.456 s / 75.11%** | +41.735..+50.552 s | 8/8 | 2.170 s / 1 build |

All five rows now pass the unchanged value gates. Across 85 ordinary
invocations, all 40 pairs are positive, output equivalence is exact, fallback
works and product-attributable failures are zero. Spring moves from the former
7/8 and 67-build native-retained result to 8/8 and four-build payback without
lowering a threshold.

Repository percentages are deliberately not averaged and mechanism
percentages are not added. These timings replace the previous V2 terminal
comparison for this exact binary and protocol; the older bundle remains
immutable before-evidence.

## Captured materialization cost

| Repository | Files / bytes captured | Collect | Pack | Manifest | Total one-time cost |
| --- | ---: | ---: | ---: | ---: | ---: |
| Spring Framework | 14,445 / 42.3 MB | 0.839 s | 0.534 s | 0.251 s | **1.625 s** |
| OpenTelemetry | 243 / 0.8 MB | 2.946 s | 0.075 s | 0.057 s | **3.078 s** |
| Apache Kafka | 3,629 / 36.1 MB | 0.750 s | 0.384 s | 0.132 s | **1.268 s** |
| Micronaut Core | 211 / 195.4 MB | 2.308 s | 1.129 s | 0.053 s | **3.492 s** |
| Apache Groovy | 3,828 / 17.7 MB | 0.449 s | 0.286 s | 0.135 s | **0.871 s** |

The bundle keeps the manifest and all hashes but omits the large `payload.pack`
files. They are runtime state, not source evidence, and the strict checker
validates their recorded digest and entry metadata without committing customer
build outputs to Git.

## Recompute

```bash
./dev/check-materialization-economics-v2 \
  benchmarks/results/poc-materialization-economics-v2/summary.json
```

The checker validates all 85 invocation records, product/repository bindings,
state-tree hashes, pack entries, exact outputs, balanced order, means, p95,
intervals, end-to-end timing decomposition, payback and fallback. The expensive
public-repository capture is intentionally not part of normal CI.

The protocol is defined in
[`poc-materialization-economics-v2.md`](../../../specs/poc-materialization-economics-v2.md).

## Scope

- POC evidence captured on the 12-CPU development host with a common 12-worker
  cap; it is not production capacity or SLO evidence.
- No production authority, soak or design-partner requirement is introduced.
- Test Optimization remains out of scope.
- The next question is profile lifetime across compatible public descendant
  changes, not another rewrite of this calibration result.
