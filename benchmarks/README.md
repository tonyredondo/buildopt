# Benchmarks

Reproducible workloads for measuring causal savings, overhead, queues, additional compute, and behavior under failure.

Seeds, images, toolchains, runner classes, and digests are part of the evidence. A benchmark never authorizes a security capability.

[`beta-v1.yaml`](./beta-v1.yaml) is the materialized `F0-032`
machine-readable workload, seed, budget, and fault matrix. Its interpretation
belongs in [`specs/benchmark-beta-v1.md`](../specs/benchmark-beta-v1.md).

The JSON-compatible YAML is validated by `./dev/check-beta-benchmark`. The
historical load/fault harnesses remain available as engineering evidence, but
the active POC does not run or require long soak qualification. Current effort
goes to paired, bounded build-time experiments against an optimized native
Gradle control. `./dev/check-beta-gradle-fixtures` owns the bounded
small/medium/large Gradle build matrix and makes no performance claim.

## Build Optimization scorecard

For a stakeholder-oriented interpretation of these results and the recommended
next experiments, see [Build Optimization performance findings](../docs/findings/build-optimization-performance.md).

The current POC verdict is `CONTINUE`, qualified only for the measured synthetic
workload classes. Contractual 4-vCPU/16-GiB runs cover the baseline,
negative-mechanism decision, accelerator-coverage matrix, combined public path,
and realistic breadth test. Safe Cache is explicit-only while the default delegates to Gradle's native
cache; Runtime Tuning candidates `W4_H6G` and `W3_H4G` are disabled; Build
Impact and the exact reviewed Task/Patch route clear the threshold across Kotlin
and Groovy; and the complete path also clears the final gate. Validate that
interpretation with:

```bash
./dev/check-poc-value-validation
```

### Fresh qualified-path ablation

The preregistered Spring/OpenTelemetry ablation is stored under
[`results/poc-full-path-ablation-v1`](./results/poc-full-path-ablation-v1).
The percentages below are independent effects against the contemporaneous
optimized native Gradle control; they are not additive.

| Repository and arm | Mean effect | Stability | Decision |
|---|---:|---:|---|
| Spring Build Impact | **2,492.375 ms / 30.86% faster** | 8/8 positive; interval +1,688.25..+3,242.25 ms | Qualified |
| OpenTelemetry Build Impact | **985.5 ms / 7.49% faster** | 3/4 positive; interval -737.5..+2,708.5 ms | Favorable but not qualified |
| OpenTelemetry Impact + hot state | **892 ms / 7.68% slower** | 1/4 positive; interval -1,451.5..+111.5 ms | Rejected |
| OpenTelemetry Impact + hot state + standard `Jar` adapter | **4,496.75 ms / 40.60% faster** | 4/4 positive; interval +3,928.5..+5,467.25 ms | Adapter arm qualified; composition rejected |

All source protocols preserved their declared outputs and fallback contracts.
The aggregate decision is `RETAIN_COMPONENT_EVIDENCE_ONLY`: the fast terminal
OpenTelemetry arm cannot hide the included hot-state regression. Protocol
revision 2 tightened this composition rule after measurements without changing
raw reports, thresholds, pairs, or source decisions.

The subsequent [clean-composition evidence](./results/poc-otel-clean-composition-v1.json)
measures Build Impact plus the standard `Jar` adapter **without hot-state
reuse** against the same optimized native control. Native Gradle averaged
10,636.5 ms and the installed candidate averaged 5,275.25 ms, saving 5,361.25
ms or **50.40%**. Pair savings were +3,825, +5,862, +5,995, and +5,763 ms; the
paired interval is +4,334.25..+5,937 ms. All four candidate arms restored
`:testing-common:jar FROM-CACHE`, preserved the same 125-file output digest,
enabled no hot state or managed runtime, and introduced no product failure. A
separate global change retained all 53 native entrypoints and completed
successfully. The terminal decision is `QUALIFY_CLEAN_OTEL_COMPOSITION`.

### Build Impact generalization

The checked [Spring generalization evidence](./results/poc-impact-generalization-v1.json)
compares installed BuildOpt revision `f5708fb32a0c0b826553f741d198e2982eed0c21`
with optimized native Gradle across compilation, test preparation, packaging,
verification, and source distribution. All performance cells use four
alternating offline pairs, include launcher and planner overhead, restore the
same native cache seed, and preserve byte-identical non-empty required outputs.

| Output family | Native mean | BuildOpt mean | Effect | Stability | Decision |
|---|---:|---:|---:|---:|---|
| Leaf compilation | 14,791.25 ms | 14,595 ms | 196.25 ms / 1.33% faster | 3/4 positive; interval -1,722.25..+1,901 ms | Not qualified |
| Shared test preparation | 13,971.75 ms | 11,333.75 ms | **2,638 ms / 18.88% faster** | 4/4 positive; interval +1,516..+3,275.5 ms | Qualified |
| Leaf packaging | 11,442 ms | 11,014.75 ms | 427.25 ms / 3.73% faster | 2/4 positive; interval -264.75..+1,433.5 ms | Not qualified |

Verification and source-distribution discovery produced incomplete graphs, so
the candidate retained the original full selector and produced the exact same
report/JAR rather than making a performance claim. Separate `buildSrc` and
`gradle.properties` mutations also returned `IMPACT_GLOBAL_CHANGE`, executed
the complete `classes` selector, and succeeded. The terminal decision is
`GENERALIZE_ONLY_QUALIFIED_CELLS`: only shared test preparation broadens the
Build Impact POC claim.

### Verification and distribution graph completion

The checked
[verification/distribution evidence](./results/poc-verification-distribution-graph-v1.json)
replaces the former name-based fallback with public Gradle task contracts.
On the fixed Spring revision, `checkstyleMain` narrows from 23 projects to a
complete 12-project Spring MVC dependency closure; `sourcesJar` narrows from
22 projects to a complete 12-project closure. Both contain no `Test` and add no
Spring-specific task rule.

Only verification was preregistered for timing. Optimized native Gradle
averaged 33,916 ms and installed BuildOpt averaged 33,812.25 ms, an apparent
103.75-ms/0.31% saving. Pair effects were -5,158, +2,921, -1,136 and +3,788 ms:
only 2/4 were positive and the lower bound was -5,158 ms. The exact Checkstyle
report matched in every pair and global fallback passed, but the terminal
decision is `RETAIN_VERIFICATION_FULL_GRAPH`. Graph capability is complete;
stable build-time value is not.

The checked [verification attribution evidence](./results/poc-verification-overhead-attribution-v1.json)
explains why this line stops. The operation trace contains 143 control and 51
candidate task rows. The 92 omitted rows total 4,249 ms of overlapping work,
but 35 are native-cache hits and another 39 are no-action, no-source, skipped
or up-to-date. The candidate task interval is diagnostically 1,581 ms shorter,
while BuildOpt's largest own phase is only 1.238233 ms of impact preparation.
No named product phase reaches the frozen 500-ms correction threshold. These
trace durations do not replace the four-pair result; the decision is
`STOP_VERIFICATION_OPTIMIZATION_NO_ACTIONABLE_BOTTLENECK`.

