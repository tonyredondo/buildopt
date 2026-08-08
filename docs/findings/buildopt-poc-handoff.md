# BuildOpt POC Handoff

## Executive Summary

- **BuildOpt is an evidence-driven Gradle optimization POC.** It runs the
  repository's existing Gradle command, preserves expected outputs and exit
  behavior, and applies an optimization only when its scope and fallback are
  explicit.
- **The clearest value comes from avoiding work and making exact expensive
  tasks reusable.** Installed Build Impact improved the reviewed Spring
  workload by 15.76%; a conservative standard-`Jar` adapter improved the
  OpenTelemetry workload by 39.92%.
- **Not every feature adds value.** Safe Cache is effectively at parity with
  Gradle's native cache, and the latest trace-selected Runtime Tuning candidate
  was 2.00% slower. Both remain outside the default optimization path.
- **The clean full path now qualifies.** The first composition was rejected
  because hot state regressed by 7.68%. The fresh rerun removed hot state and
  combined only Build Impact with the exact standard-`Jar` adapter, saving
  **5,361.25 ms/50.40%** with 4/4 positive pairs and 125 identical outputs.
- **Build Impact does not generalize uniformly.** Shared test preparation
  qualified at **18.88% faster**, while leaf compilation and packaging missed
  the frozen gate. Generic verification/source-distribution graphs are now
  complete, but Spring verification saved only **0.31%** with 2/4 positive
  pairs. Attribution found only **1.238233 ms** in the largest BuildOpt-owned
  phase, so it still retains native full-graph execution and this line stops.
- **More cache hits are not automatically valuable.** On a real Spring test
  workflow, BuildOpt restored three exact Test-fixture JARs but still made the
  complete build **11.31% slower**. After narrowing plugin registration, a
  three-arm ablation still measured it **9.53% slower** than native. The
  activation was not promoted.
- **Edge locality now has bounded value evidence.** The same Gradle HTTP client
  and eight committed Shared objects averaged 6,911.25 ms over a frozen modeled
  WAN and 4,510 ms through a prewarmed Edge: **34.74% faster**, 4/4 positive
  pairs, identical 32-MiB outputs and zero measured upstream Edge requests.
  The unchanged mechanism transferred to real Kafka `:clients:testClasses`
  under an independently derived network profile: direct Shared averaged
  8,885.25 ms and Edge 7,534 ms, **15.21% faster**, with 4/4 positive pairs and
  all 4,062 outputs identical.
- **The clean profile transfers to a third substantial repository.** On Apache
  Kafka 4.3.1, generic discovery reduced build-owned test preparation from 64
  reached projects to three and exact `Jar` reuse restored `:generator:jar`.
  Installed BuildOpt averaged **55.09% faster** (2,539.5 ms saved), with 4/4
  positive pairs and 4,062 byte-identical outputs.
- **The public POC command now consumes repository-owned profiles end to end.**
  An installed native package replayed the committed profile on the fixed
  OpenTelemetry and Kafka revisions, reproduced 125 and 4,062 historical
  outputs, and completed native full-graph fallback for global changes.
  OpenTelemetry restored an exact standard `Jar`; Kafka restored
  `:generator:jar`, while its required shaded client artifact is not a standard
  `Jar`. This is adoption evidence; it deliberately adds no new timing claim.
- **Kafka packaging now qualifies independently as Build Impact.** For a fixed
  central client change, native root `assemble` averaged 8,054 ms and installed
  BuildOpt averaged 3,416.5 ms: **57.58% faster**, saving 4,637.5 ms with 4/4
  positive pairs, an exact 10.2-MB client JAR, zero product failures and full fallback.
  A later composition seed proved `:clients:jar` is skipped and the required
  artifact is produced by custom `:clients:shadowJar`, so this saving is not
  evidence for the standard-`Jar` adapter.

## Product Idea

Gradle already provides Build Cache, up-to-date checks, incremental tasks,
Configuration Cache, parallel execution, and remote-cache integration. BuildOpt
does not replace those features. It adds a decision layer around Gradle that
aims to answer three questions safely:

