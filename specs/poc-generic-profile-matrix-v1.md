# Five-repository generic structural profile matrix

This POC measures the same installed `profile propose -> profile measure ->
profile evaluate` path on Spring Framework, OpenTelemetry Java
Instrumentation, Apache Kafka, Micronaut Core, and Apache Groovy.

The candidate contains only Build Impact. The control is each repository's
declared optimized-native Gradle workflow. Both arms use independent checkouts,
Gradle homes, and native-cache seeds. Eight temporally paired observations
alternate order, compare the required outputs byte for byte, and prove the
native full-graph fallback.

OpenTelemetry's real Spring-family workflow contains 53 qualified Gradle task
paths. The generic proposal surface therefore accepts repeated `--entrypoint`
values and maps their terminal task selectors to the exact project owners of
the change. This is a repository-independent extension: it neither recognizes
OpenTelemetry nor contains any repository-specific selection rule.

The matrix deliberately separates two questions:

1. **Structural-only comparison:** the five fresh rows produced by this
   contract are methodologically comparable.
2. **Retained full compositions:** older Jar or Edge combinations remain useful
   product evidence, but are shown separately and must not be averaged with or
   attributed to Build Impact alone.

Graph reduction is a hypothesis, not a speedup claim. A repository qualifies
only when all eight pairs are positive, mean savings exceed 500 ms and 2%, the
paired lower bound is positive, required outputs are stable and identical, and
fallback succeeds. Any unsupported workflow, failed build, timeout, output
drift, or weak result retains native Gradle.

This is POC evidence only. It does not authorize automatic activation,
production rollout, Test Optimization, soak testing, or design-partner work.
