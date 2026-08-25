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

## Central two-machine functional evidence

[`poc-central-two-machine-v1.json`](./results/poc-central-two-machine-v1.json)
records the first installed composition across isolated producer and consumer
containers. They share no workspace, Gradle User Home or BuildOpt cache. After
the central TLS service restarts, the consumer selects `CENTRAL_PORTFOLIO`,
uses a `READ_ONLY` gateway, restores one task `FROM-CACHE` and emits the exact
producer JAR. With the service stopped and local cache entries removed, it
retains the verified profile, records zero cache hits and rebuilds the same
bytes. Central credentials are absent from Gradle and logs.

The 136.836-second producer calibration, 11.398-second producer cache build,
7.330-second consumer and 5.877-second outage observations make the phases
auditable but are not comparable arms. They support no savings percentage.
Validate the result with:

```bash
./dev/check-central-two-machine \
  benchmarks/results/poc-central-two-machine-v1.json
```

That functional proof is superseded for value decisions by the terminal paired
experiment below; it remains the smaller restart/outage diagnostic.

## Central end-to-end value evidence

The [terminal central result](./results/poc-central-end-to-end-value-v1/README.md)
compares the complete installed BuildOpt path with optimized native Gradle
under the same committed central-cache opportunity.

| Repository / workflow | Graph | Central objects | Native mean | BuildOpt mean | Direct result | 95% saving interval | p95 | Payback |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Ktor `jvmJar` | 133 -> 10 | 6 / 474,183 B | 215.506 s | 37.828 s | **82.45% faster**, 8/8 pairs | 156.555..198.802 s | 243.986 -> 55.139 s | 28 builds |
| Beam `classes` | 316 -> 6 | 155 / 46,565,962 B | 48.475 s | 21.130 s | **56.41% faster**, 8/8 pairs | 22.407..32.283 s | 53.468 -> 28.319 s | 29 builds |

Every required output is exact. Selection, launcher, gateway, TLS and central
synchronization overhead are inside candidate wall time. A Ktor root
build-logic change retains the full native graph, succeeds with 13 central
cache hits and makes no performance claim. This is bounded 12-CPU POC evidence,
not the contractual golden-runner class, a soak or a universal claim.

```bash
./dev/check-central-end-to-end-value \
  benchmarks/results/poc-central-end-to-end-value-v1/summary.json
```

## Cross-commit profile lifetime evidence

The [Ktor lifetime result](./results/poc-profile-lifetime-v1/README.md) follows
one centrally published Jetty profile through a real public first-parent
sequence. Qualification measured a 58.36% steady-state reduction with a
1,443.324-second learning cost and projected 31-build break-even. The one
matching replay saved 112.198 seconds; an unrelated CORS change safely retained
native but added 220.761 seconds because the POC also discovered the new owner;
a global build-logic invalidation rejected early at native parity.

The three observed builds lost 108.490 seconds before calibration and
1,551.814 seconds including it. This is retained negative economic evidence:
the profile mechanism works, but this public window did not contain enough
matching reuse to justify learning. Validate every classification, exact
output and cumulative calculation with:

```bash
./dev/check-profile-lifetime \
  benchmarks/results/poc-profile-lifetime-v1/summary.json
```

The [economic prequalification follow-up](./results/poc-economic-prequalification-v1/README.md)
uses the same public change classes and prevents the rejected CORS profile
from triggering discovery or calibration. It finds only two analogous recent
commits against the eight-build theoretical payback floor and rejects in
192.442 ms. The observed CORS fallback penalty falls from 220.761 seconds to
13.896 seconds, a 93.71% lower cross-run penalty, while the matching Jetty
replay saves 100.744 seconds and the global build-logic case remains at 22-ms
parity. Qualification still costs 1,386.764 seconds and needs 31 matches, so
the observed window remains **-1,299.894 seconds net** rather than being
rewritten as an economic success.

```bash
./dev/check-economic-prequalification \
  benchmarks/results/poc-economic-prequalification-v1/summary.json
```

### Latest automatic breadth and materialization economics

The [materialization-economics V2 result](./results/poc-materialization-economics-v2/README.md)
runs the unchanged zero-manual-file path on Spring, OpenTelemetry, Kafka,
Micronaut and Groovy with one exact BuildOpt binary after task-graph discovery,
single-pack verified output materialization and end-to-end accounting were
composed.

| Repository | Graph | Direct timed effect | Learning / payback | Decision |
| --- | ---: | ---: | ---: | --- |
| Spring | 27 -> 10 | 9.97% faster, 8/8 | 4.070 s / 4 builds | Qualified. |
| OpenTelemetry | 1,024 -> 34 | 14.93% faster, 8/8 | 7.357 s / 1 build | Qualified. |
| Kafka | 64 -> 3 | 38.93% faster, 8/8 | 3.436 s / 2 builds | Qualified. |
| Micronaut | 75 -> 22 | 59.54% faster, 8/8 | 7.298 s / 1 build | Qualified. |
| Groovy | 37 -> 2 | 75.11% faster, 8/8 | 2.170 s / 1 build | Qualified. |

All 85 ordinary invocations preserve exact outputs and full fallback. All five
repositories pass the unchanged qualification gates and all 40 pairs improve.
Durations include Gradle plus BuildOpt wrapper work. Repository percentages
are deliberately not averaged.

```bash
./dev/check-materialization-economics-v2 \
  benchmarks/results/poc-materialization-economics-v2/summary.json
```

The [qualified-lifetime V2 result](./results/poc-qualified-lifetime-v2/README.md)
then tests whether those isolated wins survive later public commits. Current
qualification remains positive for Spring (**18.98%**), OpenTelemetry
(**11.88%**), Kafka (**18.02%**) and Micronaut (**13.67%**); Groovy retains
native at 6/8 positive pairs and a regressive candidate p95. Only OpenTelemetry
and Kafka pass exact cross-root portability. Across their seven descendant
builds, no profile is selected, all seven exact-output runs retain native and
no subject pays back. Spring and Micronaut stop before observation because two
AspectJ classes and one Micronaut JAR respectively differ across independent
native roots.

```bash
./dev/check-qualified-lifetime-v2 \
  benchmarks/results/poc-qualified-lifetime-v2/summary.json
```

The [ordinary-build lifetime breadth V3 result](./results/poc-lifetime-breadth-v3/README.md)
is the current terminal evidence. One exact executable runs the same five
public repository families under the bounded ordinary-build learning policy.
Spring, OpenTelemetry, Micronaut and Groovy stop after one requested build
because their compatible history cannot repay within five matches. Kafka uses
17 requested builds, qualifies at **21.43% with 8/8 positive pairs**, and
selects one of six eligible descendants. That selected replay saves **135.127
seconds / 75.67%**; Kafka finishes **82.527 seconds net positive** after
qualification, publication and measured fallback wrapper work.

The aggregate contains 21 requested builds, zero measurement-only builds, 27
exact-output observations and zero product failures. Only 1/5 repository
families is net positive and only 1/6 eligible descendants selects, below the
frozen 3/5 and 50% gates. Its terminal result is
`FUNCTIONAL_COVERAGE_NOT_PROVEN`. The signed +50.544-second total is descriptive
only and is neither averaged across repositories nor used to override breadth.

```bash
./dev/check-lifetime-breadth-v3 check \
  ./specs/poc-lifetime-breadth-v3.json \
  ./benchmarks/results/poc-lifetime-breadth-v3
```

The [generic POC functional-coverage decision](./results/poc-functional-coverage-decision-v1/README.md)
applies the complete frozen exit gate to that evidence. Five of eight criteria
pass, including exact outputs, zero product failures, generic selection rules,
robust Kafka qualification and bounded Kafka payback. The breadth criteria fail
at 1/5 net-positive families and 1/6 selected eligible descendants; the only
observed pre-Gradle economic rejection costs 4,098 ms, above both overhead
limits. The resulting decision is `STOP_GENERIC_POC`.

```bash
./dev/check-functional-coverage-decision
```

The [cross-commit recovery result](./results/poc-cross-commit-value-recovery-v1/README.md)
keeps Kafka's qualifier, six public descendants, workflow and output gate
unchanged while comparing the implementation before and after selected-replay
graph repair. The selected candidate changes from 166.299 seconds and -5.404
seconds saved to 42.577 seconds and **+104.975 seconds / 71.14%** saved. All
4,449 required outputs match. The six-build window changes from -22.040 seconds
to **+66.772 seconds net** after learning/publication.

```bash
./dev/check-cross-commit-value-recovery
```

The checker validates both raw results under qualified lifetime v2 and
recomputes the comparison. It keeps native-retained arm delta separate: the
after fallbacks total -31.441 seconds and are not claimed as mechanism value.
This closes one Kafka recovery hypothesis; it does not turn 71.14% into a
universal or cross-repository claim.

### Cross-commit breadth replication

The [breadth replication result](./results/poc-cross-commit-breadth-replication-v1/README.md)
applies the frozen recovery path to three non-Kafka subjects without repository
rules or weaker correctness gates. It is terminal negative breadth evidence:

| Subject | Calibration | Portability / replay | Decision |
| --- | ---: | --- | --- |
| Spring root `classes` | -206.25 ms / -0.91%, 1/8 positive | Not evaluated | Retain native; value not proven. |
| OpenTelemetry JMX `testClasses` | Not measured | Ownership rejected before calibration | Retain native; root changelog is unowned. |
| Spring JMS root `classes` | +2.590 s / +11.43%, 8/8 positive | 14/8,385 native outputs differ | Reject transport; no replay. |

