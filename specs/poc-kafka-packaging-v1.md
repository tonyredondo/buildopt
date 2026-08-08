# Kafka packaging value experiment

This experiment asks whether the installed, repository-owned BuildOpt POC
profile can beat optimized native Gradle for a real Kafka packaging change.
It does not treat graph reduction as value by itself: the candidate must reduce
wall-clock time while producing the exact client JAR and retaining native
full-graph fallback.

## Frozen comparison

The fixed Apache Kafka 4.3.1 revision receives one comment-only change in
`Metadata.java`. The optimized native control runs the repository's root
`assemble` entrypoint. The installed BuildOpt candidate consumes the committed
manifest, complete 64-project graph and qualified profile, then selects
`:clients:jar` across three projects. Only Build Impact and the exact standard
`Jar` adapter are enabled.

Dependency hydration, daemon start-up and one warm-up per arm happen before
measurement. The four accepted pairs alternate their starting arm, use the
same JDK, Gradle user home, daemon, dependency state and native build cache,
and run offline. Each observation removes generated project outputs before the
arm starts. The measured timer encloses the public command and its complete
Gradle child.

## Acceptance boundary

All four pairs must favor BuildOpt. Mean savings must exceed both 500 ms and
2%, the paired lower bound must be positive, the client JAR must be byte
identical, and no product failure may occur. A `gradle.properties` change must
still select and complete the native `assemble` fallback outside the candidate
graph.

Passing qualifies only this fixed Kafka client-packaging change shape. Failing
retains the existing qualified OpenTelemetry scope and the already proved
Kafka test-preparation scope. Neither result authorizes production rollout,
Test Optimization, or a broader packaging claim.
