# BuildOpt POC One-Pager

_Current evidence as of 2026-08-11_

## Purpose

BuildOpt tests one strict product hypothesis: a repository-independent tool can
make substantial Gradle builds faster than an already optimized native Gradle
baseline by executing a smaller, change-specific project graph, without
changing the required outputs or weakening fallback behavior.

The current POC is intentionally narrow. It is not a production-readiness
exercise and it does not claim that every repository or change can be
optimized.

## Current POC Flow

The active workflow is:

```text
repository-owned task + exact Git change + required outputs
  -> buildopt profile propose
  -> buildopt profile measure
  -> buildopt profile evaluate
  -> explicit, review-required structural profile
  -> buildopt poc
```

The proposal logic uses project structure and source ownership rather than
repository names. Measurement compares the installed BuildOpt path with
optimized native Gradle through eight isolated alternating pairs. Evaluation
requires exact outputs, a positive timing result, repeatability, and a proven
full-graph fallback.

If discovery is incomplete, inputs drift, outputs differ, timing is weak, or a
fallback fails, BuildOpt retains the original native full graph.

## Current Comparable Evidence

All rows below use the same structural-only method. Times are wall-clock means
from eight alternating pairs and include the installed BuildOpt launcher and
profile overhead.

| Repository | Full -> selected projects | Optimized native | BuildOpt | Mean saving | Positive pairs | POC decision |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| Spring Framework | 27 -> 10 | 13.940 s | 11.438 s | **2.501 s / 17.94%** | 7/8 | Retain native: one -260 ms pair fails the frozen 8/8 repeatability rule. |
| OpenTelemetry Java Instrumentation | 1,024 -> 34 | 83.934 s | 71.825 s | **12.110 s / 14.43%** | 8/8 | Qualify. |
| Apache Kafka | 64 -> 3 | 82.498 s | 13.113 s | **69.385 s / 84.11%** | 8/8 | Qualify. |
| Micronaut Core | 75 -> 22 | 27.407 s | 15.968 s | **11.439 s / 41.74%** | 8/8 | Qualify. |
| Apache Groovy | 37 -> 2 | 75.064 s | 19.629 s | **55.434 s / 73.85%** | 8/8 | Qualify. |

Every accepted observation preserved the declared required outputs byte for
byte, completed the native full-graph fallback, and recorded zero
product-attributable failures. OpenTelemetry uses the separately
preregistered v4 correction because its earlier correctness fallback changed
scheduling and therefore could not be accepted; no rejected timing was reused.

The repository percentages are independent results. They are neither averaged
across repositories nor added to results from caching, JAR adapters, Edge, or
older compositions.

## What This Proves

1. **Generic structural optimization can beat optimized native Gradle.** The
   same repository-independent mechanism qualified on four materially
   different Gradle families.
2. **The value is end to end.** The measurements include discovery, launcher,
   validation, profile loading, and Gradle execution rather than timing only an
   internal BuildOpt phase.
3. **Graph reduction has a cascade effect.** Omitting unrelated projects also
   removes their configuration, task scheduling, cache lookup, source
   processing, compilation, and packaging work. This explains the larger
   Kafka, Micronaut, and Groovy gains.
4. **A smaller graph is not sufficient by itself.** Spring removed 17 projects
   and improved its mean, but BuildOpt still declined activation because the
   complete paired result missed the frozen repeatability gate.
5. **Fail-closed selection is practical.** Unsupported, incomplete, drifted,
   weak, or incorrect candidates return to native Gradle instead of converting
   uncertainty into a performance claim.

## Current Product Conclusion

The POC now has a defensible reason to continue: BuildOpt can create material
value beyond Gradle's native incremental and cache behavior by avoiding work
before Gradle executes the requested graph. The current evidence supports a
**review-required structural Build Impact POC**, not a universal automatic
optimizer.

Structural selection is the active acceleration mechanism. Safe Cache remains
at native-cache parity rather than being presented as a speed advantage.
Runtime Tuning, Hot State, and standard-Copy activation were removed after
terminal negative evidence. Historical JAR and Edge results remain useful
research evidence, but they are not part of this comparable five-repository
claim.

## What This Does Not Prove

- It does not guarantee savings for every repository, change, task, or runner.
- It does not establish production readiness, autonomous activation, HA,
  long-duration stability, or customer operations.
- It does not authorize skipping undeclared outputs or Test Optimization.
- It does not show that BuildOpt's cache is faster than a warm native Gradle
  cache.
- It does not justify relaxing the correctness or fallback gates to qualify
  more repositories.

## Next Steps

1. **Publish the proposal as a review-only CI artifact.** Implement
   `POC-GENERIC-PROFILE-CI-001` so an owner-declared workflow supplies the
   original Gradle task, exact Git change, and required outputs. CI should
   upload the generated proposal and native-fallback reason without activating
   it automatically.
2. **Exercise that CI flow on the existing public repositories.** Confirm that
   clean runners reproduce the same proposals and that unsupported or
   incomplete discovery produces an explicit native decision.
3. **Use one unseen substantial Gradle repository as a holdout.** Run the
   unchanged propose -> measure -> evaluate workflow without repository-specific
   code or post-result parameter tuning.
4. **Keep wall-clock value as the promotion boundary.** Continue only when the
   installed path beats optimized native Gradle, preserves every declared
   output, passes the repeatability rule, and proves full fallback.
5. **Reopen other mechanisms only from new attributable evidence.** Do not
   revive Runtime Tuning, Hot State, or Copy unless a materially different,
   generic trace exposes enough recoverable critical-path work to justify a
   preregistered experiment.

## Primary Evidence

- [Terminal five-repository structural matrix](../../benchmarks/results/poc-generic-profile-matrix-v3/README.md)
- [OpenTelemetry fallback-equivalence correction](../../benchmarks/results/poc-generic-profile-matrix-v4/README.md)
- [Detailed current performance findings](./build-optimization-performance.md)
- [Implementation tracker](../../implementation-tracker.md)
