# BuildOpt POC: Current Evidence and Direction

## Idea

BuildOpt explores whether a generic decision layer can make substantial Gradle
builds faster than an already optimized native Gradle baseline. Gradle remains
the execution engine and source of truth. BuildOpt's retained accelerator acts
before Gradle: it derives a smaller graph for an exact Git change and requested
workflow, measures the complete installed path, verifies required outputs, and
replays the candidate only when safety and value gates pass. Any ambiguity,
drift, regression, or poor payback keeps optimized native Gradle authoritative.

This is an owner-operated proof of concept, not a production-ready product.
Production SLOs, soak testing, design-partner dependencies, autonomous rollout,
and Test Optimization are intentionally outside the current scope.

## What a user does today

```text
install BuildOpt
cd <any Gradle repository>
buildopt optimize <existing Gradle workflow>
```

The command requires no BuildOpt manifest, graph, profile, plugin path, or
repository-specific rule. It discovers the Git comparison, Gradle project
ownership, candidate tasks, required outputs, and structural graph; then runs
eight alternating optimized-native/BuildOpt pairs in isolated checkouts. A
qualified result becomes a private digest-bound profile for exact replay.

## Components and differentiation from Gradle

| Component | What it does | Current role |
| --- | --- | --- |
| **Structural Build Impact** | Avoids configuring and executing unrelated project/task producers for an exact change and output contract. | Primary accelerator. Gradle optimizes the graph it is asked to run; BuildOpt attempts to prove that a smaller graph is sufficient. |
| **Automatic discovery and calibration** | Derives a candidate from Git/Gradle facts, measures it against optimized native Gradle, verifies outputs and computes payback. | Required evidence layer. It prevents task-count reductions from being mistaken for wall-time value. |
| **Profile portfolio and replay** | Stores only qualified structural families and selects them after exact repository, Wrapper, workflow, graph, output and evidence bindings pass. | POC-only automatic replay; any drift falls back before Gradle starts. |
| **Safe local cache** | Separates and verifies cached outputs by repository, Wrapper and platform. | Supporting safety. It is approximately at parity with a warm native Gradle cache and is not the current speed claim. |
| **Shared / Edge cache** | Makes verified Gradle cache objects available over HTTP/HTTPS and optionally closer to builders. | Separate locality experiment; network results are never added to Structural Build Impact percentages. |
| **Build history, launcher and reports** | Preserve process behavior and explain graph reduction, measured saving, uncertainty, p95, calibration cost and fallback. | Supporting infrastructure; its overhead is included in candidate timing. |
| **Runtime Tuning, Hot State and standard Copy** | Earlier attempts to tune resources or reuse broader task state. | Retired from the executable POC after neutral, unstable or regressive evidence. |

## Current zero-configuration evidence

The latest automatic matrix uses installed development packages, public
repositories, zero manual BuildOpt files, eight pairs when calibration runs,
and a declared maximum payback of 30 matching builds.

| Repository / workflow | Projects | Native mean | BuildOpt mean | Direct effect | Current decision |
| --- | ---: | ---: | ---: | ---: | --- |
| Ktor `jvmJar` | 133 -> 10 | 33.595 s | 6.049 s | **27.546 s / 82.00% faster** | **Qualified; 27-build payback** |
| Spring `classes` | 27 -> 21 | 10.135 s | 9.072 s | **1.063 s / 10.49% faster** | Native retained; 328-build payback |
| Beam `classes` | 316 -> 6 | 20.396 s | 4.901 s | **15.495 s / 75.97% faster** | Native retained; 37-build payback |
| Groovy `classes` | 37 -> 30 | 61.497 s | 62.038 s | **0.542 s / 0.88% slower** | Native retained; value not proven |
| Kafka `testClasses` | 66 -> 36 | 8.921 s | 11.671 s | **2.750 s / 30.83% slower** | Native retained; value not proven |
| Micronaut `assemble` | unavailable | unavailable | unavailable | no timing claim | Native retained; output semantics ambiguous |

