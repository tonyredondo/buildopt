# Critical-path build-logic correction v1

This checked evidence closes `CPBLC-001..006` at the source gate. Five exact
public source trees are scanned against independently retained native
critical-path analyses. The analyses select 61 standard tasks above both 500 ms
and 2% of their build span; they are discovery inputs, not new timing samples.

The generic scanner reads 833 Gradle and conventional build-logic source files.
It finds six explicit state/cache opt-outs, all bound to non-material tasks:
Groovy's `testSingle` helper, Kafka's `copyTestXml`, OpenTelemetry's benchmark
tasks, and Spring's `eclipseSettings`. MockK has no matching opt-out. The result
is 5/5 conclusive families and 0/5 proposal families versus the required 3/5.

The failed breadth gate keeps candidate correctness, paired value, owner review,
and delivery closed. No Gradle build, public patch, new timing sample, product
failure, or speedup claim exists. The terminal decision is
`STOP_CRITICAL_PATH_BUILD_LOGIC_CORRECTION_NO_PROPOSAL_BREADTH`. This bounded
result does not assert that every possible build-logic optimization is absent.

Run `./dev/check-critical-path-build-logic-correction` to reconstruct the
material-task rows and terminal counts from committed evidence. Pass the five
`family=GIT_ROOT` bindings used by the source runner to additionally regenerate
every source-bound report from the exact Git trees.
