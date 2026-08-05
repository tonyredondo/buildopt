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

The scorecard answers a different question for each optimization instead of
combining unrelated percentages:

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
- [Spotless exact-workflow Build Impact evidence](./results/poc-spotless-impact-v1.json),
  validated by `./dev/check-poc-spotless-impact`;
- [safe-cache observations](./results/cache-parity-v1-local.json) and
  [contract](../specs/cache-parity-v1.md);
- [Runtime Tuning observations](./results/b-runtime-owner-evaluation.json) and
  [contract](../specs/runtime-owner-evaluation-v1.md);
- [Build Impact observations](./results/build-impact-performance-v1-local.json)
  and [contract](../specs/build-impact-performance-v1.md).

The three mechanism-development reports remain historical inputs. The strict
synthetic reports prove bounded combined value. The public-repository
compatibility and performance reports test whether that claim generalizes; the
answer is currently **no**. None of these documents claims universal savings
or production readiness.

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
