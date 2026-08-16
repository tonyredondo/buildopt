# BuildOpt POC: Current Evidence and Direction

## Product idea

BuildOpt explores whether a generic decision layer can make substantial Gradle
builds faster than optimized native Gradle. Gradle already caches outputs,
reuses configuration and incremental state, and executes a requested task graph
efficiently. BuildOpt's current accelerator acts before that work: for an exact
Git change, owner-approved Gradle workflow, and declared required outputs, it
derives a smaller project/task graph and falls back to the original optimized
native graph whenever ownership, outputs, evidence, or inputs are uncertain.

This is an owner-operated proof of concept. There is no automatic production
activation, soak requirement, design-partner dependency, or Test Optimization
scope.

## Current mechanisms

| Mechanism | Role | Current evidence decision |
| --- | --- | --- |
| **Structural Build Impact** | Selects the smallest proven change-specific project/task graph before Gradle runs. | **Retained accelerator.** It materially improves the five-repository matrix, a fresh Ktor family transfer, three additional build-owned workflows, six earlier source-change cells, and three materially different Ktor change classes below. |
| **Profile discovery and evaluation** | Derives the candidate from generic Gradle metadata, measures it against the owner workflow, binds inputs, and enforces native fallback. | **Required safety/evidence layer.** Review remains explicit. |
| **Safe Cache / local L1** | Isolates and verifies cached outputs by repository, Wrapper, and platform. | **Supporting safety, not the current speed claim.** It is approximately at parity with a warm native cache. |
| **Shared / Edge Cache** | Moves verified cache objects closer to developers or CI. | **Separate locality experiment.** Network-dependent results are not added to Structural Build Impact percentages. |
| **Build History and launcher** | Records evidence, preserves process behavior, and applies bypass/fallback. | **Supporting infrastructure.** Its candidate-path overhead is included in measurements. |
| **Runtime Tuning, Hot State, standard Copy** | Previously explored worker/heap changes and broader task reuse. | **Retired from the executable POC:** terminal evidence was neutral, unstable, or regressive. |

## Fresh comparable public-repository evidence

The latest preregistered balanced protocols ran two independent captures per
repository, eight alternating pairs per capture, and eight opposite-order
AB/BA blocks.
The control is optimized native Gradle with the same revision, workflow,
resources, required outputs, cache/parallel options, and warmed state. BuildOpt
timing includes its installed launcher, profile validation, planning, and
Gradle execution.

| Repository | Declared -> selected projects | Native mean | BuildOpt mean | Mean saving | Blocks | p95 native -> BuildOpt |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Spring Framework | 27 -> 10 | 13.311 s | 11.183 s | **2.128 s / 15.99%** | 8/8 positive | 15.711 -> 13.318 s |
| OpenTelemetry Java Instrumentation | 1,024 -> 34 | 87.869 s | 74.713 s | **13.156 s / 14.97%** | 8/8 positive | 100.229 -> 79.004 s |
| Apache Kafka | 64 -> 3 | 113.381 s | 14.341 s | **99.040 s / 87.35%** | 8/8 positive | 145.478 -> 15.339 s |
| Micronaut Core | 75 -> 22 | 30.411 s | 18.418 s | **11.993 s / 39.44%** | 8/8 positive | 34.006 -> 20.448 s |
| Apache Groovy | 37 -> 2 | 79.868 s | 20.767 s | **59.101 s / 74.00%** | 8/8 positive | 85.338 -> 24.183 s |
| Ktor JVM JAR workflow | 133 -> 3 | 103.724 s | 14.308 s | **89.416 s / 86.21%** | 8/8 positive | 153.818 -> 20.397 s |

All **6/6 repositories qualify independently**. Across the matrix and Ktor
transfer, all 96 raw
pairs improved, every required output matched exactly, measured task shapes
were stable, both full-graph fallbacks passed for every repository, and no
product-attributable failure occurred. Percentages are neither averaged across
repositories nor added to cache or other mechanism results.

Ktor is intentionally a JVM JAR selector rather than a claim about its full
multiplatform release build. Its two independent captures saved 92.128 and
86.705 seconds; the balanced block interval is +79.451..+98.422 seconds. Five
invalid or rejected attempts are retained, and neither their timings nor their
state enters the terminal result.

The same generic Ktor path now also qualifies three independently
preregistered change classes under that selector:

