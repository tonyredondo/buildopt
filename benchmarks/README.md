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
- [safe-cache observations](./results/cache-parity-v1-local.json) and
  [contract](../specs/cache-parity-v1.md);
- [Runtime Tuning observations](./results/b-runtime-owner-evaluation.json) and
  [contract](../specs/runtime-owner-evaluation-v1.md);
- [Build Impact observations](./results/build-impact-performance-v1-local.json)
  and [contract](../specs/build-impact-performance-v1.md).

The three mechanism-development reports remain historical inputs. The strict
reports are the current decision evidence. They prove bounded combined
value but do not yet prove realistic breadth; none claims universal savings or
production readiness.

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
