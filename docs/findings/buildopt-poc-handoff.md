# BuildOpt POC: One-Page Handoff

## The idea

BuildOpt tests whether a generic, one-command layer can reduce wall time for
substantial Gradle workflows beyond an already optimized native Gradle
baseline. Gradle remains the execution engine and safe fallback. BuildOpt
learns the relationship between a Git change, the requested workflow, the task
graph and its outputs; when the evidence is strong enough, it rebuilds only the
necessary graph and restores verified unaffected outputs.

The intended experience has no repository-specific BuildOpt files:

```text
install BuildOpt
cd <a Gradle repository>
buildopt optimize <the existing Gradle workflow>
```

This is an owner-operated proof of concept. The current question is whether
the idea creates repeatable customer value across ordinary builds and commits,
not whether it is ready for production. Soak, design-partner validation,
production SLOs, autonomous rollout and Test Optimization are outside this
decision.

## Mechanisms

| Mechanism | Purpose | Current evidence |
| --- | --- | --- |
| **Structural Build Impact** | Derive the smallest sufficient Gradle graph for the exact change and requested workflow. | Produces large isolated target wins, including Kafka's current **21.43%** qualified target saving, but compatible descendants are much rarer than target calibrations. |
| **Verified output materialization** | Restore byte-exact outputs from unaffected producers so the reduced graph remains correct. | Fail-closed correctness is strong: the current five-repository run verifies 27 exact-output builds with zero product failures. |
| **Structural profile rebinding** | Reuse learned evidence only when Wrapper, workflow, producer lineage, output contract and change family remain compatible. | Safely rejects drift and selected one of six structurally eligible Kafka descendants in the current run. |
| **Ordinary-build learning economics** | Learn only from builds the user requested and stop when the expected compatible lifetime cannot repay discovery. | Four repositories stopped after one requested build and avoided 64 additional learning builds. |
| **Local/HTTP cache and central state** | Carry verified task outputs and profiles between builds or machines. | Supporting infrastructure; useful for transport and persistence, but not the primary acceleration claim. |
| **Runtime Tuning, Hot State and standard Copy** | Earlier broad resource and state hypotheses. | Retired after neutral, unstable or regressive end-to-end evidence. |

## Current public-repository result

One exact BuildOpt executable was used across frozen Spring Framework,
OpenTelemetry Java Instrumentation, Apache Kafka, Micronaut Core and Apache
Groovy windows. The run used requested ordinary builds only, kept the robust
eight-pair qualification gate unchanged and stopped learning early when the
five-match lifetime economics could not pay back.

| Repository | Requested builds | Target evidence | Later selection | Signed lifetime net |
| --- | ---: | --- | ---: | ---: |
| Spring Framework | 1 | Early native retention; only four compatible historical matches | 0 / 0 | **-10.113 s** |
| OpenTelemetry | 1 | Early native retention; one compatible match | 0 / 0 | **-9.961 s** |
| Apache Kafka | 17 | **+6.673 s / 21.43%, 8/8 positive** | **1 / 6 eligible** | **+82.527 s** |
| Micronaut Core | 1 | Early native retention; one compatible match | 0 / 0 | **-9.149 s** |
| Apache Groovy | 1 | Early native retention; one compatible match | 0 / 0 | **-2.760 s** |

Kafka's selected descendant improves from 178.566 seconds to 43.439 seconds,
saving **135.127 seconds / 75.67%** with 4,449 exact outputs. Five other
structurally eligible descendants retain native Gradle. After 42.040 seconds
of measured fallback wrapper work and 10.560 seconds of qualification plus
publication cost, Kafka finishes **82.527 seconds net positive**.

Across all five repositories the experiment uses 21 requested builds, zero
measurement-only builds, verifies 27 exact-output observations and records
zero product failures. The signed total is +50.544 seconds, but that number is
descriptive only: repository percentages are not averaged and mechanism
percentages are not added.

## What the evidence says

- The core idea can create very large value when a learned structural profile
  remains compatible: Kafka's selected replay saves 75.67%.
- Isolated acceleration is not enough. A profile must recur across real commits
  often enough to repay discovery and publication.
- The current generic breadth is insufficient. Only **1/5 repository
  families** is net positive and only **1/6 eligible descendants (16.67%)**
  selects a profile. The preregistered pass gate required at least 3/5 net
  positive families and at least 50% selection coverage.
- Safety and fail-open behavior work: every uncertain case uses optimized
  native Gradle, outputs remain exact and product failures are zero.
- Early economics prevent waste: the four short-lived hypotheses stop after
  one requested build instead of spending 16 additional builds each.

The terminal evidence decision is therefore
`FUNCTIONAL_COVERAGE_NOT_PROVEN`. This does not mean the mechanism never works;
it means the current generic one-command POC has not shown enough repeatable
cross-commit coverage to justify claiming broad customer value.

## Recommended next decision

The next block must issue the frozen terminal decision without moving the
thresholds after seeing the data:

1. Stop presenting target calibration wins as generic customer value.
2. Decide whether to stop the generic POC or explicitly narrow the hypothesis
   to repository/workflow families with demonstrated recurring structural
   compatibility.
3. If continuing, improve automatic compatibility and selection coverage as a
   generic Gradle mechanism; do not add repository-name rules or weaken exact
   output, failure or statistical gates.
4. Keep production hardening, soak and design-partner work deferred until the
   POC first proves repeatable net wall-time value.

## Evidence

- [Lifetime breadth V3 result](../../benchmarks/results/poc-lifetime-breadth-v3/README.md)
- [Machine-readable V3 summary](../../benchmarks/results/poc-lifetime-breadth-v3/summary.json)
- [V3 protocol](../../specs/poc-lifetime-breadth-v3.md)
- [Detailed historical findings](./build-optimization-performance.md)
- [Ordinary-build learning economics](../../benchmarks/results/poc-ordinary-learning-economics-v1/README.md)
- [Structural profile rebinding](../../benchmarks/results/poc-structural-profile-rebinding-v1/README.md)
