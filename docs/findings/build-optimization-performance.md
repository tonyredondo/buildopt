# Build Optimization Performance Findings and Next Steps

## Executive Summary

- **BuildOpt has demonstrated real build-time value, but not every component is
  an accelerator.** Build Impact and bounded task optimizations are the
  strongest mechanisms. They reduce work that Gradle would otherwise execute
  while preserving the required outputs.
- **Safe Cache is useful but is not currently a competitive advantage over a
  well-configured native Gradle cache.** It improves builds that did not have
  effective caching, but is effectively at parity with Gradle's native cache.
- **The Runtime Tuning profiles tested so far should remain disabled.** The
  strict comparison against optimized native Gradle regressed build time, so
  activating tuning would make the product worse rather than better.
- **The clean composition now qualifies.** After the full-path ablation rejected
  a profile containing a 7.68% hot-state regression, the fresh OpenTelemetry
  run removed hot state and combined only Build Impact with the exact standard
  `Jar` adapter. It saved **5,361.25 ms/50.40%**, with 4/4 positive pairs, a
  +4,334.25..+5,937-ms interval, 125 identical outputs, and safe full fallback.
- **Build Impact breadth is selective, not universal.** The latest Spring
  matrix qualified shared test preparation at **18.88% faster**, rejected leaf
  compilation and packaging under the unchanged gate, and retained full-graph
  execution for incomplete verification and distribution graphs.

The current POC supports a clear decision: continue investing in mechanisms
that avoid work or safely make expensive tasks reusable, while keeping neutral
or regressive mechanisms out of the active path.

## What Each Component Contributes

In this report, a positive result means lower wall-clock build time. Comparisons
use an optimized native Gradle control whenever that evidence is available.
Percentages from different rows must not be added because the mechanisms were
measured on different workloads and scopes.

| Component | What it does | Measured build-time effect | Current conclusion |
|---|---|---:|---|
| **Safe Cache / local L1** | Reuses verified outputs in a scope isolated by repository, Wrapper, and platform. | Against cache-off: **15.9% faster in Kotlin** and **13.7% faster in Groovy**. Against native Gradle cache: **0.02% faster in Kotlin** and **0.47% slower in Groovy**. | Useful when a repository has no effective cache, but **not an accelerator over native Gradle cache**. Strict Safe Cache remains explicit-only. |
| **Runtime Tuning** | Tests bounded worker, heap, and resource profiles intended to improve Gradle execution. | The latest candidate, `W3_H4G`, was **4.3% slower** (512 ms). The earlier `W4_H6G` candidate was **54.7% slower**. | **No current value. Disabled.** Optimized native Gradle remains the stable control. |
| **Build Impact** | Maps a change to the projects and tasks needed for the requested outputs, with full-graph fallback for unknown or global changes. | Synthetic coverage: **73.5-76.0% faster**. Installed Spring path: **15.76% faster**. Generalized Spring test preparation: **18.88% faster**; leaf compilation and packaging did not qualify. | **The strongest broadly useful accelerator currently demonstrated, but only for independently qualified scopes.** |
| **Task Intelligence** | Observes and qualifies tasks only when their inputs, outputs, cache keys, and outcomes are exact enough to support an optimization. | No general direct saving. In the accepted pilot it enabled a qualified native-cache restore that saved **203 ms** on average. | A **safety and eligibility layer**, not a standalone accelerator. Its value is realized through a qualified cache or patch route. |
| **Patch Autopilot / reviewed task patch** | Produces a reviewable and reversible patch that correctly declares inputs and outputs and enables caching for an exact custom-task shape. | Exact reviewed Java recipe: **67.3% faster in Kotlin** and **68.0% faster in Groovy**. Combined installed path: **63.5-67.3% faster**. | Highly promising for **specific reviewed task contracts**. The result must not be generalized to arbitrary tasks or recipes. |
| **Graph reduction** | Replaces broad aggregate task dependencies with the typed producers required for the declared outputs. | The OpenTelemetry experiment removed **3 graph nodes and 2 executed tasks** while preserving all 125 required outputs. No standalone wall-clock percentage is claimed. | Structurally valuable, but it still needs independent timing evidence before it can be presented as a separate accelerator. |
| **Exact-bound hot-state reuse** | Reuses a previously validated Build Impact plan only when repository revision, graph, manifest, changes, Wrapper, executable, and options still match. | Reduced BuildOpt preparation from **74.97 ms to 40.34 ms**, but the fresh whole-build arm was **7.68% slower**. | **Disabled for the measured profile.** Micro-overhead reduction does not override regressive end-to-end evidence. |
| **Standard `Jar` cache adapter** | Makes only an unmodified standard Gradle `Jar` task eligible for native caching; custom archives, `Copy`, arbitrary tasks, and `Test` remain unchanged. | Before the adapter, the OpenTelemetry candidate was **1.94% slower**. After the adapter it was **39.92% faster**, saving 4,376.75 ms with 4/4 positive pairs and 125 identical outputs. | **The strongest result on a substantial public repository.** It proves the value of precise standard-task adapters, not a universal policy for every archive task. |
| **Shared and Edge Cache** | Reuse committed outputs across machines and optionally place them nearer developers or CI runners. | No defensible build-time percentage has yet been established against Gradle's native remote-cache support. | Functionally implemented, but **performance value remains unproven**. It needs a controlled network and locality benchmark before further investment. |
| **Build History** | Records durations, outcomes, cache behavior, and applied optimizations so results can be inspected and compared. | **No direct build-time saving.** | Observability that helps discover and validate optimizations; not an accelerator itself. |
| **Launcher, gateway, and telemetry** | Provide orchestration, authentication, evidence collection, and safe fallback behavior. | Add fixed overhead rather than saving work. The local-cache fast path avoids starting these components when they have no consumer. | Necessary infrastructure for instrumented flows, but it must remain off the critical path when it is not needed. |
| **Combined qualified path** | Orchestrates Build Impact and exact task optimizations through the packaged CLI and plugin while leaving unproven mechanisms disabled. | Fresh Spring Build Impact: **30.86% faster**. Clean OpenTelemetry Impact + standard `Jar`, with hot state absent: **50.40% faster**, saving 5,361.25 ms with 4/4 positive pairs. | **Qualified for the exact measured POC workloads.** Generalize change shapes and outputs before broadening the claim. |

