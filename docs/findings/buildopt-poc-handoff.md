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
| **Incremental learning and value gate** | Accumulates useful control/candidate observations and checks repeatability, uncertainty, p95, outputs, fallback and payback. | The automatic POC policy now accepts 6/8 only with a strictly positive interval and non-regressive p95; weak or incompatible evidence still retains native. |
| **Verified output materialization** | Restores exact unaffected outputs before their producers are omitted. | Fast and fail-closed. A producer-atomic quarantine now excludes every output of a task when any output is volatile, while exact outputs from other producers remain transportable. |
| **Aggregate output closure** | Derives the complete output set of custom lifecycle workflows from Gradle task dependencies and producer ownership. | 4/4 Gradle 8/9 x Groovy/Kotlin fixtures rebuild the changed producer, materialize the stable output and reproduce exact bytes without task-name, plugin, path or extension rules. This is correctness evidence, not a timing claim. |
| **Profile portfolio and central state** | Carries verified profiles and packs over HTTP/HTTPS between builds and machines. | Transport and safe cross-commit refresh have selected value on Kafka (+104.975 s) and Spring (+84.656 s). Micronaut proves exact transitive lineage and recovery, but neither its 58-entrypoint lifecycle cover (+65 ms/0.49%) nor a 63-entrypoint direct frontier (-709 ms/-5.60%) qualifies. |
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
root `classes` lost 206 ms/0.91% with only 1/8 positive pairs. Spring JMS
calibrated 2.590 seconds/11.43% faster with 8/8 positive pairs, but independent
native roots produced 14 different class files out of 8,385 outputs, so no
profile was transported or replayed. OpenTelemetry JMX initially retained
native because its public change mixed three owned JMX paths with an unowned
root changelog. The Kafka-only value claim therefore did not broaden.

Workflow-input ownership and effective-change replay now close the
OpenTelemetry discovery blocker without weakening the value gate. The same
frozen JMX change observes all four changed paths: only unowned `CHANGELOG.md`
has zero consumers and is ignored; the JMX YAML and Java test are consumed,
while the owned `jetty.md` remains relevant. Discovery completes from 1,027 to
8 projects. All eight candidates run and preserve the same 50 required outputs,
but the profile still retains native: 122.044-second control versus
119.333-second candidate saves **2.711 seconds / 2.22%**, with only 5/8 positive
pairs and a **-1.124..+7.330-second** interval. No descendant is timed because
the calibration value is not proven.

Native-volatility quarantine now closes the mechanism gap exposed by Spring
JMS. Two independent output observations are compared exactly; any differing
path quarantines its complete Gradle producer, all outputs of that producer are
rebuilt locally, and only the remaining exact outputs may be transported. The
generic fixture proves atomic quarantine, exact transported-byte verification
and fail-closed ambiguity handling. The historical Spring evidence identifies
14 volatile paths associated with two producer patterns, but it did not retain
the complete producer inventory at that exact revision. Therefore that fixture
alone added no replay or wall-time claim.

The subsequent producer-bound recapture closes that public experiment. Spring
JMS qualifies under the explicit automatic POC policy at **4.62% / 1.362 s**,
6/8 positive pairs, a +375.75..+2,435.75-ms interval and lower p95. Independent
roots retain 7,864 exact transported outputs and quarantine 352 outputs. The
first compatible descendant selects the profile and saves **84.656 seconds /
50.27%** (168.393 s -> 83.737 s). A later `spring-core` change is structurally
and economically ineligible and retains optimized native Gradle. Across both
observations, 8,033 stable outputs remain byte-exact, all 352 quarantined paths
are rebuilt locally, product failures remain zero and cumulative net is
**+59.550 seconds** after qualification/publication cost.

The latest breadth V2 matrix applies that producer-bound path to three new
frozen public windows. All three target commits show real isolated potential:
OpenTelemetry saves **8.53%**, Ktor **56.33%** and Groovy **10.56%**, each with
8/8 positive pairs and exact portable outputs. Across the nine later commits,
however, only one Ktor revision selects the profile. That replay saves
**116.030 seconds / 53.69%**, but the remaining native-retained comparisons
make the three complete windows **-168.751 s**, **-52.237 s** and **-37.684 s**
after qualification/publication. The result is 1/9 selected and 0/3 paid back,
with zero product failures. The cross-commit claim therefore remains bounded.

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
- Cross-commit value is no longer Kafka-only: the unchanged generic path now
  has one selected Kafka replay and one selected Spring replay. It is still not
  universal; other repository/workflow families must qualify independently.
- The three-family breadth V2 run makes that limitation concrete. Target
  acceleration and output portability succeed in 3/3 families, but compatibility
  and fallback economics reduce later reuse to 1/9 commits and 0/3 paid-back
  windows. A fast target benchmark is not sufficient evidence of customer value.
