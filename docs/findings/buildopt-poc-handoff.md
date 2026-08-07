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
  Gradle's native cache, and the tested Runtime Tuning profiles regressed. Both
  remain outside the default optimization path.
- **The clean full path now qualifies.** The first composition was rejected
  because hot state regressed by 7.68%. The fresh rerun removed hot state and
  combined only Build Impact with the exact standard-`Jar` adapter, saving
  **5,361.25 ms/50.40%** with 4/4 positive pairs and 125 identical outputs.
- **Build Impact does not generalize uniformly.** In the latest real Spring
  matrix, shared test preparation qualified at **18.88% faster**, while leaf
  compilation and packaging missed the frozen stability gate. Verification and
  source distribution correctly retained the full graph.

## Product Idea

Gradle already provides Build Cache, up-to-date checks, incremental tasks,
Configuration Cache, parallel execution, and remote-cache integration. BuildOpt
does not replace those features. It adds a decision layer around Gradle that
aims to answer three questions safely:

1. Can this build execute a smaller affected graph?
2. Can an expensive task be made reusable without changing its outputs?
3. Does the complete installed path beat an already optimized native Gradle
   baseline?

Unknown changes, unqualified tasks, failed validation, or
`BUILDOPT_BYPASS=1` restore the original native/full-graph path.

## Components and Differentiation from Gradle

| Component | What it does | What Gradle already provides | BuildOpt differentiation and current status |
|---|---|---|---|
| **Launcher and evidence envelope** | Runs the original command and records bounded execution evidence. | Direct Wrapper execution and normal process semantics. | Adds attribution, exact fallback, and bypass. It is infrastructure, not an accelerator, so its overhead must stay off unused paths. |
| **Safe Cache / local L1** | Reuses verified outputs inside repository/Wrapper/platform boundaries. | Native local and remote Build Cache. | Adds stricter isolation and verification, but no incremental speed advantage has been proven. Native Gradle cache remains the default. |
| **Runtime Tuning** | Evaluates worker, heap, fork, and resource profiles. | User-configured workers, JVM arguments, parallelism, and daemon behavior. | Intended to choose profiles from evidence rather than static defaults. Current candidates are disabled because they regressed. |
| **Build Impact** | Maps changed files to the projects and entrypoints required for declared outputs. | Incremental and up-to-date behavior inside the task graph requested by the user. | Selects a smaller repository-authorized graph before Gradle executes it, with full-graph fallback for ambiguous/global changes. This is the strongest broadly useful accelerator so far. |
| **Task Intelligence** | Qualifies tasks only when inputs, outputs, keys, and outcomes are exact enough. | Cacheability annotations and task validation supplied by build authors/plugins. | Adds an evidence-based eligibility layer. It enables optimizations but does not directly make a build faster. |
| **Patch Autopilot** | Produces reviewable, reversible patches for exact known task shapes. | Manual build-script/plugin changes. | Can turn a non-cacheable custom task into a safely reusable task, but only for reviewed recipes that match exactly. |
| **Exact task adapters** | Adds bounded eligibility for an unmodified standard Gradle task type. | Native caching when a task is correctly cacheable. | Uses real task traces to close one missing cacheability gap at a time. The first standard-`Jar` adapter produced the strongest public-repository result. |
| **Hot-state reuse** | Reuses a validated impact plan when every repository, graph, Wrapper, executable, and option digest still matches. | Configuration Cache and daemon reuse inside Gradle. | Reduced planning overhead, but the fresh end-to-end arm was 7.68% slower. It is disabled for this profile. |
| **Shared / Edge Cache** | Shares committed outputs and optionally places them nearer runners. | Native Gradle remote-cache protocol and third-party cache servers. | Adds authority and pending-write controls, but has no defensible performance advantage yet. |
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
| Generalized shared test preparation | **18.88% faster**, 2,638 ms saved, 4/4 positive pairs | Qualified with interval +1,516..+3,275.5 ms and 378 identical outputs. |
| Generalized leaf compilation | **1.33% faster**, 196.25 ms saved, 3/4 positive pairs | Rejected: misses 500 ms, 2%, 4/4, and positive-bound gates. |
| Generalized leaf packaging | **3.73% faster**, 427.25 ms saved, 2/4 positive pairs | Rejected: misses 500 ms, 4/4, and positive-bound gates. |
| Verification and source distribution | Full graph, exact output | Generated graphs were incomplete; no performance claim was attempted. |

### OpenTelemetry Java Instrumentation: real public repository

| Experiment | Result | Interpretation |
|---|---:|---|
| Initial installed Spring-family transfer | **10.01% faster on average**, but 3/4 positive pairs and interval crossed zero | Favorable signal, not stable enough to qualify. |
| Typed graph reduction plus hot plan | **1.94% slower**, only 2/4 positive pairs | Correct outputs and fallback, but insufficient value. The result was retained rather than hidden. |
| Exact-bound hot-state reuse | BuildOpt preparation reduced from 74.97 ms to 40.34 ms (**46.2%**) | Internal planning improvement only; not a whole-build percentage. |
| Standard-`Jar` adapter | **39.92% faster**, 4,377 ms saved, 4/4 positive pairs | Qualified installed POC value with 125 identical outputs, zero product failures, and full 53-entrypoint fallback. |
| Clean Impact + standard-`Jar` composition | **50.40% faster**, 5,361.25 ms saved, 4/4 positive pairs | Qualified without hot state; paired interval +4,334.25..+5,937 ms, identical 125-file outputs, and successful full-graph fallback. |

## Latest Generalization and Next Work

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
qualify and remain on native Gradle. Incomplete verification and distribution
graphs, plus build-logic and global-configuration changes, all retained the
original full graph and completed successfully.

The next block is `POC-TASK-TAIL-ADAPTERS-001`: select the next exact standard
Gradle task adapter from real dominant-tail traces and measure it independently.
After that, the roadmap is to optimize build-owned test preparation without
changing Test execution, investigate Runtime Tuning only around measured
bottlenecks, compare Shared/Edge directly with native remote cache, and finally
transfer the unchanged profile to a third substantial public repository.

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
