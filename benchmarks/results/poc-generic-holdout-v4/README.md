# Hibernate ORM target-workload attribution

This immutable diagnostic bundle investigates why the earlier Hibernate ORM
holdout did not satisfy the frozen eight-of-eight repeatability gate. It uses
BuildOpt revision `ef595cb584009b4a8b18153bc8f716ba6145007e`, the same public
Hibernate revision, exact `Session.java` comment change, root `assemble`
workflow, repository-defined core JARs, 12-worker optimized-native control and
Build-Impact-only candidate as the v2/v3 runs. No earlier timing observation is
reused.

Version 4 adds an excluded warm-up of the exact target revision, normalized
task/outcome fingerprints and interval-scoped Linux PSI for both arms. All
eight fresh pairs and the scheduling-equivalent full-graph fallback completed;
the three required JARs were byte-identical in every pair and in fallback.

## Result

| Metric | Optimized native | BuildOpt candidate | Result |
| --- | ---: | ---: | --- |
| Mean wall time | 221.898 s | 213.418 s | **8.480 s / 3.82% faster** |
| Positive pairs | — | 5/8 | Retain native under the unchanged gate |
| Paired 95% interval | — | — | -0.839..+17.478 s |
| Required outputs | 3 JARs | Same 3 JARs | Byte-identical in all pairs |

The result is a positive value signal, not qualifying evidence. It misses the
eight-of-eight gate, its lower interval bound is negative and the native task
fingerprint is not stable. BuildOpt therefore emitted no profile and selected
the verified full-graph fallback.

## What failed, and why it is recoverable

The candidate remained structurally stable in every target warm-up and measured
pair: 32 tasks and one task/outcome fingerprint. The native control evolved
from 300 tasks in the target warm-up through 302 and 301 tasks in the early
pairs, then remained at 300 for four pairs before returning to 301 in pair
eight. The first measured observations therefore began before the native target
workload had a repeatable execution shape.

Timing also retained a strong period effect. BuildOpt was positive in all four
`CONTROL_FIRST` pairs, but in only one of four `CANDIDATE_FIRST` pairs:

| Order | Positive pairs | Mean saving |
| --- | ---: | ---: |
| Control then candidate | 4/4 | +17.654 s |
| Candidate then control | 1/4 | -0.694 s |

The existing alternation balances that effect in the overall mean but cannot
make every raw pair positive. As a diagnostic only, grouping each consecutive
AB/BA pair into an order-adjusted crossover block yields +4.278, +9.003,
+15.395 and +5.245 seconds: four positive blocks. Those post-hoc blocks are not
qualification evidence and must not be reused by a subsequent protocol.

Linux PSI showed sustained IO pressure in every measured arm (`some` roughly
67-76% of interval wall time), negligible memory pressure and lower CPU wait in
the smaller candidate. The three negative pairs did not have a unique PSI
signature, so host pressure is a material source of variance but does not by
itself explain or excuse a failed pair.

The evidence rejects a structural-product failure: the candidate consistently
reduced 29 projects to one, executed 32 tasks instead of roughly 300 and
preserved exact outputs. It attributes the failed gate to two measurement
problems that can be corrected generically: insufficient target-workload
stabilization and unmodelled first/second-period effects. The next protocol must
capture task-path differences explicitly, establish target stability before
timing, and use fresh reciprocal AB/BA blocks as its observation unit while
retaining fail-closed output and fallback checks.

Validate the retained result without network access:

```bash
./dev/check-generic-holdout \
  benchmarks/results/poc-generic-holdout-v4 \
  specs/poc-generic-holdout-v4.json
```
