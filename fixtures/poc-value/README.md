# POC value workloads

These fixtures support the bounded `POC-VALUE-003` comparison against an
optimized native Gradle control. Each fixture contains equivalent Kotlin and
Groovy DSL entrypoints; the runner retains only the selected pair.

- `build-impact` models a changed library and affected service while a second,
  unrelated service owns deterministic work that is explicitly non-cacheable.
  Build Impact may omit only that unrelated work and must preserve the affected
  JAR plus the independent Test-owned marker.
- `reviewed-task` contains the exact reviewed Java preimage accepted by
  `CUSTOM_TASK_CONTRACT_JAVA_V1`. The candidate is the exact registered
  postimage, making eight deterministic tasks cacheable; the optimized native
  control keeps Build Cache, Configuration Cache, parallelism, and the daemon
  enabled but cannot reuse the unqualified task.

The fixtures are synthetic POC workloads, not customer or production evidence.
