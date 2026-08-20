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
| **Profile portfolio and replay** | Stores only qualified structural families under exact repository, Wrapper, workflow, graph, output, executable and evidence bindings. | A Ktor matching replay saved 112.198 s, but its observed public window did not repay learning; drift retained native. |
| **Safe local cache** | Isolates and verifies Gradle cache data by repository, Wrapper and platform. | Supporting safety; approximately at parity with an already warm native Gradle cache, not the current speed claim. |
| **Shared / Edge cache** | Offers Gradle-compatible opaque cache objects over HTTP/HTTPS and optional locality. | Separate experiment; its percentages are never added to Build Impact results. |
| **Optional central cache and state** | Shares committed Gradle outputs plus compatible portfolios, evidence and checkpoints while keeping local execution authoritative. | Under the same committed remote-cache opportunity, the complete path is **82.45% faster on Ktor** and **56.41% on Beam**, with 8/8 positive pairs and exact outputs; global Ktor build logic safely retains native. |
| **Launcher, history and reports** | Preserves process behavior and reports graph reduction, wall time, uncertainty, p95, learning cost, payback and fallback. | Supporting infrastructure; launcher overhead is included in candidate timings. |
| **Runtime Tuning, Hot State and standard Copy** | Earlier resource/state-reuse hypotheses. | Retired from the active POC after neutral, unstable or regressive evidence. |

## Latest central end-to-end value proof

The terminal central experiment gives optimized native Gradle and BuildOpt the
same already-committed remote-cache objects. Candidate time includes the
installed package, structural selection, launcher, gateway, TLS, state/cache
synchronization and Gradle execution.

| Repository / workflow | Graph | Native mean | BuildOpt mean | Saving | 95% saving interval | p95 | Payback |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Ktor `jvmJar` | 133 -> 10 | 215.506 s | 37.828 s | **177.678 s / 82.45%** | 156.555..198.802 s | 243.986 -> 55.139 s | 28 builds |
| Beam `classes` | 316 -> 6 | 48.475 s | 21.130 s | **27.345 s / 56.41%** | 22.407..32.283 s | 53.468 -> 28.319 s | 29 builds |

Both rows improve in 8/8 alternating pairs and preserve exact required
outputs. A Ktor root build-logic change keeps the full graph, succeeds through
the same central connection and makes no performance claim. The 12-CPU host
result is bounded POC evidence, not golden-runner or production evidence.

## Observed profile lifetime and economics

Steady-state speedup is not enough if the repository changes before learning
repays. The current Ktor experiment follows one centrally published Jetty
profile through a real first-parent sequence and applies the new generic
economic precheck before learning an unrelated owner:

| Event | Optimized native | BuildOpt | Direct effect |
| --- | ---: | ---: | ---: |
| Qualifying calibration | 77.419 s mean | 32.489 s mean | **58.03% faster**, but 1,386.764 s learning cost |
| Matching Jetty replay | 197.028 s | 96.284 s | **100.744 s / 51.13% saved** |
| Unrelated CORS change | 184.647 s | 198.543 s | **13.896 s / 7.53% overhead**, discovery/calibration rejected |
| Global build-logic change | 186.553 s | 186.531 s | 22-ms native parity; profile rejected early |

All required JAR bytes match. The CORS precheck uses direct project ownership,
finds two analogous commits against the theoretical eight-build minimum and
rejects in 192.442 ms without discovery or calibration. The preceding run had
observed a 220.761-second penalty on the same public change; the current
13.896-second penalty is 93.71% lower across runs and remains inside the 10%
native-retention guardrail. Across three current observations the window gains
86.870 seconds before calibration but remains **1,299.894 seconds negative
after calibration**. The projected 31-build break-even was not reached.

## Previous automatic public-package local proof

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
- Qualified learning can now move between checkouts: retained Kafka evidence
  is accepted on a source-only descendant after local replanning, while a
  `build.gradle.kts` descendant is rejected before Gradle. This is functional
  reuse evidence, not a new central wall-time claim.
- The complete connected path now beats a full-graph optimized-native control
  under the same committed central-cache opportunity on Ktor and Beam. The
  smaller two-machine fixture separately proves restart, credential isolation
  and outage fallback.
