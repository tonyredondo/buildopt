# BuildOpt Generalization Audit

## Audit question

Can the retained BuildOpt POC run unchanged on an arbitrary Gradle repository,
find a safe optimization and beat optimized native Gradle without
repository-name rules?

The current answer is: **not with the complete structural-profile unit tested
so far**. Target qualification produced substantial savings on several public
repositories, but the later chronological lifetime evaluation found repeatable
net value in only one of five families. One of six structurally eligible
descendants selected a profile, so the frozen terminal decision is
[`STOP_GENERIC_POC`](../../benchmarks/results/poc-functional-coverage-decision-v1/README.md).

That decision stopped the whole-profile hypothesis, not the bounded mechanisms
that worked. The subsequent
[adaptive fragment model](../plans/adaptive-fragment-generalization-tracker.md)
also stopped after its current installed campaign produced zero activations,
zero attributable mechanism saving and negative cumulative value.

The active preregistered direction is the
[Sticky Wrapper Learning POC](../plans/sticky-wrapper-learning-poc-tracker.md).
It changes the integration and economic unit rather than reviving stopped
profiles: one committed wrapper observes ordinary builds, uses a negligible
locally cached decision, spends only a bounded CI trial budget and may retain
either an exact runtime action or a reviewed durable Gradle change. It must
prove breadth, payback and positive cumulative value beyond optimized native
Gradle; safe fallback and cache hits alone do not pass.

## What is generalized today

| Layer | Generic behavior | Current boundary |
| --- | --- | --- |
| Installation / launcher | Native packages locate the Wrapper and preserve argv, exit and signal behavior. | Linux, macOS and Windows lifecycle is tested; comparable wall-time breadth evidence is Linux. |
| Change and workflow discovery | Derives provider/local revisions, exact changed paths and requested Gradle entrypoints. | Global/build-logic ambiguity retains native. |
| Output discovery | Reads Gradle-owned outputs and rejects missing, external, symlinked or ambiguous declarations. | A root aggregate workflow can legitimately declare a very broad output surface. |
| Structural proposal | Uses typed project/task relationships and changed-project ownership; no repository-name branch is allowed. | Unknown relationships, excessive candidate task sets and no reduction retain native. |
| Durable native catalog | Detects repeated task-contract gaps and over-broad declared graph edges, then emits digest-bound, reviewable and exactly reversible native Gradle recipes. | The current strict POC report finds the same task-contract detector in Kotlin and Groovy, with 64.1% and 74.7% savings across 16/16 exact pairs. Graph breadth is proposal-only until durable timing is measured. |
| Measurement / decision | Alternating native/candidate observations verify outputs, execution shape, interval, fallback and payback. | Observations now accumulate across useful invocations with zero measurement-only workflows; weak value still retains native. |
| Verified output materialization | Captures required outputs omitted by a candidate in digest-bound private state, then restores only exact missing bytes before candidate execution. | Composed and timed on all five public subjects; stale, missing or corrupt payloads cannot authorize candidate output. |
| Aggregate workflow partition | Groups directly changed output producers by generic lifecycle selector and variant, while exact unaffected outputs remain materializable. | Transfers to public workflows: Kafka selects 3/64 projects, Micronaut 22/75 and Groovy 2/37. |
| Portfolio / central state | Reuses exact compatible evidence across checkouts or machines. | Reuse cannot infer lifetime or value from another profile/repository. |
| Gradle-compatible cache | Supports local and optional HTTP/HTTPS reuse with safe miss/outage behavior. | Cache is supporting infrastructure; native-cache parity is not a speed advantage. |

Runtime Tuning, Hot State and standard Copy remain retired. The standard `Jar`
adapter and Patch Autopilot retain only their exact qualified scopes.

### Current durable-catalog evidence

The new [durable native catalog report](../../benchmarks/results/sticky-wrapper-durable-catalog-v1.json)
was generated from a fresh `linux-amd64-4c-16g-v1` campaign at BuildOpt
revision `1d93570c02147eda8671253663d50605bff9f25a`. A single structural
detector accepts a missing native task contract in both DSL families:

| Family | Control mean | Reviewed-patch mean | Saving | Positive pairs | Required outputs |
|---|---:|---:|---:|---:|---|
| Kotlin DSL | 2.438 s | 0.875 s | **64.1%** | 8/8 | Exact |
| Groovy DSL | 3.574 s | 0.903 s | **74.7%** | 8/8 | Exact |

The proposed source recipe is applied and reverted outside the checkout, and
plain native Gradle remains the runtime after acceptance. A second detector
proposes removing one unrelated dependency edge from a 3-project workflow in
both DSLs; its candidate preserves the observed output digest, but no durable
wall-time measurement has been made. The report therefore demonstrates
cross-DSL detector breadth and a strong reviewed-task signal, not universal
customer value or automatic patching.

