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

The current POC verdict is `CONTINUE_CONDITIONALLY`, not “value proven”. Safe
Cache is a safety enabler at native-cache parity; Runtime Tuning is below the
accelerator threshold; Build Impact has a strong but single-workload signal;
and the combined public path has not yet been measured against the optimized
native baseline. Validate that interpretation with:

```bash
./dev/check-poc-value-validation
```

The scorecard answers a different question for each optimization instead of
combining unrelated percentages:

| Mechanism | Control mean | Candidate mean | Result | Acceptance |
|---|---:|---:|---:|---|
| Safe cache, Kotlin, cache off | 9.131 s | 7.682 s | 1.449 s faster (15.9%) | Positive mean and 4/4 positive pairs |
| Safe cache, Groovy, cache off | 12.454 s | 10.754 s | 1.700 s faster (13.7%) | Positive mean and 3/4 positive pairs |
| Safe cache, Kotlin, native cache | 10.415 s | 10.412 s | 0.003 s faster (0.02%) | Within 2% native-cache parity guardrail |
| Safe cache, Groovy, native cache | 10.470 s | 10.520 s | 0.050 s slower (0.47%) | Within 2% native-cache parity guardrail |
| Runtime Tuning | 8.999 s | 8.933 s | 0.066 s faster (0.7%) | Positive lower 95% bound; artifact, OOM, queue and compute guardrails pass |
| Build Impact | 8.180 s | 5.926 s | 2.254 s faster (27.6%) | 4/4 positive pairs and required outputs identical |

Safe cache is expected to beat cache-off and remain near native cache: both
arms use Gradle's cache engine, while BuildOpt adds repository isolation and a
safe task policy. Runtime Tuning measures a bounded worker/heap profile on a
large four-CPU workload. Build Impact measures avoided compilation when one
manifest-declared project is unaffected. The Runtime Tuning percentage cannot
be added to the cache or Build Impact percentage because the runner and
workload differ.

Validate all checked-in evidence and print the machine-readable scorecard:

```bash
./dev/check-build-optimization-performance
```

The underlying evidence and contracts are:

- [safe-cache observations](./results/cache-parity-v1-local.json) and
  [contract](../specs/cache-parity-v1.md);
- [Runtime Tuning observations](./results/b-runtime-owner-evaluation.json) and
  [contract](../specs/runtime-owner-evaluation-v1.md);
- [Build Impact observations](./results/build-impact-performance-v1-local.json)
  and [contract](../specs/build-impact-performance-v1.md).

All three are bounded POC results. They preserve signed differences and do not
claim universal savings, combined-product value, or production readiness.

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
