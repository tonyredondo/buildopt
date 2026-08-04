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

## Current decision

The current state is `CONTINUE_CONDITIONALLY`:

- Safe Cache is a proven safety enabler at native-cache parity, not a standalone
  accelerator.
- Runtime Tuning has a positive but sub-threshold preliminary result.
- Build Impact has a strong preliminary single-workload result and justifies broader testing.
- Task Intelligence and Patch Autopilot have preliminary reviewed-source evidence;
  Agent discovery and hermetic enforcement remain unavailable.
- The combined product path has not yet been measured against optimized native
  Gradle, so the overall value hypothesis remains open.

The active order is: reproduce the baseline and scorecard on the contractual
runner; either qualify or disable Runtime Tuning per workload; broaden Build
Impact and reviewed-source Task/Patch evidence; then benchmark the combined
public path and make the explicit `CONTINUE` or `STOP` decision.

Run the executable decision check with:

```bash
./dev/check-poc-value-validation
```
