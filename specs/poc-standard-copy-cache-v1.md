# Standard Copy task-tail adapter POC

## Purpose

This experiment selects the next task adapter from existing task-attribution
evidence instead of from a generic Gradle task catalog. On the pinned
OpenTelemetry Spring-family workload, the remaining dominant task after the
already qualified `:testing-common:jar` is
`:testing:dependencies-shaded-for-testing:extractShadowJar`. The observed
native task took 2,807 ms.

Runtime inspection before product implementation proved that the selected task
has all of these properties on Gradle 9.6.1:

- concrete class `org.gradle.api.tasks.Copy_Decorated`;
- direct superclass `org.gradle.api.tasks.Copy`;
- exactly one `org.gradle.api.internal.project.taskfactory.StandardTaskAction`;
- destination `build/extracted/shadow`, below the owning project's build
  directory; and
- no custom task action.

An uncommitted feasibility probe made only that exact task cache-eligible. The
first invocation executed it and the second restored it `FROM-CACHE`. The
probe is not accepted performance evidence.

## Selection boundary

The adapter may grant native build-cache eligibility only when all frozen
runtime checks match:

- Gradle is exactly 9.6.1;
- concrete class, direct superclass, action count, and action class match the
  values above;
- the destination is non-empty and remains below the owning project's build
  directory; and
- Gradle's own cache key, input snapshot, output snapshot, enabled state, and
  cache-disable reasons permit the store or load.

Custom `Copy` subclasses, `Sync`, tasks with `doFirst` or `doLast`, destinations
outside the project build directory, custom archive tasks, `Test`, and every
other task remain unchanged. A mismatch is a native Gradle fallback, not an
error and not a widened policy. The adapter is explicit, removable through
`BUILDOPT_BYPASS=1`, and incompatible with `--no-build-cache`.

## Preregistered comparisons

All accepted timing uses OpenTelemetry Java Instrumentation v2.30.0 at the
fixed revision, Temurin 21, Gradle 9.6.1, 12 workers, offline dependency
resolution, one shared hot Gradle runtime per comparison, four alternating
pairs, a restored native-cache seed, and the installed BuildOpt package.

Three comparisons answer different questions and must not be merged by adding
percentages:

1. `COPY_ADAPTER_ONLY`: optimized native Gradle against the installed full
   graph with only the exact standard-`Copy` adapter enabled.
2. `INCREMENTAL_COPY_ON_QUALIFIED_PROFILE`: the already qualified Build Impact
   plus standard-`Jar` profile against the same profile plus the standard-`Copy`
   adapter.
3. `COMPLETE_PROFILE_VS_NATIVE`: optimized native Gradle against Build Impact
   plus both exact task adapters. This is the end-to-end cascade measurement;
   it captures configuration, graph reduction, task execution, and critical
   path effects directly.

The native control keeps upstream build cache, parallel execution, and
Configuration Cache. Every arm uses the same checkout, mutation, entrypoints,
JDK, Wrapper, options, cache seed, and daemon state. Candidate-only warm-ups
may populate entries that native Gradle cannot create because the task is not
cache-eligible. Warm-ups are never timing observations.

Required evidence includes byte-identical, non-empty manifests for the
selected class output, the standard Jar, and the extracted Copy tree. Every
candidate must report the expected `FROM-CACHE` hit. The full-graph fallback
must retain all 53 entrypoints for a global change.

Each acceleration claim independently requires at least 500 ms and 2% mean
saving, four positive pairs, a positive deterministic paired-bootstrap 95%
lower bound, exact required outputs, and zero product-attributable failures.
No failed or unfavorable pair may be discarded and thresholds cannot move
after timing.

## POC boundary

This work tests whether one exact standard task adapter adds incremental and
cumulative value. It does not authorize a general `Copy` policy, repository-
specific task names in product code, Hot State, Runtime Tuning, Safe/Shared/
Edge Cache, Test Optimization, production rollout, soak testing, or design-
partner work.