### Exact standard-Copy cascade

The checked [standard-Copy cascade evidence](./results/poc-standard-copy-cascade-v1.json)
measures the next trace-selected task adapter in three separate four-pair
comparisons. The direct complete-profile comparison captures graph reduction,
task reuse, critical-path changes, and their interaction; component percentages
are never added.

| Comparison | Native/control mean | Candidate mean | Effect | Stability | Decision |
|---|---:|---:|---:|---:|---|
| Copy adapter only | 15,840.5 ms | 11,556.25 ms | 4,284.25 ms / 27.05% faster | 3/4 positive; interval -3,334.25..+8,846.5 ms | Not qualified |
| Copy added to Impact + Jar | 9,603 ms | 7,211.75 ms | 2,391.25 ms / 24.90% faster | 3/4 positive; interval -2,568.25..+7,092 ms | Not qualified |
| Complete Impact + Jar + Copy versus native | 8,276 ms | 3,899 ms | **4,377 ms / 52.89% faster** | 4/4 positive; interval +4,130.25..+4,653.25 ms | Complete profile qualified |

All twelve observations preserve the same non-empty 21,818-file digest and the
global-change probe completes through the full native graph. The terminal
decision is `RETAIN_STANDARD_COPY_EVIDENCE_ONLY`: the whole profile proves a
stable cascade, but the direct incremental gate does not authorize activating
Copy. This prevents a fast terminal profile from concealing an unstable
component.

### Normal-build task-tail closure

The checked [normal-build task-tail review](./results/poc-normal-build-tail-expansion-v1.json)
re-evaluates the remaining retained Spring and OpenTelemetry tails before any
more adapter code is authorized. It adds no timing claim: source evidence is
bound by SHA-256 and checked against the existing 500-ms, exact-standard-task,
bounded-effects, native-cache, Test-ownership, and incremental-value gates.

The review finds no actionable candidate. Standard `Jar` is already active;
standard `Copy` failed its direct incremental gate; custom `ShadowJar` is below
the floor and already restored from native cache; configured `JavaExec` is
below the floor and has broader process effects. The terminal decision is
`STOP_NORMAL_BUILD_TASK_ADAPTER_EXPANSION_NO_ACTIONABLE_TAIL`, so the POC moves
to build-owned test work instead of implementing a generic adapter merely to
continue the sequence.

### Spring test-build JAR reuse

The checked [Spring test-build evidence](./results/poc-spring-test-build-optimization-v1.json)
measures a fixed `:spring-jms:test` filter while BuildOpt changes only
build-owned standard-JAR eligibility. Every arm executes the same 8 tests with
zero failures, errors or skips; every pair preserves the same 15 JARs and test
case set. BuildOpt restores three exact `testFixturesJar` producers per pair,
while native Gradle executes them.

That technical success does not translate into value. Native Gradle averages
6,503.5 ms and BuildOpt 7,238.75 ms: BuildOpt loses 735.25 ms/11.31%, has 0/4
positive pairs and a -1,449..-113-ms interval. The terminal decision is
`STOP_STANDARD_JAR_REUSE_FOR_SPRING_TEST_BUILD`; the switch remains
diagnostic-only for reproducibility, while user-facing activation remains
limited to its already qualified Build Impact scope.

### Optimization-overhead ablation

The checked [three-arm Spring evidence](./results/poc-optimization-overhead-ablation-v1.json)
measures the same filtered test workflow after narrowing standard-JAR
registration to one root plugin and typed `Jar` task collections. Four rotated
rounds compare optimized native Gradle, native Gradle loading only BuildOpt's
init/plugin classpath, and the exact adapter restoring three standard JARs.

| Arm | Mean | Difference |
|---|---:|---:|
| Optimized native Gradle | 6,422.75 ms | Control |
| Init/plugin classpath only | 7,484.50 ms | 1,061.75 ms slower than native |
| Exact standard-JAR adapter | 7,035.00 ms | 449.50 ms faster than init-only; **612.25 ms/9.53% slower than native** |

The adapter is positive in only 2/4 rounds and its paired interval is
-1,785.75..+235 ms. Every arm runs the same eight tests, produces the same 15
JARs (6,728,787 bytes), and has zero product-attributable failures. The
init-only differences vary materially with arm order, so they diagnose where
to investigate rather than authorizing a causal fixed-cost claim. The only
activation decision comes from the complete native comparison:
`KEEP_NATIVE_FOR_UNQUALIFIED_STANDARD_JAR_WORKFLOW`.

### Targeted Runtime Tuning research

The checked [Spring Runtime evidence](./results/poc-runtime-research-v1.json)
tests one preregistered resource hypothesis rather than searching profiles
until one wins. Both arms run the same offline `testClasses` workload with the
same heap, cache state, source mutation, task outcomes and 378 required class
outputs; only `--max-workers` changes from the native 12 to 6.

| Arm | Mean | Difference |
|---|---:|---:|
| Optimized native Gradle, 12 workers | 9,556.75 ms | Control |
| Bounded Runtime candidate, 6 workers | 9,748.25 ms | **191.5 ms/2.00% slower** |

The candidate is positive in only 2/4 pairs and its paired interval is
-973.5..+590.5 ms. Every pair preserves the same 378 outputs and sorted task
outcomes, with zero product failures. The terminal decision is
`RETAIN_NATIVE_12_WORKERS`: Runtime Tuning remains disabled for this workload,
and the frozen protocol forbids another worker search after this result.

### Controlled remote-cache locality

The checked [remote-cache evidence](./results/poc-remote-cache-value-v1.json)
isolates one variable: where Gradle reads the same eight committed Shared
objects. Both arms use Gradle 9.6.1 `HttpBuildCache`, identical authentication,
disabled local and Configuration caches, read-only measured runs and 32 MiB of
byte-identical required outputs. The control reads Shared through a fixed
80-ms/20-MiB/s modeled WAN; the candidate reads a prewarmed BuildOpt Edge on
loopback.

| Arm | Mean | Measured Shared traffic |
|---|---:|---:|
| Gradle HTTP cache direct to Shared | 6,911.25 ms | 8 GETs / 33,569,614 bytes per pair |
| Gradle HTTP cache through prewarmed Edge | 4,510.00 ms | 0 GETs / 0 bytes per pair |

Edge saves **2,401.25 ms/34.74%**, all four pairs are positive (+2,605,
+2,307, +2,479 and +2,214 ms), and the deterministic paired interval is
+2,260.5..+2,542 ms. All tasks are `FROM-CACHE`, required outputs and task
outcomes are identical, and there are zero product-attributable failures. The
terminal decision is `QUALIFY_EDGE_LOCALITY_FOR_CONTROLLED_REMOTE_CACHE_POC`.
This is a locality result under the frozen network profile, not evidence that
Shared storage itself is faster than another Gradle-compatible origin.