| Ktor change | Full -> selected projects | Native mean | BuildOpt mean | Mean saving | Blocks |
| --- | ---: | ---: | ---: | ---: | ---: |
| Upstream dependency source | 133 -> 3 | 98.269 s | 13.954 s | **84.314 s / 85.80%** | 8/8 positive |
| JVM service resource | 133 -> 9 | 57.238 s | 7.720 s | **49.517 s / 86.51%** | 8/8 positive |
| Mixed production source, two modules | 133 -> 9 | 104.877 s | 23.096 s | **81.781 s / 77.98%** | 8/8 positive |

All **48/48 raw pairs** and **24/24 reciprocal blocks** improve. Required JARs
are byte-identical, candidate p95 is lower, task shapes are stable and all six
selective fallbacks pass. Two independent root-configuration proposals retain
native Gradle without a timing claim. These percentages remain independent;
they are not averaged with each other or with the original Ktor transfer.
The accepted matrix restarted from zero after a diagnostic run exposed a
missing preregistered Gradle option in the generic runner. That rejected run is
preserved for audit and contributes no timing to this table.

The result replaces no historical evidence. In particular, Spring's previous
7/8 decision remains valid for its old strict protocol; the fresh balanced run
uses 16 new pairs and qualifies with 8/8 positive reciprocal blocks, a positive
1.803..2.386 s block-bootstrap interval, and improved p95.

## Additional build-owned workflow evidence

Three workflows that previously failed closed on byte-level output
nondeterminism were rerun with explicit owner-reviewed semantic contracts.
Exact bytes remain the default; only the declared repository-root or archive
container metadata is excluded from comparison.

| Repository and workflow | Full -> selected projects | Native mean | BuildOpt mean | Mean saving | Blocks |
| --- | ---: | ---: | ---: | ---: | ---: |
| Apache Groovy `jar` | 37 -> 2 | 72.319 s | 19.455 s | **52.864 s / 73.10%** | 8/8 positive |
| Apache Kafka `checkstyleMain` | 64 -> 2 | 82.835 s | 58.209 s | **24.627 s / 29.73%** | 8/8 positive |
| Apache Kafka `shadowJar` | 64 -> 2 | 40.728 s | 13.625 s | **27.103 s / 66.55%** | 8/8 positive |

All **48/48 raw pairs** improved. Required outputs were semantically identical,
candidate p95 was lower, task shapes were stable, two native fallbacks passed
per workflow, and product-attributable failures were zero. Percentages remain
workflow-specific and are not combined with the five-repository table.

## Change-shape breadth

The latest preregistered matrix asks whether those reviewed workflow profiles
transfer beyond one exact source edit. Each source cell has two independent
eight-pair captures; build-logic and global changes are separate untimed safety
cells.

| Repository and workflow | Change | Full -> selected projects | Native mean | BuildOpt mean | Mean saving |
| --- | --- | ---: | ---: | ---: | ---: |
| Apache Groovy `jar` | leaf source | 37 -> 2 | 71.227 s | 18.843 s | **52.384 s / 73.54%** |
| Apache Groovy `jar` | shared source | 37 -> 2 | 79.557 s | 27.207 s | **52.349 s / 65.80%** |
| Kafka `checkstyleMain` | metadata source | 64 -> 2 | 82.431 s | 59.353 s | **23.078 s / 28.00%** |
| Kafka `checkstyleMain` | client-utils source | 64 -> 2 | 87.522 s | 61.177 s | **26.345 s / 30.10%** |
| Kafka `shadowJar` | clients source | 64 -> 2 | 44.828 s | 14.956 s | **29.872 s / 66.64%** |
| Kafka `shadowJar` | generator source | 64 -> 2 | 37.079 s | 7.587 s | **29.492 s / 79.54%** |

All **96/96 raw pairs** and **48/48 reciprocal blocks** improved. Every output,
shape, lower-bound, p95, selective fallback, and zero-product-failure gate
passed. Eight independent build-logic/global captures correctly returned
`NATIVE_FULL_GRAPH / GLOBAL_CHANGE_REQUIRES_FULL_GRAPH` and made no timing
claim. The percentages remain independent and are not averaged.

## Calibration economics and efficiency

Two fresh setup captures per selective cell now expose how long it takes to
reach those steady-state savings. The installed-workflow view includes the
real combined output-preflight/discovery command and all candidate warm-ups;
the full POC view additionally includes native-control warm-ups used only to
prove the comparison.

