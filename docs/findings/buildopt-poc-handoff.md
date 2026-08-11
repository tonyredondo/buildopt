# BuildOpt POC: Current Evidence and Direction

## Executive Summary

- **BuildOpt is testing whether a generic decision layer can make substantial
  Gradle builds faster than optimized native Gradle.** Its current accelerator
  selects a smaller change-specific project graph before Gradle executes it,
  while preserving repository-declared outputs and the original full graph as
  the fallback.
- **The current evidence supports continuing the POC.** The same structural
  method qualified on OpenTelemetry, Kafka, Micronaut, and Groovy, reducing
  wall time by **14.43% to 84.11%**. Spring improved by **17.94%** and the unseen
  Hibernate holdout by **7.80%**, but both correctly retained native Gradle
  because one of eight pairs was slower.
- **Cross-repository CI reproducibility passes 5/5, and the blind transfer is
  now complete.** The read-only Action recreated every terminal proposal on
  clean hosted runners with zero graph drift. The unchanged generic path then
  discovered a 29-to-1 Hibernate candidate and saved 19.386 seconds on average;
  the strict 7/8 result remained native rather than manufacturing a win. A
  separate diagnostic recovered that first negative after better daemon
  stabilization, proving it was investigable rather than a structural reason
  to discard the candidate; a target-workload replay is now preregistered.

## The Project in One Minute

Gradle already provides Build Cache, Configuration Cache, incremental tasks,
up-to-date checks, parallel execution, and remote-cache integration. Those
features optimize work inside the task graph requested by the build.

BuildOpt explores an additional layer: given the original Gradle workflow, an
exact Git change, and the outputs the repository requires, can it prove that a
smaller graph is sufficient and materially faster? If the answer is uncertain
or the measured value is weak, BuildOpt runs the optimized native full graph.

The current POC flow is:

```text
repository-owned Gradle task + exact Git change + required outputs
  -> buildopt profile propose
  -> buildopt profile measure
  -> buildopt profile evaluate
  -> explicit review
  -> buildopt poc or optimized native Gradle
```

There are no repository-name rules and no automatic production activation.

## Mechanisms and Current Role

| Mechanism | What it does | Difference from native Gradle | Current POC decision |
| --- | --- | --- | --- |
| **Structural Build Impact** | Maps a change and required outputs to the smallest proven project/task entrypoint set. | Avoids configuring and executing unrelated parts of the requested graph; Gradle's incremental features normally act after the graph has been requested. | **Active accelerator.** This is the mechanism measured consistently across the six current public repositories. |
| **Profile measurement and evaluation** | Captures isolated paired timings, exact outputs, drift bindings, and fallback evidence before producing a profile. | Adds a cross-build evidence and activation policy rather than another Gradle execution optimization. | **Required safety layer.** Review remains explicit. |
| **Safe Cache / local L1** | Reuses verified outputs within repository, Wrapper, and platform boundaries. | Adds isolation and verification around native cache semantics. | **Not a speed differentiator.** It is at parity with a warm native Gradle cache and is not part of the current structural claim. |
| **Exact task optimization / Patch Autopilot** | Makes one exactly understood task shape reusable through a bounded adapter or reviewable patch. | Repairs or augments cacheability that the repository has not declared correctly. | **Promising but scoped research.** It must qualify independently for each generic task contract before joining the main path. |
| **Shared / Edge Cache** | Serves committed outputs from shared or nearer storage. | Adds controlled locality around Gradle's remote-cache protocol. | **Bounded supporting evidence.** Network-dependent results are kept separate from the structural matrix. |
| **Build History and launcher** | Records evidence, preserves process behavior, and applies bypass/fallback. | Provides orchestration and attribution, not avoided Gradle work. | **Supporting infrastructure.** Its overhead is included in installed-path measurements. |
| **Runtime Tuning, Hot State, standard Copy** | Previously tested worker/heap changes, plan reuse, and broader task adaptation. | Attempted to tune or reuse work beyond the retained structural path. | **Retired.** Terminal evidence was neutral, unstable, or regressive. |

## Current Wall-Time Evidence

The comparison baseline is optimized native Gradle using the same repository
revision, Gradle workflow, runner resources, required outputs, and applicable
native cache/parallel settings. Each row contains eight isolated alternating
pairs. BuildOpt time includes proposal consumption, validation, launcher, and
Gradle execution.

| Repository | Full -> selected projects | Native mean | BuildOpt mean | Mean saving | Positive pairs | Decision |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| Spring Framework | 27 -> 10 | 13.940 s | 11.438 s | **2.501 s / 17.94%** | 7/8 | Retain native: one -260 ms pair fails the frozen 8/8 repeatability gate. |
| OpenTelemetry Java Instrumentation | 1,024 -> 34 | 83.934 s | 71.825 s | **12.110 s / 14.43%** | 8/8 | Qualify. |
| Apache Kafka | 64 -> 3 | 82.498 s | 13.113 s | **69.385 s / 84.11%** | 8/8 | Qualify. |
| Micronaut Core | 75 -> 22 | 27.407 s | 15.968 s | **11.439 s / 41.74%** | 8/8 | Qualify. |
| Apache Groovy | 37 -> 2 | 75.064 s | 19.629 s | **55.434 s / 73.85%** | 8/8 | Qualify. |
| Hibernate ORM holdout | 29 -> 1 | 248.481 s | 229.095 s | **19.386 s / 7.80%** | 7/8 | Retain native: one -1.118 s pair fails the frozen 8/8 gate. |