The aggregate is three subjects, one positive calibration, zero portable
profiles, zero selected replays and zero product failures. Spring's 14 differing
class files remain explicit native-output volatility; they are not normalized or
ignored. OpenTelemetry's unowned root changelog remains an ownership gap; it is
not bypassed by a repository-specific exception. Therefore the one-workflow
Kafka value claim does not broaden.

```bash
./dev/check-cross-commit-breadth-replication
```

The earlier negative V2 result remains the immutable breadth baseline. It does
not invalidate graph reduction; it exposed the cross-commit eligibility and
portable-output limitations addressed by the recovery implementation.

### Cross-commit breadth V2

The [breadth V2 result](./results/poc-cross-commit-breadth-v2/README.md)
retests the producer-bound path on frozen OpenTelemetry, Ktor and Groovy
families. Each target qualifies with 8/8 positive pairs and exact portable
outputs: **8.53%**, **56.33%** and **10.56%** lower mean wall time,
respectively.

The later public commits do not sustain that isolated value. OpenTelemetry and
Groovy select zero of three descendants. Ktor selects one and saves
116.030 seconds/53.69%, but its two native-retained comparisons lose 152.684
seconds. After learning/publication, the three windows finish -168.751,
-52.237 and -37.684 seconds. The matrix is therefore 1/9 selected, 0/3 paid
back and zero product failures; the cross-commit claim remains bounded.

```bash
./dev/check-cross-commit-breadth-v2 \
  benchmarks/results/poc-cross-commit-breadth-v2
```

This result does not average repository percentages. Its signed millisecond
total is descriptive window accounting, not a universal performance estimate.
It redirects the next POC work toward cheaper native retention and broader
generic compatibility rather than more isolated calibration wins.

### Native-volatility quarantine

The [quarantine result](./results/poc-native-volatility-quarantine-v1/README.md)
closes the generic mechanism gap exposed by the Spring JMS portability failure.
One volatile output quarantines its complete producer, while outputs from stable
producers remain eligible for exact transport. The executable fixture compares
four outputs, identifies one volatile path, quarantines two producer outputs and
transports two stable outputs. Tampered transported bytes and incomplete
attribution fail closed.

The associated public finding remains intentionally narrower: Spring JMS had 14
different files among 8,385 outputs after an 11.43% positive calibration, and
existing observations associate those patterns with two producers. The exact
revision did not retain the complete producer-bound output set, so no filtered
pack, selected replay or new performance result is claimed.

The later [producer-bound lifetime result](./results/poc-producer-bound-quarantine-lifetime-v2/README.md)
recaptures the complete public boundary and applies that mechanism end to end.
Spring qualifies at 4.62% with 6/8 positive pairs, a strictly positive interval
and lower p95. One compatible descendant selects the profile and saves 84.656
seconds/50.27%; one incompatible `spring-core` descendant retains optimized
native Gradle. Both preserve 8,033 stable outputs exactly and rebuild the same
352 quarantined paths locally. The two-build window finishes 59.550 seconds net
positive with zero product failures.

```bash
./dev/check-producer-bound-quarantine-lifetime
```

The subsequent [Micronaut generalization result](./results/poc-producer-bound-lifetime-generalization-v1/README.md)
keeps the producer-bound protocol unchanged. Its public qualification reduces
70 projects to 2 and saves **4,092.375 ms / 10.33%**, with 7/8 positive pairs
and a positive interval. Target-revision quarantine transports 193 exact
outputs and rebuilds one volatile HTTP-server JAR locally.

The first preregistered descendant retains optimized native Gradle because its
source/checkpoint bindings drift. Those two native arms nevertheless differ in
one Jackson JAR: two generated classes reverse the operands passed to
`Set.of`. BuildOpt selected no profile and restored no outputs, so this is
native cross-revision volatility rather than product corruption. Exact hashes
remain mandatory; no descendant performance result is claimed, the remaining
descendants are not timed, and the experiment finishes **-11.676 seconds net**.

```bash
./dev/check-producer-bound-lifetime-generalization
```

The [cross-revision volatility portfolio result](./results/poc-cross-revision-volatility-portfolio-v1/README.md)
then captures a fresh diagnostic pair at the frozen learning revision. It
compares 11,187 outputs, quarantines 476 outputs from five volatile Kotlin
producers and leaves 10,711 outputs exactly transportable. On the later public
revision, canonical repository and workflow identity remain stable while the
Gradle Wrapper and output contract drift. The product returns structured
`NATIVE_RETAINED` and stops before all timing pairs, so no replay percentage is
claimed. The current-revision native pair also observes two different volatile
JAR producers among 186 outputs, demonstrating that a single pair cannot be
treated as a universal producer list.

```bash
./dev/check-cross-revision-volatility-portfolio
```

The [portfolio compatibility preflight result](./results/poc-portfolio-compatibility-preflight-v1/README.md)
moves the context decision ahead of the extra native observation. On the
frozen Micronaut rejection, the required customer `assemble` completed in
526.089 seconds; the preflight then identified Wrapper and output-contract
drift, retained native and proved that no independent native clone or second
Gradle workflow started. It avoids one incompatible measurement-only build but
does not count the customer build as a saving or claim a performance
percentage.

```bash
./dev/check-portfolio-compatibility-preflight
```

The same evidence preregisters a direct-child Micronaut window with identical
repository, workflow and Wrapper bindings. Its output contract must still pass
runtime preflight before any independent observation or timing is allowed.

The [compatible descendant discovery result](./results/poc-compatible-descendant-discovery-v1/README.md)
reruns the frozen OpenTelemetry JMX window after the ownership correction. The
candidate is functionally correct in 8/8 pairs and reduces 1,027 projects to 8,
but its 2.22% mean saving has only 5/8 positive pairs and a saved-time interval
that crosses zero. The profile remains unpublished and the Kafka-only
cross-commit value claim is not broadened.

The preceding [automatic breadth V2 result](./results/poc-automatic-breadth-transfer-v2/README.md)
is retained as immutable before-evidence; it qualified 4/5 and exposed the
learning/materialization cost addressed by this follow-up.

### Incremental ordinary-build learning

The [incremental learning result](./results/poc-incremental-learning-v1/README.md)
replaces the synchronous 16-build calibration transaction with one exact-bound
observation per useful `buildopt optimize` invocation. The executable fixture
completed one baseline plus eight alternating pairs with nine full-graph
workflows, eight structural candidates and **zero measurement-only workflow
runs**. All required JAR digests matched and fallback remained valid.

The three-project fixture saved only 50.125 ms/0.90% on average, produced 4/8
positive pairs and a -275.75..+405.125-ms interval, while 19,247 ms of
incremental BuildOpt work projected 384-build payback. The unchanged gates
therefore retained native Gradle. This result proves the cheaper learning
transaction and honest rejection; it is not a transferable speed claim and is
not numerically combined with the public-repository breadth results.

```bash
./dev/check-incremental-learning
```

### Verified unaffected-output materialization

The [materialization result](./results/poc-verified-output-materialization-v1/README.md)
tests the clean-workspace correctness gap separately from timing. A full
three-project `assemble` records three required JARs. The reduced candidate
then rebuilds the changed `service-a` JAR and restores the unaffected
`library-c` and `service-b` JARs from exact content-addressed state. Baseline,
candidate and native fallback produce the same three-file digest.

After a retained blob is deliberately corrupted, BuildOpt rejects the state
before candidate execution and runs the full graph successfully. Unit coverage
also rejects stale workspace bytes without overwriting them and prevents a
corrupt state set from producing a partial materialization. This evidence
authorizes the mechanism only; it contains no wall-time or payback claim.

```bash
./dev/check-verified-output-materialization
```

### Aggregate workflow partition

The [aggregate partition result](./results/poc-aggregate-workflow-partition-v1/README.md)
tests the remaining broad-lifecycle correctness gap. A generic 66-project
`assemble` workflow has one directly changed owner and 65 transitive consumers.
The previous flat candidate required 66 entrypoints and exceeded the unchanged
64-entrypoint bound. The partitioned candidate runs only `:core:assemble`,
materializes 65 revision-bound consumer JARs and produces the same complete
66-JAR digest as the full graph.

This result does not contain timing or payback evidence. It proves a generic
producer/selector/variant partition and exact clean-workspace composition;
the next five-public-repository transfer owns the performance decision.

```bash
./dev/check-aggregate-workflow-partition
```

## Build Optimization scorecard

For the decision-ready product summary, see the [current POC one-pager](../docs/findings/buildopt-poc-handoff.md).
The [detailed performance findings](../docs/findings/build-optimization-performance.md)
retain mechanism-specific and historical experiments for engineering review.

### Latest generic blocker closure

The [workflow-input ownership evidence](./results/poc-workflow-input-ownership-v1/README.md)
replays the OpenTelemetry JMX change that previously retained native on mixed
owned and unowned paths. Complete finalized Gradle task-input evidence ignores
only unconsumed root `CHANGELOG.md`; the YAML and Java test remain consumed and
the owned `jetty.md` remains relevant. Discovery completes from 1,027 to 8
projects with zero product failures.

Calibration is skipped, so this is structural correctness evidence rather than
a new performance row. The Kafka-only selected-replay value claim remains
unchanged.

### Current automatic one-command terminal result

The [published terminal evidence](./results/poc-magic-end-to-end-value-v2/README.md)
tests the customer-shaped `buildopt optimize` path with public `v0.6.1`,
fresh package/checkouts/BuildOpt state and zero manual BuildOpt files.