The early isolated Runtime Tuning experiment reported a 0.7% saving, but the
later and stricter comparison against optimized native Gradle superseded that
result: the selected candidate regressed by 4.3%. The current activation
decision correctly follows the stronger evidence.

## What the Evidence Actually Proves

### The clearest value comes from avoiding work

Build Impact wins because it changes the amount of work Gradle must execute.
The Spring evidence remains positive after including package installation,
launcher startup, manifest loading, graph validation, and fallback machinery.
That makes its **15.76% installed-path saving** more decision-useful than a
larger synthetic-only percentage.

### Precise task adapters can unlock large gains

The OpenTelemetry investigation initially ended with BuildOpt **1.94% slower**
than native Gradle. Task attribution identified a repeated non-cacheable
standard `Jar` task as the dominant tail. Adding one conservative adapter for
that exact Gradle task type changed the result to **39.92% faster**, with a
positive paired interval and identical outputs.

The important lesson is not that every `Jar` should be changed. The lesson is
that profiling a real build, finding a repeated expensive tail, and adding a
small exact adapter can produce more value than applying broad configuration
changes.

### Cache safety and cache speed are different value propositions

Safe Cache protects isolation and publication boundaries. That is valuable,
but the current data shows that it does not materially outperform the native
Gradle cache engine. BuildOpt should not claim a speed advantage where the
underlying mechanism is the same. The default should continue to use native
Gradle caching unless the stricter security behavior is explicitly required.

### Runtime Tuning is not ready

The tested resource profiles did not beat an optimized native Gradle control.
Enabling them would contaminate the gains from Build Impact and task adapters.
Runtime Tuning should remain a research track with a hard activation rule: no
profile is applied unless it produces repeatable incremental value for the
current workload class.

### The terminal result must not hide a regressive component

The fresh ablation retained all four source reports. Spring Build Impact saved
2,492.375 ms/30.86% with 8/8 positive pairs. OpenTelemetry Build Impact alone
had a favorable 985.5-ms/7.49% mean but did not qualify; adding hot state made
the build 892 ms/7.68% slower. The terminal arm with the standard `Jar`
adapter saved 4,496.75 ms/40.60%, but the complete composition was rejected
because it still included that regressive hot-state mechanism.

