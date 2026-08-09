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
  latest trace-selected Spring experiment changed only the worker cap from 12
  to 6 and made the build **191.5 ms/2.00% slower**. Earlier synthetic profiles
  also regressed, so activating tuning would make the product worse rather
  than better.
- **The clean composition now qualifies.** After the full-path ablation rejected
  a profile containing a 7.68% hot-state regression, the fresh OpenTelemetry
  run removed hot state and combined only Build Impact with the exact standard
  `Jar` adapter. It saved **5,361.25 ms/50.40%**, with 4/4 positive pairs, a
  +4,334.25..+5,937-ms interval, 125 identical outputs, and safe full fallback.
- **Build Impact breadth is selective, not universal.** The Spring matrix
  qualified shared test preparation at **18.88% faster** and rejected leaf
  compilation/packaging. Verification and source-distribution graphs are now
  generically complete, but verification improved only **0.31%** with 2/4
  positive pairs. Attribution found only **1.238233 ms** in the largest
  BuildOpt-owned phase, so it remains on native full-graph execution and this
  optimization line is closed.
- **Direct JAR reuse did not improve the measured Spring test build.** It
  restored three exact Test-fixture JARs but regressed the complete unchanged
  test workflow by **735.25 ms/11.31%**, so the activation was not promoted.
  A follow-up three-arm ablation narrowed plugin registration but still found
  the complete adapter **612.25 ms/9.53% slower** than native Gradle.