- Native fallback is now close to native cost for every cheap negative case in
  that matrix. Seven of eight retained revisions decide before Gradle, attach
  no output observer and total **9.020 seconds** of directly measured wrapper
  work; the maximum is **2.878 seconds**. All eight authoritative native builds
  preserve exact outputs with zero product failures. The frozen hypothesis
  predicted only six early decisions: Groovy `be211c1b` modifies five nested
  build scripts, so the generic build-logic classifier safely resolved it
  earlier than expected. The selected Ktor control remains exact and saves
  **126.225 seconds / 56.47%**. Single-pair fallback wall deltas remain
  descriptive Gradle noise, not a BuildOpt overhead claim.
- The remaining Groovy ownership ambiguity is now explicitly classified rather
  than widened. Public revision `1ff25776` changes a configuration-time
  `versions.properties` input with no source owner or task consumer. BuildOpt
  returns `CONFIGURATION_INPUT_OWNERSHIP_UNPROVEN`, executes native `classes`
  in 29.710 seconds and preserves all 3,890 required outputs exactly. The fresh
  target calibration that preceded the descendant replay averaged 4.05%
  savings with 6/8 positive pairs, but its -0.262..+2.625-second interval
  crossed zero, so no profile was published. This proves safe attribution and
  also shows that the earlier isolated Groovy value was not repeatable enough
  in this run to justify cross-commit reuse.
- The generic configuration-input follow-up closes that ambiguity without a
  repository rule. Gradle 8.14.3 and 9.6.1, in Groovy and Kotlin DSL, detect,
  report and invalidate a `ProviderFactory.fileContents` input in 4/4 fixture
  cases. The report identifies the file and build-logic read origin, but not
  complete semantic project ownership. BuildOpt therefore retains the full
  requested workflow; no timing is run after this safety precondition fails.
- The first unchanged Micronaut generalization did not broaden that claim. Its
  qualification was **10.33% faster** with 7/8 positive pairs, but the first
  native-retained descendant generated a different Jackson JAR even though no
  BuildOpt profile was selected. The differing bytecode reverses two `Set.of`
  operands, proving that producer volatility can move between revisions.
- The fresh cross-revision portfolio safely learned five volatile Kotlin
  producers from 11,187 outputs, quarantining 476 while leaving 10,711 exact
  outputs transportable. The later revision changed its Wrapper and output
  contract, so BuildOpt returned `NATIVE_RETAINED` before timing. Its native
  pair observed two different volatile JAR producers. Portfolio safety is
  proven, but no new Micronaut replay value is claimed.
- Portfolio compatibility now runs before the additional diagnostic native
  observation. The frozen Micronaut customer build took 526.089 seconds and
  then failed closed on Wrapper/output-contract drift. BuildOpt started zero
  independent native observations and zero timing pairs, avoiding one
  incompatible measurement-only build without misreporting the required
  customer build as a saving.
- The compatible direct-child Micronaut window then passed all four context
  bindings. It selected 22 of 70 projects and captured 190 outputs/172.5 MB in
  2.537 seconds after a 623.348-second ordinary build. Learning left 10,319 of
  11,187 outputs transportable, but eight volatile intermediate producers had
  no proven transitive lineage to the final materialized outputs. BuildOpt
  returned `PORTFOLIO_PRODUCER_LINEAGE_UNAVAILABLE`, ran zero timing pairs and
  claimed no saving. That result authorized the lineage experiment below; it
  did not authorize transport or a performance claim.
- That connection is now implemented generically. The first lineage-aware
  replay quarantined the correct 89 outputs but rebuilt only 11 entrypoints;
  exact verification detected `REQUIRED_OUTPUT_DRIFT` and the full-graph
  fallback restored the required digest. The corrected graph-proven frontier
  rebuilds 58 entrypoints across 52/70 projects, transports 101 outputs and
  completes all eight pairs with one exact digest and zero product failures.
  It is correct but not valuable enough: 13.318 s native versus 13.253 s
  BuildOpt saves **65 ms/0.49%**, only 5/8 pairs improve, the interval crosses
  zero and p95 regresses from 14.267 to 16.967 s.
- The follow-up direct-producer frontier also fails the value test. It keeps
  one exact digest, 101 transported and 89 rebuilt outputs, but uses 63
  entrypoints, still selects 52/70 projects and changes 12.656-second native to
  13.366-second BuildOpt: **709 ms/5.60% slower**, 3/8 positive, interval
  -1.878..+0.500 seconds and worse p95. Gradle accounts for 681 ms of the mean
  regression. The experiment is recorded and reverted; the POC keeps the
  previously verified graph-proven cover.
