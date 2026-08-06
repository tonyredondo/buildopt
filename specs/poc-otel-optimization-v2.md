# Standard-Jar optimized OpenTelemetry experiment

This preregistered POC is a new terminal validation after the failed optimized
OpenTelemetry result. It uses the same pinned
`v2.30.0` source, 53-task native Gradle control, runner, cache seed, mutation,
outputs, four alternating pairs and frozen value gate. It does not reuse or
reinterpret any previous timing.

The candidate replaces the aggregate `testClasses` entrypoint for the changed
Spring-autoconfigure project with the four typed compile producers that create
the required class output. The generic discovery adapter recognizes these by
Gradle task type rather than repository-specific task names. The complete
1,024-project graph reaches 148 projects for the control and 46 for every
candidate producer, contains no `Test` task and has no unknown relationship.

The candidate also enables an exact-bound hot state. The first candidate
warm-up validates and stores the Build Impact plan. Every measured candidate
arm must reuse it only when repository revision, change set, manifest, graph,
generated state, Wrapper, executable and Gradle options are byte-identical.
The candidate explicitly enables native-cache eligibility for the exact
unmodified `:testing-common:jar` producer. One unmeasured candidate warm-up
populates that entry; all measured candidate arms must report `FROM-CACHE`.
The native control shares the same checkout, Gradle user home and daemon but
cannot query that entry because Gradle leaves the task non-cacheable by
default. This removes daemon/startup imbalance without changing control work.
No Managed L1, gateway, handshake or telemetry may start.

The separate global-change probe must miss the exact hot state, retain all 53
control entrypoints with `IMPACT_GLOBAL_CHANGE`, and build successfully. The
Jar adapter may remain enabled during fallback, but it cannot authorize a
selective graph or change any required output.

Qualification requires all of the following without moving thresholds or
discarding pairs: mean saving of at least 500 ms and 2%, a positive paired
bootstrap lower bound, four positive pairs, identical non-empty outputs, zero
product-attributable failures and successful safe fallback. Parity alone is
not value.

The experiment covers build-owned test preparation only. It does not select,
skip, shard or reorder tests, widen managed Tier 1 caching, add a
repository-specific product rule, change production authority, or require a
soak, design partner or public release. A result is accepted only after this
specification, runner and checker have been committed before measurement.