- **Edge locality has controlled POC value.** Reading the same committed
  32-MiB object set through a prewarmed loopback Edge was **34.74% faster**
  than direct Shared reads over the frozen 80-ms/20-MiB/s link, saving
  2,401.25 ms with 4/4 positive pairs and zero measured upstream Edge reads.
  The unchanged mechanism then transferred to Kafka `:clients:testClasses`
  under an independently derived 337-ms/6,994,831-B/s profile: **15.21%
  faster**, saving 1,351.25 ms with 4/4 positive pairs and 4,062 exact outputs.
  This qualifies Edge locality for both bounded profiles, not Shared storage
  generally or every network.

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
| **Runtime Tuning** | Tests bounded worker, heap, and resource profiles intended to improve Gradle execution. | The latest real Spring candidate capped 12 workers to 6 and was **2.00% slower** (191.5 ms), with 2/4 favorable pairs and interval -973.5..+590.5 ms. Earlier synthetic `W3_H4G` and `W4_H6G` candidates were **4.3%** and **54.7% slower**. | **No current value. Disabled.** Optimized native Gradle remains the stable control. |
| **Build Impact** | Maps a change to the projects and tasks needed for the requested outputs, with full-graph fallback for unknown or global changes. | Synthetic coverage: **73.5-76.0% faster**. Installed Spring path: **15.76% faster**. Generalized Spring test preparation: **18.88% faster**. Kafka client packaging scope: **57.58% faster**. Spring verification is graph-complete but saved only **0.31%**; attribution found no product phase above **1.238233 ms**. | **The strongest broadly useful accelerator currently demonstrated, but only for independently qualified scopes.** The Kafka result proves graph reduction, not standard-`Jar` reuse of the required shaded artifact. |
| **Task Intelligence** | Observes and qualifies tasks only when their inputs, outputs, cache keys, and outcomes are exact enough to support an optimization. | No general direct saving. In the accepted pilot it enabled a qualified native-cache restore that saved **203 ms** on average. | A **safety and eligibility layer**, not a standalone accelerator. Its value is realized through a qualified cache or patch route. |
| **Patch Autopilot / reviewed task patch** | Produces a reviewable and reversible patch that correctly declares inputs and outputs and enables caching for an exact custom-task shape. | Exact reviewed Java recipe: **67.3% faster in Kotlin** and **68.0% faster in Groovy**. Combined installed path: **63.5-67.3% faster**. | Highly promising for **specific reviewed task contracts**. The result must not be generalized to arbitrary tasks or recipes. |
| **Graph reduction** | Replaces broad aggregate task dependencies with the typed producers required for the declared outputs. | The OpenTelemetry experiment removed **3 graph nodes and 2 executed tasks** while preserving all 125 required outputs. No standalone wall-clock percentage is claimed. | Structurally valuable, but it still needs independent timing evidence before it can be presented as a separate accelerator. |
| **Exact-bound hot-state reuse** | Reuses a previously validated Build Impact plan only when repository revision, graph, manifest, changes, Wrapper, executable, and options still match. | Reduced BuildOpt preparation from **74.97 ms to 40.34 ms**, but the fresh whole-build arm was **7.68% slower**. | **Disabled for the measured profile.** Micro-overhead reduction does not override regressive end-to-end evidence. |
| **Standard `Jar` cache adapter** | Makes only an unmodified standard Gradle `Jar` task eligible for native caching; custom archives, `Copy`, arbitrary tasks, and `Test` remain unchanged. | OpenTelemetry Build Impact: **39.92% faster**, saving 4,376.75 ms. Direct Spring test-build use: **11.31% slower** initially; after narrowing registration, the three-arm ablation remained **9.53% slower** than native. Kafka composition stopped before timing because `:clients:jar` was skipped and custom `:clients:shadowJar` produced the required artifact. | **Qualified only inside the measured OpenTelemetry composition.** It is not qualified for Kafka's shaded client artifact. Correct cacheability and cache hits are insufficient when the avoided work is too small or a different task owns the output. |
| **Shared and Edge Cache** | Reuse committed outputs across machines and optionally place them nearer developers or CI runners. | Synthetic 32-MiB profile: **34.74% faster**, saving 2,401.25 ms. Kafka transfer under an independently derived 337-ms/6,994,831-B/s profile: **15.21% faster**, saving 1,351.25 ms. Both have 4/4 positive pairs, exact outputs and zero measured Edge upstream reads. | **Edge locality transfers across two bounded profiles.** Shared provides origin and authority semantics; no independent Shared acceleration or universal-network claim is made. |
| **Build History** | Records durations, outcomes, cache behavior, and applied optimizations so results can be inspected and compared. | **No direct build-time saving.** | Observability that helps discover and validate optimizations; not an accelerator itself. |
| **Launcher, gateway, and telemetry** | Provide orchestration, authentication, evidence collection, and safe fallback behavior. | Add fixed overhead rather than saving work. The local-cache fast path avoids starting these components when they have no consumer. | Necessary infrastructure for instrumented flows, but it must remain off the critical path when it is not needed. |
| **Combined qualified path** | Orchestrates Build Impact and exact task optimizations through the packaged CLI and plugin while leaving unproven mechanisms disabled. | Fresh Spring Build Impact: **30.86% faster**. Clean OpenTelemetry Impact + standard `Jar`: **50.40% faster**. Fresh normalized Kafka Impact + Edge: **82.35% faster**, saving 35,405.5 ms with 4/4 positive pairs. | **Qualified only for exact measured workloads that pass both value and safety gates.** Kafka now passes exact cached, global-change, and HTTP-503 fallback gates for its fixed change and network profile. |

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

### Optimize the optimizer, but gate on the complete build

The Spring overhead ablation separates optimized native Gradle from loading
only BuildOpt's init/plugin classpath and from enabling the exact JAR adapter.
Native averages 6,422.75 ms, init/plugin-only averages 7,484.50 ms, and the
adapter averages 7,035 ms. The three cache hits recover 449.50 ms relative to
the init-only mean, but the complete installed arm remains 612.25 ms/9.53%
slower than native, with only 2/4 positive rounds and an interval of
-1,785.75..+235 ms.

Those init-only differences vary substantially with arm order, so they are a
diagnostic signal rather than a universal fixed-cost estimate. The product
decision does not depend on assigning every millisecond: an unqualified
workflow stays on native Gradle. BuildOpt keeps the exact adapter only in the
OpenTelemetry Build Impact scope where the complete path independently clears
the value gate.

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

The latest bounded test used the retained Spring trace to preregister one
worker-oversubscription hypothesis. The same offline `testClasses` workload
averaged 9,556.75 ms at the native 12 workers and 9,748.25 ms at 6 workers. The
candidate lost 191.5 ms/2.00%, had only 2/4 positive pairs and a paired interval
of -973.5..+590.5 ms, while preserving the same 378 outputs and task outcomes.
The terminal decision is `RETAIN_NATIVE_12_WORKERS`; the protocol explicitly
forbids another worker search after this no-value result.

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