| Workflow and change | Saving/build | Installed break-even before -> now | Full POC break-even before -> now |
| --- | ---: | ---: | ---: |
| Groovy `jar`, leaf source | 52.384 s | **10 -> 9 builds** | **20 -> 19 builds** |
| Groovy `jar`, shared source | 52.349 s | **11 -> 10 builds** | **22 -> 21 builds** |
| Kafka Checkstyle, metadata source | 23.078 s | **31 -> 26 builds** | **55 -> 50 builds** |
| Kafka Checkstyle, client-utils source | 26.345 s | **27 -> 24 builds** | **49 -> 46 builds** |
| Kafka `shadowJar`, clients source | 29.872 s | **15 -> 13 builds** | **28 -> 26 builds** |
| Kafka `shadowJar`, generator source | 29.492 s | **14 -> 13 builds** | **27 -> 25 builds** |

The efficiency follow-up fuses structural discovery, replays only exact
digest-bound proposals, and removes the third candidate target warm-up only
after two exact task fingerprints match. Cold discovery is **8.01% to 21.08%
lower** across all six cells; exact replay takes **0.281 to 1.261 seconds**.
All six installed and full-POC break-even counts improve without changing any
terminal saving, output, tail, fallback or failure result. Checkout remains
visible but is excluded as shared native/BuildOpt work.

The same economics protocol now covers the three qualified Ktor change
classes. First-time cold discovery plus candidate stabilization repays after
**7 builds** for dependency source, **10 builds** for a JVM resource and
**8 builds** for the mixed-source edit. Exact proposal replay is
0.321–0.376 seconds; including fresh stabilization, reevaluation repays after
**2, 4 and 3 builds**. Every replay artifact and terminal output binding is
exact, all six drift probes reject replay, and the global fallback stays
untimed native Gradle.

## What this demonstrates

- **The core idea transfers.** One generic implementation derived material
  graph reductions across six very different, substantial Gradle codebases,
  including a new Kotlin Multiplatform family, without repository-name logic.
- **The value transfers across change shapes.** Two distinct changes per
  reviewed Groovy/Kafka workflow and dependency, resource, and multi-module
  Ktor changes qualify independently, while build-logic and global changes
  retain native Gradle instead of forcing a reduction.
- **Avoiding the graph compounds value.** Fewer selected projects remove
  configuration, scheduling, cache lookup, compilation, and packaging work;
  this is why Kafka and Groovy show especially large savings.
- **The gain survives product overhead.** Candidate timings include BuildOpt's
  installed path rather than an internal task microbenchmark.
- **Correctness and tails remain mandatory.** Mean speed alone cannot qualify
  a scope; outputs, measured shapes, p95, order balance, and native fallbacks
  must all pass.
- **Owner-reviewed output semantics recover real workflows.** Narrow,
  digest-bound text and archive contracts allow BuildOpt to compare correct
  reports and packages without treating timestamps, archive order, or an
  isolated checkout prefix as business payload. Undeclared drift still fails
  closed.
- **Calibration is cheaper but still not free.** The original six reviewed
  profiles repay fused discovery plus adaptive candidate warm-ups after 9–26
  qualifying builds; exact repeat evaluation repays after 4–12. The three
  Ktor cells repay first-time calibration after 7–10 builds and exact replay
  evaluation after 2–4. The POC proves repeated-build value, not instant
  first-run payback.
- **The reviewed profiles transfer to the public package.** Public `v0.3.1`
  reproduced all six selective decisions through `buildopt poc`; all six
  manifest-drift probes retained native Gradle and all six contemporary
  candidate/native outputs were semantically equal. The replay preserves the
  existing timing qualifications rather than creating new percentages.
- **The new Ktor family also transfers to the public package.** Public
  `v0.3.2` reproduced all three dependency, resource and mixed-source plans;
  adding one unqualified Gradle option retained native Gradle 3/3 before
  execution, and all contemporary candidate/native JARs matched exactly.
  Historical Ktor percentages remain unchanged.
- **Cross-date output comparison is now explicit and fail-closed.** Four Kafka
  cells match naturally across independent captures. Two real Groovy JAR
  probes match after declaring only `BuildDate + BuildTime`; the previous
  contract rejects the date change, and the reviewed contract still rejects
  undeclared version and class-payload drift. Historical profiles and timing
  values remain unchanged.

## Current limits

The first two one-command automation blocks are now executable rather than
only a roadmap. In the real two-project Gradle fixture, `buildopt optimize`
discovers packaging, custom verification, distribution and `testClasses`
proposals with zero hand-authored BuildOpt files; every case derives a 2→1
project graph and writes seven private review documents. For a measurable
candidate, the same command now runs eight balanced pairs, exact output checks,
full fallback and break-even evaluation, then resumes matching evidence without
remeasurement. An under-budget run makes no claim. Unsupported relationships,
a root build change and an ambiguous Git base independently retain optimized
native Gradle. This validates orchestration, not a new transferable performance
percentage: the fixture's intentional delay produced 1.299 s/66.43% mean
saving across 8/8 positive pairs and repaid its 77.922-s learning cost after 60
matching builds under the fixture's explicit 100-build ceiling. Those values
test the math and do not extend the real-repository evidence above.