1. Can this build execute a smaller affected graph?
2. Can an expensive task be made reusable without changing its outputs?
3. Does the complete installed path beat an already optimized native Gradle
   baseline?

The qualified mechanisms are now exposed through one explicit repository-owned
POC command:

```text
buildopt poc --changes-file .buildopt-changes
```

`buildopt-qualified-profile.json` binds the repository, pipeline, Build Impact
state and clean mechanism set. Before Gradle starts, the command reports the
selected or full graph, exact adapters, expected outputs and disabled
mechanisms. It enables only Build Impact plus the exact standard-`Jar` adapter
for a qualified alternative; Safe Cache, Runtime Tuning, Hot State, `Copy` and
Shared/Edge remain outside this path. Unknown/global changes and bypass use
native full-graph execution. This improves POC usability without broadening the
performance or production claim.

Unknown changes, unqualified tasks, failed validation, or
`BUILDOPT_BYPASS=1` restore the original native/full-graph path.

## Components and Differentiation from Gradle

| Component | What it does | What Gradle already provides | BuildOpt differentiation and current status |
|---|---|---|---|
| **Launcher and evidence envelope** | Runs the original command and records bounded execution evidence. | Direct Wrapper execution and normal process semantics. | Adds attribution, exact fallback, and bypass. It is infrastructure, not an accelerator, so its overhead must stay off unused paths. |
| **Safe Cache / local L1** | Reuses verified outputs inside repository/Wrapper/platform boundaries. | Native local and remote Build Cache. | Adds stricter isolation and verification, but no incremental speed advantage has been proven. Native Gradle cache remains the default. |
| **Runtime Tuning** | Evaluates worker, heap, fork, and resource profiles. | User-configured workers, JVM arguments, parallelism, and daemon behavior. | Intended to choose profiles from evidence rather than static defaults. A bounded Spring 12-to-6-worker experiment was 191.5 ms/2.00% slower, so current candidates remain disabled. |
| **Build Impact** | Maps changed files to the projects and entrypoints required for declared outputs. | Incremental and up-to-date behavior inside the task graph requested by the user. | Selects a smaller repository-authorized graph before Gradle executes it, with full-graph fallback for ambiguous/global changes. This is the strongest broadly useful accelerator so far. |
| **Task Intelligence** | Qualifies tasks only when inputs, outputs, keys, and outcomes are exact enough. | Cacheability annotations and task validation supplied by build authors/plugins. | Adds an evidence-based eligibility layer. It enables optimizations but does not directly make a build faster. |
| **Patch Autopilot** | Produces reviewable, reversible patches for exact known task shapes. | Manual build-script/plugin changes. | Can turn a non-cacheable custom task into a safely reusable task, but only for reviewed recipes that match exactly. |
| **Exact task adapters** | Adds bounded eligibility for an unmodified standard Gradle task type. | Native caching when a task is correctly cacheable. | Uses real task traces to close one missing cacheability gap at a time. The standard-`Jar` adapter qualified; the later standard-`Copy` adapter works exactly but remains disabled because its isolated and incremental timing was unstable. |
| **Hot-state reuse** | Reuses a validated impact plan when every repository, graph, Wrapper, executable, and option digest still matches. | Configuration Cache and daemon reuse inside Gradle. | Reduced planning overhead, but the fresh end-to-end arm was 7.68% slower. It is disabled for this profile. |
| **Shared / Edge Cache** | Shares committed outputs and optionally places them nearer runners. | Native Gradle remote-cache protocol and third-party cache servers. | Edge qualified at **34.74% faster** on the synthetic 80-ms/20-MiB/s profile and **15.21% faster** on Kafka under an independently derived 337-ms/6,994,831-B/s profile. Shared remains the authoritative origin; no standalone Shared or universal-network speed claim is made. |
| **Build History** | Stores redacted sessions, timing, cache, and optimization evidence. | Logs, Build Scans, and external observability tooling. | Provides local POC evidence and comparison. It improves diagnosis, not build time directly. |

## Initial Performance Evidence

All qualified comparisons preserve their declared outputs and retain
unfavorable observations. Percentages from different rows are not additive.

### Controlled synthetic Kotlin and Groovy workloads