Do not spend the next cycle building a general automatic tuner. The bounded
Spring worker-cap hypothesis failed and is closed without trying 7, 8, 9 or
another post-result value. Reopen Runtime Tuning only when a materially
different retained trace identifies a different bottleneck such as heap
pressure, garbage collection, queueing, configuration work or daemon startup.
Test one hypothesis at a time and retain `STABLE_CONTROL` whenever the paired
interval crosses zero or the gain is operationally insignificant.

Potential research areas are:

- workload-specific worker and heap selection rather than one global profile;
- daemon and JVM warm-up cost for CI and short-lived builds;
- configuration-phase and Kotlin DSL compilation overhead;
- dependency resolution and artifact-transform reuse;
- compiler avoidance and incremental compilation boundaries.

The completed test-build experiment strengthens this priority. Reusing three
small exact `testFixturesJar` producers did not shorten the critical path:
native Gradle averaged 6,503.5 ms and BuildOpt 7,238.75 ms, with 0/4 positive
pairs. The follow-up registration optimization and three-arm ablation did not
rescue it: the complete adapter remained 612.25 ms/9.53% slower than native.
The next candidate must therefore come from an observed resource or
critical-path bottleneck, not from another low-cost cacheable task.

### 6. Retain Edge only where locality can pay for itself

The controlled experiment now answers the bounded locality question. Direct
Shared reads averaged 6,911.25 ms; the same objects from a prewarmed Edge
averaged 4,510 ms, saving 34.74% with exact outputs and zero measured upstream
requests. Edge is therefore worth retaining for remote runners behind a
material network boundary. It should not be presented as a universal win on
low-latency networks, and the result does not establish that Shared alone is
faster than another Gradle-compatible remote origin.

The Kafka transfer strengthens that conclusion without changing the mechanism.
Under a network profile derived before timing from Kafka's own fixed source
archive, direct Shared averaged 8,885.25 ms and prewarmed Edge 7,534 ms. The
1,351.25-ms/15.21% saving cleared the same gate with 4/4 positive pairs, a
+788.25..+1,883-ms interval, four identical cache hits and 4,062 exact outputs.
The smaller percentage is expected because only four of Kafka's tasks consumed
7.65 MB from Shared; locality still shortened the complete build consistently.

### 7. Use real repositories with substantial builds

Short two-second repositories are useful for compatibility but poor for
performance decisions because startup noise dominates. Performance expansion
should use public repositories with:

- a stable offline or dependency-pinned workload;
- enough work for the candidate mechanism to matter;
- deterministic, hashable outputs;
- repeatable mutations and entrypoints;
- no repository-specific rule added to BuildOpt merely to win the benchmark.

Spring Framework and OpenTelemetry remain the primary POC laboratories. Apache
Kafka 4.3.1 now supplies the third workload class: 64 Gradle projects combining
Java, Scala, generated protocol sources, shaded packaging, and test
preparation. The unchanged clean profile qualified there without adding a
Kafka-specific product rule.

## Recommended Activation Policy

| Mechanism | Experimental profile | Default path | Required next proof |
|---|---:|---:|---|
| Native Gradle cache | Enabled | Enabled | Continue parity and output checks |
| Build Impact | Enabled for qualified scopes | Explicit command | Generalize across real change and output scopes |
| Hot-state reuse | Disabled for the measured profile | Disabled | Beat the same end-to-end control independently before reconsideration |
| Standard `Jar` adapter | Enabled only in the qualified Build Impact scope | Not universal | Require independent end-to-end value for every new workflow; optimized direct Spring use remained 9.53% slower |
| Reviewed task patch | Enabled only for exact matching contracts | Review required | Add recipes only after independent qualification |
| Strict Safe Cache | Disabled | Disabled | Beat or justify cost versus native Gradle cache |
| Runtime Tuning | Disabled | Disabled | Positive incremental evidence against optimized native Gradle |
| Shared / Edge Cache | Edge qualified independently and in the normalized Kafka composition | Explicit repository-owned v2 POC profile | Keep the 82.35% claim bound to the exact Kafka source precondition and modeled network profile; preserve native Shared behavior outside a matched candidate |
| Installed Kafka composition | Exact packaged profile qualified at 80.51% with 8/8 positive pairs | Explicit repository-owned v2 POC profile | Replicate installed-path value independently on Spring and OpenTelemetry before changing the portfolio claim; do not average repository percentages |
| Deterministic profile discovery | No new timing; exact retained Kafka profile reproduced | Read-only, review-required analyzer | Keep native fallback for Spring, OpenTelemetry, drift, incomplete/unknown graphs and selected Test tasks; never turn discovery into autonomous activation |

