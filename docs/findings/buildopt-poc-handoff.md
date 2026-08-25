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
| **Cross-repository hypothesis prior** | Use generic task implementation, plugin, Gradle and graph/output features to decide what to investigate first in a new repository. | Four opaque source scopes rank three hypothesis classes identically for two holdouts and after all repository identities are replaced. Priors never transfer correctness, value or activation. |
| **Durable patch-opportunity learning** | Detect repeated expensive task-contract problems, propose an owner-reviewed reversible source patch and validate it independently. | One generic detector feeds an exact reviewed recipe that saves **67.28% Kotlin** and **68.01% Groovy** across 16 native Gradle pairs with exact outputs. BuildOpt is not needed after acceptance; recipe coverage remains specific. |
| **Conflict-aware fragment planner** | Compose only qualified fragments whose dependencies, exclusions, authorities and direct joint economics remain valid; otherwise use native Gradle. | Direct timing now shows that the reviewed patch and Build Impact can save 68.56% Groovy and 79.32% Kotlin together, but Kotlin Build Impact reaches only 6/8 positive isolated pairs. The frozen constituent gate therefore retains qualified fragments instead of authorizing the composition. |
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

## Current fragment-composition result

AF-011 ran 48 new Gradle 9.6.1 pairs on one controlled workflow, always against
the complete optimized native workflow and with identical required outputs.

| Direct comparison | Groovy | Kotlin | Decision |
| --- | ---: | ---: | --- |
| Build Impact | **46.45% / 2.180 s**, 7/8 | **43.97% / 1.973 s**, 6/8 | Groovy qualifies; Kotlin misses repeatability |
| Reviewed task patch | **37.95% / 1.749 s**, 7/8 | **35.08% / 1.705 s**, 8/8 | Both qualify |
| Direct composition | **68.56% / 2.947 s**, 8/8 | **79.32% / 3.025 s**, 8/8 | Faster in both, but not authorized |
| HTTP cache locality, independent scope | **35.07% / 2.482 s**, 4/4 | Same controlled cache contract | Qualifies independently |

Every lower confidence bound is positive, outputs are byte-identical and
product failures are zero. The composed percentages are measured directly and
are not sums. The outcome is `RETAIN_BEST_SINGLE_FRAGMENT`: a positive
composition cannot hide the failed 7/8 constituent gate. Cache locality is not
claimed as part of that composition because its remote-cache object contract
is different.

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
- Frozen shadow decomposition explains part of the coverage loss: all **6/6**
  eligible Kafka descendants retain a compatible structural fragment even
  though only **1/6** retains the complete profile. Five cases are partial, so
  output freshness or economics can invalidate materialization without erasing
  the structural opportunity. This is compatibility evidence, not a timing
  result.

Applying the complete frozen gate yields **`STOP_GENERIC_POC`**. Five criteria
pass: matrix completeness, exact outputs/zero failures, generic selection,
robust Kafka qualification and bounded Kafka payback. Three fail:

- 1/5 net-positive repository families versus the required 3/5;
- 1/6 selected eligible descendants (16.67%) versus the required 50%; and
- one observed pre-Gradle economic rejection at 4,098 ms, above the 500-ms
  median and 1,000-ms p95 limits.

## Conclusion and next step

The current generic structural-profile hypothesis stops here. This is not a
claim that BuildOpt never works: Kafka pays back strongly, Spring and other
bounded experiments remain valid mechanism evidence, and exact-output plus
fail-open controls work. It is a rejection of the broad claim that the current
one-command implementation already delivers repeatable net wall-time value to
ordinary Gradle repositories.