- Task-level attribution explains why neither exact frontier creates owner
  value. The safe graph-proven candidate executes 110 fewer tasks and removes
  4,731 ms of cumulative task work, but none of those tasks appears on the
  control critical path. Its main-build task span grows 248.375 ms and its
  longest hard-dependency chain grows 178.875 ms. The unchanged causal pairs
  save only 197 ms/1.51%, with 5/8 positive pairs and an interval crossing
  zero. The terminal decision is to stop the Micronaut quarantine line.

## Next steps

1. Rebind qualified profiles across commits from canonical workflow, Wrapper,
   producer-lineage, output-contract and change-family compatibility rather
   than incidental revision identity.
2. Learn duration, graph, portability and volatility only from ordinary
   requested builds, and require expected payback within five compatible
   matches before spending calibration work.
3. Run one unchanged binary over frozen Spring, OpenTelemetry, Kafka,
   Micronaut and Groovy lifetime windows. Require exact outputs, zero product
   failures, at least three net-positive repository families and selection on
   at least half of structurally eligible non-global descendants.
4. Issue a terminal continue/stop decision for the generic one-command POC.
   Production hardening, soak and design partners remain outside this gate.

## Evidence

- [Aggregate output closure result](../../benchmarks/results/poc-aggregate-output-closure-v1/README.md)
- [Machine-readable aggregate closure](../../benchmarks/results/poc-aggregate-output-closure-v1/summary.json)
- [Latest recovery result](../../benchmarks/results/poc-cross-commit-value-recovery-v1/README.md)
- [Workflow-input ownership result](../../benchmarks/results/poc-workflow-input-ownership-v1/README.md)
- [Machine-readable ownership summary](../../benchmarks/results/poc-workflow-input-ownership-v1/summary.json)
- [Source ownership compatibility result](../../benchmarks/results/poc-source-ownership-compatibility-v1/README.md)
- [Configuration-input binding result](../../benchmarks/results/poc-configuration-input-binding-v1/README.md)
- [Machine-readable configuration-input summary](../../benchmarks/results/poc-configuration-input-binding-v1/summary.json)
- [Native-volatility quarantine result](../../benchmarks/results/poc-native-volatility-quarantine-v1/README.md)
- [Machine-readable quarantine summary](../../benchmarks/results/poc-native-volatility-quarantine-v1/summary.json)
- [Producer-bound Spring lifetime result](../../benchmarks/results/poc-producer-bound-quarantine-lifetime-v2/README.md)
- [Micronaut lifetime generalization result](../../benchmarks/results/poc-producer-bound-lifetime-generalization-v1/README.md)
- [Cross-revision volatility portfolio result](../../benchmarks/results/poc-cross-revision-volatility-portfolio-v1/README.md)
- [Machine-readable volatility portfolio summary](../../benchmarks/results/poc-cross-revision-volatility-portfolio-v1/summary.json)
- [Portfolio compatibility preflight result](../../benchmarks/results/poc-portfolio-compatibility-preflight-v1/README.md)
- [Machine-readable preflight summary](../../benchmarks/results/poc-portfolio-compatibility-preflight-v1/summary.json)
- [Compatible portfolio value result](../../benchmarks/results/poc-compatible-portfolio-value-v1/README.md)
- [Machine-readable compatible portfolio summary](../../benchmarks/results/poc-compatible-portfolio-value-v1/summary.json)
- [Transitive producer lineage result](../../benchmarks/results/poc-transitive-producer-lineage-v1/README.md)
- [Machine-readable lineage summary](../../benchmarks/results/poc-transitive-producer-lineage-v1/summary.json)
- [Minimal quarantine frontier result](../../benchmarks/results/poc-minimal-quarantine-rebuild-frontier-v1/README.md)
- [Machine-readable minimal-frontier summary](../../benchmarks/results/poc-minimal-quarantine-rebuild-frontier-v1/summary.json)
- [Quarantine critical-path attribution](../../benchmarks/results/poc-quarantine-critical-path-attribution-v1/README.md)
- [Machine-readable critical-path attribution](../../benchmarks/results/poc-quarantine-critical-path-attribution-v1/attribution.json)
- [Cross-commit breadth V2 result](../../benchmarks/results/poc-cross-commit-breadth-v2/README.md)
- [Machine-readable breadth V2 summary](../../benchmarks/results/poc-cross-commit-breadth-v2/summary.json)
- [Native-retention fast-path result](../../benchmarks/results/poc-native-retention-fast-path-v1/README.md)
- [Machine-readable native-retention summary](../../benchmarks/results/poc-native-retention-fast-path-v1/summary.json)
- [Compatible descendant discovery result](../../benchmarks/results/poc-compatible-descendant-discovery-v1/README.md)
- [Machine-readable compatible descendant summary](../../benchmarks/results/poc-compatible-descendant-discovery-v1/summary.json)
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
