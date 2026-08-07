# Clean OpenTelemetry optimization composition

This preregistered POC follows the full-path ablation that retained component
evidence but rejected the first OpenTelemetry composition. It uses the same
pinned `v2.30.0` source, 53-task optimized native Gradle control, 12-CPU
runner, cache seed, mutation, outputs, four alternating pairs, and frozen value
gate. It does not reuse or reinterpret any previous timing.

The candidate combines only two mechanisms:

- Build Impact replaces the broad Spring-family graph with the four typed
  compile producers required for the declared class outputs;
- the exact standard-`Jar` adapter makes only the unmodified
  `:testing-common:jar` producer eligible for Gradle's native cache.

Exact-bound hot-state reuse is deliberately absent. Every candidate invocation
loads and validates the manifest, graph, generated state, change set, Wrapper,
executable, and Gradle options before selecting work. Any hot-state diagnostic
is a protocol failure. This isolates whether the non-regressive composition
preserves the standard-`Jar` advantage without the arm that was 7.68% slower
in the fresh ablation.

One unmeasured candidate warm-up populates the candidate-only `Jar` cache
entry. Every measured candidate must report `:testing-common:jar FROM-CACHE`.
The control shares the checkout, Gradle user home, daemon, dependency cache,
and restored cache seed, but Gradle leaves that task non-cacheable by default.
No Managed L1, gateway, handshake, telemetry, or Runtime Tuning may start.

The separate `gradle.properties` probe must retain all 53 control entrypoints
with `IMPACT_GLOBAL_CHANGE` and build successfully. The `Jar` adapter may stay
enabled during fallback, but it cannot authorize a selective graph or alter a
required output.

Qualification requires an unchanged mean saving of at least 500 ms and 2%, a
positive paired bootstrap lower bound, four positive pairs, identical non-empty
outputs, zero product-attributable failures, and successful full-graph
fallback. No pair may be discarded and no threshold may move after timing.

The experiment covers build-owned test preparation only. It does not select,
skip, shard, retry, or reorder tests; widen managed caching; add a
repository-specific product rule; change production authority; or require a
soak, design partner, or public release. The specification, runner, and both
checkers must be committed before measurement.
