# Gradle correlation fixture

`F0-040A` provides the independent repository used by the bounded `SPK-001`
correlation experiment. The fixture tasks use public Gradle APIs; the spike
analyzer is deliberately version-pinned to Gradle's internal build-operation
trace.

The fixture contains two subprojects, `alpha` and `beta`. Each registers the
same cacheable task implementation, input payload, and output property while
writing to a project-specific output path. A filesystem barrier requires both
task actions to start before either completes, proving that the initial miss
path ran concurrently. Because task identity and output location do not alter
the native cache key, both tasks must report the same key.

The original executable check performs two otherwise identical Gradle 9.6.1
runs with the real repository Wrapper:

1. an empty local build cache produces both equivalent outputs in parallel;
2. the next clean run reuses Configuration Cache and restores both tasks from
   the local build cache.

Run it with:

```bash
./dev/check-gradle-correlation-fixture
```

The original `F0-040A` check alone does not claim
`taskExecutionId → cacheKey → PUT` correlation. That bounded question is closed by
[`SPK-001`](../../specs/gradle-correlation-v1.md).

The expanded matrix adds Worker API no-isolation and process-isolation tasks, a
real child JVM, ordinary failure, cancellation, a loopback HTTP Build Cache,
and Gradle 8.14.3. Its analyzer structurally walks build-operation parent IDs
and checks the resulting remote stores against actual HTTP `PUT` requests:

```bash
./dev/run -- ./dev/check-gradle-correlation-spike --full
```

Task-owned stores correlate exactly, including the Worker and child-process
paths. Cold Kotlin DSL builds also emit remote stores for script compilation
and accessor generation that have no task ancestor. The supported result is
therefore `UNAVAILABLE`: any such `UNATTRIBUTED` event aborts the complete
attempt. Warming shared Gradle state is intentionally excluded from acceptance
because it can hide those stores.