Every accepted observation preserved the declared required outputs byte for
byte, completed a scheduling-equivalent native full-graph fallback, and
recorded zero product-attributable failures. OpenTelemetry uses its separately
preregistered v4 correction because the earlier fallback changed scheduling;
none of the rejected timing was reused.

Repository percentages are independent. They are not averaged across
repositories and are not added to cache, task-adapter, or Edge results.

### Hibernate is retained while its variance is investigated

The immutable v3 diagnostic did not replace the 7.80% v2 value result. It added
a second excluded base-revision warm-up and reran all eight pairs from zero.
The first pair changed from **−1.118 s to +11.883 s**, proving that the original
negative was recoverable. The complete v3 run reached only **4/8 positive
pairs**, **2.50% mean savings**, and an interval of **−6.604..+20.190 s** while
both arms continued accelerating. Native Gradle therefore remains active.

The next frozen replay warms the exact target workload, binds normalized task
and outcome fingerprints, and records interval-scoped Linux host pressure.
Those diagnostics may explain a failure but cannot remove a pair or relax the
8/8 gate.

## Clean-CI Reproducibility

Hosted run
[`31467370391`](https://github.com/tonyredondo/buildopt/actions/runs/31467370391)
executed the review-only Action from immutable revision `18caa8f` on five
independent Ubuntu runners. Every repository reproduced its exact owner input,
source change, proposal, project counts, manifest, graph, generated binding,
fallback input, and checksums. The result was **5/5 `MATCH`, 0 drift**, and zero
active profiles.

Spring's successful replay does not reverse its value decision: it confirms
the 27-to-10 proposal is reproducible, while the prior 7/8 wall-time result
still requires native Gradle. The other four value decisions remain bound to
their existing paired timings. No timing was executed during this replay.

## What the Tests Demonstrate

- **The idea transfers.** Four different public Gradle families qualified
  without repository-specific product logic, and the unseen Hibernate family
  produced a complete output-equivalent candidate without product changes.
- **Avoided work compounds.** Omitting projects also removes their
  configuration, scheduling, cache lookup, compilation, and packaging work;
  this explains the larger Kafka, Micronaut, and Groovy gains.
- **Correctness is necessary but not sufficient.** Spring preserved outputs
  and improved on average, yet retained native Gradle because the complete
  result missed the repeatability gate.
- **The gain survives product overhead.** Timings include BuildOpt validation,
  launcher, profile loading, and Gradle execution rather than an internal
  microbenchmark.
- **Owner output declarations remain a usability risk.** The first Hibernate
  attempt safely stopped before timing because its modules use `target` instead
  of Gradle's default `build` directory. The POC needs an earlier generic
  output-contract preflight before expensive paired measurement.

## Current Conclusion

BuildOpt now has a defensible POC value proposition: it can improve substantial
Gradle build wall time beyond native caching and incremental execution by
proving that less of the repository needs to run for a specific change and
declared output set.

The evidence does **not** support calling BuildOpt a universal optimizer yet.
The right product shape today is a review-required structural optimization
assistant that produces evidence, qualifies only repeatable wins, and otherwise
keeps native Gradle authoritative.

## Recommended Next Steps

1. **Make output ownership discoverable and fail early.** Derive or validate
   the repository's real Gradle output directories before warm-ups and paired
   measurement, retaining native when ownership remains ambiguous.
2. **Measure only reviewed candidates.** A CI proposal remains an observation,
   not value evidence. Run isolated paired measurement only after its graph,
   outputs and fallback are accepted.
3. **Broaden workflow coverage deliberately.** Test distinct customer build
   shapes such as compilation, packaging, verification, and build-owned test
   preparation. Test Optimization remains separate.
4. **Make onboarding repository-owned and keep wall time authoritative.** Users
   should provide a Gradle command, change source, and output contract—not
   hand-authored graphs. Promote only installed paths that preserve outputs,
   pass repeatability, prove fallback, and materially beat native Gradle.

## Scope and Evidence Sources

This is POC evidence, not production readiness, autonomous activation,
long-duration validation, customer operations, or a universal savings claim.
Test Optimization remains outside Build Optimization.

- [Terminal five-repository structural matrix](../../benchmarks/results/poc-generic-profile-matrix-v3/README.md)
- [OpenTelemetry fallback-equivalence correction](../../benchmarks/results/poc-generic-profile-matrix-v4/README.md)
- [Hosted review-only CI artifact run](https://github.com/tonyredondo/buildopt/actions/runs/31464264563)
- [Five-repository clean-CI replay](../../benchmarks/results/poc-generic-profile-ci-replay-v1/README.md)
- [Unseen Hibernate ORM holdout](../../benchmarks/results/poc-generic-holdout-v2/README.md)
- [Hibernate warm-up diagnosis](../../benchmarks/results/poc-generic-holdout-v3/README.md)
- [Detailed performance findings and historical research](./build-optimization-performance.md)
- [Implementation tracker](../../implementation-tracker.md)
