# Optimized OpenTelemetry Spring-family experiment

This preregistered POC is the terminal validation of the optimizations derived
from the earlier OpenTelemetry Spring-family result. It uses the same pinned
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
The measured build outputs and native Gradle cache are not stored in this
state. The separate global-change probe must miss the state, retain all 53
control entrypoints with `IMPACT_GLOBAL_CHANGE`, and build successfully.

Protocol revision 2 was recorded after both unmeasured warm-ups completed and
before any observation began. The runner referenced the candidate pair log in
its hot-state assertion before assigning that path, so Bash stopped on an
unbound variable. Revision 2 moves the same assertion after both pair arms;
the implementation, control, candidate, measurement, outputs and value gate
are unchanged. The failed run contributed no accepted timing.

Qualification requires all of the following without moving thresholds or
discarding pairs: mean saving of at least 500 ms and 2%, a positive paired
bootstrap lower bound, four positive pairs, identical non-empty outputs, zero
product-attributable failures and successful safe fallback. Parity alone is
not value.

The experiment covers build-owned test preparation only. It does not select,
skip, shard or reorder tests, adds no repository-specific product rule, changes
no production authority, and requires no soak, design partner or public
release. A result is accepted only after this specification, runner and
checker have been committed before measurement.
