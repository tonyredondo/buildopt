# Unseen substantial Gradle holdout

## Question

Can the unchanged generic `profile propose -> measure -> evaluate` path find
and qualify useful structural Build Impact on a substantial Gradle repository
that did not participate in the design of the mechanism?

## Frozen subject

Hibernate ORM is frozen at public revision
`2b448a59d332326f0cd0691c868425124d55cbb5`. It was selected before any
BuildOpt proposal or target-workflow timing because it is a substantial
multi-project Gradle build, requires JDK 25, and already enables Gradle's
parallel execution and build cache. A non-measured `help` preflight established
only that the frozen checkout and toolchain are runnable.

The repository-owned contract is:

- original workflow: root `assemble`;
- exact change: append one fixed comment to
  `hibernate-core/src/main/java/org/hibernate/Session.java`;
- required outputs: `hibernate-core/build/libs/**`;
- optimized-native control: the same workflow with daemon, build cache,
  parallel execution, Configuration Cache disabled, and 12 workers;
- candidate mechanism: generic structural Build Impact only.

No candidate entrypoint or graph reduction is preregistered. Discovery may
return a complete structural candidate or retain the native full graph.

## Unchanged value gate

An accepted candidate is measured over eight isolated alternating pairs. It
qualifies only when all eight pairs are positive, mean savings exceed both
500 ms and 2%, the paired 95% lower bound is positive, required outputs are
stable and byte-identical, no product-attributable failure occurs, and the
full native graph succeeds as the fallback.

An unsupported, uncertain, incomplete, or weak result retains optimized native
Gradle. Repository-specific product logic, post-result tuning, threshold
changes, discarded failed observations, and automatic activation are forbidden.
A general correctness defect may be fixed, but the same repository revision,
change, workflow, outputs, control, and gates must then be rerun from zero and
the failed attempt retained as evidence.

## Reproduce

The online runner packages the current committed BuildOpt revision, fetches
the frozen public source, and writes evidence outside the repository:

```bash
./dev/run-generic-holdout /absolute/evidence/directory
```

Validate a captured or committed bundle without network access:

```bash
./dev/check-generic-holdout /absolute/evidence/directory
```

This is POC evidence only. It does not authorize production use, automatic
activation, Test Optimization, soak testing, or design-partner work.