| Repository / workflow | Graph | Native mean | BuildOpt mean | Direct result | Payback |
| --- | ---: | ---: | ---: | ---: | ---: |
| Ktor `jvmJar` | 133 -> 10 | 38.810 s | 7.830 s | **79.82% faster**, 8/8 pairs | 26 builds |
| Beam `classes` | 316 -> 6 | 65.081 s | 24.958 s | **61.65% faster**, 8/8 pairs | 28 builds |

Both rows have positive paired intervals, lower candidate p95, exact required
outputs, stable task shapes, successful full-graph fallback and zero
product-attributable failures. An honest Ktor root build-logic case completes
native Gradle and retains it with `GLOBAL_CHANGE_REQUIRES_FULL_GRAPH` without
calibration or a performance claim.

Both arms receive identical content-bound dependencies and native-cache seeds;
daemon/configuration warmup is outside measured samples. The result therefore
compares the installed POC with optimized native Gradle rather than a cold
download or unequal cache opportunity.

Validate every retained result, pair, output hash, p95, fallback, economic
calculation, package binding and native case with:

```bash
./dev/check-magic-end-to-end-value-v2
```

### Historical automatic diagnostic matrix

The earlier [v1 automatic matrix](./results/poc-magic-end-to-end-value-v1/README.md)
remains immutable diagnostic evidence. It originally qualified only Ktor;
Spring and Beam were directly positive but exceeded the 30-build payback,
Groovy and Kafka regressed, and Micronaut safely stopped before timing. The
later [Beam calibration-cost preflight](./results/poc-magic-calibration-cost-v1/README.md)
made Beam economically viable by reducing comparable learning cost from
1,097.547 to 988.145 seconds and payback from 33 to 26 builds. Those development
runs motivated, but are not substituted for, the public-package terminal v2
capture.

```bash
./dev/check-magic-end-to-end-value
./dev/check-magic-calibration-cost
```

The reviewed-profile results below remain useful feasibility evidence. They
must not be presented as the current zero-configuration onboarding result.

### Balanced public-repository qualification

The [terminal v2 evidence](./results/poc-statistical-qualification-v2/README.md)
re-measures Spring, OpenTelemetry, Kafka, Micronaut, and Groovy under one
preregistered protocol and one exact BuildOpt revision. Each repository has two
independent eight-pair captures; adjacent opposite-order observations form
eight AB/BA blocks.

| Repository | Optimized native mean | BuildOpt mean | Direct result | Decision |
| --- | ---: | ---: | ---: | --- |
| Spring Framework | 13.311 s | 11.183 s | **15.99% faster**, 8/8 blocks | Qualify |
| OpenTelemetry | 87.869 s | 74.713 s | **14.97% faster**, 8/8 blocks | Qualify |
| Apache Kafka | 113.381 s | 14.341 s | **87.35% faster**, 8/8 blocks | Qualify |
| Micronaut Core | 30.411 s | 18.418 s | **39.44% faster**, 8/8 blocks | Qualify |
| Apache Groovy | 79.868 s | 20.767 s | **74.00% faster**, 8/8 blocks | Qualify |

All 80 raw pairs improved. Every row also passes its material-mean, positive
median/lower-bound, 6-of-8 repeatability, non-regressive p95, exact-output,
stable-shape, two-fallback, and zero-product-failure gates. Percentages remain
repository-specific and are not averaged or added to other mechanisms.

This is the latest directly comparable five-repository matrix. Older sections
below remain immutable historical evidence and may use different protocols;
they must not be substituted into this table.

### Clean-CI proposal replay

The [five-repository hosted replay](./results/poc-generic-profile-ci-replay-v1/README.md)
recreated Spring, OpenTelemetry, Kafka, Micronaut, and Groovy from their exact
public revisions on five independent GitHub-hosted runners. All **5/5** owner
inputs, changes, candidate plans, manifests, declared graphs, generated
bindings, fallback inputs, and checksums matched their terminal references;
drift was **0/5** and no active profile was written. This proves proposal
reproducibility only. It reruns no timing and does not change the wall-time
decisions below.

### Generic workflow breadth

The [hosted workflow-breadth result](./results/poc-generic-workflow-breadth-v1/README.md)
uses one unchanged confirmed owner-input contract for packaging, typed
verification, distribution, and build-owned test preparation. All four cells
selected only the changed `service-a` task, omitted `service-b`, rebuilt the
declared output byte for byte, and executed no Gradle `Test`. An arbitrary
executable workflow retained native before structural state or timing.

This is capability and fallback evidence only. It contains zero timing
observations and creates no performance claim.

### Public workflow-family value

The [public workflow value result](./results/poc-generic-workflow-value-v1/README.md)
then applies the installed path to substantial Groovy, Kafka, and Spring
workflows under one preregistered eight-pair protocol. Spring
`testClasses` preserved exact class outputs and averaged **18.47% / 2.695 s**
faster with a positive 95% interval, but retained native Gradle because only
7/8 pairs improved under the frozen 8/8 gate.

The other three cells failed closed before timing could become value evidence:
Groovy embeds `BuildTime` in an otherwise reproducible JAR, Kafka Checkstyle
embeds the isolated checkout path, and Kafka explicitly preserves timestamps
and non-reproducible order in the measured fat JAR. Two native Kafka rebuilds
had different archive hashes despite identical names and payloads for all
4,378 entries. The terminal result is therefore **0/4 qualified** with all four
families retaining native Gradle. This is useful correctness evidence and a
concrete next generalization target; it is not a universal workflow-value
claim.

### Owner-reviewed semantic output equivalence

The subsequent [terminal output-equivalence bundle](./results/poc-generic-output-equivalence-v1/README.md)
repeats the three representation-blocked workflows with strict, digest-bound
owner contracts. Byte identity remains the default. Groovy permits only its
declared `BuildTime` property to vary, Kafka Checkstyle relocates only the
isolated repository root in `main.xml`, and Kafka `shadowJar` compares
canonical ZIP entry content rather than container timestamps and order.

| Repository and workflow | Optimized native mean | BuildOpt mean | Direct result | Decision |
| --- | ---: | ---: | ---: | --- |
| Apache Groovy `jar` | 72.319 s | 19.455 s | **52.864 s / 73.10% faster**, 8/8 blocks | Qualify for review |
| Apache Kafka `checkstyleMain` | 82.835 s | 58.209 s | **24.627 s / 29.73% faster**, 8/8 blocks | Qualify for review |
| Apache Kafka `shadowJar` | 40.728 s | 13.625 s | **27.103 s / 66.55% faster**, 8/8 blocks | Qualify for review |

All 48 raw pairs improve. Every semantic output comparison passes, candidate
p95 is lower, measured and final warm-up task shapes are stable, both native
fallbacks pass per workflow, and product-attributable failures remain zero.
The conformance suite still rejects undeclared report, property, and archive
payload drift. No repository-name product rule or automatic activation was
added, and percentages remain workflow-specific rather than averaged.

### Generic change-shape breadth

The [terminal change-breadth matrix](./results/poc-generic-change-breadth-v1/README.md)
tests two distinct source edits per workflow plus independent build-logic and
global-configuration fallbacks. The same output-owner-rooted implementation is
used throughout; the product contains no Groovy or Kafka branch.

| Repository and workflow | Change | Optimized native mean | BuildOpt mean | Direct result | Decision |
| --- | --- | ---: | ---: | ---: | --- |
| Apache Groovy `jar` | leaf source | 71.227 s | 18.843 s | **73.54% faster**, 8/8 blocks | Qualify for review |
| Apache Groovy `jar` | shared source | 79.557 s | 27.207 s | **65.80% faster**, 8/8 blocks | Qualify for review |
| Kafka `checkstyleMain` | metadata source | 82.431 s | 59.353 s | **28.00% faster**, 8/8 blocks | Qualify for review |
| Kafka `checkstyleMain` | client-utils source | 87.522 s | 61.177 s | **30.10% faster**, 8/8 blocks | Qualify for review |
| Kafka `shadowJar` | clients source | 44.828 s | 14.956 s | **66.64% faster**, 8/8 blocks | Qualify for review |
| Kafka `shadowJar` | generator source | 37.079 s | 7.587 s | **79.54% faster**, 8/8 blocks | Qualify for review |

All 96 raw pairs and 48 reciprocal blocks improve; outputs, task shapes,
tails, 12 selective fallbacks, and zero-failure gates pass. Four additional
build-logic/global cells produce eight independent
`NATIVE_FULL_GRAPH / GLOBAL_CHANGE_REQUIRES_FULL_GRAPH` decisions and no
timing claim. Percentages remain cell-specific and are not averaged.

### Calibration economics

The [terminal calibration bundle](./results/poc-calibration-economics-v1/README.md)
separates two fresh setup captures per selective change-breadth cell from the
already-qualified terminal saving. The installed-workflow view counts the real
combined output-preflight/discovery command and candidate warm-ups; the full
POC view also counts native-control warm-ups.

Installed payback is **10–11 builds** for Groovy JAR, **27–31 builds** for
Kafka Checkstyle, and **14–15 builds** for Kafka `shadowJar`. Complete
comparative-POC payback is **20–22**, **49–55**, and **27–28 builds**
respectively. Offline Gradle distribution materialization adds under 1.4
seconds per capture and does not change any rounded POC break-even. Checkout is
measured but excluded as shared native/BuildOpt work. Results and percentages
remain cell-specific and are never averaged.