| Mechanism | Kotlin | Groovy | Decision |
|---|---:|---:|---|
| Safe Cache versus cache disabled | **15.9% faster** | **13.7% faster** | Useful when caching is absent. |
| Safe Cache versus native Gradle cache | **0.02% faster** | **0.47% slower** | Native-cache parity; no BuildOpt acceleration claim. |
| Runtime Tuning `W3_H4G` | **4.3% slower** overall (512 ms) | Not DSL-specific | Disabled; the earlier `W4_H6G` profile was 54.7% slower. |
| Build Impact | **76.0% faster** (1,939 ms) | **73.5% faster** (2,155 ms) | Qualified for the controlled affected-work class. |
| Reviewed Task/Patch | **67.3% faster** (1,369 ms) | **68.0% faster** (2,349 ms) | Qualified only for the exact reviewed custom-task recipe. |
| Combined installed Impact path | **77.5% faster** | **84.1% faster** | Qualified synthetic POC value; not a universal claim. |

### Spring Framework: real public repository

| Experiment | Result | Interpretation |
|---|---:|---|
| Direct `testClasses` Build Impact | **28.72% faster**, 2,098 ms saved, 8/8 positive pairs | Generic affected-project selection qualified with 378 identical outputs. |
| Installed `buildopt impact` path | **15.76% faster**, 1,260 ms saved, 8/8 positive pairs | Includes package, launcher, manifest, and graph-validation overhead. This is the most representative Spring result. |
| Additional `spring-webmvc` leaf scope | **13.50% faster**, 4/4 positive pairs | Additional positive per-scope evidence. |
| Shared `spring-core` to `spring-jms` scope | **10.89% faster on average**, but only 3/4 positive pairs | Failed the frozen stability gate; no broad shared-change claim. |
| Global two-fork test tuning | **24.07% slower** | Rejected. Every test was retained, but the candidate was materially worse. |
| Selective two-fork test tuning | **1.57% slower** and `:spring-test:test` failed | Rejected and not retried. Work returned to build-owned test preparation. |
| Trace-selected six-worker `testClasses` cap | **2.00% slower**, 191.5 ms lost, 2/4 positive pairs | Rejected. Native 12 workers averaged 9,556.75 ms; six workers averaged 9,748.25 ms, with identical 378 outputs and task outcomes. |
| Exact Test-fixture JAR reuse around unchanged tests | **11.31% slower**, 735.25 ms lost, 0/4 positive pairs | Rejected. All 8 tests ran with identical cases and 15 identical JARs; three cache hits did not offset adapter overhead. |
| Optimized JAR-adapter overhead ablation | **9.53% slower**, 612.25 ms lost, 2/4 positive rounds | Native averaged 6,422.75 ms; init/plugin-only 7,484.50 ms; adapter 7,035 ms. Hits recovered 449.50 ms versus init-only, but end-to-end value still failed. |
| Generalized shared test preparation | **18.88% faster**, 2,638 ms saved, 4/4 positive pairs | Qualified with interval +1,516..+3,275.5 ms and 378 identical outputs. |
| Generalized leaf compilation | **1.33% faster**, 196.25 ms saved, 3/4 positive pairs | Rejected: misses 500 ms, 2%, 4/4, and positive-bound gates. |
| Generalized leaf packaging | **3.73% faster**, 427.25 ms saved, 2/4 positive pairs | Rejected: misses 500 ms, 4/4, and positive-bound gates. |
| Verification graph completion | **0.31% faster**, 103.75 ms saved, 2/4 positive pairs | Generic graph is complete and exact, but performance is unstable; retain native full graph. |
| Verification overhead attribution | 143 native versus 51 candidate task rows; **1.238233 ms** largest BuildOpt phase | The 92 omitted rows mostly restore or no-op and overlap; no 500-ms correction exists, so stop this line. |
| Source distribution graph completion | Complete 12-project candidate; not timed | Capability proved generically; prior leaf packaging did not qualify, so no new timing was authorized. |

### OpenTelemetry Java Instrumentation: real public repository