This is exactly why mechanisms are ablated independently. The clean rerun then
removed hot state and improved the same OpenTelemetry workload by 5,361.25 ms
or 50.40%, with 4/4 positive pairs and a strictly positive interval. The result
qualifies the clean composition for this exact workload; it does not rehabilitate
hot state or authorize a universal claim.

## Recommended Direction

### 1. Generalize the qualified clean profile

The clean profile now contains only the remaining non-regressive mechanisms:

- native Gradle local cache as the cache baseline;
- Build Impact with full-graph fallback;
- exact standard-task adapters such as the current `Jar` adapter;
- reviewed task patches only when their contract matches exactly;
- output equivalence, failure attribution, and immediate bypass throughout.

It must continue to exclude Runtime Tuning, strict Safe Cache, hot state, and
Edge Cache until each demonstrates incremental value. Generalization should now
cover leaf, shared, build-logic, and global changes plus compilation,
test-preparation, verification, packaging, and distribution outputs. This is
**unified orchestration**, not "turn every feature on."

### 2. Keep the same native control while broadening workload coverage

A single native-versus-everything comparison still does not prove transfer.
For every new change/output cell, retain these attributable arms:

1. optimized native Gradle;
2. native Gradle plus Build Impact;
3. Build Impact plus the qualified standard `Jar` adapter;
4. the same candidate with native/full-graph fallback probes.

Use Spring Framework and OpenTelemetry first because they already have stable
protocols and meaningful build durations. Add a third repository only after an
unchanged profile exercises a genuinely new workload class. This does not
authorize test selection or skipping test execution; Test Optimization remains
separate.

Every arm should retain all signed observations and require:

- at least 500 ms and 2% mean saving for an acceleration claim;
- a positive paired 95% lower bound;
- byte-identical required outputs;
- zero product-attributable failures;
- a tested full-graph or native fallback;
- no hidden change to the control's Gradle cache, daemon, parallelism, or
  Configuration Cache settings.

### 3. Generalize Build Impact before expanding the product surface

Build Impact is the best candidate for broader POC value. The next iterations
should cover more real change shapes and requested outputs:

- leaf, shared-library, build-logic, dependency, and global configuration
  changes;
- compilation, test preparation, verification, packaging, and distribution
  entrypoints;
- multi-project Kotlin and Groovy builds with different plugin families;
- unknown changes that must prove they restore the complete original graph.

The completed matrix confirms that distinction. Shared test preparation saved
2,638 ms/18.88%, with 4/4 positive pairs and a +1,516..+3,275.5-ms interval.
Leaf compilation saved only 196.25 ms/1.33% with a negative lower bound, and
leaf packaging saved 427.25 ms/3.73% with only 2/4 positive pairs. Verification
and source distribution reported incomplete graphs and retained exact-output
full-graph execution. Therefore only the test-preparation cell is generalized;
all other measured cells remain native or full graph.

### 4. Stop normal-build adapter expansion when the measured tail is exhausted

The retained Spring and OpenTelemetry traces have now been re-evaluated under
the same evidence gate. They do not justify another adapter: standard `Jar` is
already qualified; standard `Copy` failed its direct incremental gate; custom
`ShadowJar` is below the 500-ms floor and already restored from native cache;
and configured `JavaExec` is below the floor with external-process effects too
broad for a generic adapter. Spring produced no additional exact standard-task
candidate.

This is a bounded stop for the current traces, not a claim that every possible
normal-build optimization has been discovered. A new adapter should reopen
only when a materially different workload exposes a dominant, exact,
unmodified task with bounded effects and independent incremental value.

### 5. Keep Runtime Tuning as targeted research

Do not spend the next cycle building a general automatic tuner. First identify
workloads where the native profile shows a measurable resource bottleneck:
worker starvation, heap pressure, garbage collection, queueing, or daemon
startup. Test one hypothesis at a time and retain `STABLE_CONTROL` whenever the
paired interval crosses zero or the gain is operationally insignificant.

Potential research areas are:

- workload-specific worker and heap selection rather than one global profile;
- daemon and JVM warm-up cost for CI and short-lived builds;
- configuration-phase and Kotlin DSL compilation overhead;
- dependency resolution and artifact-transform reuse;
- compiler avoidance and incremental compilation boundaries.

### 6. Benchmark Shared and Edge Cache only against the native alternative

Shared and Edge Cache should not be prioritized until the experiment can model
a real remote-cache decision. Compare them with Gradle's native remote cache
under controlled latency, bandwidth, object size, hit rate, and runner locality.
Measure both the full build and cache-service overhead. If BuildOpt cannot
improve end-to-end time or provide a required safety property at acceptable
cost, defer the component.

