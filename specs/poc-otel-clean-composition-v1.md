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

Protocol revision 2 assigns packaging its own private temporary Gradle home.
The first attempt inherited a read-only global Gradle home and stopped while
building the local package, before the source archive was downloaded and
before any preflight, warm-up, or observation. The product, control, candidate,
outputs, ordering, and value gate are unchanged; zero timing was accepted.

Protocol revision 3 removes an orphaned `--repository-revision` argument from
the candidate and fallback runner invocations. The second attempt completed
source preparation, discovery, and the control warm-up, then stopped before the
candidate warm-up because revision 2 passed the hot-state revision without the
required hot-state directory. That is an invalid CLI pair, not a product defect
or performance result. No measured observation ran or was accepted. The
product, control, candidate mechanisms, outputs, ordering, and value gate remain
unchanged, and the complete experiment must restart from fresh temporary state.

## Result

The complete revision-3 execution qualifies the clean composition. Optimized
native Gradle averaged 10,636.5 ms and installed BuildOpt averaged 5,275.25 ms,
saving 5,361.25 ms or 50.40%. Pair savings were +3,825, +5,862, +5,995, and
+5,763 ms; the deterministic paired interval is +4,334.25..+5,937 ms. Every
candidate restored `:testing-common:jar FROM-CACHE`, enabled no hot state or
managed runtime, and preserved the same non-empty 125-file output digest. Zero
product-attributable failures occurred. The global-change probe restored all 53
control entrypoints and completed successfully.

The terminal decision is `QUALIFY_CLEAN_OTEL_COMPOSITION`. Qualification is
limited to this exact POC workload and does not authorize Hot State, production
rollout, Test Optimization, or a universal savings claim.
