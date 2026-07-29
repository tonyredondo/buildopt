# Gradle correlation fixture

`F0-040A` provides the first executable repository for the bounded `SPK-001`
correlation experiment. It is intentionally independent from the product
plugin and uses only public Gradle APIs.

The fixture contains two subprojects, `alpha` and `beta`. Each registers the
same cacheable task implementation, input payload, and output property while
writing to a project-specific output path. A filesystem barrier requires both
task actions to start before either completes, proving that the initial miss
path ran concurrently. Because task identity and output location do not alter
the native cache key, both tasks must report the same key.

The executable check performs two otherwise identical Gradle 9.6.1 runs with
the real repository Wrapper:

1. an empty local build cache produces both equivalent outputs in parallel;
2. the next clean run reuses Configuration Cache and restores both tasks from
   the local build cache.

Run it with:

```bash
./dev/check-gradle-correlation-fixture
```

This fixture does not claim `taskExecutionId → cacheKey → PUT` correlation.
`SPK-001` owns that experiment, including Worker API, child-process,
failure/cancellation, and Gradle 8.14.x expansion. Its result may be exact
correlation or the safe all-attempt fallback.