### 7. Use real repositories with substantial builds

Short two-second repositories are useful for compatibility but poor for
performance decisions because startup noise dominates. Performance expansion
should use public repositories with:

- a stable offline or dependency-pinned workload;
- enough work for the candidate mechanism to matter;
- deterministic, hashable outputs;
- repeatable mutations and entrypoints;
- no repository-specific rule added to BuildOpt merely to win the benchmark.

Spring Framework and OpenTelemetry satisfy enough of these conditions to remain
the primary POC laboratories. A third repository should be added only when it
exercises a different workload class or invalidates a current assumption.

## Recommended Activation Policy

| Mechanism | Experimental profile | Default path | Required next proof |
|---|---:|---:|---|
| Native Gradle cache | Enabled | Enabled | Continue parity and output checks |
| Build Impact | Enabled for qualified scopes | Explicit command | Generalize across real change and output scopes |
| Hot-state reuse | Disabled for the measured profile | Disabled | Beat the same end-to-end control independently before reconsideration |
| Standard `Jar` adapter | Enabled for the exact standard task type | Not universal | Replicate on another substantial public repository |
| Reviewed task patch | Enabled only for exact matching contracts | Review required | Add recipes only after independent qualification |
| Strict Safe Cache | Disabled | Disabled | Beat or justify cost versus native Gradle cache |
| Runtime Tuning | Disabled | Disabled | Positive incremental evidence against optimized native Gradle |
| Shared / Edge Cache | Disabled for performance claims | Operator opt-in | Controlled native-remote-cache comparison |

## Recommended Next Block

The next implementation block is **build-owned test-build optimization**:

1. profile test compilation, resource processing, fixture preparation, and
   packaging on the fixed substantial public repositories;
2. select one build-owned bottleneck from measured wall-clock evidence;
3. preserve every requested `Test` task, test selection, order, retry, shard,
   and outcome;
4. measure the candidate independently and as part of the complete qualified
   profile against optimized native Gradle;
5. retain native execution whenever the unchanged 500-ms/2%/positive-bound
   gate or exact-output/test-outcome checks fail.

The task-tail review is complete with zero actionable normal-build candidates.
This next block explores a different source of repeated build work without
crossing into Test Optimization.

## Open Questions

- Does Build Impact remain positive across full `build`, packaging, and
  verification entrypoints, not only selected test-preparation outputs?
- Can Runtime Tuning find a workload-specific win without a long search cost or
  an unstable profile?
- Does Shared or Edge Cache improve end-to-end time over a native Gradle remote
  cache under realistic network conditions?
- How much of the combined gain remains when the same qualified profile is
  transferred unchanged to a third substantial public repository?

## Evidence Boundaries

- This is proof-of-concept evidence, not a universal savings or production
  readiness claim.
- Synthetic results validate mechanisms; Spring and OpenTelemetry results are
  more representative but remain repository- and workload-specific.
- Percentages from different experiments are not additive.
- All qualified experiments preserve their declared outputs and safe fallback.
- Test Optimization, long-duration soak testing, design partners, HA, and
  production operations are outside this evaluation.

The detailed measurements and validation commands are available in the
[benchmark index](../../benchmarks/README.md). The most relevant checked
artifacts are:

- [negative Safe Cache and Runtime Tuning decision](../../benchmarks/results/poc-value-negative-mechanisms-v1.json);
- [Build Impact and reviewed task coverage](../../benchmarks/results/poc-value-coverage-v1.json);
- [combined public-path evidence](../../benchmarks/results/poc-value-combined-v1.json);
- [installed Spring Build Impact evidence](../../benchmarks/results/poc-spring-installed-impact-v1.json);
- [OpenTelemetry hot-state evidence](../../benchmarks/results/poc-otel-hot-state-v1.json);
- [final OpenTelemetry optimization evidence](../../benchmarks/results/poc-otel-optimization-v2.json).
- [fresh full-path ablation and retained component evidence](../../benchmarks/results/poc-full-path-ablation-v1/summary.json).
- [qualified clean OpenTelemetry composition](../../benchmarks/results/poc-otel-clean-composition-v1.json).

Validate the current scorecard with:

```bash
./dev/check-build-optimization-performance
./dev/check-poc-value-validation
```
