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
| **Structural Build Impact** | Selects only producers needed by the exact change and required workflow outputs. | Primary accelerator; the automatic path qualifies on all five public repositories under unchanged gates. |
| **Automatic discovery** | Derives Git ownership, Gradle output relationships and a candidate without repository-name rules. | Complete on all five breadth subjects and on `classes`, `testClasses` and `assemble`. |
| **Incremental learning and value gate** | Collects one control or candidate observation per useful invocation and checks outputs, interval, tail, fallback and payback. | Adds zero measurement-only workflows; one-time cost now repays in one to four matching builds across all five subjects. |
| **Verified output materialization** | Restores exact unaffected outputs before their producers are omitted from a clean workspace. | One manifest-bound pack plus bounded parallel restore qualifies all five builds; corrupt or missing state still falls back before accepting output. |
| **Aggregate workflow partition** | Splits broad lifecycle workflows into changed-owner work plus exact revision-bound unaffected outputs. | Kafka selects 3/64 projects, Micronaut 22/75 and Groovy 2/37 while preserving complete outputs. |
| **Profile portfolio / optional central state** | Reuses qualified evidence under exact repository, Wrapper, workflow, graph, output and executable bindings. | Functional locally and across machines; lifetime value for the five newly qualified profiles is the next measurement. |
| **Gradle-compatible cache** | Reuses verified task outputs locally or through optional HTTP/HTTPS storage. | Supporting infrastructure near native-cache parity, not the principal speed claim. |
| **Runtime Tuning, Hot State and standard Copy** | Earlier broad resource/state hypotheses. | Retired after neutral, unstable or regressive end-to-end evidence. |

## Latest public-repository result

One exact BuildOpt binary ran 85 ordinary invocations on five frozen public
repositories. Every candidate reproduced the exact required-output digest;
full-graph fallback remained valid and product-attributable failures were zero.

| Repository / workflow | Graph | Native -> candidate | Mean saving | 95% saving interval | Pairs | Learning / payback | Decision |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Spring `testClasses` | 27 -> 10 | 11.062 -> 9.960 s | **1.102 s / 9.97%** | +0.369..+2.166 s | 8/8 | 4.070 s / 4 builds | Qualified. |
| OpenTelemetry Spring family | 1,024 -> 34 | 75.861 -> 64.534 s | **11.328 s / 14.93%** | +8.866..+14.535 s | 8/8 | 7.357 s / 1 build | Qualified. |
| Kafka `testClasses` | 64 -> 3 | 8.246 -> 5.036 s | **3.210 s / 38.93%** | +2.727..+3.738 s | 8/8 | 3.436 s / 2 builds | Qualified. |
| Micronaut `assemble` | 75 -> 22 | 23.997 -> 9.710 s | **14.287 s / 59.54%** | +12.724..+15.903 s | 8/8 | 7.298 s / 1 build | Qualified. |
| Groovy `classes` | 37 -> 2 | 61.855 -> 15.398 s | **46.456 s / 75.11%** | +41.735..+50.552 s | 8/8 | 2.170 s / 1 build | Qualified. |

The headline is **5/5 automatically qualified and 40/40 positive pairs**.
Every duration is complete installed wall time: Gradle plus BuildOpt discovery,
materialization, verification and remaining wrapper work. Spring now clears the
same thresholds at four-build payback; no exception or weaker gate was added.

The run replaces per-file durable blobs with one manifest-bound pack, bounded
parallel hashing/restoration and direct creation of absent verified outputs.
Capturing Spring's 14,445 files/42.3 MB costs 1.625 seconds; its entire one-time
learning overhead is 4.070 seconds. Content and entry hashes remain mandatory,
and unavailable or corrupt state retains native Gradle.

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
  useful ordinary invocations. Shared discovery and exact one-time accounting
  reduce projected payback from 19–67 builds in the preceding V2 result to
  1–4 builds without removing any measured customer invocation.
- Generic task-graph closure avoids executing unrelated tasks inside selected
  projects while exact materialization preserves the complete workflow output.

It does not yet prove that every change family will match a qualified profile
often enough to produce cumulative customer value. Qualification economics and
cross-commit lifetime are separate questions.

## Next steps

1. Replay the five qualified profiles over compatible public descendant
   changes and measure cumulative savings, selection cost, invalidation and
   fallback rather than assuming calibration value equals lifetime value.
2. Surface learning progress, expected payback and native-retained reasons in
   the one-command experience so users do not need internal JSON.
3. Retain generic task-graph and output contracts while adding new public
   change families; do not infer universal transfer from these five rows.

## Evidence

- [Latest five-repository result](../../benchmarks/results/poc-materialization-economics-v2/README.md)
- [Machine-readable summary](../../benchmarks/results/poc-materialization-economics-v2/summary.json)
- [V2 protocol](../../specs/poc-materialization-economics-v2.md)
- [Detailed performance findings](./build-optimization-performance.md)
- [Generalization audit](./buildopt-generalization-audit.md)
- [One-command roadmap](../plans/one-command-onboarding-roadmap.md)
- [Implementation tracker](../../implementation-tracker.md)
