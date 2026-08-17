# BuildOpt POC: One-Page Handoff

## The idea

BuildOpt tests whether a generic decision layer can make substantial Gradle
builds faster than an already optimized native Gradle baseline. Gradle remains
the execution engine and source of truth. BuildOpt inspects the exact Git
change and requested workflow, derives a smaller sufficient Gradle graph,
measures the complete installed path, verifies required outputs, and reuses the
candidate only while exact bindings and value gates hold. Ambiguity, drift,
poor economics or a global change keeps optimized native Gradle authoritative.

This is an owner-operated proof of concept, not a production-ready product.
Soak qualification, design-partner evidence, production SLOs, autonomous
promotion and Test Optimization are outside the current scope.

## The intended experience

```text
install BuildOpt
cd <a Gradle repository>
buildopt optimize <the existing Gradle workflow>
```

No BuildOpt manifest, graph, profile, plugin path or output contract is required
in the target repository. The first invocation discovers and measures a
candidate in private state. Later exact matches may replay it; any failed
binding falls back before Gradle starts.

## Components

| Component | What it contributes | Current evidence |
| --- | --- | --- |
| **Structural Build Impact** | Runs only the project/task producers required by the exact change and requested outputs. | Primary accelerator; closes the public one-command gate on Ktor and Beam. |
| **Automatic discovery and calibration** | Derives Git/Gradle ownership, outputs and graph; runs balanced native/candidate pairs; checks uncertainty, p95, outputs, fallback and payback. | Prevents a smaller task count from being mistaken for customer value. |
| **Profile portfolio and replay** | Stores only qualified structural families under exact repository, Wrapper, workflow, graph, output, executable and evidence bindings. | POC-only automatic replay; drift retains native. |
| **Safe local cache** | Isolates and verifies Gradle cache data by repository, Wrapper and platform. | Supporting safety; approximately at parity with an already warm native Gradle cache, not the current speed claim. |
| **Shared / Edge cache** | Offers Gradle-compatible opaque cache objects over HTTP/HTTPS and optional locality. | Separate experiment; its percentages are never added to Build Impact results. |
| **Optional central state** | Preserves compatible portfolios, evidence and resumable checkpoints without making remote state optimization authority. | Contract, restart-safe CAS/SQLite storage and scoped TLS 1.3 access are complete; synchronization and cross-machine value are not implemented yet. |
| **Launcher, history and reports** | Preserves process behavior and reports graph reduction, wall time, uncertainty, p95, learning cost, payback and fallback. | Supporting infrastructure; launcher overhead is included in candidate timings. |
| **Runtime Tuning, Hot State and standard Copy** | Earlier resource/state-reuse hypotheses. | Retired from the active POC after neutral, unstable or regressive evidence. |

## Latest automatic public-package proof

The terminal run used immutable public
[`v0.6.1`](https://github.com/tonyredondo/buildopt/releases/tag/v0.6.1),
fresh package/checkouts/BuildOpt state, zero manual BuildOpt files, eight
alternating pairs and a maximum 30-build payback. Both arms received identical
content-bound Gradle dependencies and native-cache seeds; unmeasured
daemon/configuration warmup prevented download or cache asymmetry from becoming
claimed value.

| Repository / workflow | Graph | Native mean | BuildOpt mean | Saving | 95% saving interval | p95 | Payback |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Ktor `jvmJar` | 133 -> 10 | 38.810 s | 7.830 s | **30.979 s / 79.82%** | 24.679..38.922 s | 61.575 -> 11.957 s | 26 builds |
| Beam `classes` | 316 -> 6 | 65.081 s | 24.958 s | **40.123 s / 61.65%** | 33.867..51.470 s | 102.621 -> 25.946 s | 28 builds |

Both rows improve in 8/8 pairs, preserve every required output hash, maintain
stable task shapes, lower p95, pass full-graph fallback and record zero
product-attributable failures. Percentages are repository-specific and are not
averaged or added.

The honest negative is a Ktor root build-logic change. The complete native
`jvmJar` workflow succeeded in 46 seconds; BuildOpt returned
`GLOBAL_CHANGE_REQUIRES_FULL_GRAPH`, performed no calibration and made no
performance claim. Safe non-activation is part of the success criterion.

## Supporting breadth, not a substitute for onboarding proof

The latest comparable reviewed-profile matrix independently showed structural
potential on additional public repositories:

| Repository | Optimized native | BuildOpt | Direct result |
| --- | ---: | ---: | ---: |
| Spring Framework | 13.311 s | 11.183 s | **15.99% faster** |
| OpenTelemetry | 87.869 s | 74.713 s | **14.97% faster** |
| Apache Kafka | 113.381 s | 14.341 s | **87.35% faster** |
| Micronaut Core | 30.411 s | 18.418 s | **39.44% faster** |
| Apache Groovy | 79.868 s | 20.767 s | **74.00% faster** |

All 40 balanced blocks improved with exact outputs and full fallback. These
profiles required reviewed owner inputs, so they demonstrate mechanism breadth
but do not replace the zero-configuration Ktor/Beam result.

## What is proven

- The same repository-independent implementation can discover and qualify
  material graph reduction in two different substantial Gradle families.
- The complete public installed path beats optimized native Gradle, not a
  cache-disabled strawman.
- Correctness and economics are executable gates: exact outputs, positive
  interval, lower p95, fallback and payback all have to pass.
- The system can say no safely; global or uncertain work stays native without a
  fabricated timing claim.
- Public-package compatibility matters: the rejected `v0.6.0` run exposed a
  real Configuration Cache defect, which was fixed and republished as
  `v0.6.1` before terminal timing restarted.

## What is not proven

- A universal improvement for every repository, workflow or change.
- That a qualified profile survives enough future commits to realize projected
  payback; lifetime is not yet observed.
- Cross-machine value from centralized cache and learned state.
- Production reliability, security posture or autonomous rollout.

## Recommended next steps

1. **Connect Gradle through the local verifying gateway.** Use the proven
   central TLS/token boundary without exposing the upstream token to Gradle;
   prove one clean producer, one read-only consumer and outage fallback.
2. **Measure profile lifetime across commits.** Replace projected payback with
   observed matching replays, invalidations and cumulative net saving.
3. **Add generic economic prequalification.** Use task shapes and graph cost to
   avoid an expensive eight-pair calibration when a candidate is unlikely to
   repay.
4. **Improve graph precision without repository rules.** Target task/variant,
   ABI and output relationships that currently make some Groovy, Kafka or
   Micronaut workflows too broad or uneconomic.
5. **Repeat the automatic path on the breadth repositories.** The POC should
   discover value from the same one command; reviewed profiles remain
   supporting evidence until then.

## Evidence

- [Terminal one-command result](../../benchmarks/results/poc-magic-end-to-end-value-v2/README.md)
- [Machine-readable terminal summary](../../benchmarks/results/poc-magic-end-to-end-value-v2/summary.json)
- [Terminal contract](../../specs/poc-magic-end-to-end-value-v2.md)
- [Historical automatic diagnostic matrix](../../benchmarks/results/poc-magic-end-to-end-value-v1/README.md)
- [Comparable reviewed-profile matrix](../../benchmarks/results/poc-statistical-qualification-v2/README.md)
- [One-command roadmap](../plans/one-command-onboarding-roadmap.md)
- [Optional central storage contract](../../specs/poc-central-storage-contract-v1.md)
- [Restart-safe typed central state](../../specs/poc-central-state-storage-v1.md)
- [Implementation tracker](../../implementation-tracker.md)