## Historical target-qualification evidence

| Repository | Automatic graph | Direct timed effect | Economics / decision |
| --- | ---: | ---: | --- |
| Spring Framework | 27 -> 10 | 12.71% faster, 7/8, positive interval | 88.668-s learning; 67-build payback; native retained. |
| OpenTelemetry Java Instrumentation | 1,024 -> 34 | 14.97% faster, 8/8, positive interval | 201.913-s learning; 19-build payback; qualified. |
| Apache Kafka | 64 -> 3 | 54.92% faster, 8/8, positive interval | 70.808-s learning; 14-build payback; qualified. |
| Micronaut Core | 75 -> 22 | 66.24% faster, 8/8, positive interval | 114.284-s learning; 8-build payback; qualified. |
| Apache Groovy | 37 -> 2 | 75.97% faster, 8/8, positive interval | 73.857-s learning; 2-build payback; qualified. |

All 85 ordinary invocations preserve exact required outputs and pass full-graph
fallback. This remains useful mechanism evidence because the target
repositories contain no BuildOpt files and the same binary makes all five
decisions. It is not the current lifetime conclusion: target qualification does
not prove that a profile will recur often enough across later commits.

## Historical adaptive longitudinal evidence

| Repository | Comparable descendants | Selected | Signed longitudinal net | Row outcome |
|---|---:|---:|---:|---|
| Spring Framework | 2 | 1 | **+59.550 s** | `NET_POSITIVE` |
| OpenTelemetry Java Instrumentation | 3 | 0 | **-168.751 s** | `NET_NEGATIVE` |
| Apache Kafka | 6 | 1 | **+88.219 s** | `NET_POSITIVE` |
| Micronaut Core | 0 / 1 attempted | 0 | no attributable delta | `INCONCLUSIVE` |
| Apache Groovy | 3 | 0 | **-37.684 s** | `NET_NEGATIVE` |

AF-013 recomputes these rows from prior direct source-checkout measurements; it
is not a favorable rerun, a fresh timing claim or the scorecard for the current
implementation. The 14 comparable observations all preserve exact required
outputs and report zero product failures. Every signed delta and the one-time
qualification/publication cost remain visible. Micronaut is not treated as
zero: its descendant JAR differed byte for byte, so the safety gate retained
native and the row remains inconclusive.

Only 2/5 rows are net positive versus the frozen terminal breadth requirement
of 3/5. OpenTelemetry's worst individual regression is -104.572 seconds, while
Spring and Kafka each contain one negative build despite positive cumulative
value. OpenTelemetry and Groovy contain only three later observations, while
Micronaut has no comparable descendant pair; later implementation work also
changed ownership and aggregate-output rejection paths. The terminal decision
is therefore deferred until AF-014B..D evaluate the current installed binary on
larger frozen cohorts. AF-014A has already proved the isolated installed
measurement apparatus without creating a value claim. Repository percentages
are not averaged and mechanism effects are not added.

## Why graph reduction alone is insufficient

Three distinct constraints remain visible in the current data:

1. **Learning economics.** Spring saves 1.329 seconds per build but spends
   88.668 seconds learning, so a real reduction can still be a bad customer
   transaction.
2. **Complete outputs.** Kafka and Groovy aggregate workflows require outputs
   from many otherwise unaffected projects in a clean workspace. Omitting
   those producers without materializing verified outputs would make the build
   faster but wrong.
3. **Aggregate task breadth.** The former Micronaut 73-entrypoint rejection is
   now handled through generic partitioning, but incomplete producer/output
   relationships must still retain native on unknown repositories.

The implementation addresses these shapes through task producers, lifecycle
selectors, variants and exact output relationships. It does not branch on
repository identity or borrow an old profile's expected percentage.

## Implementation-generic versus evidence-specific

Repository names and frozen mutations are valid in runners and immutable
evidence because the experiment must identify what was tested. Product code
reasons only about:

- typed projects, tasks, dependencies, variants and entrypoints;
- exact change ownership and global-change rules;
- required outputs and their producer/digest relationships;
- repository, revision, Wrapper, graph, workflow, output and executable
  bindings; and
- measured wall time, uncertainty, failures, fallback and payback.

No latest terminal decision depends on a repository-name rule. A new Gradle
repository can run `buildopt optimize <workflow>` and receives either a
measured candidate or an explicit native verdict.

## Next generalization steps

The detailed order, outcomes and documentation obligations live in the
[Adaptive Fragment Generalization POC Tracker](../plans/adaptive-fragment-generalization-tracker.md).
Its first proof must happen before another broad timing campaign:

1. define independently compatible fragments and immutable lifecycle state
   (**complete**);
2. make compatibility lookup negligible and prove fragment applicability over
   frozen histories without lookahead (**complete: 6/6 eligible Kafka
   descendants retain a structural fragment; only 1/6 retained the whole
   profile**);