### Real-repository Edge transfer

The checked [Kafka transfer evidence](./results/poc-remote-cache-transfer-v1.json)
moves the unchanged Shared/Edge implementation to Apache Kafka 4.3.1
`:clients:testClasses`. Its network profile was frozen before timing from the
median of three SHA-identical source-archive downloads: 337 ms per response and
6,994,831 bytes/s. Both arms use Gradle 9.2.1 native `HttpBuildCache`, the same
six committed objects, prepared dependency state, disabled local and
Configuration caches, and the same 4,062 required outputs.

| Arm | Mean | Measured Shared traffic |
|---|---:|---:|
| Gradle HTTP cache direct to Shared | 8,885.25 ms | 4 GETs / 7,652,777 bytes per pair |
| Gradle HTTP cache through prewarmed Edge | 7,534.00 ms | 0 GETs / 0 bytes per pair |

Edge saves **1,351.25 ms/15.21%**. All four pairs are positive (+2,107,
+1,211, +533 and +1,554 ms), and the paired interval is
+788.25..+1,883 ms. All four cache-hit outcomes, 4,062 outputs and their
24,722,721 bytes are identical. This transfers the locality result to a real
repository and a materially different network shape; it remains an opt-in POC
claim rather than universal or production evidence.

```bash
./dev/check-poc-remote-cache-transfer-v1-result \
  benchmarks/results/poc-remote-cache-transfer-v1.json
```

```bash
./dev/check-poc-full-path-ablation
./dev/test-poc-full-path-ablation
./dev/check-poc-otel-clean-composition-v1-result
./dev/test-poc-otel-clean-composition-v1
./dev/check-poc-impact-generalization
./dev/test-poc-impact-generalization
./dev/check-poc-standard-copy-cascade \
  benchmarks/results/poc-standard-copy-cascade-v1.json
./dev/check-poc-normal-build-tail-expansion
./dev/test-poc-normal-build-tail-expansion
./dev/check-poc-spring-test-build-optimization \
  benchmarks/results/poc-spring-test-build-optimization-v1.json
./dev/check-poc-optimization-overhead-ablation \
  benchmarks/results/poc-optimization-overhead-ablation-v1.json
./dev/check-poc-runtime-research \
  benchmarks/results/poc-runtime-research-v1.json
./dev/check-poc-remote-cache-value \
  benchmarks/results/poc-remote-cache-value-v1.json
```

The scorecard answers a different question for each optimization instead of
combining unrelated percentages:

The current substantial public-repository qualification is Spring Framework
test preparation. Generic root-build discovery reduced native `testClasses`
from 7,306.125 ms to 5,208 ms on average: 2,098.125 ms/28.72% saved, eight of
eight positive pairs, and a paired bootstrap interval of +1,464.5 to +2,780 ms.
The [checked evidence](./results/poc-spring-test-preparation-v2.json) preserves
all 378 affected class outputs, runs no root-build `Test`, retains the common
`:buildSrc:test UP-TO-DATE`, and records zero product failures. This qualifies
only unchanged transfer to OpenTelemetry; it is not a general or production
claim. That transfer was executed but did not qualify: six completed
OpenTelemetry pairs were all positive and descriptively saved 18,560.667 ms
(59.03%) on average, then the native control failed in pair 7 inside upstream
Byte Buddy processing with `zip file closed`. The protocol forbids discarding
the failed pair, so pair 8 was not started and the terminal decision is
`STOP_OTEL_TRANSFER_INCOMPLETE_UPSTREAM_CONTROL`. See the [checked terminal
evidence](./results/poc-otel-test-preparation-v1.json); the six-pair diagnostic
is not an eight-pair value claim.

The same fixed Spring mutation was then exercised through an isolated installed
native package and the public `buildopt impact` command. The optimized native
control averaged 7,996.125 ms and the installed candidate 6,736 ms, saving
1,260.125 ms/15.76%. All eight pairs were positive, the paired bootstrap
interval was +624.125 to +2,019 ms, all 378 declared outputs were identical,
and launcher plus manifest/graph validation overhead was included. This is a
POC claim for that reviewed output scope, not authorization to replace Spring's
root `testClasses` graph generally.

Validate the installed-path evidence with:

```bash
./dev/check-poc-spring-installed-impact
```

The complete report is
[`poc-spring-installed-impact-v1.json`](./results/poc-spring-installed-impact-v1.json).

The preregistered installed breadth matrix then tested two additional Spring
scopes. `WEBMVC_LEAF` reduced the mean from 15,962 to 13,807.5 ms, saving
2,154.5 ms/13.50% with interval +880.5..+3,428.5 ms and 4/4 positive pairs.
`CORE_TO_JMS_SHARED` reduced the mean from 16,435.75 to 14,645.25 ms, saving
1,790.5 ms/10.89% with interval +378..+3,128.5 ms, but one of four pairs was
380 ms slower. The shared cell therefore failed the frozen 4/4-positive rule.
The global `gradle.properties` cell correctly retained root `testClasses`.
The terminal decision is `RETAIN_SINGLE_INSTALLED_SPRING_SCOPE`: the matrix
does not authorize a broad shared-change claim, even though the webmvc cell
qualified and the shared mean was favorable.

Validate the full matrix with:

```bash
./dev/check-poc-spring-impact-breadth
```

The subsequent stable OpenTelemetry Spring-family transfer completed all four
installed-package pairs. Native Gradle averaged 14,961.5 ms and BuildOpt
13,464.5 ms, a descriptive saving of 1,497 ms/10.01%, with exact non-empty
outputs and zero product failures. It did not qualify: one pair regressed by
465 ms and the paired interval crossed zero at -61.5..+3,671.5 ms. The separate
global-change probe safely restored all 53 entrypoints. The terminal decision
is `RETAIN_SPRING_ONLY_INSTALLED_EVIDENCE`; the favorable mean does not
authorize a stable OpenTelemetry claim or another retry.

Validate the terminal evidence with:

```bash
./dev/check-poc-otel-spring-family-v2-result \
  benchmarks/results/poc-otel-spring-family-v2.json
```

The installed-path phase profile then isolated BuildOpt from Gradle by running
the exact validated 435,875-byte OpenTelemetry graph through a neutral child.
Across 16 retained samples, BuildOpt-owned preparation averaged 47.70 ms:
36.43 ms for graph load/validation, 9.74 ms for impact evaluation and about
1.29 ms for the remaining planner/setup work. Total launcher time including
the neutral child averaged 49.51 ms and never exceeded 77.39 ms. This is too
small to explain the earlier 465-ms regressive pair; the next optimization
target is the selected Gradle work and its hot-state stability, not generic
launcher micro-optimization. This diagnostic makes no build-time claim.

Validate all 16 samples and recomputed means with:

```bash
./dev/check-poc-otel-overhead
```

