# Qualified full-path ablation v1

This POC experiment answers whether the coordinated BuildOpt path preserves
the value already observed in Spring Framework and OpenTelemetry Java
Instrumentation, and which qualified mechanism is responsible. It does not
turn every implemented feature on.

The matrix declares six logical arms: optimized native Gradle, Build Impact,
Impact plus exact hot state, qualified standard-task adapters, exact reviewed
task patches, and the complete qualified profile. A logical arm runs only when
the exact repository/workload contract authorizes it. Otherwise the result
records `NOT_AUTHORIZED_FOR_WORKLOAD`; absence is never represented as zero
cost or zero saving.

The experiment re-executes four frozen source protocols after this
preregistration is committed:

- Spring installed Build Impact;
- OpenTelemetry installed Build Impact;
- OpenTelemetry Build Impact plus exact hot state;
- OpenTelemetry Build Impact, exact hot state, and the standard `Jar` adapter.

Each executable candidate is compared only with the optimized native Gradle
control measured by the same protocol. Cross-protocol differences are
descriptive because the measurements are not one shared randomized runtime.
Percentages are never added. Spring has no separately qualified hot-state,
standard-task adapter, or reviewed patch for this workload. OpenTelemetry has
no exact reviewed task patch. The complete qualified profile is therefore
Build Impact on Spring and Impact plus hot state plus the `Jar` adapter on
OpenTelemetry.

The final POC claim requires both complete profiles to save at least 500 ms and
2%, have a positive paired lower bound, preserve non-empty byte-identical
required outputs, introduce zero product failures, and retain the frozen
native/full-graph fallback. A source-protocol failure stops the run without a
retry or discarded pair.

Run the fresh ablation from a clean committed checkout:

```bash
./dev/run-poc-full-path-ablation \
  /absolute/path/to/poc-full-path-ablation-v1
./dev/check-poc-full-path-ablation \
  /absolute/path/to/poc-full-path-ablation-v1
```

The output directory contains the four complete source reports plus the
aggregate `summary.json`. This is POC evidence only. It does not authorize
production rollout, Test Optimization, Runtime Tuning, strict Safe Cache,
Shared/Edge Cache, soak testing, or a universal savings claim.