## Recommended Next Block

The verification attribution block is complete. The native trace contains 143
task rows and the candidate 51. The 92 omitted rows have 4,249 ms of cumulative
duration, but 35 are cache hits and another 39 are no-action, no-source,
skipped or up-to-date; parallel durations cannot be treated as additive
critical-path savings. The diagnostic candidate task interval is 1,581 ms
shorter, while BuildOpt's largest own phase is only 1.238233 ms. There is no
500-ms generic product bottleneck to remove, so verification stays native and
this line is closed without another timing run.

The Edge transfer is complete. On real Kafka `:clients:testClasses`, direct
Shared averaged 8,885.25 ms and prewarmed Edge 7,534 ms, saving
1,351.25 ms/15.21%. The same unchanged gate passed with 4/4 positive pairs,
interval +788.25..+1,883 ms, identical task outcomes and all 4,062 outputs.

The first attempted composition stopped before timing. Its unmeasured seed
proved that Kafka's required `kafka-clients-4.3.1.jar` is produced by custom
`:clients:shadowJar`, while `:clients:jar` is `SKIPPED`. No object was committed
to Shared, Edge was never opened, and no warm-up or measured pair ran. The
57.58% packaging result therefore remains valid for the fixed Build Impact
scope, but it cannot be attributed to exact standard-`Jar` reuse.

The first corrected composition is complete. Native full `assemble` through
Shared averaged 43,345 ms and installed Build Impact through prewarmed Edge
averaged 7,423 ms. All four diagnostic pairs were positive (+42,084, +34,881,
+35,269 and +31,454 ms), and the paired interval was strictly positive. Those
numbers are not additive component percentages and they are not a qualified
claim: forced Edge failure completed successfully but rebuilt custom
`shadowJar` with different bytes from the cached artifact.

The reproducibility block is now complete. Two independent uncached baseline
builds produced different 10,204,023-byte JAR digests while preserving the
same logical payload and entry order; their ZIP metadata fingerprints differed.
Kafka explicitly configured non-reproducible archive order and preserved file
timestamps. Changing only those two temporary source properties produced the
same `3ffd994e...3349` digest in two independent builds and preserved the
baseline payload. A fifth build received remote-cache HTTP 503, rebuilt
locally, and reproduced the same normalized digest after Gradle disabled the
cache.