The next graph probe replaced the aggregate `testClasses` candidate with four
typed `AbstractCompile` output producers. Gradle discovery remained complete
for all 1,024 projects and the same 46-project reach; arbitrary custom tasks
and every `Test` dependency still fail closed. In a hot output-reset
comparison, the producer set removed three task nodes and two executed tasks
while preserving the exact same 125-file output manifest. This closes a real
generic work reduction but deliberately makes no material timing claim.

```bash
./dev/check-poc-otel-graph-reduction
```

Exact-bound POC hot state then reduced BuildOpt preparation from a 74.97-ms
miss to 40.34 ms across 16 hits (46.2%), with identical selection. Changes,
Wrapper or revision drift all miss and re-enter the normal planner.

```bash
./dev/check-poc-otel-hot-state
```

The subsequent terminal OpenTelemetry run retained every preregistered pair.
Typed producers and exact hot state preserved all 125 outputs and the global
full-graph fallback, but did not stabilize wall-clock value. Native Gradle
averaged 13,382.5 ms and BuildOpt 13,641.75 ms: BuildOpt was 259.25 ms/1.94%
slower. Pair differences were `+2729`, `-67`, `-4019`, and `+320` ms, producing
a paired interval of −2,934.25..+2,030 ms and only 2/4 positive pairs. The
frozen gate therefore remains open; these measurements must not be retried or
filtered as if they qualified.

```bash
./dev/check-poc-otel-optimization-v1-result \
  benchmarks/results/poc-otel-optimization-v1.json
```

Task attribution then identified Gradle's repeated non-cacheable standard
`:testing-common:jar` packaging as the dominant candidate tail. The explicit
POC adapter makes only an unmodified standard `Jar` eligible for Gradle's
native cache; custom `Jar`, `Copy`, arbitrary and Test tasks remain unchanged.
The newly preregistered installed-path experiment used one shared hot Gradle
runtime and completed all four pairs at `+6158`, `+3661`, `+4004`, and `+3684`
ms. Native Gradle averaged 10,964.75 ms and BuildOpt 6,588 ms: a 4,376.75-ms
or 39.92% mean saving. The paired interval was +3,672.5..+5,539.5 ms and all
4/4 pairs were positive. Every pair preserved the same 125-file digest, zero
product failures occurred, and the global change restored all 53 entrypoints.

```bash
./dev/check-poc-otel-optimization-v2-result \
  benchmarks/results/poc-otel-optimization-v2.json
```

This qualifies stable POC value for the exact OpenTelemetry Spring-family
build-preparation workload. It is not a production-wide Jar policy or a claim
that unrelated builds receive the same percentage.

| Mechanism | Mean result | Exact paired 95% interval | Classification |
|---|---:|---:|---|
| Default native-cache fallback, Kotlin | 79 ms faster (8.9%) | +6 to +156 ms | `NO_VALUE_NO_ACTION`; same cache mechanism, no acceleration claim |
| Default native-cache fallback, Groovy | 1,051 ms faster (56.6%) | +486 to +1,572 ms | `NO_VALUE_NO_ACTION`; same cache mechanism, no acceleration claim |
| Runtime Tuning `W3_H4G` | 512 ms slower (4.3%) | −2,818 to +1,302 ms | `NO_VALUE_NO_ACTION`; `STABLE_CONTROL_ONLY` |
| Build Impact, Kotlin | 1,939 ms faster (76.0%) | +1,899 to +1,982 ms | `THRESHOLD_MET_REQUIRED_BREADTH` |
| Build Impact, Groovy | 2,155 ms faster (73.5%) | +1,869 to +2,414 ms | `THRESHOLD_MET_REQUIRED_BREADTH` |
| Reviewed Task/Patch, Kotlin | 1,369 ms faster (67.3%) | +1,142 to +1,624 ms | `THRESHOLD_MET_REVIEWED_CUSTOM_TASK` |
| Reviewed Task/Patch, Groovy | 2,349 ms faster (68.0%) | +1,245 to +3,421 ms | `THRESHOLD_MET_REVIEWED_CUSTOM_TASK` |
| Combined Impact, Kotlin | 2,193 ms faster (77.5%) | +2,058 to +2,397 ms | `THRESHOLD_MET_REQUIRED_BREADTH` |
| Combined Impact, Groovy | 3,265 ms faster (84.1%) | +2,518 to +3,912 ms | `THRESHOLD_MET_REQUIRED_BREADTH` |
| Combined reviewed Task/Patch, Kotlin | 1,441 ms faster (67.3%) | +1,159 to +1,722 ms | `THRESHOLD_MET_REQUIRED_BREADTH` |
| Combined reviewed Task/Patch, Groovy | 1,905 ms faster (63.5%) | +724 to +3,055 ms | `THRESHOLD_MET_REQUIRED_BREADTH` |

The reports retain all four signed pairs, including unfavorable samples. Every
required output is identical, Runtime Tuning has zero OOM delta, and no
product-attributable failures occurred. The apparent fallback timing difference
is not attributable to BuildOpt because control and candidate both use Gradle's
native cache; its evidence closes regression removal only. Percentages are not
added because the mechanisms use different controls and workloads.
`POC-VALUE-001..004` are completed decision gates. The final `CONTINUE` verdict
means only that the idea merits more POC work for the qualified synthetic
classes. The reviewed Patch result applies only to
`CUSTOM_TASK_CONTRACT_JAVA_V1`; it does not qualify unrelated recipes.

Validate all checked-in evidence and print the machine-readable scorecard:

```bash
./dev/check-build-optimization-performance
```

The underlying evidence and contracts are:

- [contractual POC baseline](./results/poc-value-baseline-v1.json) and
  [value contract](../specs/poc-value-validation-v1.md);
- [negative-mechanism decisions](./results/poc-value-negative-mechanisms-v1.json),
  validated by `./dev/check-poc-value-negative-mechanisms`;
- [accelerator coverage matrix](./results/poc-value-coverage-v1.json),
  validated by `./dev/check-poc-value-coverage`;
- [combined public-path matrix](./results/poc-value-combined-v1.json),
  validated by `./dev/check-poc-value-combined`;
- [initial realistic change-class matrix](./results/poc-breadth-v1.json) and
  [post-attribution repeat](./results/poc-breadth-v2.json), validated by
  `./dev/check-poc-breadth`;
- [installed-path phase attribution](./results/poc-overhead-v1.json), validated
  by `./dev/check-poc-overhead`;
- [isolated control-first](./results/poc-stability-v1-control-first.json) and
  [candidate-first](./results/poc-stability-v1-candidate-first.json) stability
  reports plus their [checked decision](./results/poc-stability-v1-decision.json),
  validated by `./dev/check-poc-stability`;
- [temporally paired control-first](./results/poc-pairing-v1-control-first.json)
  and [candidate-first](./results/poc-pairing-v1-candidate-first.json) reports
  plus their [checked decision](./results/poc-pairing-v1-decision.json),
  validated by `./dev/check-poc-pairing`;
- [calibrated Groovy control-first](./results/poc-groovy-v1-control-first.json)
  and [candidate-first](./results/poc-groovy-v1-candidate-first.json) reports
  plus their [checked decision](./results/poc-groovy-v1-decision.json),
  validated by `./dev/check-poc-groovy`;
