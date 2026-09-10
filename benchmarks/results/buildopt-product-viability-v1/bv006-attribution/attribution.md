# Collector attribution result

Status: **verified diagnostic**, 2026-09-09. Readiness remains unqualified.

All 20 fixed native requests succeeded: four warmups and sixteen measured requests.
Each measured root workflow has 1256 UP-TO-DATE, 44 NO-SOURCE and one EXECUTED
task. All three Checkstyle tasks are UP-TO-DATE. The four build contexts have
complete matching graph/phase coverage; no native failure or sample was removed.

## Observed collector phases

Median current-thread CPU, in milliseconds. The top callbacks below do not
include nested children twice. These measurements include diagnostic overhead;
they are neither measured product savings nor critical-path wall delay.

| Build context | Tasks/request | N callback CPU | I callback CPU |
|---|---:|---:|---:|
| `:build-conventions` | 5 | 4.190 | 4.100 |
| `:build-tools` | 11 | 5.232 | 4.976 |
| `:build-tools-internal` | 9 | 4.866 | 4.567 |
| `:` | 1301 | 541.334 | 552.130 |

| Root phase, inclusive of its children | N CPU ms | I CPU ms |
|---|---:|---:|
| `graphCallback` | 238.655 | 243.440 |
| `graphConstruction` | 229.712 | 233.210 |
| `checkstyleEnumeration` | 57.694 | 56.757 |
| `dependencies` | 75.992 | 79.365 |
| `outputEnumeration` | 41.939 | 42.409 |
| `beforeCallback` | 51.537 | 52.388 |
| `afterCallback` | 250.078 | 255.124 |
| `afterRecord` | 109.668 | 109.362 |
| `json` | 17.930 | 19.039 |
| `encoding` | 19.418 | 20.381 |
| `sink` | 81.813 | 85.333 |

Root callback wall sums have medians 1009.580 ms for N and 1087.414 ms
for I. They can overlap across task threads and include native provider
realization. Adding nested rows, subtracting these sums from native duration,
or treating the sum as incremental critical-path delay would be invalid.

## Complete measured paired durations

Diagnostic native duration minus the adjacent plain native duration, in ms.
The added diagnostic work may affect execution; these numbers cannot qualify
the original collector or supply an optimizer saving.

| Pair | N diagnostic minus plain | I diagnostic minus plain |
|---:|---:|---:|
| 0 | -22697.146361 | 934.694348 |
| 1 | 4757.978894 | -883.374757 |
| 2 | 31.477550 | 6051.508267 |
| 3 | -2895.219197 | -1066.751871 |

The inherited lower-nearest-rank medians are -2895.219197 ms
and -883.374757 ms. Their absolute difference is
**2011.844440 ms**, above the unchanged 100-ms envelope.
Wrapper p95 is 1.609903 ms, below 10 ms.

The wide native variation is not explained by an arm-specific collector
phase in these observations. JVM compilation and GC counters vary; causality
has not been isolated. Whole-cgroup CPU includes daemon, CLI, worker and
descendants over a wider interval. It cannot be assigned to the collector
or supervisor by subtraction. Undelegated I/O accounting remains UNAVAILABLE.

## Decision

No production collector/worker rewrite is selected. Serialization-only or
file-open-only work is not supported as a primary repair of this arm gate.
Preserve the earlier failed uninstrumented 20-request result and this entire diagnostic.
Use the [prospectively registered precision design](precision-design.md):
one fixed 80-request pilot can select or reject one independent confirmation size.
Both original point gates and both declared block intervals remain required.

The complete graph/event/native/daemon bindings, per-request phase counters,
resource observations and closed-unit checks are retained in the task state
`bv006-attribution/attribution-result.json`. The prior immutable diagnostic
plan, script, registration, interpretation rules, preflight setup failure
and successful preflight remain alongside the original command receipts.
