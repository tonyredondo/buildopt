# BuildOpt POC: One-Page Handoff

## The idea

BuildOpt tests whether a generic layer can make substantial Gradle workflows
faster than an already optimized native Gradle baseline. Gradle remains the
execution engine and safe fallback. BuildOpt derives a smaller sufficient task
graph from the exact Git change and requested workflow, materializes verified
unaffected outputs, and enables the candidate only when measured value,
correctness, portability and compatibility all hold.

The intended experience is one command and no repository-specific BuildOpt
files:

```text
install BuildOpt
cd <a Gradle repository>
buildopt optimize <the existing Gradle workflow>
```

This is an owner-operated proof of concept, not a production product. Soak,
design-partner evidence, production SLOs, autonomous promotion and Test
Optimization are outside the current scope.

## Components and current status

| Component | What it does | Current conclusion |
| --- | --- | --- |
| **Structural Build Impact** | Selects only the changed producers and tasks needed by the requested workflow. | The primary accelerator: current isolated calibrations reduce Spring 27→10, OpenTelemetry 1,008→34, Kafka 66→3, Micronaut 70→21 and Groovy 37→2 projects. |
| **Automatic discovery** | Derives Git ownership, finalized workflow inputs, Gradle task/output relationships and candidate graphs without repository-name rules. | Works across `classes`, `testClasses`, `assemble` and the five unrelated public repositories. Complete task-input evidence now lets a mixed OpenTelemetry change ignore only its unconsumed root changelog while retaining module-owned and consumed paths. |
| **Incremental learning and value gate** | Accumulates useful control/candidate observations and checks repeatability, uncertainty, p95, outputs, fallback and payback. | Prevents weak current evidence from being promoted: Groovy retains native at 6/8 and a regressive p95. |
| **Verified output materialization** | Restores exact unaffected outputs before their producers are omitted. | Fast and fail-closed. A producer-atomic quarantine now excludes every output of a task when any output is volatile, while exact outputs from other producers remain transportable. |
| **Profile portfolio and central state** | Carries verified profiles and packs over HTTP/HTTPS between builds and machines. | Transport and safe cross-commit refresh work and one Kafka descendant saved 104.975 seconds. Unchanged replication on Spring and OpenTelemetry found generic ownership and native-output portability blockers, so broader value remains unproven. |
| **Gradle-compatible cache** | Reuses verified task outputs locally or through optional HTTP/HTTPS storage. | Supporting infrastructure near native-cache parity, not the principal speed claim. |
| **Runtime Tuning, Hot State and standard Copy** | Earlier broad resource/state hypotheses. | Retired after neutral, unstable or regressive end-to-end evidence. |

## Latest public-repository evidence

One exact installed executable was used for all five subjects. The current run
requalified each frozen public change, checked native output portability across
independent roots and then followed every preregistered public descendant when
the profile was portable.

| Repository / workflow | Current calibration | Portability | Later builds | Cumulative conclusion |
| --- | ---: | --- | ---: | --- |
| Spring `testClasses` | **18.98% faster**, 8/8 | Rejected: 2 AspectJ classes differ | 0 | **-7.592 s net**; non-portable. |
| OpenTelemetry Spring family | **11.88% faster**, 8/8 | 269 exact outputs | 1 | 0 selections, 1 native fallback; **-13.255 s net**. |
| Kafka `testClasses` | **18.02% faster**, 8/8 | 4,440 exact outputs | 6 | 0 selections, 6 native fallbacks; **-39.961 s net**. |
| Micronaut `assemble` | **13.67% faster**, 8/8 | Rejected: 1 JAR differs | 0 | **-15.457 s net**; non-portable. |
| Groovy `classes` | 6.82% faster, 6/8; p95 worse | Not evaluated | 0 | **-1.835 s net**; current value not proven. |

That frozen V2 baseline was **4/5 qualified, 2/4 portable, 0/7 selected
replays, 7/7 native fallbacks, and 0/5 paid back**. The recovery experiment
then reran the same Kafka qualifier and six descendants after generic product
changes:

| Recovery measurement | Before | After |
| --- | ---: | ---: |
| Selected descendant candidate | 166.299 s | **42.577 s** |
| Attributable selected-replay saving | -5.404 s | **+104.975 s / 71.14%** |
| Six-build cumulative net after learning/publication | -22.040 s | **+66.772 s** |

The selected candidate became 123.722 seconds faster and preserved all 4,449
required outputs exactly. The five native-retained after observations total
-31.441 seconds of uncontrolled arm variation; that value remains visible in
window economics but is not attributed to BuildOpt. Repository percentages are
not averaged and mechanism percentages are not added. These runs used the
12-CPU development host with a common 12-worker cap.