- [shared-source Groovy control-first](./results/poc-shared-groovy-v1-control-first.json)
  and [candidate-first](./results/poc-shared-groovy-v1-candidate-first.json)
  reports plus their [checked decision](./results/poc-shared-groovy-v1-decision.json),
  validated by `./dev/check-poc-shared-groovy`;
- [leaf-source Kotlin control-first](./results/poc-leaf-kotlin-v1-control-first.json)
  and [candidate-first](./results/poc-leaf-kotlin-v1-candidate-first.json)
  reports plus their [checked decision](./results/poc-leaf-kotlin-v1-decision.json),
  validated by `./dev/check-poc-leaf-kotlin`;
- [remaining Kotlin control-first](./results/poc-kotlin-stability-v1-control-first.json)
  and [candidate-first](./results/poc-kotlin-stability-v1-candidate-first.json)
  reports plus their [checked decision](./results/poc-kotlin-stability-v1-decision.json),
  validated by `./dev/check-poc-kotlin-stability`;
- [terminal Kotlin boundary decision](./results/poc-kotlin-boundary-v1.json),
  validated by `./dev/check-poc-kotlin-boundary`;
- [public-repository compatibility evidence](./results/poc-real-world-compatibility-v1.json),
  validated by `./dev/check-poc-real-world-compatibility`;
- [public-repository performance evidence](./results/poc-real-world-performance-v1.json),
  validated by `./dev/check-poc-real-world-performance`;
- [public-workflow diagnostic evidence](./results/poc-real-world-diagnostics-v1.json),
  validated by `./dev/check-poc-real-world-diagnostics`;
- [Spring Framework diagnostic evidence](./results/poc-spring-diagnostic-v1.json),
  validated by `./dev/check-poc-spring-diagnostic`;
- [Spring Framework test-preparation evidence](./results/poc-spring-test-preparation-v2.json),
  validated by `./dev/check-poc-spring-test-preparation-result`;
- [OpenTelemetry terminal transfer evidence](./results/poc-otel-test-preparation-v1.json),
  validated by `./dev/check-poc-otel-test-preparation-result`;
- [Spotless exact-workflow Build Impact evidence](./results/poc-spotless-impact-v1.json),
  validated by `./dev/check-poc-spotless-impact`;
- [safe-cache observations](./results/cache-parity-v1-local.json) and
  [contract](../specs/cache-parity-v1.md);
- [Runtime Tuning observations](./results/b-runtime-owner-evaluation.json) and
  [contract](../specs/runtime-owner-evaluation-v1.md);
- [Build Impact observations](./results/build-impact-performance-v1-local.json)
  and [contract](../specs/build-impact-performance-v1.md).
- [Build Impact generalization evidence](./results/poc-impact-generalization-v1.json),
  validated by `./dev/check-poc-impact-generalization` and
  `./dev/test-poc-impact-generalization`.
- [Apache Kafka third-repository transfer evidence](./results/poc-third-repository-transfer-v1.json),
  validated by `./dev/check-poc-third-repository-transfer-v1-result`.
- [Apache Kafka packaging evidence](./results/poc-kafka-packaging-v1.json),
  validated by `./dev/check-poc-kafka-packaging-v1-result`.

### Third substantial repository transfer

The unchanged clean Build Impact plus exact standard-`Jar` profile qualifies
on Apache Kafka 4.3.1. Four alternating offline pairs compare optimized native
root `testClasses` with installed BuildOpt selecting `:clients:testClasses`.
Native averages 4,609.5 ms and BuildOpt 2,070 ms, saving 2,539.5 ms/55.09%; all
four pairs are positive and the interval is +1,625.5..+4,093 ms. Every pair
preserves the same 4,062 required output files, the candidate alone restores
`:generator:jar`, no Gradle `Test` executes, and a build-logic change restores
the full 64-project graph. This is output-scoped POC transfer evidence, not a
universal savings or production-readiness claim.

### Installed qualified-profile adoption

The [adoption result](./results/poc-qualified-profile-adoption-v1.json) checks
the public POC workflow after qualification rather than measuring it again. An
installed Linux AMD64 package consumes the committed OpenTelemetry and Kafka
profiles, emits the plan before Gradle, replays the exact standard `Jar`, and
reproduces the historical 125- and 4,062-file output digests. Global
`gradle.properties` changes retain the native full graph and reach work outside
each candidate scope. `./dev/check-poc-qualified-profile-adoption` validates
the record. It contains no durations and creates no new performance claim.

### Kafka client packaging value

The [preregistered packaging result](./results/poc-kafka-packaging-v1.json)
compares optimized native root `assemble` with an installed qualified profile
selecting `:clients:jar` for the same fixed `Metadata.java` change. Both arms
use Gradle 9.2.1, JDK 25, separate warm Gradle homes, offline execution and the
same 12-CPU host.

| Pair | Order | Native `assemble` | BuildOpt client JAR | Saving |
|---:|---|---:|---:|---:|
| 1 | Native to BuildOpt | 9,811 ms | 3,941 ms | 5,870 ms |
| 2 | BuildOpt to native | 7,947 ms | 3,830 ms | 4,117 ms |
| 3 | Native to BuildOpt | 7,137 ms | 3,087 ms | 4,050 ms |
| 4 | BuildOpt to native | 7,321 ms | 2,808 ms | 4,513 ms |

Native averages 8,054 ms and BuildOpt 3,416.5 ms, saving **4,637.5 ms or
57.58%**. All four pairs are positive, the conservative lower bound used by
this contract is +4,050 ms, and every arm produces the exact
SHA-256-bound 10,204,023-byte client JAR. No Gradle `Test` runs, no unqualified
mechanism is enabled, product failures are zero, and a `gradle.properties`
change completes native root `assemble` outside the candidate graph. The claim
is limited to this Kafka client-packaging change shape.

The three mechanism-development reports remain historical inputs. The strict
synthetic reports prove bounded combined value. The public-repository
compatibility and early performance reports showed that the first generic
candidate did not transfer uniformly. Later preregistered Spring,
OpenTelemetry, and Kafka experiments qualify narrower mechanisms and the clean
profile for exact output scopes. The evidence therefore supports bounded
transfer, not universal savings or production readiness.

### Public-repository performance result

`POC-REALWORLD-002` ran 48 preregistered pairs (96 measured builds) on exact
Spotless, Mockito, and SpotBugs revisions in the digest-pinned 4-CPU/16-GiB
runner. Native Gradle and installed BuildOpt used separate homes, persistent
private daemons, offline execution, opposite starting orders, byte-identical
required outputs, and the unchanged 2% parity and 500-ms/2%/positive-bound
accelerator thresholds.