- Public-package compatibility matters: the rejected `v0.6.0` run exposed a
  real Configuration Cache defect, which was fixed and republished as
  `v0.6.1` before terminal timing restarted.
- Profile lifetime is now measured for one real Ktor sequence. It proves that
  a large matching replay win can still lose overall when matches are rare and
  unrelated fallback performs expensive discovery.
- Economic prequalification can prevent that unrelated discovery generically:
  it uses the verified graph and bounded Git history, not a Ktor/CORS rule, and
  retains native before spending the eight-pair learning budget.

## What is not proven

- A universal improvement for every repository, workflow or change.
- That another profile or repository has Ktor's observed lifetime; useful
  lifetime remains profile-, workflow- and change-distribution-specific.
- A universal central-path improvement for every repository, workflow, change,
  network or hardware class. The retained Ktor global case deliberately makes
  no performance claim.
- Production reliability, security posture or autonomous rollout.

## Recommended next steps

1. **Transfer the automatic path to the breadth repositories.** Run the same
   zero-manual-file command on Spring, OpenTelemetry, Kafka, Micronaut and
   Groovy; diagnose generic blockers and accept value only under exact output,
   fallback, repeatability and economic gates.
2. **Improve graph precision without repository rules.** Target task/variant,
   ABI and output relationships that currently make some Groovy, Kafka or
   Micronaut workflows too broad or uneconomic.
3. **Reduce qualification cost.** The precheck prevents obviously uneconomic
   new learning, but a qualifying Ktor profile still costs 1,386.764 seconds
   and needs 31 matching replays.
4. **Measure full compositions, not added percentages.** Keep cache, graph
   reduction and qualified task adapters attributable, then test their combined
   wall time on the same workflow and change.
5. **Transfer central value to another runner/network class.** Keep the same
   equal-opportunity protocol and confirm that selection plus central reuse
   remains positive without turning the POC into production qualification.

## Evidence

- [Terminal one-command result](../../benchmarks/results/poc-magic-end-to-end-value-v2/README.md)
- [Machine-readable terminal summary](../../benchmarks/results/poc-magic-end-to-end-value-v2/summary.json)
- [Terminal contract](../../specs/poc-magic-end-to-end-value-v2.md)
- [Historical automatic diagnostic matrix](../../benchmarks/results/poc-magic-end-to-end-value-v1/README.md)
- [Comparable reviewed-profile matrix](../../benchmarks/results/poc-statistical-qualification-v2/README.md)
- [Economic prequalification result](../../benchmarks/results/poc-economic-prequalification-v1/README.md)
- [Economic prequalification machine evidence](../../benchmarks/results/poc-economic-prequalification-v1/summary.json)
- [One-command roadmap](../plans/one-command-onboarding-roadmap.md)
- [Optional central storage contract](../../specs/poc-central-storage-contract-v1.md)
- [Restart-safe typed central state](../../specs/poc-central-state-storage-v1.md)
- [Central Gradle-cache gateway proof](../../specs/poc-central-gradle-cache-v1.md)
- [Central state-sync proof](../../specs/poc-central-state-sync-v1.md)
- [Automatic central profile reuse](../../specs/poc-central-optimize-integration-v1.md)
- [Isolated two-machine proof](../../specs/poc-central-two-machine-v1.md)
- [Two-machine machine evidence](../../benchmarks/results/poc-central-two-machine-v1.json)
- [Central end-to-end result](../../benchmarks/results/poc-central-end-to-end-value-v1/README.md)
- [Central end-to-end machine evidence](../../benchmarks/results/poc-central-end-to-end-value-v1/summary.json)
- [Central end-to-end contract](../../specs/poc-central-end-to-end-value-v1.md)
- [Ktor profile-lifetime result](../../benchmarks/results/poc-profile-lifetime-v1/README.md)
- [Ktor profile-lifetime machine evidence](../../benchmarks/results/poc-profile-lifetime-v1/summary.json)
- [Profile-lifetime contract](../../specs/poc-profile-lifetime-v1.md)
- [Implementation tracker](../../implementation-tracker.md)