The unchanged breadth replication then tested three non-Kafka subjects. Spring
root `classes` lost 206 ms/0.91% with only 1/8 positive pairs. OpenTelemetry JMX
retained native before calibration because its public change mixed three owned
JMX paths with an unowned root changelog. Spring JMS calibrated 2.590 seconds/
11.43% faster with 8/8 positive pairs, but independent native roots produced 14
different class files out of 8,385 outputs, so no profile was transported or
replayed. The terminal total is **3 subjects, 1 positive calibration, 0 portable
profiles, 0 selected replays and 0 product failures**. The Kafka-only value claim
therefore does not broaden.

Workflow-input ownership now closes the OpenTelemetry discovery blocker without
changing that performance conclusion. The same frozen JMX change observes all
four changed paths: only unowned `CHANGELOG.md` has zero consumers and is
ignored; the JMX YAML and Java test are consumed, while the owned `jetty.md`
remains relevant. Discovery completes from 1,027 to 8 projects with zero product
failures. Calibration is skipped, so this is structural evidence rather than a
new wall-time result.

Native-volatility quarantine now closes the mechanism gap exposed by Spring
JMS. Two independent output observations are compared exactly; any differing
path quarantines its complete Gradle producer, all outputs of that producer are
rebuilt locally, and only the remaining exact outputs may be transported. The
generic fixture proves atomic quarantine, exact transported-byte verification
and fail-closed ambiguity handling. The historical Spring evidence identifies
14 volatile paths associated with two producer patterns, but it did not retain
the complete producer inventory at that exact revision. Therefore this block
adds no replay or wall-time claim; the public experiment must be recaptured.

## What this proves

- Generic graph reduction can beat optimized native Gradle on isolated,
  substantial workflows without repository-specific product branches.
- Calibration speed is not customer value. A profile must remain portable,
  refreshable, and eligible on later commits before its learning cost can repay.
- Fail-open behavior is working: every uncertain descendant ran optimized
  native Gradle and preserved exact outputs.
- Cross-commit reuse can now create attributable value: the one selected Kafka
  descendant saves 104.975 seconds and the six-build window finishes 66.772
  seconds net positive after learning and publication.
- The current POC is not yet general. Unchanged replication found one weak
  calibration, one ownership rejection and one positive-but-nonportable
  calibration. Those are useful generic product gaps, not evidence to broaden
  the one-workflow Kafka claim.

## Next steps

1. Discover descendant windows structurally: preregister a refresh followed by
   an omitted-owner change that can actually select the refreshed profile.
2. Recapture complete producer-bound native observations during those windows,
   apply the quarantine, rebuild volatile producers locally and measure the
   resulting selected replay against optimized native Gradle.
3. Broaden the value claim only after at least two non-Kafka subjects produce
   positive attributable selected replays with exact transported outputs.

## Evidence

- [Latest recovery result](../../benchmarks/results/poc-cross-commit-value-recovery-v1/README.md)
- [Workflow-input ownership result](../../benchmarks/results/poc-workflow-input-ownership-v1/README.md)
- [Machine-readable ownership summary](../../benchmarks/results/poc-workflow-input-ownership-v1/summary.json)
- [Native-volatility quarantine result](../../benchmarks/results/poc-native-volatility-quarantine-v1/README.md)
- [Machine-readable quarantine summary](../../benchmarks/results/poc-native-volatility-quarantine-v1/summary.json)
- [Native-volatility quarantine protocol](../../specs/poc-native-volatility-quarantine-v1.md)
- [Workflow-input ownership protocol](../../specs/poc-workflow-input-ownership-v1.md)
- [Machine-readable recovery summary](../../benchmarks/results/poc-cross-commit-value-recovery-v1/summary.json)
- [Recovery protocol](../../specs/poc-cross-commit-value-recovery-v1.md)
- [Breadth replication result](../../benchmarks/results/poc-cross-commit-breadth-replication-v1/README.md)
- [Machine-readable breadth summary](../../benchmarks/results/poc-cross-commit-breadth-replication-v1/summary.json)
- [Breadth replication protocol](../../specs/poc-cross-commit-breadth-replication-v1.md)
- [Five-repository lifetime baseline](../../benchmarks/results/poc-qualified-lifetime-v2/README.md)
- [Detailed performance findings](./build-optimization-performance.md)
- [Generalization audit](./buildopt-generalization-audit.md)
- [One-command roadmap](../plans/one-command-onboarding-roadmap.md)
- [Implementation tracker](../../implementation-tracker.md)
