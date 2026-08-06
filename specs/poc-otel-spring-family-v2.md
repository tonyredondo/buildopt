# OpenTelemetry Spring-family transfer experiment

This preregistered POC starts a new experiment on the stable OpenTelemetry Java
Instrumentation `v2.30.0` release. It does not retry or reinterpret the earlier
terminal experiment: that older revision, root control and incomplete result
remain unchanged.

The first compatibility probe rejected root `testClasses` after it exhausted a
fixed 20-minute preflight budget. No timing from that probe is evidence. The
replacement control is the repository's Spring instrumentation family, a
boundary already distinguished by the upstream pull-request workflow. Gradle
exposes 53 `testClasses` tasks under `:instrumentation:spring`; their exact,
sorted names are checked in separately. A successful offline preflight reached
340 actionable tasks in 2m45s without executing a Gradle `Test` task.

BuildOpt discovery sees 1,024 projects. The 53 control entrypoints reach 148
projects, while the unchanged Spring-autoconfigure alternative reaches 46.
Both graphs are complete, contain no test-execution tasks and have no unknown
relationships.

Protocol revision 3 was recorded before warmups or accepted observations. The
first runner materialized different alternative/artifact IDs and an extra
global path from the manifest used to preregister the graph, so byte-level
graph validation stopped it after preflight. Revision 3 restores the exact
preregistered manifest metadata. It also records the existing fail-safe reason
for `gradle.properties` precisely as `IMPACT_UNKNOWN_CHANGE_PATH`; execution
still retains all 53 entrypoints. Control, candidate, measurement and value
gate are unchanged.

Protocol revision 4 was recorded after one control warmup completed and the
candidate stopped before starting Gradle. Zero complete warmup pairs and zero
measured observations were accepted. The shared repository-file reader had
applied the 256-KiB manifest limit to the 435,875-byte declared graph even
though the graph parser already has its own 2-MiB bound. Revision 4 passes each
artifact's existing bound to the shared reader and gives failures an accurate
artifact name. It does not increase any parser limit or change the control,
candidate, outputs, measurement, resources, pair order or value gate.

The accepted measurement will use an installed package and four alternating
pairs. Both arms receive the same offline dependencies, native-cache seed,
fixed source mutation, clean outputs, 12 workers and Gradle optimizations. The
candidate must reproduce every byte below the declared autoconfigure class
output root. A separate `gradle.properties` change must restore all 53 control
entrypoints and makes no performance claim.

Qualification still requires at least 500 ms and 2% mean saving, a positive
paired-bootstrap lower bound, four positive pairs, identical non-empty outputs
and zero product failures. Failed observations cannot be discarded and the
thresholds cannot move.

This experiment covers build-owned test preparation only. It does not select,
skip, shard or reorder tests; it changes no production authority, requires no
public release, soak or design partner, and makes no production-readiness
claim.