| Repository | No change | Leaf source | Repository decision |
|---|---:|---:|---|
| Mockito | 239.25 ms / 9.74% faster; parity | 744.25 ms / 18.22% faster; 95% interval `[77, 1526.25]` ms | `QUALIFIED` |
| SpotBugs | 72.75 ms / 3.06% slower; outside parity | 384.125 ms / 14.23% faster; 95% interval `[-659.25, 1459.125]` ms | `RETAIN_BOUNDED_CLAIM` |
| Spotless | 7.375 ms / 1.02% slower; parity | 31.125 ms / 1.89% faster; 95% interval `[-6.375, 69.5]` ms | `RETAIN_BOUNDED_CLAIM` |

All required outputs were identical and product-attributable failures were
zero. Only Mockito qualified. Spotless omitted work but did not contain enough
avoidable work to clear the threshold; SpotBugs neither retained no-change
parity nor produced stable absolute leaf savings. The terminal verdict is
`RETAIN_BOUNDED_SYNTHETIC_CLAIM`: no thresholds move, percentages are not
combined, and no general public-repository claim is authorized.

### Public-workflow diagnostic result

`POC-REALWORLD-DIAGNOSTICS-001` profiles the exact upstream workflows once;
it is not a candidate-versus-control benchmark and claims no savings. The
digest-pinned 4-CPU/16-GiB run records wall time and native Gradle profile
phases without summing overlapping task durations.

| Repository | Exact workflow wall time | Startup + configuration | Diagnostic decision |
|---|---:|---:|---|
| Spotless | 165.173 s | 2.561 s (1.55%) | Preregister Build Impact on the exact workflow, including `testClasses` |
| Mockito | 629.165 s | 2.957 s (0.47%) | Preregister value measurement for build-owned `compileTestJava` |
| SpotBugs | 271.920 s | 1.790 s (0.66%) | No material build-preparation hypothesis; requested `Test` execution dominates |

Spotless already enables native parallel execution, build cache, and
Configuration Cache. Its expensive work spans independent projects, so the
supported follow-up is change-aware project scoping with byte-identical main
and test classes. The original `E-171` decision treated Mockito's test-heavy
workflow as if all of that cost were outside BuildOpt.
[`poc-public-build-tasks-v1`](../specs/poc-public-build-tasks-v1.md) corrects
that interpretation without changing the raw profile: the 593.290-second
Mockito build spends 242.690 seconds (40.91%) in the build-owned
`:mockito-core:compileTestJava` task over 402 sources. It now receives a
separate test-build value experiment against optimized native Gradle. SpotBugs
spends 242.120 seconds (89.0% of wall time) in `:spotbugs-tests:test`, while
its visible `compileTestJava` cost is only 1.119 seconds, so no material
build-preparation hypothesis is registered for that workflow.

Neither preregistration claims savings. Spotless must preserve both production
and test classes. Mockito must beat, not merely match, the optimized native
cache before the mechanism may be tested in the complete workflow with every
requested test unchanged.

### Spotless exact-workflow Build Impact result

`POC-SPOTLESS-IMPACT-001` measured eight offline, alternating pairs in the
digest-pinned 4-CPU/16-GiB runner. Both arms ran the complete
`spotlessCheck`; only the second command differed between the root aggregate
and the affected `plugin-gradle` subgraph. The candidate omitted
`:lib-extra:compileTestJava`, produced the same 209 required class files and
executed no Gradle `Test` task.

The control averaged 7,030.875 ms and the candidate 6,734.5 ms: 296.375 ms
(4.22%) saved, with 5/8 positive pairs and a paired-bootstrap 95% interval of
`[-55.5, 662]` ms. This fails the frozen 500-ms and positive-lower-bound gates,
so the terminal decision is `STOP_SPOTLESS_ALTERNATIVE`. The signed samples
remain evidence, but no savings claim or exact-workflow activation is allowed.

### Mockito test-build Safe Cache result

`POC-MOCKITO-TEST-BUILD-001` measured eight offline, alternating pairs on the
exact Mockito revision in the digest-pinned 4-CPU/16-GiB runner. Both arms ran
the unchanged `:mockito-core:testClasses` graph with Gradle 8.14.2/JDK 21 and
restored `:mockito-core:compileTestJava`; BuildOpt used its private Tier One L1
while control used Gradle's optimized native local cache. Every pair produced
the same 1,260 test-class files, executed no Gradle `Test`, and had no product
failure.

Control averaged 2,385 ms and BuildOpt 2,103.625 ms: an apparent 281.375 ms
(11.80%) saving, with 5/8 positive pairs and a paired-bootstrap 95% interval
of `[-498.5, 1109.25]` ms. This misses the frozen 500-ms floor and positive
lower bound, so the terminal decision is
`STOP_SAFE_CACHE_FOR_MOCKITO_TEST_BUILD`. The complete Mockito workflow was
therefore not run, and no savings claim or unchanged rerun is authorized.

### Spring Framework diagnostic

`POC-SPRING-DIAGNOSTIC-001` profiles a pinned Spring Framework revision on the
local Linux AMD64 host with all 12 CPUs and Temurin 25. The online preflight is
excluded from measurement. Each offline cell starts with clean project outputs
and the exact same post-preflight native Gradle cache seed.

| Cell | Wall clock | Executed / cache hits | Configuration phases | Tests |
|---|---:|---:|---:|---:|
| `assemble` | 33.320 s | 45 / 38 | 15.882 s | 4 implicit `buildSrc` tests |
| `testClasses` | 31.850 s | 67 / 83 | 13.860 s | 4 implicit `buildSrc` tests |
| `check` | 73.290 s | 118 / 176 | 12.918 s | 41,276; 0 failed; 208 skipped |

The profile shows two distinct opportunities. `spring-jms:compileJava` remains
real changed-project work, while `assemble` and `testClasses` still configure
the full 27-project graph despite a leaf change. That authorizes a paired Build
Impact experiment. The complete `check` run is dominated by the 52.374-second
`checkstyleNohttp` task and retains every test, so only Runtime Tuning is
authorized there. Task durations overlap and are not added into a wall-clock
savings claim. This is diagnostic evidence, not proof that BuildOpt is faster.

### Calibrated Groovy result

`POC-GROOVY-001` attributed the reproduced no-change regression to a redundant
launcher process on the uninstrumented native Gradle path and replaced that
process in place on Unix. The generic fixture remains unchanged. A bounded
Groovy profile increases only the existing deterministic non-cacheable
verification rounds so the leaf scenario contains enough avoidable work to
test the unchanged 500-ms threshold honestly.

Both arms run consecutively in one strict 4-CPU/16-GiB container to remove
container identity as a timing variable. They retain separate workspaces,
installations, writable state, Gradle homes, and daemons.

| Starting order | No change | Leaf source |
|---|---:|---:|
| Control first | 196.25 ms / 10.4% faster; parity met | 884.25 ms / 41.2% faster; threshold met |
| Candidate first | 71.75 ms / 3.9% faster; parity met | 641.125 ms / 33.8% faster; threshold met |

All 32 observations preserved byte-identical required outputs, exact execution
shape, valid Configuration Cache behavior, and zero product-attributable
failures. These percentages describe separate batches and are not averaged or
added. The result broadens only the bounded Groovy no-change/leaf claim; it is
not a universal or production claim.