This qualifies the narrow reproducibility input, not the prior 82.87% timing.
The fresh v2 composition then applied it equally to both arms and collected
four new pairs. Native + Shared averaged 42,992.75 ms; installed Impact + Edge
averaged 7,587.25 ms, saving 35,405.5 ms/**82.35%** with 4/4 positive pairs and
interval +30,162..+42,487.75 ms. All measured outputs were the exact normalized
JAR, Edge made zero measured Shared requests, the global change retained the
full graph, and HTTP 503 disabled remote cache and rebuilt identical bytes.
The terminal decision is `QUALIFY_KAFKA_IMPACT_EDGE_COMPOSITION`, bounded to
this change and network profile.

The exact repository-owned profile was then measured through the packaged
`buildopt poc` command rather than experiment-only wiring. Optimized native
Gradle plus Shared averaged 34,347.25 ms; the installed profile averaged
6,694.375 ms, saving 27,652.875 ms/**80.51%**. All 8/8 pairs were positive and
the corrected deterministic bootstrap interval was
+24,826.5..+29,903.625 ms. Exact normalized output, zero candidate origin
requests, native full-graph global fallback and byte-identical HTTP-503 local
fallback all passed. This independently qualifies installed-profile value for
the fixed Kafka revision, change, output and network model; it does not make
the percentage universal or additive with another mechanism result.

The subsequent installed matrix tested whether similarly bounded profiles
replicate as a broad POC claim. They do not. Spring Build Impact averaged
1,895 ms/14.33% faster and preserved exact outputs and fallback, but one of
eight pairs regressed by 57 ms; the unchanged all-positive gate therefore
retains optimized native Gradle. OpenTelemetry produced no accepted timing:
impact discovery was terminated by signal after a successful unmeasured
preflight, so the evidence records `PREPARATION_FAILED` rather than inferring
value. Kafka independently retained 28,523.25 ms/81.85% savings with 8/8
positive pairs, interval +26,603.5..+30,509 ms, exact normalized output and
both safety fallbacks. With only one of three families qualified, the POC must
specialize instead of claiming a general accelerator or averaging percentages.

The **qualified-profile usability and scope synthesis** block is now complete:

1. `buildopt poc --changes-file PATH` reads one repository-owned versioned
   profile instead of experiment-only flags;
2. activation remains opt-in and limited to declared output scopes;
3. the CLI reports the selected/full graph, exact adapters, expected outputs
   and disabled mechanisms before Gradle starts;
4. fallback uses the original entrypoints without the standard-`Jar` adapter;
5. Runtime, Hot State, Copy and Safe Cache remain forced out of the command;
6. a v2 profile may activate only the separately qualified Build Impact plus
   read-only Edge composition, with exact file-SHA preconditions and an
   explicit loopback endpoint;
7. global scope, precondition drift, missing/invalid Edge and bypass select the
   native full graph before Gradle, while HTTP 503 executes locally with exact
   output bytes.

That installed adoption block is now complete. On the same fixed revisions,
the packaged command restored `:testing-common:jar` and `:generator:jar`,
reproduced the historical 125/4,062 output digests, and completed full-graph
fallback for global changes. It captured no timing, so the earlier qualified
50.40% and 55.09% results remain its historical performance claims.

The follow-up Kafka packaging experiment is also complete. Native root
`assemble` averaged 8,054 ms; the installed three-project client-Jar candidate
averaged 3,416.5 ms, saving 4,637.5 ms/57.58%. All four pairs were positive,
the smallest saving was 4,050 ms, the exact 10.2-MB JAR matched, and the global
fallback passed. This is fresh Build Impact performance evidence, limited to
the declared Kafka client-packaging scope. The later seed diagnosis showed
that the required artifact is produced by custom `shadowJar`; the standard
`Jar` adapter must not receive credit for this result.

Normal-build task tails, direct test-build JAR reuse and the trace-selected
six-worker Runtime candidate are closed for the current evidence. They should
reopen only for a materially different dominant task or bottleneck, not to
manufacture another optimization.

## Open Questions

- Does a materially different retained trace expose a Runtime bottleneck that
  justifies one new preregistered hypothesis?
- Does one retained installed-path trace expose at least 500 ms of recoverable,
  product-addressable critical-path work across two repository or workload
  families? If not, close the hypothesis with no action.

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
- [terminal installed qualified-profile matrix](../../benchmarks/results/poc-qualified-profile-matrix-v1/summary.json).
- [fresh full-path ablation and retained component evidence](../../benchmarks/results/poc-full-path-ablation-v1/summary.json).
- [qualified clean OpenTelemetry composition](../../benchmarks/results/poc-otel-clean-composition-v1.json).
- [targeted Spring Runtime decision](../../benchmarks/results/poc-runtime-research-v1.json).
- [Apache Kafka packaging evidence](../../benchmarks/results/poc-kafka-packaging-v1.json).
- [repository-owned Kafka composition usability evidence](../../benchmarks/results/poc-kafka-composition-usability-v1.json).
- [installed Kafka profile value evidence](../../benchmarks/results/poc-kafka-installed-profile-value-v1.json).
- [verification/distribution graph evidence](../../benchmarks/results/poc-verification-distribution-graph-v1.json).
- [verification overhead attribution](../../benchmarks/results/poc-verification-overhead-attribution-v1.json).

Validate the current scorecard with:

```bash
./dev/check-build-optimization-performance
./dev/check-poc-value-validation
./dev/check-poc-runtime-research
./dev/check-poc-verification-distribution-graph-v1-result
./dev/check-poc-verification-overhead-attribution-v1-result
```