No generic whole-profile implementation block followed automatically from that
failed gate. The explicitly preregistered
[Adaptive Fragment Generalization POC](../plans/adaptive-fragment-generalization-tracker.md):
replace a complete structural profile with independently compatible producer,
subgraph, task, patch and cache-locality fragments; learn their signed economics
from ordinary builds; compose only fragments with current correctness and value
authority; and retain native Gradle through a sub-second no-value path. The new
tracker preserves the same five-repository baseline, exact-output and zero-
failure requirements, and makes cumulative longitudinal value—not isolated
target speedup—the terminal decision. AF-001..AF-009 now provide typed fragment
identity/state, sub-millisecond lookup and a no-lookahead shadow result with
100% fragment retention across the six eligible descendants, plus immutable
signed economics. The retained Kafka composition recomputes to `+82.527 s`
after `135.127 s` gross saving, `42.040 s` synchronous wrapper cost and
`10.560 s` one-time qualification/publication cost; payback occurs on the
second requested descendant. Existing evidence cannot attribute that gross
saving to one fragment. AF-006 now provides immutable ordinary-build
checkpoints: exact restart succeeds, insufficient evidence remains
`OBSERVED`/`SHADOW`, and a synthetic value regression suspends only the affected
fragment and its dependent. AF-007 adds a name-independent prior: replacing all
source/holdout identities leaves the structural ranking unchanged, and source
outcomes can reorder exploration while fresh target correctness and value stay
mandatory. The vectors make no timing claim. AF-008 now proves that a generic
repeated-task detector can lead to one durable reviewed patch: the exact recipe
saves 1.369 seconds/67.28% in Kotlin and 2.349 seconds/68.01% in Groovy across
16 native Gradle pairs, with exact outputs and no BuildOpt runtime dependency.
That does not make the recipe generic or automatic. AF-009 now composes only
compatible fragments whose predicted net value remains positive, without
adding isolated effects: it closes
dependencies, rejects one-way or two-way conflicts, preserves every exact
constituent authority and falls back to native Gradle when the direct joint
prediction is absent, ambiguous or below `100 ms`. The proof is synthetic and
does not claim runtime saving. AF-010 now executes that boundary on real Gradle:
six control/candidate scenarios restore four exact unaffected outputs, rebuild
two changed producers locally and retain the complete original workflow for
global, ambiguous or missing state. Every producer and final bundle is exact
and product failures are zero. AF-011 now measures each mechanism and the
complete compatible composition directly. The composed arms are strongly
positive, but Kotlin Build Impact misses the fixed repeatability gate, so the
planner retains independently qualified fragments instead of activating the
joint path. AF-012 now persists the same portfolio and economic ledger locally
and over the existing HTTPS state plane. Exact bytes restore on a clean second
machine, CAS conflicts remain visible, corrupt state rejects, a verified local
generation survives outage and a clean offline machine retains native Gradle.
Adaptive documents generate zero Gradle cache-plane requests. AF-013 preserves
the frozen historical measurements without rerunning favorable arms:
Spring closes at `+59.550 s`, Kafka at `+88.219 s`, OpenTelemetry at
`-168.751 s` and Groovy at `-37.684 s`. Micronaut remains `INCONCLUSIVE`
because its first descendant failed byte reproducibility before an attributable
timing pair could be accepted. All 14 comparable builds have exact outputs and
zero product failures, but only 2/5 rows are positive versus the frozen 3/5
breadth target. This is an audit of earlier behavior, not the current scorecard:
OpenTelemetry and Groovy have only three later commits, Micronaut has no valid
descendant pair and several rejection paths changed afterward. AF-014A now
proves one current-SHA installed package, separate arm state, chronological
learning, exact selected/native-retained/bypass behavior and reconciled phase
timing. Its controlled fixture is apparatus evidence, not a speedup. AF-014B
has now frozen 20 primary commits plus ten ordered reserves per family before
timing. The five chains contain 100 primary observations, exact topology,
generic change shapes and immutable workflow/output/JDK scope. AF-014C/D will
collect at least 15 comparable builds per terminal row and attribute activated
mechanisms, fallback cost and residual variation.
AF-015 will decide from that current campaign only.
Production hardening, soak, design partners and Test Optimization remain
outside this POC.

## Evidence

- [Lifetime breadth V3 result](../../benchmarks/results/poc-lifetime-breadth-v3/README.md)
- [Terminal functional-coverage decision](../../benchmarks/results/poc-functional-coverage-decision-v1/README.md)
- [Machine-readable V3 summary](../../benchmarks/results/poc-lifetime-breadth-v3/summary.json)
- [V3 protocol](../../specs/poc-lifetime-breadth-v3.md)
- [Detailed historical findings](./build-optimization-performance.md)
- [Ordinary-build learning economics](../../benchmarks/results/poc-ordinary-learning-economics-v1/README.md)
- [Structural profile rebinding](../../benchmarks/results/poc-structural-profile-rebinding-v1/README.md)
- [Adaptive fragment shadow replay](../../specs/poc-adaptive-fragment-shadow-v1.md)
- [Machine-readable shadow result](../../benchmarks/results/adaptive-fragment-shadow-v1.json)
- [Adaptive fragment economics](../../specs/poc-adaptive-fragment-economics-v1.md)
- [Active Build Impact fragment evidence](../../benchmarks/results/adaptive-fragment-activation-v1.json)
- [Machine-readable economic ledger](../../benchmarks/results/adaptive-fragment-economics-v1.json)
- [Ordinary-build learner contract](../../specs/poc-adaptive-fragment-online-v1.md)
- [Machine-readable learner proof](../../benchmarks/results/adaptive-fragment-online-v1.json)
- [Cross-repository prior contract](../../specs/poc-adaptive-fragment-prior-v1.md)
- [Machine-readable prior proof](../../benchmarks/results/adaptive-fragment-prior-v1.json)
- [Patch-opportunity contract](../../specs/poc-adaptive-fragment-patch-opportunity-v1.md)
- [Machine-readable durable patch proof](../../benchmarks/results/adaptive-fragment-patch-opportunity-v1.json)
- [Conflict-aware planner contract](../../specs/poc-adaptive-fragment-planner-v1.md)
- [Machine-readable planner proof](../../benchmarks/results/adaptive-fragment-planner-v1.json)
- [Direct fragment-composition result](../../benchmarks/results/adaptive-fragment-composition-v1.json)
- [Fresh HTTP cache-locality result](../../benchmarks/results/adaptive-cache-locality-v1.json)
- [Adaptive state portability contract](../../specs/poc-adaptive-state-portability-v1.md)
- [Adaptive longitudinal matrix](../../benchmarks/results/adaptive-fragment-longitudinal-v1.json)
- [Adaptive longitudinal protocol](../../specs/poc-adaptive-fragment-longitudinal-v1.md)