| Experiment | Result | Interpretation |
|---|---:|---|
| Initial installed Spring-family transfer | **10.01% faster on average**, but 3/4 positive pairs and interval crossed zero | Favorable signal, not stable enough to qualify. |
| Typed graph reduction plus hot plan | **1.94% slower**, only 2/4 positive pairs | Correct outputs and fallback, but insufficient value. The result was retained rather than hidden. |
| Exact-bound hot-state reuse | BuildOpt preparation reduced from 74.97 ms to 40.34 ms (**46.2%**) | Internal planning improvement only; not a whole-build percentage. |
| Standard-`Jar` adapter | **39.92% faster**, 4,377 ms saved, 4/4 positive pairs | Qualified installed POC value with 125 identical outputs, zero product failures, and full 53-entrypoint fallback. |
| Clean Impact + standard-`Jar` composition | **50.40% faster**, 5,361.25 ms saved, 4/4 positive pairs | Qualified without hot state; paired interval +4,334.25..+5,937 ms, identical 125-file outputs, and successful full-graph fallback. |
| Standard-`Copy` adapter alone | **27.05% faster on average**, 4,284.25 ms saved, 3/4 positive pairs | Not qualified: interval -3,334.25..+8,846.5 ms crosses zero. |
| Incremental `Copy` on Impact + `Jar` | **24.90% faster on average**, 2,391.25 ms saved, 3/4 positive pairs | Not qualified: the direct incremental interval -2,568.25..+7,092 ms crosses zero. |
| Complete Impact + `Jar` + `Copy` profile | **52.89% faster**, 4,377 ms saved, 4/4 positive pairs | The directly measured cascade qualifies globally with interval +4,130.25..+4,653.25 ms and 21,818 identical outputs; percentages were not added. Copy still remains disabled because its incremental authorization gate failed. |

### Apache Kafka: third real-repository transfer

| Experiment | Result | Interpretation |
|---|---:|---|
| Clean Impact + exact standard-`Jar` profile | **55.09% faster**, 2,539.5 ms saved, 4/4 positive pairs | The unchanged installed profile reduced the conservative graph from 64 projects to three and restored `:generator:jar`; interval +1,625.5..+4,093 ms, 4,062 identical outputs, no Gradle `Test`, and successful full-graph fallback. |
| Client packaging through installed profile | **57.58% faster**, 4,637.5 ms saved, 4/4 positive pairs | Native root `assemble` averaged 8,054 ms; BuildOpt selected the three-project packaging scope and averaged 3,416.5 ms. The smallest pair saving was 4,050 ms; the exact 10.2-MB JAR and global fallback passed. Later diagnosis attributes this to Build Impact scope reduction, not standard-`Jar` reuse. |
| Build Impact + prewarmed Edge | **82.87% diagnostic difference**, 35,922 ms mean, 4/4 positive pairs | Native full `assemble` through Shared averaged 43,345 ms; the installed candidate averaged 7,423 ms and restored the same cached `shadowJar`. Not qualified: forced Edge failure completed but rebuilt the custom JAR with different bytes. |
| Custom `shadowJar` reproducibility | **5/5 clean builds passed the fixed safety protocol** | Original builds kept the same payload/order but drifted in ZIP metadata. Two normalized builds and an HTTP-503 fallback produced identical `3ffd994e...3349` bytes. This qualifies an input, not a performance percentage. |
| Fresh normalized Build Impact + Edge | **82.35% faster**, 35,405.5 ms saved, 4/4 positive pairs | Native + Shared averaged 42,992.75 ms; installed Impact + Edge averaged 7,587.25 ms. Interval +30,162..+42,487.75 ms, exact normalized output, full-graph global fallback, zero measured candidate-origin traffic, and byte-identical HTTP-503 fallback. Qualified only for this change and network profile. |
| Exact installed Kafka profile | **80.51% faster**, 27,652.875 ms saved, 8/8 positive pairs | The packaged `buildopt poc` path retained the qualified value: native + Shared averaged 34,347.25 ms and the installed repository-owned profile 6,694.375 ms. Corrected bootstrap interval +24,826.5..+29,903.625 ms, exact normalized output, zero origin requests, global full-graph fallback and byte-identical HTTP-503 fallback. |
| Installed qualified-profile matrix | **Kafka 81.85% faster; Spring 14.33% faster but unqualified; OpenTelemetry no accepted observations** | Kafka passed 8/8 pairs and all safety gates. Spring preserved output/fallback but passed only 7/8 pairs, including one -57-ms regression. OpenTelemetry impact discovery was terminated by signal after preflight, so it retains native with no performance claim. Only 1/3 families qualifies; do not average these percentages. |