### Calibration efficiency

The [terminal efficiency bundle](./results/poc-calibration-efficiency-v1/README.md)
repeats all six setup cells after fusing structural discovery, adding exact
digest-bound proposal replay and proving a two-of-three candidate-stability
stop against the previously recorded task fingerprints.

Cold discovery is **8.01% to 21.08% lower** in every cell. Exact proposal
replay takes **0.281 to 1.261 seconds** and regenerates all six contract
artifacts byte for byte; an added Gradle option misses the cache in all 12
captures. Installed break-even improves from 10–31 to **9–26 qualifying
builds**, while complete-POC break-even improves from 20–55 to **19–50**.
Existing terminal savings are immutable and remain cell-specific.

### Public installed qualified-profile replay

The [installed-profile replay bundle](./results/poc-installed-profile-replay-v1/README.md)
tests the six reviewed profiles through the public `v0.3.1` Linux AMD64
package rather than the source-tree research runner. All six exact profiles
select their reviewed entrypoints; harmless byte drift in the bound manifest
then produces six `FULL_GRAPH / PROFILE_PRECONDITION_FAILED` decisions. In
every cell the selective candidate and its contemporary native fallback have
the same owner-reviewed semantic output, and the embedded terminal
qualification remains unchanged.

Four Kafka outputs also match the historical terminal digest. Two Groovy JARs
do not under their immutable historical contract because it excludes
`BuildTime` but not the date-dependent `BuildDate`; direct native Gradle on the
same frozen checkout matches the current BuildOpt digest. Cross-capture
identity is therefore diagnostic for that historical bundle.

```bash
./dev/check-installed-profile-replay-result
./dev/test-installed-profile-replay-result
```

### Reviewed cross-date output equivalence

The [cross-date bundle](./results/poc-cross-date-output-equivalence-v1/README.md)
closes that diagnostic gap for future evidence without editing old profiles.
Four Kafka cells retain their natural independent-capture matches. Two fresh
Groovy real-JAR probes change only `BuildDate`: the historical
`BuildTime`-only contract rejects both changes, while the reviewed
`BuildDate + BuildTime` contract matches both. Changing undeclared
`ImplementationVersion` or a `.class` payload still produces a mismatch in
2/2 cells.

The aggregate is **6/6 cross-date comparable**, with six unchanged historical
eight-pair qualifications and zero product failures. The Groovy boundary is a
controlled correctness probe, not a new timed build, so no saving or
percentage is recomputed.

```bash
./dev/check-cross-date-output-equivalence-result
./dev/test-cross-date-output-equivalence-result
```

### Unseen Hibernate ORM holdout

The [terminal holdout bundle](./results/poc-generic-holdout-v2/README.md)
applies the unchanged generic installed path to Hibernate ORM, which was not
part of proposal development or the five-repository matrix. From root
`assemble`, one exact `Session.java` change and repository-declared core JARs,
BuildOpt discovered a 29-to-1-project candidate without repository-specific
product logic.

| Metric | Optimized native | BuildOpt candidate | Result |
| --- | ---: | ---: | --- |
| Mean wall time | 248.481 s | 229.095 s | **19.386 s / 7.80% faster** |
| Positive pairs | — | 7/8 | Retain native under the frozen 8/8 gate |
| Paired 95% interval | — | — | **+9.719..+29.210 s** |
| Required outputs | 3 JARs | Same 3 JARs | Byte-identical in all pairs |

The full-graph fallback succeeded and no product-attributable failure occurred.
The result supports the generic structural hypothesis but does not qualify the
Hibernate scope. The separately retained
[v1 attempt](./results/poc-generic-holdout-v1-attempt1/README.md) contains zero
timings: it failed closed because the initial owner output glob used Gradle's
default `build` directory instead of Hibernate's configured `target`
directory. Validate both records with:

```bash
./dev/check-generic-holdout
./dev/check-generic-holdout \
  benchmarks/results/poc-generic-holdout-v1-attempt1 \
  specs/poc-generic-holdout-v1.json
```

The later [generic output-contract preflight](./results/poc-generic-output-contract-v1/README.md)
replays that original wrong declaration through the installed proposal path.
After one exact owner-workflow execution it reports the declared pattern as
empty, exposes the Gradle-owned `hibernate-core/target/libs` candidates, and
retains native before structural discovery, warm-up or timing. Validate the
frozen observation with:

```bash
./dev/check-generic-output-contract-evidence
```

This adds a fail-early usability/correctness result, not another performance
percentage.

The [v3 diagnostic bundle](./results/poc-generic-holdout-v3/README.md) then
investigates rather than discarding the v2 negative. A second excluded base
warm-up reduced control/candidate preparation by 87.30%/85.80% and recovered
the first pair to +11.883 seconds, proving the original −1.118-second result was
not a structural reason to reject BuildOpt. The complete fresh run still
reached only 4/8 positive pairs: 2.50% mean savings with interval
−6.604..+20.190 seconds. Task outcomes stayed at 301/32 through pair 7, while
both arms continued accelerating and the control changed to 300 tasks in pair
8. Version 3 therefore retains native and motivates target-workload warm-up,
exact task fingerprinting and interval-scoped host PSI; no old timing is
discarded or reused.

The separately preregistered
[v4 attribution bundle](./results/poc-generic-holdout-v4/README.md) executes
that correction from fresh arms. It reduces the raw means to 221.898 seconds
for native and 213.418 seconds for BuildOpt, a positive 8.480-second/3.82%
signal, but retains native at 5/8 positive pairs with interval
-0.839..+17.478 seconds. The candidate is structurally stable at 32 tasks;
the native control moves between 300, 301 and 302 tasks, and all four
control-first pairs are positive versus one of four candidate-first pairs.
Four post-hoc AB/BA crossover blocks are positive, so the result diagnoses a
recoverable warm-up/period problem rather than a structural-product failure.
Those blocks are diagnostic only and cannot qualify a future run.

The fresh [v5 reciprocal result](./results/poc-generic-holdout-v5/README.md)
then runs the preregistered correction rather than reusing those blocks. Two
independent batches produce eight reciprocal observations. Native Gradle
averages 216.724 seconds and BuildOpt 203.991 seconds, saving **12.733
seconds/5.88%** with interval **+6.808..+19.859 seconds** and **8/8 positive
blocks**. Exact target shapes remain at 300/32 tasks, required JARs are
byte-identical and both full-graph fallbacks pass. The terminal decision is
`REVIEW_STRUCTURAL_PROFILE`, not automatic activation.

### Terminal generic structural matrix

The [terminal v3 five-repository bundle](./results/poc-generic-profile-matrix-v3/README.md),
separately preregistered [OpenTelemetry v4 correction](./results/poc-generic-profile-matrix-v4/README.md),
and unseen [Hibernate v5 crossover](./results/poc-generic-holdout-v5/README.md)
provide the latest comparable Build-Impact-only evidence:

| Repository | Native mean | BuildOpt mean | Direct result | Decision |
| --- | ---: | ---: | ---: | --- |
| Spring Framework | 13.940 s | 11.438 s | **17.94% faster**, 7/8 positive | Retain native under the frozen 8-of-8 gate |
| OpenTelemetry | 83.934 s | 71.825 s | **14.43% faster**, 8/8 positive | Qualified by v4 fallback-equivalence proof |
| Apache Kafka | 82.498 s | 13.113 s | **84.11% faster**, 8/8 positive | Qualified |
| Micronaut Core | 27.407 s | 15.968 s | **41.74% faster**, 8/8 positive | Qualified |
| Apache Groovy | 75.064 s | 19.629 s | **73.85% faster**, 8/8 positive | Qualified |
| Hibernate ORM | 216.724 s | 203.991 s | **5.88% faster**, 8/8 reciprocal blocks | Review structural profile |

All accepted observations preserve required outputs byte for byte and include
launcher/planning overhead. OpenTelemetry v4 changes only the untimed fallback
execution mode; its timed conditions and thresholds are identical to v3. Do
not average these repository percentages or add them to other mechanism
effects. Hibernate uses reciprocal blocks because its preregistered diagnostic
proved raw pairs were materially order-sensitive; every value and correctness
threshold remains unchanged.

### Historical whole-profile composition evidence

This earlier scorecard records differently composed mechanisms and is not
directly comparable with the terminal structural-only matrix above.

[`results/poc-general-build-value-v1.json`](./results/poc-general-build-value-v1.json)
binds the direct end-to-end result for every exact target composition without
rerunning unchanged inputs or adding component percentages:

| Target | Complete measured mechanisms | Direct result | Installed replication |
| --- | --- | ---: | --- |
| Spring Framework | Build Impact | **30.86% faster** | Positive 14.33%, but 7/8 pairs failed the frozen all-positive gate |
| OpenTelemetry | Build Impact + standard `Jar` | **50.40% faster** | No accepted replication timing after preparation failure |
| Kafka | Build Impact + read-only Edge | **82.35% faster** | **81.85% faster**, 8/8 pairs |

Hot State is excluded because it directly regressed on OpenTelemetry. Runtime
Tuning, Hot State, and standard `Copy` are retired from the executable POC;
Safe Cache and Test Optimization are not part of these
target compositions. The exact scorecard has three positive direct scopes but
only one strict installed replication, so it supports continued structural
experiments rather than a general accelerator claim. Validate source digests,
calculations, outputs and replication status with:

```bash
./dev/check-poc-general-build-value
```

### Fresh structural transfer: Micronaut Core

