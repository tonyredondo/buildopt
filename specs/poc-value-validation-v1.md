# BuildOpt POC value validation v1

This contract answers one decision: whether BuildOpt produces enough net build-time
reduction over a well-configured native Gradle baseline to justify continuing the
experiment.

The POC validates ideas, not production readiness. It does not require an eight-hour
soak, an external design partner, high availability, enterprise identity, or a
production promotion sample. Those concerns become relevant only after the value
hypothesis passes.

## Decision rule

BuildOpt distinguishes accelerators from safety enablers:

- An **accelerator** passes only when paired evidence against native Gradle has
  identical required outputs, no additional product-attributable failure, a positive
  95% lower bound, and a point estimate of at least `max(500 ms, 2%)`.
- A **safety enabler** may pass at parity when its absolute regression is at most 2%
  and it is required by a downstream accelerator. Parity is not advertised as a
  speed improvement.
- If an optimization cannot clear its value threshold for the current compatibility
  and workload class, BuildOpt does not activate it. `NO_VALUE_NO_ACTION` is a POC
  invariant.

Mechanism results are never added together. The combined product path needs its own
paired experiment against the same native Gradle command and workload.

## Bounded evidence

Mechanism exploration may use four alternating pairs to decide whether an idea is
worth further work. The combined POC value gate requires at least eight alternating
pairs in each of two representative workload classes, including Kotlin and Groovy
DSL coverage. It preserves every raw observation and rejects output divergence,
failed builds, asymmetric warm state, or a weaker baseline.

The native control enables every applicable first-party Gradle feature used by the
candidate: Build Cache, Configuration Cache, daemon/process policy, and parallelism.
BuildOpt must win against that control, not against an intentionally unoptimized
build.

`POC-VALUE-003` fixes four cells before measurement: Build Impact and the exact
reviewed custom-task/Patch route, each in Kotlin and Groovy. Every cell uses one
unmeasured warm-up followed by eight alternating control/candidate pairs. Build
outputs are removed before each arm while its isolated Gradle home, daemon,
Configuration Cache, and Build Cache remain warm. The report keeps all signed
differences and computes a deterministic 4,096-resample paired bootstrap with a
fixed 32-bit LCG; the independent checker recomputes every statistic from the raw
rows. A structurally valid unfavorable result remains valid evidence and is
classified `NO_VALUE_NO_ACTION`; only the higher-level decision contract requires
all four cells to pass before moving to the combined gate.

## Current decision

The current state is `CONTINUE`. This means the idea merits further POC work for
the qualified synthetic workload classes; it does not mean that BuildOpt has proved
universal savings or production readiness. The contractual four-CPU runs in
[`poc-value-baseline-v1.json`](../benchmarks/results/poc-value-baseline-v1.json),
[`poc-value-negative-mechanisms-v1.json`](../benchmarks/results/poc-value-negative-mechanisms-v1.json),
[`poc-value-coverage-v1.json`](../benchmarks/results/poc-value-coverage-v1.json),
and [`poc-value-combined-v1.json`](../benchmarks/results/poc-value-combined-v1.json)
closed `POC-VALUE-001..004` with these results:

- Safe Cache did not demonstrate incremental value over Gradle's native local
  cache. It is classified `NO_VALUE_NO_ACTION`, is explicit-only through
  `BUILDOPT_SAFE_CACHE=1`, and the zero-configuration path now delegates cache
  reuse to Gradle. This removes the earlier Groovy regression without claiming
  that BuildOpt accelerated the same native mechanism.
- Runtime Tuning profiles `W4_H6G` and `W3_H4G` both failed the accelerator rule.
  The strict `W3_H4G` run saved −512 ms (−4.3%) with a 95% interval from
  −2,818 to +1,302 ms. Only `STABLE_CONTROL` may be applied; the candidates are
  classified `NO_VALUE_NO_ACTION`.
- Build Impact clears the accelerator threshold for the bounded
  `UNRELATED_NON_CACHEABLE_WORK` class in both DSLs. Eight-pair means saved
  1,939 ms (76.0%) in Kotlin and 2,155 ms (73.5%) in Groovy; both lower 95%
  bounds are positive and every required output is identical.
- The exact reviewed-source `CUSTOM_TASK_CONTRACT_JAVA_V1` route clears the
  threshold in both DSLs. Eight-pair means saved 1,369 ms (67.3%) in Kotlin and
  2,349 ms (68.0%) in Groovy while all eight task outputs remained identical.
  This qualifies that exact Task Intelligence/Patch route, not every Patch
  Autopilot recipe. Agent discovery and hermetic enforcement remain unavailable.
- The combined public path was measured with unproven Safe Cache and Runtime
  profiles disabled. Against the same optimized native Gradle control it saved
  2,193 ms (77.5%) for Kotlin and 3,265 ms (84.1%) for Groovy in the unrelated
  non-cacheable-work class, and 1,441 ms (67.3%) for Kotlin and 1,905 ms
  (63.5%) for Groovy on the exact reviewed custom-task route. All four paired
  intervals had positive lower bounds, every required output was identical, and
  product-attributable failures remained zero. The Groovy custom-task cell keeps
  its one negative observation rather than filtering it out.

The final claim is deliberately `QUALIFIED_SYNTHETIC_WORKLOADS_ONLY`. These
percentages are the complete-path results and are not added to the earlier
mechanism percentages. The owner may now decide whether another POC phase should
broaden workload coverage; soak, design partners, HA, enterprise identity, and
other production work are not implied or required by this result.

Run the executable decision check with:

```bash
./dev/check-poc-value-validation
```