## Latest Generalization and Next Work

The qualified Kafka composition has now moved from experiment-only wiring into
a repository-owned v2 POC profile and has been measured through the exact
installed command. `buildopt poc` exposes the normalized source SHA, selected
graph, read-only loopback Edge endpoint and disabled mechanisms before Gradle
starts. Eight fresh pairs retained **80.51%** mean savings with a strictly
positive bootstrap interval; global/unknown changes and endpoint failures keep
the native or local fallback safe. The claim remains bounded to the fixed
Kafka revision, change, output and modeled network profile.

The terminal installed matrix narrows the next step further. Deterministic
profile discovery should reproduce only the qualified Kafka profile from
checked manifests, graphs, traces and evidence. Spring and OpenTelemetry are
negative fixtures: discovery must select their native paths, not encode their
repository names or resurrect an unqualified profile. Broad automatic profile
generation is not authorized by the current 1/3 result.

**Continue the POC, but activate only measured value.** The fresh ablation
qualified Spring Build Impact at 2,492.375 ms/30.86% saved with 8/8 positive
pairs. OpenTelemetry Build Impact alone saved 985.5 ms/7.49% but did not meet
the stability gate; adding exact hot state regressed by 892 ms/7.68%. Adding
the standard `Jar` adapter produced a strong 4,496.75-ms/40.60% terminal gain,
but that composition was rejected because it contained the regressive hot-state
arm. All raw evidence and unfavorable observations were retained.

The clean rerun then removed hot state and qualified at 5,361.25 ms/50.40%
saved, with 4/4 positive pairs, a +4,334.25..+5,937-ms interval, identical
outputs, zero product failures, and successful full-graph fallback.

The subsequent Spring generalization matrix kept those same gates and produced
one transferable result: shared test preparation averaged 13,971.75 ms natively
and 11,333.75 ms with BuildOpt, saving 2,638 ms/18.88%, with 4/4 positive pairs
and a +1,516..+3,275.5-ms interval. Leaf compilation and packaging did not
qualify and remain on native Gradle. Verification and distribution were then
made complete through public Gradle task contracts. The measured verification
scope averaged only 103.75 ms/0.31% faster with 2/4 positive pairs and a
-5,158-ms lower bound, so it also remains on native Gradle. Build-logic and
global-configuration changes continue to retain the original full graph.

The standard-`Copy` experiment confirms the cascade concern directly. The
complete profile is stable at 52.89% faster, but Copy alone and Copy's direct
incremental contribution are not stable enough to activate. This is why the
POC measures the whole profile as well as each increment: graph reduction can
shorten the critical path and amplify later reuse, while a favorable terminal
profile must not conceal an unqualified component.

The follow-up normal-build tail review found no further actionable adapter in
the retained real traces. Standard `Jar` is already active, standard `Copy`
remains unauthorized, custom `ShadowJar` is below the value floor and already
served by native cache, configured `JavaExec` is below the floor with broader
effects, and Spring exposed no new exact standard task. This closes only the
current trace set; a materially different dominant tail may reopen the work.

The build-owned test experiment then preserved the exact requested test
boundary but rejected direct standard-JAR reuse: native Gradle averaged
6,503.5 ms and BuildOpt 7,238.75 ms, a 735.25-ms/11.31% regression with an
interval of -1,449..-113 ms. The result retained all four pairs, 8/8 tests per
arm, three differential cache hits and 15 identical JARs. The POC therefore
kept the switch diagnostic-only rather than promoting a technically correct
but slower feature.