3. learn value and decay from ordinary builds using a signed economic ledger
   (**complete: immutable checkpoints accept requested builds only, resume
   under exact bindings and isolate regression to dependent fragments**);
4. activate and directly measure only compositions whose individual fragments
   retain correctness and positive value authority; and
5. install one current-SHA package and prove isolated, stateful, phase-attributed
   control/candidate execution (**complete: 18 alternating learning
   observations plus exact selected, forward native-retained and bypass
   scenarios; apparatus evidence only**);
6. freeze 20 primary first-parent commits plus an ordered reserve queue per
   family before timing (**complete: 100 primary commits and 50 reserves bound
   to exact topology, paths, workflows and outputs with no timing fields**);
7. collect at least 15 comparable builds per terminal row with chronological
   learning and exact outputs (**complete: 20/20 per repository, 100 exact
   pairs, zero product failures, 25 positive/75 negative and -368.623 seconds
   cumulative signed value**);
8. attribute activated mechanisms, fallback cost and residual variation
   (**complete: 179.029 seconds recorded BuildOpt cost, -189.593 seconds
   residual, zero activated-mechanism saving, 0.531-second p50 and 8.656-second
   p95 native retention**); and
9. recompute the frozen terminal decision from current evidence only
   (**complete: 9/15 criteria pass; activation 0/71, breadth 0/5, positive
   confidence 0/5, portfolio -368.623 seconds, payback 0 and native-retention
   tails fail; outcome `STOP_ADAPTIVE_FRAGMENT_POC`**).

## POC conclusion

BuildOpt's defensible idea remains an evidence-gated optimizer on top of native
Gradle, not a faster reimplementation of Gradle's cache. The current complete-
profile implementation demonstrates bounded target value and strong safety,
but not generic lifetime customer value. The shadow result supports the
coverage hypothesis but does not yet establish
wall-time value: the subgraph candidate remains structurally compatible in all
six eligible Kafka descendants, including five where whole-profile reuse is
invalid. AF-005 makes the retained Kafka composition economically auditable:
`135.127 - 42.040 - 10.560 = +82.527 seconds`, with observed payback at the
second requested descendant. It deliberately does not assign that saving to an
individual fragment. The current adaptive-fragment campaign did not turn that
bounded result into generic customer value: no fragment activated in 71
eligible builds, 0/5 families were positive, cumulative value was -368.623
seconds and native-retention tails missed their limits. AF-015 therefore stops
the generic successor rather than reinterpreting the evidence. Any future POC
must preregister a materially different mechanism and cannot inherit authority
from these inactive fragments.

AF-006 proves the online lifecycle mechanically but adds no real timing result:
five synthetic requested-build updates move three fragments through
`OBSERVED`, `SHADOW` and `QUALIFIED`; a later negative value suspends one family
and its dependent while an unrelated family stays qualified. AF-007 now proves
that task-implementation, plugin-version, Gradle and structural features can
rank local exploration without repository-name behavior: replacing four source
and two holdout identities leaves the three-class ranking unchanged, while
changing transferred positive/non-positive evidence changes only exploration
priority. Every candidate still requires local correctness and value and
authorizes no activation. These are synthetic ranking vectors, not a target
timing or breadth result. AF-008 now connects one generic repeated-task
detector to an exact reviewed Patch Autopilot recipe: ten unsafe inputs reject,
the replacement applies/reverts outside the checkout, and the frozen native
Gradle evidence saves 67.28% in Kotlin and 68.01% in Groovy across 16 exact-
output pairs. The improvement survives without BuildOpt at execution time.
This proves durable value for one recipe, not generic patch coverage. AF-009
then proved exact conflict-aware planning without timing. AF-010 now proves the
missing execution boundary on real Gradle: an unrelated change restores two
independent producers, each localized change rebuilds only its affected
producer, and global, ambiguous or incomplete state runs the complete original
workflow. Six of six final bundles and producer-output sets match byte for byte
with zero product failures. This establishes fragment-specific invalidation and
activation correctness. AF-011 subsequently timed the full composition
directly: both composed DSL arms were strongly positive and exact, but Kotlin
Build Impact reached only 6/8 positive isolated pairs. The frozen constituent
gate therefore retained the independently qualified fragments instead of
authorizing the joint path.

AF-013 then closed five historical evidence rows without hiding adverse
observations: Spring and Kafka are net positive, OpenTelemetry and Groovy net
negative, and Micronaut inconclusive after an output-reproducibility rejection.
This reaches 2/5 positive families, not the preregistered 3/5. The
source-checkout matrix is therefore historical audit evidence, not permission
to claim current generic value. The current claim now requires a larger,
preregistered installed-package campaign rather than exact repetition of
outdated decisions.
