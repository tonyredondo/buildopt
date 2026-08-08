# Installed qualified-profile matrix protocol

This protocol tests whether the public `buildopt poc` path retains value on
three fixed public repositories when every repository enables only mechanisms
that already qualified for that exact workload:

- Spring Framework: Build Impact only;
- OpenTelemetry Java Instrumentation: Build Impact plus the exact standard-Jar
  adapter;
- Apache Kafka: Build Impact plus read-only Edge locality.

Each cell keeps its previously fixed revision, mutation, optimized native
Gradle control, output contract, pair order and fallback. A fresh package from
the measured BuildOpt revision must be used. Spring and OpenTelemetry use the
repository-owned profile through `buildopt poc`; Kafka uses the installed v2
profile and its independently modeled Shared/Edge boundary.

## Value gate

Every repository is evaluated independently. A cell qualifies only when it
saves at least 500 ms and 2%, has a positive paired 95% lower bound, has no
negative pair, preserves byte-identical non-empty required outputs and records
zero product-attributable failures. Thresholds are immutable after the first
accepted observation and failed pairs may not be discarded.

The matrix permits broad continuation only when at least two independent
repository families qualify. It never averages repository percentages or adds
mechanism effects. A failed cell retains optimized native Gradle for that
scope and narrows the claim.

## Execution

After committing this preregistration, run:

```bash
./dev/run-poc-qualified-profile-matrix-v1 \
  /absolute/path/to/result-directory \
  /absolute/path/to/installed/buildopt \
  /absolute/path/to/kafka-source.tar.gz
./dev/check-poc-qualified-profile-matrix-v1-result \
  /absolute/path/to/result-directory/summary.json
```

The runner retains the three full cell documents next to `summary.json`.

## Boundaries

This is POC evidence for the fixed revisions and scopes. It is not a production
readiness, soak, design-partner, universal-repository or Test Optimization
claim.