The follow-up optimized the adapter itself: one root-plugin registration now
targets only typed `Jar` collections instead of applying through every project
and observing every task. A separate four-round ablation then isolated native,
init/plugin-only, and adapter arms. The adapter recovered 449.50 ms relative
to init-only on the mean, but remained 612.25 ms/9.53% slower than native, with
2/4 positive rounds and an interval crossing zero. This closes the question
for the current Spring workflow: native Gradle is the value-aware selection;
cache-hit count cannot override the complete build result.

The bounded Runtime experiment then selected worker oversubscription from the
retained Spring trace before timing. Reducing the exact `testClasses` workload
from 12 to 6 workers lost 191.5 ms/2.00%, produced only 2/4 positive pairs and
an interval of -973.5..+590.5 ms, while preserving all 378 outputs and sorted
task outcomes. The terminal decision is `RETAIN_NATIVE_12_WORKERS`; no further
worker search is allowed for this trace.

The controlled remote-cache experiment then qualified Edge locality. Direct
Shared reads averaged 6,911.25 ms; the same eight objects from prewarmed Edge
averaged 4,510 ms, saving 2,401.25 ms/34.74%. All four pairs were positive,
the paired interval was +2,260.5..+2,542 ms, all 32-MiB outputs and task
outcomes matched, and Edge made zero measured upstream requests. The claim is
deliberately limited to this network profile.

The unchanged Edge mechanism then transferred to Kafka's real
`:clients:testClasses` workload. A separately derived 337-ms/6,994,831-B/s
profile was frozen before cache timing. Direct Shared averaged 8,885.25 ms and
Edge 7,534 ms, saving 1,351.25 ms/15.21%; all four pairs were positive, the
interval was +788.25..+1,883 ms, four cache-hit outcomes matched, and all 4,062
required outputs were identical.

The attempted Kafka composition then stopped before timing because its
standard-`Jar` premise was false: `:clients:jar` was `SKIPPED` and custom
`:clients:shadowJar` produced the required artifact. Shared received zero
objects, Edge was never opened, and no warm-up or measured pair ran. This
corrects the next step: compose only Kafka-qualified Build Impact and Edge
locality, without adding percentages or crediting the standard-`Jar` adapter.

That corrected experiment is now complete. Its four pairs saved 42,084,
34,881, 35,269 and 31,454 ms, a diagnostic 82.87% mean difference with an
entirely positive interval. Measured cached outputs matched, Build Impact
selected the intended scope, Edge made zero origin requests and global fallback
passed. The composition still does not qualify: an Edge 503 caused Gradle to
fall back and finish normally, but the custom `shadowJar` rebuilt different
bytes.

The follow-up reproducibility block has now isolated and corrected that safety
input. Two clean baseline JARs had the same logical payload and entry order but
different ZIP metadata. With reproducible archive order and timestamps
disabled, two independent builds and a third HTTP-503 fallback all produced
the same `3ffd994e...3349` JAR digest. This authorizes a fresh preregistered
composition run; it does not retroactively turn the earlier 82.87% diagnostic
into a qualified claim.

The fresh v2 composition has now used that input in both arms without reusing
the old observations. Native full `assemble` through Shared averaged
42,992.75 ms and installed Build Impact through prewarmed Edge averaged
7,587.25 ms. The four new savings were +46,159, +35,139, +28,850 and
+31,474 ms, producing an **82.35%** mean reduction and a
+30,162..+42,487.75-ms interval. Every arm restored the same normalized JAR,
the candidate made zero measured Shared requests, global fallback passed, and
HTTP 503 rebuilt the exact bytes locally. The composition is qualified only
for this Kafka change and frozen network profile.

The third-repository transfer then applied the unchanged clean profile to
Apache Kafka 4.3.1, a 64-project Java/Scala/generated-source build. Native root
`testClasses` averaged 4,609.5 ms; installed BuildOpt selected
`:clients:testClasses` and averaged 2,070 ms, saving 2,539.5 ms/55.09%. All four
pairs were positive, the interval remained above zero, 4,062 required outputs
were identical, and build-logic drift restored the full graph. This closes the
recorded ten-block performance roadmap with transfer evidence rather than a
repository-specific rule.

