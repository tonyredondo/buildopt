# BuildOpt POC: One-Page Handoff

## The idea

BuildOpt tests whether a generic decision layer can make substantial Gradle
builds faster than an already optimized native Gradle baseline. Gradle remains
the execution engine and safe fallback. BuildOpt inspects the exact Git change
and requested workflow, derives a smaller sufficient graph, materializes exact
unaffected outputs, measures the complete installed path and reuses only
evidence that remains structurally and economically valid.

This is an owner-operated proof of concept, not a production product. Soak,
design-partner evidence, production SLOs, autonomous promotion and Test
Optimization are outside the current scope.

## Intended customer experience

```text
install BuildOpt
cd <a Gradle repository>
buildopt optimize <the existing Gradle workflow>
```

The target repository needs no BuildOpt manifest, graph, profile, plugin path
or output contract. An ambiguous, global, unprofitable or drifted case runs
optimized native Gradle and makes no performance claim.

## Components

| Component | What it does | Current POC conclusion |
| --- | --- | --- |
| **Structural Build Impact** | Selects only producers needed by the exact change and required workflow outputs. | Primary accelerator; the automatic path is faster on 5/5 public repositories and qualifies on 4/5. |
| **Automatic discovery** | Derives Git ownership, Gradle output relationships and a candidate without repository-name rules. | Complete on all five breadth subjects and on `classes`, `testClasses` and `assemble`. |
| **Incremental learning and value gate** | Collects one control or candidate observation per useful invocation and checks outputs, interval, tail, fallback and payback. | Adds zero measurement-only workflows; qualifies four repositories and safely retains Spring. |
| **Verified output materialization** | Restores exact unaffected outputs before their producers are omitted from a clean workspace. | Composed on all five public builds; corruption or missing state falls back to native before accepting output. |
| **Aggregate workflow partition** | Splits broad lifecycle workflows into changed-owner work plus exact revision-bound unaffected outputs. | Kafka selects 3/64 projects, Micronaut 22/75 and Groovy 2/37 while preserving complete outputs. |
| **Profile portfolio / optional central state** | Reuses qualified evidence under exact repository, Wrapper, workflow, graph, output and executable bindings. | Functional locally and across machines; lifetime value after the latest breadth qualification is the next measurement. |
| **Gradle-compatible cache** | Reuses verified task outputs locally or through optional HTTP/HTTPS storage. | Supporting infrastructure near native-cache parity, not the principal speed claim. |
| **Runtime Tuning, Hot State and standard Copy** | Earlier broad resource/state hypotheses. | Retired after neutral, unstable or regressive end-to-end evidence. |

## Latest public-repository result

One exact BuildOpt binary ran 85 ordinary invocations on five frozen public
repositories. Every candidate reproduced the exact required-output digest;
full-graph fallback remained valid and product-attributable failures were zero.

| Repository / workflow | Graph | Native -> candidate | Mean saving | 95% saving interval | Pairs | Learning / payback | Decision |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Spring `testClasses` | 27 -> 10 | 10.456 -> 9.127 s | **1.329 s / 12.71%** | +0.282..+2.572 s | 7/8 | 88.668 s / 67 builds | Native retained. |
| OpenTelemetry Spring family | 1,024 -> 34 | 74.529 -> 63.376 s | **11.154 s / 14.97%** | +7.903..+14.223 s | 8/8 | 201.913 s / 19 builds | Qualified. |
| Kafka `testClasses` | 64 -> 3 | 9.774 -> 4.406 s | **5.368 s / 54.92%** | +3.513..+7.337 s | 8/8 | 70.808 s / 14 builds | Qualified. |
| Micronaut `assemble` | 75 -> 22 | 23.108 -> 7.802 s | **15.306 s / 66.24%** | +14.245..+16.498 s | 8/8 | 114.284 s / 8 builds | Qualified. |
| Groovy `classes` | 37 -> 2 | 65.652 -> 15.776 s | **49.876 s / 75.97%** | +45.842..+53.680 s | 8/8 | 73.857 s / 2 builds | Qualified. |

The headline is **4/5 automatically qualified and 5/5 faster**. Spring is not
promoted despite a positive mean and interval because only 7/8 pairs improve
and learning would need 67 matching builds to repay, above the unchanged limit
of 30. This is expected fail-open behavior, not a hidden failure.

The run also exposed and fixed a generic small-file bottleneck. Per-file
durability made Spring materialization add about 629 seconds. Batching directory
durability after atomic writes reduced the first post-fix candidate overhead to
about 2.7 seconds while preserving content hashes and native fallback.

Repository percentages are not averaged and mechanism percentages are not
added. The evidence was captured on the 12-CPU development host with a common
12-worker cap; it validates the POC idea, not production capacity or SLOs.

## What the data proves

- Generic graph reduction can beat optimized native Gradle on substantial,
  unrelated Gradle repositories without target-specific product branches.
- Exact output composition closes the gap between a smaller graph and the
  complete customer workflow; the candidate does not win by omitting required
  deliverables.
- Incremental learning turns the former synchronous 16-build transaction into
  useful ordinary invocations and reduces payback from 101–710 builds in V1 to
  2–19 for four qualified repositories.
- The gate remains selective: Spring improves but stays native when cumulative
  economics are not good enough.

It does not yet prove that every change family will match a qualified profile
often enough to produce cumulative customer value, nor that Spring's remaining
learning cost is irreducible.

## Next steps

1. Attribute and reduce Spring's remaining learning/materialization cost with
   generic content reuse; preserve exact hashes, clean-workspace semantics and
   native fallback.
2. Replay the four qualified profiles over compatible public descendant
   changes and measure cumulative savings, selection cost, invalidation and
   fallback rather than assuming calibration value equals lifetime value.
3. Surface learning progress, expected payback and native-retained reasons in
   the one-command experience so users do not need internal JSON.

## Evidence

- [Latest five-repository result](../../benchmarks/results/poc-automatic-breadth-transfer-v2/README.md)
- [Machine-readable summary](../../benchmarks/results/poc-automatic-breadth-transfer-v2/summary.json)
- [V2 protocol](../../specs/poc-automatic-breadth-transfer-v2.md)
- [Detailed performance findings](./build-optimization-performance.md)
- [Generalization audit](./buildopt-generalization-audit.md)
- [One-command roadmap](../plans/one-command-onboarding-roadmap.md)
- [Implementation tracker](../../implementation-tracker.md)