The same calibrated boundary was then applied to the historical shared-source
Groovy failure. No additional product change was required:

| Starting order | Shared source |
|---|---:|
| Control first | 814.125 ms / 40.1% faster; +639.25 ms lower bound |
| Candidate first | 619.75 ms / 33.1% faster; +380.75 ms lower bound |

Build Impact executed exactly two affected verifications instead of the five
executed by the optimized native control. All 16 pairs preserved byte-identical
required outputs, Configuration Cache reuse, and zero product failures. The
historical failure is therefore attributed to the already corrected launcher
and measurement boundary, not to a remaining shared-source product defect.

The equivalent calibrated Kotlin boundary also resolves the historical
leaf-source mismatch without another product change:

| Starting order | Leaf source |
|---|---:|
| Control first | 1,112.875 ms / 49.6% faster; +870.5 ms lower bound |
| Candidate first | 833.75 ms / 39.4% faster; +601.25 ms lower bound |

Build Impact executed exactly one affected verification instead of the five
executed by the optimized native control. All 16 pairs were positive and
preserved byte-identical required outputs, Configuration Cache reuse, and zero
product failures. The percentages remain separate by batch and broaden only
this bounded synthetic Kotlin leaf cell.

The remaining shared-source and build-logic Kotlin cells were then repeated on
the same calibrated boundary. Correctness remained exact, but classification
did not reproduce:

| Starting order | Shared source | Build logic |
|---|---:|---:|
| Control first | +963.75 ms / 45.2%; threshold met | −21.375 ms / −0.65%; parity met |
| Candidate first | +468.125 ms / 27.2%; 500-ms floor missed | −137.625 ms / −4.1%; parity missed |

Shared-source stayed faster in all 16 pairs, but one batch missed the unchanged
absolute floor by 31.875 ms. Build-logic retained exact five-versus-five
execution but crossed the −2% parity boundary. The checked verdict is therefore
`MEASUREMENT_UNSTABLE`, not a product regression, and authorizes no product
change or broader claim.

The terminal boundary review uses all four recorded batches per remaining
Kotlin cell. Shared-source classifications alternate `PASS/FAIL/PASS/FAIL`;
the latest failed batch still has 8/8 positive pairs and 27.2% mean savings,
but misses the unchanged 500-ms floor by 31.875 ms. Build-logic alternates
`FAIL/PASS/PASS/FAIL`, executes five verifications in both arms, and has no
stable attributable regression. Every batch preserves correctness and records
zero product failures.

Because no new causal hypothesis remains, another unchanged repetition could
only sample which side of a fixed threshold the noise lands on. The checked
terminal verdict is therefore `STOP_RETAIN_BOUNDED_CLAIM`: retain previously
qualified synthetic claims, keep these two cells outside generalization, and
authorize neither product tuning nor another replication in the current POC.

### Realistic change-class result

`POC-BREADTH-001` tested whether the bounded result generalizes to a five-project
Kotlin/Groovy graph. The initial report qualified 2/8 cells. `POC-OVERHEAD-001`
then proved that the installed candidate used the native-only path, loaded no
init/project plugin, and had one avoidable candidate-only `XDG_CACHE_HOME`.
After removing only that asymmetry and leaving every threshold unchanged, the
repeat qualifies 4/8 cells. The checked decision remains
`RETAIN_QUALIFIED_SYNTHETIC_WORKLOADS_ONLY`.

| Change | Kotlin | Groovy |
|---|---:|---:|
| No change | 45 ms faster (4.9%); parity met | 168 ms slower (12.1%); parity failed |
| Leaf source | 543 ms faster (38.1%); threshold met | 1,309 ms faster (63.6%); threshold met |
| Shared source | 877 ms faster (49.1%); threshold met | 938 ms slower (95.8%); threshold failed |
| Build logic | 679 ms slower (27.0%); parity failed | 119 ms slower (5.4%); parity failed |

Every required output was byte-identical, selection/fallback task counts were
exact, Configuration Cache behavior matched the scenario, and no product failure
occurred. The failure is value/performance, not correctness. Percentages are not
added across cells. The isolation experiment below determines whether those
classifications survive removal of inter-arm carryover.

### Isolated-arm stability result

`POC-STABILITY-001` removed writable-state carryover by running control and
candidate in separate strict containers, each with its own workspace, Gradle
home, and daemon. Two complete batches reversed the global arm order while
retaining the same fixture, warm-up, mutations, samples, outputs, and thresholds.

| Batch | Qualifying cells | Classification changes |
|---|---:|---:|
| Control first | 0/8 | 4 versus candidate-first |
| Candidate first | 4/8 | 4 versus control-first |

All 256 underlying arm measurements preserved the expected execution shape,
Configuration Cache behavior, byte-identical required outputs, and zero
product-attributable failures. Four classifications changed solely with global
arm order, so the checked verdict is `MEASUREMENT_UNSTABLE` and
`POC-STABILITY-G01` is `FAILED`. This is valid negative POC evidence: it neither
authorizes another product change nor broadens the claim. The next experiment
must interleave isolated control/candidate microbatches close in time so runner
drift cannot dominate an otherwise isolated comparison.

### Temporally paired stability result

`POC-PAIRING-001` kept the isolated arm containers alive concurrently and ran
each control/candidate pair consecutively. Pair order alternated inside every
cell, and the second batch reversed the starting arm. All 128 pairs started the
second arm within 542 ms while retaining private workspaces, Gradle homes,
daemons, installs, and state.

| Cell | Control-first start | Candidate-first start | Reproduced? |
|---|---:|---:|---:|
| No-change Kotlin | +0.4%; parity | +2.1%; parity | yes |
| No-change Groovy | -40.1%; failed | -54.7%; failed | yes |
| Leaf Kotlin | +31.0%; failed 500-ms floor | -6.7%; failed | yes |
| Leaf Groovy | +43.0%; threshold met | +58.1%; threshold met | yes |
| Shared Kotlin | +54.2%; threshold met | +8.6%; failed | no |
| Shared Groovy | +34.3%; failed interval | +3.7%; failed | yes |
| Build-logic Kotlin | -3.4%; failed parity | +19.0%; parity | no |
| Build-logic Groovy | +14.5%; parity | +13.2%; parity | yes |

Required outputs were byte-identical, task selection and Configuration Cache
behavior were exact, and product-attributable failures remained zero. Both
batches qualify 4/8 cells, but only six classifications reproduce, so
`POC-PAIRING-G01` remains failed. The decision authorizes product experiments
only for the reproduced failures: no-change Groovy, leaf Kotlin, and shared
Groovy. It explicitly blocks tuning the two mismatched Kotlin cells from this
evidence. Percentages are not added across cells.

## Historical v0.2 public onboarding performance