[`results/poc-structural-transfer-v1.json`](./results/poc-structural-transfer-v1.json)
records the next fresh installed replication on Micronaut Core. The public
revision is substantial: complete root `assemble` executed 360 tasks, and
repository-independent discovery found a complete 75-project control reach and
22-project candidate reach, a potential 53-project/70.67% omission.

The initial installed candidate correctly retained the full graph because
Micronaut's cyclic source-set expansion made one direct root appear owned by 39
projects. That immutable zero-observation result remains in
[`results/poc-structural-transfer-v1-native-stop.json`](./results/poc-structural-transfer-v1-native-stop.json).
The generic ownership correction now preserves the expanded boundary for
conservative affected closure while using only original project roots for
direct ownership. Fresh discovery resolves exactly one owner,
`:micronaut-http-client-jdk`, inside the candidate.

The unchanged replay then qualified against optimized native Gradle:

| Metric | Optimized native control | Installed structural candidate | Difference |
|---|---:|---:|---:|
| Mean wall clock | 24,067.125 ms | 6,505.5 ms | **17,561.625 ms / 72.97% faster** |
| Alternating pairs | 8 | 8 | **8/8 positive** |
| 95% paired interval | — | — | **+17,018.875..+18,118.375 ms** |
| Required output | 3 JARs | Same 3 JARs | Byte-identical in every pair |

A global `gradle.properties` change restored the full graph, no Gradle `Test`
task ran, and no other optimization mechanism was enabled. This qualifies the
fixed structural Micronaut scope; it is not a universal savings claim. Validate
the current evidence and the historical fail-closed result with:

```bash
./dev/check-poc-structural-transfer-v1
./dev/check-poc-structural-transfer-v1 \
  benchmarks/results/poc-structural-transfer-v1-native-stop.json
```

### Generic installed structural-profile adoption

[`results/poc-structural-profile-adoption-v1.json`](./results/poc-structural-profile-adoption-v1.json)
measures the reviewed profile through the packaged product path rather than
through experiment-only flags. `buildopt profile qualify` first materializes a
Build-Impact-only v4 profile from the independently checked structural result;
the timed candidate then invokes only `buildopt poc --changes-file`.

| Metric | Optimized native control | Installed qualified profile | Difference |
|---|---:|---:|---:|
| Mean wall clock | 23,642.75 ms | 6,581.625 ms | **17,061.125 ms / 72.16% faster** |
| Alternating pairs | 8 | 8 | **8/8 positive** |
| 95% paired interval | — | — | **+16,243.75..+17,942.25 ms** |
| Required output | 3 JARs | Same 3 JARs | Byte-identical in every pair |

Profile validation, planning and launcher overhead are included; only profile
materialization is outside the timed path. The installed result is 0.81
percentage points below the direct structural experiment, so adoption retains
almost all measured value. A global change and a whitespace-only graph drift
both restored the full graph. The product implementation contains no
repository-name rule, but this percentage remains bound to the fixed Micronaut
revision, change and outputs. Validate it with:

```bash
./dev/check-poc-structural-profile
./dev/check-poc-structural-profile-adoption-v1
```

### Fresh generic measurement: Apache Groovy classes

[`results/poc-apache-groovy-classes-v1`](./results/poc-apache-groovy-classes-v1/)
contains the complete checked input and output bundle from Apache Groovy 5.0.8:
manifest, generated graph, generated-state binding, exact changed paths,
fallback paths, eight-pair measurement, and the derived review-required
profile.

| Metric | Optimized native control | Installed structural candidate | Difference |
|---|---:|---:|---:|
| Mean wall clock | 92,350.625 ms | 46,119.875 ms | **46,230.75 ms / 50.06% faster** |
| Alternating pairs | 8 | 8 | **8/8 positive** |
| 95% paired interval | — | — | **+44,190.25..+47,846.875 ms** |
| Required output | 66 class files | Same 66 class files | Byte-identical in every pair |
| Project reach | 37 | 2 | 35/37 or 94.59% omitted |

Both arms used the same optimized-native options, including build cache,
parallel execution and four workers. Launcher/planning overhead is included.
A global `gradle.properties` change restored the full graph. The earlier
distribution candidate was rejected because its ZIP bytes differed, and root
`assemble` was stopped before timing because it included unrelated docs work.
The exact classes result therefore qualifies only this declared output scope.

Validate hashes, calculations, profile determinism and tamper fallback with:

```bash
./dev/check-poc-apache-groovy-classes-v1
```

### Public-repository onboarding replay

[`results/poc-generic-profile-realworld-v1`](./results/poc-generic-profile-realworld-v1/)
records the installed `buildopt profile propose` workflow starting from fresh
checkouts rather than retained BuildOpt inputs. Apache Groovy reproduced the
37-to-2 project plan and Micronaut reproduced the 75-to-22 plan. Both accepted
proposal passes ran offline after excluded dependency preparation.

The structural plans are exactly the ones already qualified, so this block did
not rerun timing or create a new percentage. The retained **50.06%** Groovy and
**72.16%** Micronaut results remain the value evidence; this replay proves that
the user-facing setup can rediscover those candidates without hand-authored
BuildOpt JSON. Validate the complete evidence bundle with:

```bash
./dev/check-generic-profile-realworld
```

The current POC verdict is `CONTINUE` for exact evidence-qualified scopes, not
for arbitrary repositories. Contractual 4-vCPU/16-GiB runs cover the baseline,
negative-mechanism decision, accelerator-coverage matrix, combined public path,
and realistic breadth test. Safe Cache is explicit-only while the default delegates to Gradle's native
cache; Runtime Tuning candidates `W4_H6G` and `W3_H4G` are retired; Build
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

### Retired Runtime Tuning research

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
`RETAIN_NATIVE_12_WORKERS`: Runtime Tuning was retired from the POC after this
result. Its runner and activation code were removed; the frozen evidence is
retained to prevent another parameter search over the same failed mechanism.

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
| Runtime Tuning `W3_H4G` | 512 ms slower (4.3%) | −2,818 to +1,302 ms | `NO_VALUE_NO_ACTION`; mechanism now retired |
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

### Kafka remote-composition precondition result

The [qualified remote-composition evidence](./results/poc-qualified-remote-composition-v1.json)
is a terminal diagnostic, not a timing result. Before committing any cache
object or opening Edge, the installed seed proved that Kafka's required
`kafka-clients-4.3.1.jar` is produced by custom `:clients:shadowJar` while
`:clients:jar` is `SKIPPED`. The observed shaded artifact also did not match the
historical standard-`Jar` premise.

The independent checker therefore accepts only
`STOP_KAFKA_REMOTE_COMPOSITION_INVALID_STANDARD_JAR_PRECONDITION`: zero Shared
objects, no Edge server, no warm-up and zero measured pairs. The prior 57.58%
Kafka packaging result remains valid for its fixed Build Impact scope, but it
must not be attributed to the standard-`Jar` adapter. The corrected next
experiment may compose Kafka-qualified Build Impact with Edge locality only.

```bash
./dev/check-poc-qualified-remote-composition-v1-result
```

### Kafka Build Impact and Edge composition

The corrected [terminal composition evidence](./results/poc-kafka-impact-edge-composition-v1.json)
removes the disproved standard-`Jar` premise and combines only Kafka-qualified
Build Impact with prewarmed Edge locality. The native control runs full root
`assemble` through Shared; the installed candidate selects the three-project
packaging scope and reads the same admissible committed objects through Edge.

| Pair | Order | Native + Shared | Impact + Edge | Diagnostic saving |
|---:|---|---:|---:|---:|
| 1 | Native first | 52,143 ms | 10,059 ms | 42,084 ms |
| 2 | Candidate first | 42,040 ms | 7,159 ms | 34,881 ms |
| 3 | Native first | 41,630 ms | 6,361 ms | 35,269 ms |
| 4 | Candidate first | 37,567 ms | 6,113 ms | 31,454 ms |