The evidence is bound to exact revisions, changes, workflows, outputs, and a
12-CPU host. The matrices cover multiple source shapes, a JVM resource,
multiple simultaneous paths, and conservative build-logic/global fallback,
but they do not prove that every change in these repositories wins or that
every Gradle repository can be activated automatically. Profiles remain
review-required and native Gradle remains authoritative on drift or ambiguity.

The evidence now covers those three known output representations, but only
through explicit reviewed contracts: Groovy JAR metadata embeds build date and
time, Kafka Checkstyle embeds isolated workspace paths, and Kafka `shadowJar`
preserves timestamp/order differences. A new output shape remains byte-exact
until an owner declares and validates its semantics. The controlled Groovy
date probe proves the semantic boundary on real JARs; it is not a second timed
build on another day or authority to ignore arbitrary timestamps.

## Recommended next steps

The product-direction north star is now one owner-invoked command:

```text
install BuildOpt -> open a Gradle repository -> buildopt optimize build
```

That command must automatically discover, calibrate, evaluate, materialize and
replay repository-owned profiles. The magic is orchestration rather than
weaker evidence: unsupported or ambiguous workflows retain optimized native
Gradle, and automatic selection remains POC-only with production authority
false. The ordered implementation and scorecard are frozen in the
[one-command onboarding roadmap](../plans/one-command-onboarding-roadmap.md).

1. **Materialize a generic profile portfolio.** The command now derives and
   calibrates a candidate with zero manual BuildOpt files, eight balanced
   pairs, equivalent outputs, full fallback, positive value and bounded
   break-even. Next it must store multiple independently bound change-family
   profiles without repository-name rules.
2. **Automate portfolio selection and CI persistence.** Match qualified
   change families, retain native on drift and reuse only exact calibration
   state through one-input GitHub/GitLab onboarding.
3. **Prove the complete effect.** Measure install-to-decision time, net
   installed-path wall time and break-even on fresh substantial public
   repositories; never add isolated mechanism percentages.

## Evidence

- [Balanced five-repository qualification](../../benchmarks/results/poc-statistical-qualification-v2/README.md)
- [Machine-readable terminal matrix](../../benchmarks/results/poc-statistical-qualification-v2/summary.json)
- [Preregistered v2 contract](../../specs/poc-statistical-qualification-v2.md)
- [Public workflow-family output barriers](../../benchmarks/results/poc-generic-workflow-value-v1/README.md)
- [Terminal semantic output-equivalence qualification](../../benchmarks/results/poc-generic-output-equivalence-v1/README.md)
- [Terminal generic change-breadth qualification](../../benchmarks/results/poc-generic-change-breadth-v1/README.md)
- [Terminal calibration economics](../../benchmarks/results/poc-calibration-economics-v1/README.md)
- [Terminal calibration efficiency](../../benchmarks/results/poc-calibration-efficiency-v1/README.md)
- [Public installed profile replay](../../benchmarks/results/poc-installed-profile-replay-v1/README.md)
- [Reviewed cross-date output equivalence](../../benchmarks/results/poc-cross-date-output-equivalence-v1/README.md)
- [Ktor new-family transfer](../../benchmarks/results/poc-new-family-transfer-v1/README.md)
- [Ktor new-family change breadth](../../benchmarks/results/poc-new-family-change-breadth-v1/README.md)
- [Ktor calibration economics](../../benchmarks/results/poc-new-family-calibration-economics-v1/README.md)
- [Ktor public installed profile replay](../../benchmarks/results/poc-new-family-installed-profile-replay-v1/README.md)
- [One-command POC onboarding contract](../../specs/poc-magic-onboarding-contract-v1.md)
- [Automatic one-command discovery contract](../../specs/poc-magic-auto-discovery-v1.md)
- [Automatic one-command calibration contract](../../specs/poc-magic-calibration-v1.md)
- [Generalization audit](./buildopt-generalization-audit.md)
- [One-command POC onboarding roadmap](../plans/one-command-onboarding-roadmap.md)
- [Detailed historical performance findings](./build-optimization-performance.md)
- [Implementation tracker](../../implementation-tracker.md)