Ktor, Spring and Beam improved in 8/8 pairs with positive intervals, lower p95,
exact required outputs and successful full-graph fallback. Beam is the clearest
new structural signal: the candidate reduces 316 projects to six and replay
wall time by 75.97%. Its complete first-decision cost is 558.913 seconds,
however, so repayment needs 37 matching builds rather than the allowed 30.

Groovy explains why automatic graph precision matters: the current project
dependency graph keeps a large strongly connected component, so automatic
selection reaches 30 of 37 projects and provides no value. Kafka shows that a
smaller graph alone is insufficient: it halves the project graph but executes
slightly more expensive work and is 30.83% slower. Micronaut correctly makes no
timing claim because a root `assemble` request fans out broadly and the generic
output contract cannot yet prove a bounded terminal deliverable set.

The first Beam attempt also exposed a general calibration bug: isolated clones
replaced the public `origin` URL with a local path, while Beam derives metadata
from that URL. BuildOpt now preserves the public origin without fetching from
it; the corrected run passed all eight pairs and fallback.

## What the evidence proves

- The core structural idea can deliver large customer-path savings on real,
  substantial Gradle repositories.
- The automatic path can discover a 133 -> 10 Ktor candidate and qualify it
  without a hand-authored BuildOpt file.
- The same generic implementation finds a 316 -> 6 Beam candidate with strong,
  repeatable replay value.
- Safety gates work: negative, uneconomic and semantically ambiguous candidates
  retain native Gradle rather than forcing an optimization.
- Project count is diagnostic, not value. Only measured installed wall time,
  outputs, uncertainty, tails, fallback and payback can qualify a profile.

It does **not** yet prove the desired general product experience. The terminal
gate requires two economically qualified repository families from a published
package and fresh install-to-decision captures. Current automatic count is
**1/2**. OpenTelemetry is not included in this automatic matrix because its raw
temporary result was not retained; historical reviewed-profile evidence remains
versioned separately and is not substituted for missing current data.

## Historical feasibility versus current automation

Earlier owner-reviewed profiles produced substantial improvements across
Spring, OpenTelemetry, Kafka, Micronaut, Groovy and Ktor, including 14.97% to
87.35% lower wall time in their terminal protocols. Those experiments prove
that structural graph reduction has broad potential when workflow scope and
output semantics are supplied and reviewed. They do not prove that today's
zero-configuration command discovers the same scopes automatically. The table
above is therefore the current customer-shaped status and takes precedence for
onboarding decisions.

## Recommended next steps

1. **Reduce generic first-decision cost.** Reuse immutable dependency snapshots
   under exact content bindings, avoid repeated multi-gigabyte copying, and
   reuse base-cache preparation only when repository/Wrapper/dependency inputs
   prove equivalence. Beam needs at least 94.1 seconds removed to repay within
   30 matching builds at the observed saving.
2. **Add an economic prequalification signal.** Use observed executed,
   from-cache, up-to-date and no-source task shapes to avoid expensive full
   calibration when graph reduction is unlikely to remove costly work, as in
   Kafka.
3. **Improve graph precision generically.** Investigate task/variant and
   ABI-aware dependency relationships so Groovy SCCs and Micronaut root
   fan-out can be reduced without repository-specific rules.
4. **Repeat the terminal gate only after two families qualify.** Publish one
   immutable release, install it into fresh Ktor and Beam checkouts/homes, run
   the same command, preserve every raw result and require 2/2 economics plus
   at least one native-retained negative.
5. **Observe profile lifetime across commits.** Payback is a projection until
   exact graph/output applicability is measured over real follow-up changes.

## Evidence

- [Current automatic one-command matrix](../../benchmarks/results/poc-magic-end-to-end-value-v1/README.md)
- [Machine-readable summary](../../benchmarks/results/poc-magic-end-to-end-value-v1/summary.json)
- [End-to-end value contract](../../specs/poc-magic-end-to-end-value-v1.md)
- [One-command onboarding roadmap](../plans/one-command-onboarding-roadmap.md)
- [Detailed historical performance findings](./build-optimization-performance.md)
- [Generalization audit](./buildopt-generalization-audit.md)
- [Implementation tracker](../../implementation-tracker.md)