The means are 43,345 ms and 7,423 ms: a diagnostic 35,922-ms/**82.87%**
difference with 4/4 positive pairs and interval
+32,407.75..+40,283.25 ms. Every measured arm restored custom
`:clients:shadowJar`, matched the same cached artifact, kept the standard-Jar
adapter disabled, and the candidate made zero Shared requests.

This is **not a qualified savings claim**. When Edge returned HTTP 503, native
Gradle correctly disabled the remote cache and completed the candidate build,
but rebuilding custom `shadowJar` produced different bytes from the cached
artifact. The mandatory fallback-equivalence gate therefore failed and the
decision is `RETAIN_SEPARATE_KAFKA_IMPACT_AND_EDGE`. The next experiment must
attribute and stabilize that custom archive before any recomposition.

```bash
./dev/check-poc-kafka-impact-edge-composition-v1-result
```

### Kafka shadow JAR reproducibility

The [reproducibility evidence](./results/poc-kafka-shadowjar-reproducibility-v1.json)
attributes the failed fallback without measuring performance. Two independent
clean builds using Kafka's original archive settings produced different
10,204,023-byte JAR SHA-256 values (`5539a273...2d96` and
`ce25a3f3...b0a1`). Their logical payload and entry-order fingerprints were
identical; only ZIP metadata differed.

Two more clean builds changed exactly `reproducibleFileOrder` to `true` and
`preserveFileTimestamps` to `false`. Both produced
`3ffd994e...3349` with the original logical payload. A fifth build received
HTTP 503 from the remote cache, Gradle disabled that cache and rebuilt locally,
and the resulting JAR still had the same normalized digest. The terminal
decision is `QUALIFY_KAFKA_SHADOWJAR_REPRODUCIBILITY_INPUT`: a newly
preregistered composition may use this source configuration, but no timing or
savings claim follows from the reproducibility block itself.

```bash
./dev/check-poc-kafka-shadowjar-reproducibility-v1-result
```

### Qualified Kafka Build Impact and Edge composition

The fresh [v2 composition evidence](./results/poc-kafka-impact-edge-composition-v2.json)
uses the qualified archive settings before dependency preparation and derives
seed, control, and candidate from the same normalized source. It reuses no v1
timing observation.

| Pair | Order | Native + Shared | Impact + Edge | Saving |
|---:|---|---:|---:|---:|
| 1 | Native first | 54,404 ms | 8,245 ms | 46,159 ms |
| 2 | Candidate first | 41,402 ms | 6,263 ms | 35,139 ms |
| 3 | Native first | 38,488 ms | 9,638 ms | 28,850 ms |
| 4 | Candidate first | 37,677 ms | 6,203 ms | 31,474 ms |

Native averages 42,992.75 ms and the installed candidate 7,587.25 ms, saving
**35,405.5 ms or 82.35%**. All four pairs are positive and the paired interval
is +30,162..+42,487.75 ms. Every measured arm restores the same normalized
10,204,023-byte `shadowJar`; candidate Edge reads produce zero Shared traffic.
A global change selects the full graph. When Edge returns HTTP 503, Gradle
disables the remote cache, succeeds locally, and reproduces the exact
`3ffd994e...3349` output. The terminal decision is
`QUALIFY_KAFKA_IMPACT_EDGE_COMPOSITION` for this POC workload and modeled
network profile only.

```bash
./dev/check-poc-kafka-impact-edge-composition-v2-result
```

### Repository-owned Kafka composition profile

The checked [usability evidence](./results/poc-kafka-composition-usability-v1.json)
binds the qualified 82.35% Kafka result to the v2 `buildopt poc` surface. One
repository-owned profile records the exact normalized `build.gradle` digest;
one explicit `--edge-url` supplies a read-only loopback Edge origin. The plan,
selected endpoint, ambient-endpoint masking, real Gradle cache hit, global and
precondition fallbacks, missing/invalid endpoint fallbacks, and byte-exact HTTP
503 fallback all pass.

No new timing was collected. The performance numbers are referenced from the
unchanged v2 composition evidence and are not recomputed or combined with
other component percentages. Validate the usability boundary with:

```bash
./dev/check-poc-kafka-composition-usability
```

### Installed Kafka profile value

The [installed-profile evidence](./results/poc-kafka-installed-profile-value-v1.json)
measures the exact repository-owned v2 profile through the packaged
`buildopt poc` command. It does not reuse any timing from the earlier
experiment-only composition. Eight fresh alternating pairs compare optimized
native Gradle plus Shared Cache with installed Build Impact plus the read-only
Edge profile under the same fixed Kafka source, normalized archive input and
337-ms/6,994,831-B/s network model.

| Pair | Order | Native + Shared | Installed profile | Saving |
|---:|---|---:|---:|---:|
| 1 | Native first | 31,622 ms | 12,164 ms | 19,458 ms |
| 2 | Candidate first | 38,745 ms | 7,093 ms | 31,652 ms |
| 3 | Native first | 35,004 ms | 6,519 ms | 28,485 ms |
| 4 | Candidate first | 36,669 ms | 4,811 ms | 31,858 ms |
| 5 | Native first | 33,707 ms | 5,760 ms | 27,947 ms |
| 6 | Candidate first | 34,261 ms | 5,692 ms | 28,569 ms |
| 7 | Native first | 32,812 ms | 5,604 ms | 27,208 ms |
| 8 | Candidate first | 31,958 ms | 5,912 ms | 26,046 ms |

Native averages 34,347.25 ms and the installed profile 6,694.375 ms, saving
**27,652.875 ms or 80.51%**. All eight pairs are positive and the corrected
deterministic bootstrap interval is +24,826.5..+29,903.625 ms. The archived
evidence proves that statistical correction changed no observation. Every arm
reproduces the exact normalized `shadowJar`; the candidate makes zero origin
requests, global drift selects native full `assemble`, and HTTP 503 completes
locally with identical bytes. The terminal decision is
`QUALIFY_INSTALLED_KAFKA_PROFILE_VALUE`, bounded to this Kafka revision,
change, output and modeled network profile.

```bash
./dev/check-poc-kafka-installed-profile-value-v1-result
```

### Installed qualified-profile matrix

The [terminal matrix summary](./results/poc-qualified-profile-matrix-v1/summary.json)
remeasures the complete installed path independently on the fixed Spring,
OpenTelemetry and Kafka revisions. It does not average repository percentages
or add mechanism effects.

| Cell | Control mean | Candidate mean | Result | Decision |
|---|---:|---:|---:|---|
| Spring Build Impact | 13,226.375 ms | 11,331.375 ms | **1,895 ms / 14.33% faster**, 7/8 positive, interval +981..+3,111.75 ms | Retain native: one pair regressed by 57 ms, so the unchanged all-positive gate failed. |
| OpenTelemetry Impact + standard `Jar` | — | — | Zero accepted observations | Retain native: impact discovery was terminated by signal after the successful unmeasured preflight; no performance claim is made. |
| Kafka Impact + read-only Edge | 34,848.25 ms | 6,325 ms | **28,523.25 ms / 81.85% faster**, 8/8 positive, interval +26,603.5..+30,509 ms | Qualify the fixed Kafka profile with exact output and both safety fallbacks. |

Only one of three independent families qualifies, below the two-family broad
continuation rule. The terminal decision is `SPECIALIZE_QUALIFIED_PROFILES`:
keep Spring and OpenTelemetry on optimized native Gradle, and continue only
with deterministic, reviewable discovery of the bounded Kafka profile.

```bash
./dev/check-poc-qualified-profile-matrix-v1-result \
  benchmarks/results/poc-qualified-profile-matrix-v1/summary.json
```

### Deterministic qualified-profile discovery

The checked [discovery report](./results/poc-profile-discovery-v1.json) binds
the terminal matrix, Kafka cell, complete Build Impact graph, generated-state
digest, component trace digests, normalized source input, exact output, and
reviewed profile contract. It reproduces the committed v2 Kafka profile and
its 61-project omission plan without a repository-name allowlist.

Spring and OpenTelemetry are negative fixtures because their matrix cells did
not qualify; they emit native full-graph decisions rather than profiles. The
same fallback covers evidence drift, incomplete or unknown graph state,
selected Test tasks, generated-state drift, and precondition drift. This block
adds no timing and does not broaden Kafka's 81.85% result.

```bash
./dev/check-poc-profile-discovery
```

### Trace-gated hypothesis decision

The checked [trace decision](./results/poc-trace-hypothesis-v1.json) analyzes
only the immutable installed synthetic phase trace and Spring verification
trace. It does not collect another timing sample. Required outputs are exact
and product failures remain zero across the three evaluated workload or
repository families.

| Phase | Largest positive observed delta | Authorization result |
|---|---:|---|
| BuildOpt-specific setup | 1.238233 ms | Below 500 ms in every family |
| Launcher and Gradle-client startup | 364.875 ms | Below 500 ms and not causally recoverable |
| Configuration before tasks | 682 ms | Only Spring exceeds 500 ms; not product-attributed and not reproduced |
| Gradle finalization | 97 ms | Below 500 ms and not product-attributed |
| Launcher and Gradle-client teardown | 87 ms | Below 500 ms and not causally recoverable |

Existing Build Impact task-interval savings are deliberately excluded because
they are not a new hypothesis. The 4,249 ms of Spring control-only task time is
also excluded because parallel task durations overlap and are not an additive
critical path. The terminal decision is `NO_ACTIONABLE_HYPOTHESIS`.

```bash
./dev/check-poc-trace-hypothesis-v1
```

### Terminal POC portfolio decision

The checked [portfolio decision](./results/poc-portfolio-decision-v1.json)
applies the preregistered continue/specialize/stop policy to the installed
matrix, qualified Kafka cell, deterministic discovery result, and trace-gated
no-action decision. It collects no new timing.

Only one of three repository families qualifies, so the terminal result is
`SPECIALIZE_BOUNDED_KAFKA_PROFILE`. Kafka retains its exact reviewed profile at
28,523.25 ms / 81.85% mean savings with 8/8 positive pairs and complete
fallback. Spring and OpenTelemetry retain optimized native Gradle. Repository
percentages are not averaged, mechanism effects are not added, and the broad
accelerator claim is withdrawn.

```bash
./dev/check-poc-portfolio-decision-v1
```

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

## Historical v1 five-repository generic structural profile matrix

This section preserves the first terminal v1 attempt. The latest comparable
results are the v3/v4 table near the top of this document; v1 observations are
not reused or silently replaced.

[`results/poc-generic-profile-matrix-v1/summary.json`](./results/poc-generic-profile-matrix-v1/summary.json)
applies one installed, repository-independent `profile propose -> profile
measure -> profile evaluate` path to Spring Framework, OpenTelemetry Java
Instrumentation, Apache Kafka, Micronaut Core, and Apache Groovy. The fresh
candidate contains Build Impact only; older Jar and Edge compositions are
retained separately and are not attributed to structural reduction.

| Repository | Project reach | Optimized native | Structural candidate | Result | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| Spring Framework | 27 -> 10 | 12.338 s | 11.495 s | **6.83% faster**, 5/8 positive, interval crosses zero | Native |
| OpenTelemetry | 1,024 -> 34 | — | — | No accepted result; pair 6 gap was 5.444 s | Native |
| Kafka | 64 -> 3 | 87.873 s | 13.080 s | **85.12% faster**, 8/8 positive | Qualified |
| Micronaut Core | 75 -> 22 | 25.699 s | 14.848 s | **42.22% faster**, 8/8 positive | Qualified |
| Apache Groovy | 37 -> 2 | 69.446 s | 19.455 s | **71.99% faster**, 8/8 positive | Qualified |

Every completed row preserves byte-identical required outputs and proves the
native full-graph fallback. OpenTelemetry's first five pairs were positive,
but they are deliberately excluded because the sixth pair exceeded the frozen
five-second inter-arm gap. The matrix therefore reports three qualified cells,
one weak cell, and one unavailable cell rather than averaging repository
percentages or selecting favorable partial data.

Validate the committed bundle without network access:

```bash
./dev/check-generic-profile-matrix \
  benchmarks/results/poc-generic-profile-matrix-v1
```

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

## Historical Runtime owner evaluation evidence

[`results/b-runtime-owner-evaluation.json`](./results/b-runtime-owner-evaluation.json)
records the public four-CPU owner run for the retired Runtime Optimizer. The run
drives 200 durable pre-outcome A/A assignments with delayed exactly-once
rewards, then measures four real alternating Gradle pairs for A/A and the
finite `W4_H6G` candidate. It passes sample-ratio, p95/p99, queue, OOM,
additional-compute, and byte-identical artifact guardrails.

```bash
./dev/check-runtime-owner-evaluation
```

This is historical evidence only. Stricter later comparisons superseded its
early positive signal, and the Runtime Tuning implementation and workflow were
removed after the terminal no-value decision.


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

## Ktor new-family transfer

[`results/poc-new-family-transfer-v1/`](./results/poc-new-family-transfer-v1/README.md)
is the terminal transfer test for the unchanged generic structural path on
Ktor's public JVM JAR workflow. Both independent captures derived
`:ktor-http:jvmJar` from the reviewed workflow/output/change inputs and reduced
133 projects to three without repository-name product logic.

Across 16 raw pairs, optimized native Gradle averaged 103.724 seconds and
installed BuildOpt averaged 14.308 seconds, saving **89.416 seconds/86.21%**.
All eight reciprocal blocks improved, the deterministic block interval is
+79.451..+98.422 seconds, p95 improves from 153.818 to 20.397 seconds, exact
JAR bytes and task shapes match, and both full-graph fallbacks pass. Five
rejected attempts are preserved separately and contribute no accepted timing.

Revalidate the captures, aggregate and negative tamper fixtures with:

```bash
./dev/check-new-family-transfer
./dev/test-new-family-transfer-result
```

## Ktor new-family change breadth

[`results/poc-new-family-change-breadth-v1/`](./results/poc-new-family-change-breadth-v1/README.md)
tests three materially different changes under the same public Ktor `jvmJar`
workflow and one untimed root-configuration fallback. Dependency source saves
**84.314 seconds/85.80%**, a JVM service resource saves **49.517
seconds/86.51%**, and a two-module mixed-source edit saves **81.781
seconds/77.98%** against optimized native Gradle.

All 48 raw pairs and 24 reciprocal blocks improve, required JARs are
byte-identical, tails improve, task shapes remain stable and all six selective
fallbacks pass. Both root-configuration proposals retain the native full
graph without a timing claim. Percentages remain cell-specific and are not
averaged.

The first complete diagnostic run is retained under `incidents/` but is not
terminal evidence: the generic runner omitted one preregistered Ktor Gradle
option. BuildOpt was corrected to propagate the frozen option list exactly,
and all eight accepted captures restarted from zero on revision `35065d3`.

Recompute the terminal matrix and its negative tamper fixtures with:

```bash
./dev/check-new-family-change-breadth-result
./dev/test-new-family-change-breadth-result
```

## Ktor calibration economics

[`results/poc-new-family-calibration-economics-v1/`](./results/poc-new-family-calibration-economics-v1/README.md)
binds two fresh setup captures to each qualified Ktor change cell without
rerunning or rewriting its terminal value evidence. First-time discovery plus
candidate stabilization repays after **7 builds** for dependency source,
**10 builds** for a JVM resource and **8 builds** for the mixed-source change.

Exact digest-bound proposal replay takes 0.321–0.376 seconds. Including fresh
candidate stabilization, repeat evaluation repays after **2, 4 and 3 builds**
respectively. All replay artifacts are byte-identical, every option-drift
probe misses, target fingerprints converge, required output digests remain
exact and the global-configuration fallback is not timed. Cell economics are
not averaged.

Recompute the result and its negative tamper fixtures with:

```bash
./dev/check-new-family-calibration-economics-result
./dev/test-new-family-calibration-economics-result
```

## Ktor public installed profile replay

[`results/poc-new-family-installed-profile-replay-v1/`](./results/poc-new-family-installed-profile-replay-v1/README.md)
installs immutable public `v0.3.2` and replays all three terminal Ktor profiles
through `buildopt poc` in clean external checkouts. The dependency, resource
and mixed-source cells select their exact reviewed entrypoints. Adding only
`--stacktrace` to the complete qualified option list produces three
`FULL_GRAPH / PROFILE_GRADLE_OPTIONS_DRIFT` plans before Gradle execution.

All three candidate/native fallback output sets match by exact bytes, all
three historical output digests also match as a diagnostic, every terminal
qualification is unchanged and product-attributable failures are zero. This
bundle creates no new timing claim or averaged percentage.

Revalidate the result and its negative tamper fixtures with:

```bash
./dev/check-new-family-installed-profile-replay-result
./dev/test-new-family-installed-profile-replay-result
```

## Compatible producer-portfolio value

[`results/poc-compatible-portfolio-value-v1/`](./results/poc-compatible-portfolio-value-v1/README.md)
tests a fresh Micronaut producer-volatility portfolio on a nearby source-code
revision whose repository, workflow, Wrapper and runtime output-contract
bindings are exact. Learning compares 11,187 outputs, quarantines 868 from
nine volatile producers and leaves 10,319 transportable. The evaluation
reduces 70 projects to 22 and captures 190 unaffected outputs/172.5 MB in
2.537 seconds, but eight historical intermediate producers have no proven
transitive lineage to the final materialized outputs.

BuildOpt therefore returns
`NATIVE_RETAINED / PORTFOLIO_PRODUCER_LINEAGE_UNAVAILABLE`, records the exact
missing producer list and starts zero timing pairs. This is terminal safety and
architecture evidence, not a performance percentage. Revalidate it with:

```bash
./dev/check-compatible-portfolio-value
```

## Transitive producer lineage

[`results/poc-transitive-producer-lineage-v1/`](./results/poc-transitive-producer-lineage-v1/README.md)
repeats the same frozen Micronaut window after recording the complete Gradle
task lineage for every materialized output. The diagnostic run proves why
transport and execution must change together: 89 outputs were safely
quarantined, but rebuilding only 11 direct/changed entrypoints caused
`REQUIRED_OUTPUT_DRIFT`; full-graph recovery preserved the exact digest.

The corrected generic rebuild frontier uses 58 entrypoints across 52 of 70
projects, transports 101 outputs and rebuilds 89. It completes eight pairs
with one exact digest, a successful fallback and zero product failures. Mean
wall time changes from 13,318.25 to 13,253.25 ms, a **65-ms/0.49%** saving,
but only 5/8 pairs improve, the interval is -1,114.375..+1,166.75 ms and p95
regresses from 14,267 to 16,967 ms. Correctness is proven; value is not.

```bash
./dev/check-transitive-producer-lineage
```

## Minimal quarantine rebuild frontier

[`results/poc-minimal-quarantine-rebuild-frontier-v1/`](./results/poc-minimal-quarantine-rebuild-frontier-v1/README.md)
tests the narrower hypothesis opened by the lineage result. An experimental
revision replaces project lifecycle covers with the minimal direct-producer DAG
frontier while keeping the same frozen Micronaut revisions, producer portfolio,
output gate, fallback and eight alternating pairs.

The direct frontier is exact but not faster. It uses 63 entrypoints instead of
58, still selects 52/70 projects and completes with 101 transported outputs,
89 locally rebuilt outputs, one required-output digest, successful fallback
and zero product failures. Optimized native averages 12,656.125 ms and BuildOpt
13,365.5 ms: **709.375 ms/5.60% slower**, with 3/8 positive pairs, a
-1,877.625..+500.25-ms interval and p95 regression from 13,589 to 15,053 ms.
Within the capture, Gradle contributes 680.625 ms of the regression and wrapper
work 28.75 ms.

The terminal decision is `RETAIN_GRAPH_PROVEN_PROJECT_FRONTIER`. The
experimental implementation is reverted because it reduces neither the
project graph nor wall time. Revalidate the negative evidence with:

```bash
./dev/check-minimal-quarantine-rebuild-frontier
```

## Quarantine critical-path attribution

[`results/poc-quarantine-critical-path-attribution-v1/`](./results/poc-quarantine-critical-path-attribution-v1/README.md)
answers the final Micronaut question with task-level Gradle build operations
and the resolved hard-dependency DAG. Diagnostic traces are not used as causal
timings; the unchanged eight alternating wall-time pairs remain authoritative.

The graph-proven candidate executes 110 fewer tasks and reduces cumulative
task duration by 4,731 ms. None of those 110 tasks appears on a control
critical path. Candidate main-build task span instead grows by 248.375 ms and
the longest hard-dependency chain grows by 178.875 ms. The causal result saves
only 197 ms/1.51%, with 5/8 positive pairs and an interval of
-703.5..+951.125 ms, so it remains unqualified.

The terminal decision is `STOP_MICRONAUT_QUARANTINE_LINE`: removed work is
parallel/off-critical, while the remaining chain is required by the exact
owner-visible output boundary. Validate the checked attribution with:

```bash
./dev/check-quarantine-critical-path-attribution
```

## Native-retention fast path

[`results/poc-native-retention-fast-path-v1/`](./results/poc-native-retention-fast-path-v1/README.md)
repeats the eight native-retained cross-commit breadth V2 revisions with direct
BuildOpt pre/post execution timing. Seven fallbacks decide before Gradle,
attach no output observer and total 9,020 ms of wrapper work; the maximum is
2,878 ms. The one ownership-ambiguous Groovy change still declares its
post-discovery observer. All eight execute the authoritative optimized-native
command, preserve exact required outputs and report zero product failures.

The original six/two hypothesis is rejected because Groovy `be211c1b` changes
five nested `build.gradle` files and the generic build-logic classifier safely
resolves it before Gradle. That stronger observed coverage is recorded as a
mismatch rather than rewriting the preregistration. The selected Ktor control
also remains exact and saves 126.225 seconds/56.47%. Validate the terminal
evidence with:

```bash
./dev/check-native-retention-fast-path
```

## Source ownership compatibility

[`results/poc-source-ownership-compatibility-v1/`](./results/poc-source-ownership-compatibility-v1/README.md)
investigates the only post-discovery ownership ambiguity in the frozen breadth
V2 fallback set. A fresh public Apache Groovy `classes` observation is complete
at 37 projects and 166 tasks, but `versions.properties` has no source owner or
task consumer and is read during configuration. BuildOpt now reports
`CONFIGURATION_INPUT_OWNERSHIP_UNPROVEN` and retains native Gradle instead of
collapsing this distinct case into generic source ambiguity.

The independent native replay completed in 29.710 seconds and matched all 3,890
required outputs byte for byte. The full lifetime attempt stopped earlier when
fresh target calibration saved a descriptive 1.200 seconds/4.05% with 6/8
positive pairs but a -0.262..+2.625-second interval. The value gate therefore
published no profile and the descendant window was not replayed. This negative
result is retained rather than rerun for a favourable sample. Validate it with:

```bash
./dev/check-source-ownership-compatibility
```

## Configuration-input binding

[`results/poc-configuration-input-binding-v1/`](./results/poc-configuration-input-binding-v1/README.md)
tests the remaining configuration-input question independently of Apache
Groovy. Gradle 8.14.3 and 9.6.1, with Groovy and Kotlin DSL, store and reuse
Configuration Cache state, invalidate it after the root input changes and
preserve the expected affected/unaffected JAR behavior in 4/4 cases.

The supported report exposes the input and build-logic read origin, not a
complete semantic project owner. The terminal result is full-workflow native
retention, with no performance replay after the safety precondition fails:

```bash
./dev/check-configuration-input-binding
./dev/check-configuration-input-binding --fixture
```

## Aggregate output closure

[`results/poc-aggregate-output-closure-v1/`](./results/poc-aggregate-output-closure-v1/README.md)
proves the generic correctness prerequisite that follows configuration-input
retention. A custom lifecycle task has no direct output and its two producers
write arbitrary extension-neutral files outside BuildOpt's conventional output
paths. Gradle 8.14.3 and 9.6.1 with both DSLs complete 4/4 cases: the changed
producer is rebuilt, the stable output is materialized without executing its
producer, and the candidate reproduces the baseline two-file digest exactly.

This evidence closes output coverage only. It is deliberately revision-bound
and makes no wall-time claim:

```bash
./dev/check-aggregate-output-closure
```

## Structural profile rebinding

[`results/poc-structural-profile-rebinding-v1/`](./results/poc-structural-profile-rebinding-v1/README.md)
separates structural compatibility from Git revision identity. Two contexts
with different commits and checkout roots produce one canonical fingerprint;
Wrapper, workflow, producer-lineage, output-contract and change-family drift
are rejected in 5/5 cases, and four incomplete/ambiguous lineage or output
cases fail closed.

The real central integration proof also preserves evidence ancestry and keeps
exact output bytes revision-bound, refreshing them through native Gradle when
necessary. This is compatibility evidence, not a new performance result:

```bash
./dev/check-structural-profile-rebinding
```

## Ordinary-build learning economics

[`results/poc-ordinary-learning-economics-v1/`](./results/poc-ordinary-learning-economics-v1/README.md)
proves that the POC spends learning effort only when ordinary requested builds
can plausibly repay it. The fixed horizon is five structurally compatible
matches, and the existing robust qualification still needs eight alternating
pairs.

The deterministic decision evidence stops an insufficient-lifetime hypothesis
after one requested build, avoiding 16 remaining observations, and stops a
six-match payback after three requested builds, avoiding 14. A positive first
pair projects three matches to payback but authorizes only continued learning.
This is economic-control evidence, not a repository speedup:

```bash
./dev/check-ordinary-learning-economics
```

## JVM Agent spike evidence

[`results/spk-002-agent.json`](./results/spk-002-agent.json) records the one
warm, order-sensitive JDK 21 sample emitted while closing `SPK-002`. It is
descriptive only: the prototype is `UNAVAILABLE` for access tracing because it
observes class loads rather than method calls. The result never activates an
overhead or promotion gate.

## Adaptive fragment lookup overhead

[`adaptive-fragment-lookup-v1-local.json`](./results/adaptive-fragment-lookup-v1-local.json)
records the bounded `AF-003` decision-time experiment. The same six generic
lookup scenarios run on the frozen Spring Framework, OpenTelemetry Java
Instrumentation, Apache Kafka, Micronaut Core and Apache Groovy subjects. The
repository identifiers select evidence rows only; they do not change product
behavior.

The 12-CPU Linux run completes 30 decisions with a **0.025-ms median**,
**0.039-ms p95** and **0.061-ms maximum**. Ten decisions return compatible
candidates, five declare binding-drift suspension and fifteen retain native
Gradle. The measured decision loop starts Gradle zero times, makes zero remote
calls, materializes zero outputs and mutates no lifecycle state.

This proves cheap lookup, not build acceleration or activation correctness.
Effects are not added to any build-time percentage. Validate the frozen report,
run a live report and exercise the tamper rejection with:

```bash
./dev/check-adaptive-fragment-index
```

## Adaptive fragment frozen-history coverage

[`adaptive-fragment-shadow-v1.json`](./results/adaptive-fragment-shadow-v1.json)
records the `AF-004` shadow decomposition over the existing five-repository
lifetime histories. It runs no new build and makes no timing claim. Only Kafka
had a qualified profile to decompose; the other four subjects had already
retained native execution before calibration.

The replay reproduces the one complete selection among six eligible Kafka
descendants. All six retain the structural `SUBGRAPH` candidate, while four
suspend `OUTPUT_MATERIALIZATION` after refreshed bytes and one leaves it
unevaluated after the economic gate retains native execution. That is **6/6
(100%)** descendant fragment retention, **5** partial-compatibility decisions
and zero future observations, activation authorizations or measurement-only
builds.

This supports the fragment coverage hypothesis and permits the next active
implementation blocks. It does not show that a partial fragment saves wall
time. Recompute the report and exercise provenance and tamper rejection with:

```bash
./dev/check-adaptive-fragment-shadow
```

## Adaptive fragment economic ledger

[`adaptive-fragment-economics-v1.json`](./results/adaptive-fragment-economics-v1.json)
is the recomputable `AF-005` ledger over retained evidence plus deliberately
synthetic edge vectors. Kafka remains a `COMPOSITION`: the frozen evidence can
recompute `135,127 - 42,040 - 10,560 = +82,527 ms` and observed payback at the
second requested descendant, but cannot attribute the gross saving between the
subgraph and materialization fragments.

The synthetic signed vector ends at `+40 ms`; a `-100 ms` activated build plus
its `20 ms` synchronous cost reduces cumulative value by exactly `120 ms`, and
two references to the same `500 ms` asynchronous event charge it once. A
separate negative vector reports `-350 ms` against a `100 ms` regret budget
without clipping. Fixed-horizon projections are derived policy outputs, not
observed timings, activation authority or percentages that can be added to
other effects.

```bash
./dev/check-adaptive-fragment-economics
```

## Adaptive fragment ordinary-build learning

[`adaptive-fragment-online-v1.json`](./results/adaptive-fragment-online-v1.json)
is the deterministic `AF-006` state-machine proof. Five requested ordinary
builds contribute 15 exact comparable fragment samples; no measurement-only
build is created. All three fragments remain `OBSERVED` after one build, move
to `SHADOW` after two and become `QUALIFIED` after four.

The fifth synthetic signed observation takes one base fragment to `-250 ms`.
Only that family and its declared dependent suspend; an unrelated fragment
remains `QUALIFIED` at `+200 ms`. An interrupted generation resumes only with
its exact canonical digest, repository scope and context bindings. These values
exercise lifecycle behavior and make no wall-time or activation claim.

```bash
./dev/check-adaptive-fragment-online
```
