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
| **Structural Build Impact** | Selects the smallest proven change-specific project/task graph before Gradle runs. | **Retained accelerator.** It materially improves all five fresh public-repository subjects and all three additional build-owned workflow subjects below. |
| **Profile discovery and evaluation** | Derives the candidate from generic Gradle metadata, measures it against the owner workflow, binds inputs, and enforces native fallback. | **Required safety/evidence layer.** Review remains explicit. |
| **Safe Cache / local L1** | Isolates and verifies cached outputs by repository, Wrapper, and platform. | **Supporting safety, not the current speed claim.** It is approximately at parity with a warm native cache. |
| **Shared / Edge Cache** | Moves verified cache objects closer to developers or CI. | **Separate locality experiment.** Network-dependent results are not added to Structural Build Impact percentages. |
| **Build History and launcher** | Records evidence, preserves process behavior, and applies bypass/fallback. | **Supporting infrastructure.** Its candidate-path overhead is included in measurements. |
| **Runtime Tuning, Hot State, standard Copy** | Previously explored worker/heap changes and broader task reuse. | **Retired from the executable POC:** terminal evidence was neutral, unstable, or regressive. |

## Fresh comparable public-repository evidence

The preregistered v2 protocol ran two independent captures per repository,
eight alternating pairs per capture, and eight opposite-order AB/BA blocks.
The control is optimized native Gradle with the same revision, workflow,
resources, required outputs, cache/parallel options, and warmed state. BuildOpt
timing includes its installed launcher, profile validation, planning, and
Gradle execution.

| Repository | Full -> selected projects | Native mean | BuildOpt mean | Mean saving | Blocks | p95 native -> BuildOpt |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Spring Framework | 27 -> 10 | 13.311 s | 11.183 s | **2.128 s / 15.99%** | 8/8 positive | 15.711 -> 13.318 s |
| OpenTelemetry Java Instrumentation | 1,024 -> 34 | 87.869 s | 74.713 s | **13.156 s / 14.97%** | 8/8 positive | 100.229 -> 79.004 s |
| Apache Kafka | 64 -> 3 | 113.381 s | 14.341 s | **99.040 s / 87.35%** | 8/8 positive | 145.478 -> 15.339 s |
| Micronaut Core | 75 -> 22 | 30.411 s | 18.418 s | **11.993 s / 39.44%** | 8/8 positive | 34.006 -> 20.448 s |
| Apache Groovy | 37 -> 2 | 79.868 s | 20.767 s | **59.101 s / 74.00%** | 8/8 positive | 85.338 -> 24.183 s |

All **5/5 repositories qualify independently**. Across the matrix, all 80 raw
pairs improved, every required output matched exactly, measured task shapes
were stable, both full-graph fallbacks passed for every repository, and no
product-attributable failure occurred. Percentages are neither averaged across
repositories nor added to cache or other mechanism results.

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

## What this demonstrates

- **The core idea transfers.** One generic implementation derived material
  graph reductions across five very different, substantial Gradle codebases
  without repository-name logic.
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
- **Discovery has a cost.** Distribution preparation, graph discovery, and
  warm-up can be expensive—especially for OpenTelemetry and Micronaut—and are
  outside stable-state pair timing. The POC proves repeated-build value, not
  yet instant first-run payback.

## Current limits

The evidence is bound to exact revisions, changes, workflows, outputs, and a
12-CPU host. It does not prove that every change in these repositories wins or
that every Gradle repository can be activated automatically. Profiles remain
review-required and native Gradle remains authoritative on drift or ambiguity.

The evidence now covers those three known output representations, but only
through explicit reviewed contracts: Groovy JAR metadata embeds build time,
Kafka Checkstyle embeds isolated workspace paths, and Kafka `shadowJar`
preserves timestamp/order differences. A new output shape remains byte-exact
until an owner declares and validates its semantics.

## Recommended next steps

1. **Add change-shape breadth.** Preregister multiple source, shared-library,
   build-logic, and global changes across the existing public repositories and
   workflow families. Qualify each cell independently so one exact profile is
   not mistaken for repository-wide support.
2. **Measure calibration economics.** Record discovery, warm-up, and steady
   state separately, then report the break-even number of repeated builds.
   Reuse only digest-bound facts and never move setup cost into a hidden bucket.
3. **Replay qualified profiles through the public installed path.** Confirm
   the same value, semantic-output contract, drift handling, and native
   fallback outside the research runner.
4. **Add one new substantial repository family.** Use the unchanged generic
   path and owner-input model; do not add a repository-name rule to force a
   result.
5. **Keep wall time authoritative.** Continue only mechanisms that materially
   beat optimized native Gradle under correctness, repeatability, and tail
   guards; retire or retain native for everything else.

## Evidence

- [Balanced five-repository qualification](../../benchmarks/results/poc-statistical-qualification-v2/README.md)
- [Machine-readable terminal matrix](../../benchmarks/results/poc-statistical-qualification-v2/summary.json)
- [Preregistered v2 contract](../../specs/poc-statistical-qualification-v2.md)
- [Public workflow-family output barriers](../../benchmarks/results/poc-generic-workflow-value-v1/README.md)
- [Terminal semantic output-equivalence qualification](../../benchmarks/results/poc-generic-output-equivalence-v1/README.md)
- [Generalization audit](./buildopt-generalization-audit.md)
- [Detailed historical performance findings](./build-optimization-performance.md)
- [Implementation tracker](../../implementation-tracker.md)