[The hosted result](./results/onboarding-performance-v1-hosted.json) preserves
the pre-fast-path `0.2.0` baseline. It measures the actual no-configuration
command from the README on an isolated 4-CPU
GitHub-hosted runner and immutable public Kotlin and Groovy pilots. Four
alternating pairs compare BuildOpt separately with cache disabled and with
Gradle's unrestricted native local cache.

| Pilot | Control | Control mean | BuildOpt mean | Difference |
|---|---|---:|---:|---:|
| Kotlin | Cache off | 8.916 s | 7.905 s | 1.010 s faster (11.3%) |
| Groovy | Cache off | 9.586 s | 7.625 s | 1.962 s faster (20.5%) |
| Kotlin | Native cache | 7.233 s | 7.818 s | 0.585 s slower (8.1%) |
| Groovy | Native cache | 7.368 s | 7.394 s | 0.026 s slower (0.3%) |

All eight hosted cache-off pairs improved and every paired distribution was
byte-identical. The less favorable native-cache observations remain in the
report. This is historical descriptive POC evidence, not the current scorecard
or a production or golden-runner claim.

[The independent local result](./results/onboarding-performance-v1-local.json)
used the same protocol on a 12-CPU host:

| Pilot | Control | Control mean | BuildOpt mean | Difference |
|---|---|---:|---:|---:|
| Kotlin | Cache off | 9.857 s | 7.815 s | 2.042 s faster (20.7%) |
| Groovy | Cache off | 14.226 s | 11.276 s | 2.951 s faster (20.7%) |
| Kotlin | Native cache | 10.762 s | 11.812 s | 1.050 s slower (9.8%) |
| Groovy | Native cache | 10.754 s | 10.876 s | 0.123 s slower (1.1%) |

Across both environments, all 16 cache-off pairs improved and every output
matched. Validate both immutable results without rerunning builds:

```bash
./dev/check-onboarding-performance \
  benchmarks/results/onboarding-performance-v1-hosted.json
./dev/check-onboarding-performance \
  benchmarks/results/onboarding-performance-v1-local.json
```

To create a fresh report, provide an absent absolute output path, an installed
BuildOpt binary and version, and both Git checkouts:

```bash
./dev/run-onboarding-benchmark \
  /tmp/onboarding-performance.json \
  "$(command -v buildopt)" \
  0.2.0 \
  /path/to/buildopt-pilot \
  /path/to/buildopt-pilot-groovy
./dev/check-onboarding-performance /tmp/onboarding-performance.json
```

The exact design and claim boundary live in
[the onboarding performance specification](../specs/onboarding-performance-v1.md).


## Owner-controlled pilot deployment evidence

[`results/a1-001-owner-controlled-pilot.json`](./results/a1-001-owner-controlled-pilot.json)
records the first signed installed release on the public synthetic
`tonyredondo/buildopt-pilot` repository. Two successful authenticated runs
produced schema-valid sessions, byte-identical distributions, and eight native
managed-L1 `compileJava` hits on replay while the custom task remained under
Tier 1 default deny. The initial GitHub billing block and the successful public
runner retry are both retained in the immutable evidence.

This result closes the deployment item only. It makes no causal-savings,
signed-Shared-authority, external-user, or eight-hour-soak claim. Revalidate
its immutable contract with:

```bash
./dev/check-owner-controlled-pilot-deployment
```

## Owner-operated causal POC evidence

[`results/a1-006-owner-poc-evaluation.json`](./results/a1-006-owner-poc-evaluation.json)
records four paired alternating measurements on each immutable public Kotlin and
Groovy pilot. Both repositories have positive lower 95% paired-bootstrap bounds,
non-regressive p95, byte-identical distributions, and zero excluded/failing
outcomes. Revalidate the checked-in artifact without rerunning builds:

```bash
./dev/check-owner-poc-evaluation
```

This closes the owner-operated POC gate only; the result remains PRELIMINARY,
does not authorize production promotion, and does not claim the deferred soak
or external design-partner evidence.

## Runtime owner evaluation evidence

[`results/b-runtime-owner-evaluation.json`](./results/b-runtime-owner-evaluation.json)
records the public four-CPU owner run for the Runtime Optimizer. The same run
drives 200 durable pre-outcome A/A assignments with delayed exactly-once
rewards, then measures four real alternating Gradle pairs for A/A and the
finite `W4_H6G` candidate. It passes sample-ratio, p95/p99, queue, OOM,
additional-compute, and byte-identical artifact guardrails.

```bash
./dev/check-runtime-owner-evaluation
```

This closes the owner-operated POC gates `B-G01` and `B-G03`; it does not run
the deferred eight-hour soak or authorize production promotion.


## Task Intelligence accepted-patch evidence

[`results/c1-task-intelligence-pilot.json`](./results/c1-task-intelligence-pilot.json)
binds the reviewed custom-task source patch, accepted public PR, exact state path,
and four alternating post-merge causal pairs. All control arms executed, all
candidate arms restored `FROM-CACHE`, the output bytes matched, and the mean
saving was 203 ms with a positive 147-ms lower 95% bound.

```bash
./dev/check-task-intelligence-poc
```

The Agent and helper remain fail-closed unavailable routes; only the reviewed
source contract is active. This is POC evidence, not the deferred soak or
production-promotion authority.

## Walking-skeleton overhead evidence

[`results/ws-009-golden-lane.json`](./results/ws-009-golden-lane.json) is the
first strict, descriptive `WS-009` observation. It contains four alternating
native/wrapper pairs, retains the first pair and signed negative differences,
and binds the exact runner contract, metric catalog, envelope, launcher,
server, and plugin digests. It is not causal evidence and does not activate a
promotion gate.

The report is produced only by the strict 4-vCPU/16-GiB golden-container path
and is subsequently validated without being rewritten:

```bash
./dev/run-golden-lane-container --require-runner-class
```

The measurement contract lives in
[`specs/walking-skeleton-overhead-v1.md`](../specs/walking-skeleton-overhead-v1.md).

## A0 no-hit overhead evidence

[`results/a0-g06-no-hit-overhead.json`](./results/a0-g06-no-hit-overhead.json)
is the qualified four-pair `A0-G06` report from the pinned
4-CPU/16-GiB runner. It records authenticated forced L2 misses with fresh L1
and output state for every long wrapper arm, byte-identical required JARs, and
the independent short branch where policy omits L2 before execution and the
miss server observes zero requests.

The report binds every measurement input by SHA-256 and applies the fixed
500-ms/2% long-session p95 limits. It is an A0 engineering gate, not causal
savings or beta-promotion evidence. Revalidate it with:

```bash
./dev/check-no-hit-overhead
```

The measurement contract lives in
[`specs/no-hit-overhead-v1.md`](../specs/no-hit-overhead-v1.md).

## JVM Agent spike evidence

[`results/spk-002-agent.json`](./results/spk-002-agent.json) records the one
warm, order-sensitive JDK 21 sample emitted while closing `SPK-002`. It is
descriptive only: the prototype is `UNAVAILABLE` for access tracing because it
observes class loads rather than method calls. The result never activates an
overhead or promotion gate.