The subsequent adoption replay installed the current native package and used
only `buildopt poc --changes-file .buildopt-changes` plus repository-owned
state. OpenTelemetry restored `:testing-common:jar` and reproduced its exact
125-file output digest; Kafka restored `:generator:jar` and reproduced its
exact 4,062-file digest. A `gradle.properties` change on each repository
reported `FULL_GRAPH` before Gradle, disabled the adapter, reached work outside
the candidate graph, and completed successfully. No durations were captured:
the earlier qualified measurements remain the only performance claims.

The next preregistered value experiment then measured that same public command
for a Kafka packaging output rather than test preparation. Root `assemble`
reached the complete 64-project graph; the candidate reached three projects.
Native averaged 8,054 ms and BuildOpt 3,416.5 ms, saving 4,637.5 ms/57.58%.
All four alternating pairs were positive, the exact client JAR matched after
every arm, and global drift restored native `assemble`. Packaging is therefore
qualified only for this declared Kafka output and mutation.

The verification attribution is now terminal. The trace removed 92 task rows,
but 35 were cache hits and another 39 were no-action, no-source, skipped or
up-to-date; their 4,249-ms cumulative duration overlaps. The candidate task
interval was 1,581 ms shorter, while BuildOpt's largest own phase was only
1.238233 ms. No correction is authorized and verification stays native.

The installed matrix is terminal and selects specialization, not broad
continuation. The next block generates only the retained Kafka profile
deterministically from checked manifests, graphs, traces and evidence. It must
be reviewable, source-bound and fail closed to the native full graph on drift
or uncertainty. Spring and OpenTelemetry remain negative fixtures, and no
repository-name rule, hidden allowlist or automatic activation is permitted.

## Boundaries and References

This is proof-of-concept evidence, not a universal savings or production
readiness claim. Test Optimization, soak testing, design partners, HA, and
production operations are outside the current scope.

- [Detailed performance findings and recommendations](./build-optimization-performance.md)
- [Benchmark evidence index](../../benchmarks/README.md)
- [Current implementation tracker and next roadmap](../../implementation-tracker.md)
- [Synthetic combined evidence](../../benchmarks/results/poc-value-combined-v1.json)
- [Installed Spring evidence](../../benchmarks/results/poc-spring-installed-impact-v1.json)
- [Final OpenTelemetry evidence](../../benchmarks/results/poc-otel-optimization-v2.json)
- [Fresh full-path ablation](../../benchmarks/results/poc-full-path-ablation-v1/summary.json)
- [Qualified clean OpenTelemetry composition](../../benchmarks/results/poc-otel-clean-composition-v1.json)
- [Build Impact generalization evidence](../../benchmarks/results/poc-impact-generalization-v1.json)
- [Standard-Copy cascade evidence](../../benchmarks/results/poc-standard-copy-cascade-v1.json)
- [Kafka remote-composition terminal evidence](../../benchmarks/results/poc-qualified-remote-composition-v1.json)
- [Spring test-build stop evidence](../../benchmarks/results/poc-spring-test-build-optimization-v1.json)
- [Optimization-overhead ablation](../../benchmarks/results/poc-optimization-overhead-ablation-v1.json)
- [Targeted Runtime decision](../../benchmarks/results/poc-runtime-research-v1.json)
- [Controlled remote-cache locality evidence](../../benchmarks/results/poc-remote-cache-value-v1.json)
- [Apache Kafka transfer evidence](../../benchmarks/results/poc-third-repository-transfer-v1.json)
- [Installed qualified-profile adoption evidence](../../benchmarks/results/poc-qualified-profile-adoption-v1.json)
- [Terminal installed qualified-profile matrix](../../benchmarks/results/poc-qualified-profile-matrix-v1/summary.json)
- [Apache Kafka packaging evidence](../../benchmarks/results/poc-kafka-packaging-v1.json)
- [Kafka shadow JAR reproducibility evidence](../../benchmarks/results/poc-kafka-shadowjar-reproducibility-v1.json)
- [Installed Kafka profile value evidence](../../benchmarks/results/poc-kafka-installed-profile-value-v1.json)
- [Verification/distribution graph evidence](../../benchmarks/results/poc-verification-distribution-graph-v1.json)
- [Verification overhead attribution](../../benchmarks/results/poc-verification-overhead-attribution-v1.json)
