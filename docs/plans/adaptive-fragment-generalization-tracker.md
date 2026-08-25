# Adaptive Fragment Generalization POC Tracker

**Status:** `IN_PROGRESS`<br>
**Current block:** `AF-004 — Frozen-history shadow decomposition`<br>
**Decision baseline:** the current whole-profile hypothesis remains
[`STOP_GENERIC_POC`](../../benchmarks/results/poc-functional-coverage-decision-v1/README.md)<br>
**Last updated:** 2026-08-25

## 1. Purpose

This tracker governs the next BuildOpt proof-of-concept hypothesis:

> BuildOpt can create generic cumulative value across ordinary repository
> evolution by learning small, independently invalidated optimization fragments,
> composing only the fragments that remain correct and economically positive,
> and returning to optimized native Gradle with negligible overhead everywhere
> else.

The previous POC learned and replayed complete structural profiles. That work
proved substantial target-build savings, exact-output restoration and safe
fallback, but did not prove generic lifetime value. Only one of five repository
families was net positive, one of six eligible descendants selected a profile,
and the only measured pre-Gradle economic rejection cost 4,098 ms. This tracker
does not reinterpret or overwrite that result. It preregisters a different unit
of learning and activation.

This is a POC value-validation plan. It is not a production-readiness,
availability, multi-tenancy, soak, design-partner or autonomous-production
rollout plan.

## 2. Product hypothesis

Gradle remains the execution engine and authoritative fallback. BuildOpt does
not replace Gradle's local or remote Build Cache, Configuration Cache,
incremental execution, daemon, file-system watching or task scheduler.

BuildOpt adds an adaptive control plane that answers four questions for each
ordinary requested build:

1. Which previously validated optimization fragments remain compatible with
   this repository state, workflow and requested outputs?
2. Which compatible fragments are likely to reduce end-to-end wall time after
   all BuildOpt overhead is included?
3. Which non-conflicting fragments can be composed without weakening the
   correctness authority of any individual action?
4. What new candidate optimization is worth observing or proposing from this
   build, without requiring the developer to repeat it manually?

The unit of reuse changes from a complete repository/workflow/change profile to
an independently versioned fragment, such as:

- an unaffected project or producer subgraph;
- an exact output-materialization boundary;
- an officially cacheable task or reviewed task contract;
- a qualified standard-task adapter;
- a bounded remote-cache locality policy; or
- a reviewable build-logic patch candidate.

A changed or invalid fragment is suspended by itself. Compatible fragments
remain eligible. Cross-repository evidence may prioritize a hypothesis for the
same task implementation or plugin family, but it can never authorize
correctness or activation in a different repository.

## 3. Non-goals and retained decisions

The following boundaries apply to every block:

- no repository-name branches, hand-authored per-repository product rules or
  copied expected percentages;
- no weakening of exact-output, fallback, failure or causal-measurement gates;
- no claim that Safe Cache is faster than an already effective native Gradle
  cache;
- no reactivation of Runtime Tuning, Hot State, standard Copy or global tuning
  without a separately preregistered new mechanism hypothesis;
- no automatic caching of arbitrary tasks from observational evidence alone;
- no Test Optimization, test selection, sharding, retries or flake policy;
- no eight-hour soak, design-partner gate, HA, enterprise identity, SLO or
  production promotion work;
- no additional customer build solely to collect a natural-learning
  observation; isolated terminal experiments remain allowed only when their
  protocol is preregistered and their cost is reported separately; and
- no addition or averaging of percentages measured on different mechanisms,
  repositories or workload boundaries.

## 4. Terminal POC scorecard

The terminal decision will be recomputed from immutable evidence. A successful
result must pass every criterion below without post-result threshold movement.

| Criterion | Terminal requirement |
|---|---|
| Correctness | Every selected build preserves the preregistered semantic outputs; byte-exact outputs remain byte exact; zero product-attributable failures. |
| Generic implementation | No repository-name product rules and no manually authored BuildOpt profile required by the evaluated repositories. |
| Native substrate | Control and candidate use equivalent optimized Gradle daemon, Build Cache, Configuration Cache, VFS and dependency state unless the isolated mechanism explicitly changes one of them. |
| Fragment activation coverage | At least 50% of structurally eligible descendant builds activate at least one independently compatible fragment. |
| Repository-family breadth | At least three of five frozen public repository families finish the longitudinal window with positive cumulative net wall-time value. |
| Longitudinal confidence | At least three families have a positive paired or sequence-aware lower confidence bound at the preregistered horizon. |
| Time to value | Report cumulative value at builds 1, 5, 10 and 20; a qualifying family must repay within 20 compatible requested builds or the complete shorter frozen window. |
| Native-retention cost | Pre-Gradle no-value decisions have median below 500 ms and p95 below 1,000 ms over at least 30 observations spanning all five families. |
| Ordinary learning | Natural learning uses requested builds only; measurement-only builds are zero outside explicitly isolated terminal experiments. |
| Bounded regret | Report every negative commit and the worst per-build regression; future expected value cannot hide an unbounded current penalty. |
| Attribution | Each fragment is measured independently before any composed path; the complete composition is then measured directly and component percentages are not added. |
| Safe evolution | Wrapper, plugin, workflow, producer, output or correctness-authority drift suspends only affected fragments and never restores unverified bytes. |

The terminal outcome is one of:

- `CONTINUE_ADAPTIVE_FRAGMENT_POC`: every criterion passes;
- `SPECIALIZE_BOUNDED_FRAGMENT_CLASSES`: correctness passes and bounded value
  exists, but generic breadth or lifetime value fails; or
- `STOP_ADAPTIVE_FRAGMENT_POC`: the new unit of learning does not create
  defensible generic cumulative value.

## 5. Measurement model

Every longitudinal evaluation is chronological and prequential: decision `N`
may use only state observed before build `N`. No future commit, outcome or
compatibility fact may influence an earlier selection.

For a frozen sequence of requested builds:

```text
cumulativeNet(N) =
    sum(nativeGradleWallMs[i] - buildOptWallMs[i], i = 1..N)
    - additionalOutOfBandLearningCostMs(N)
```

Synchronous launcher, matching, plugin, gateway, state, cache materialization
and finalization costs are already inside `buildOptWallMs` and must not be
subtracted twice. Additional isolated calibration or background runner cost is
reported separately and included once in the economic ledger.

Each evidence bundle must retain:

- repository, revisions, chronological order and exact requested workflow;
- native and candidate commands and environment fingerprints;
- daemon, dependency, local/remote build-cache and Configuration Cache state;
- selected, rejected, suspended and newly observed fragments;
- required-output and execution-shape comparisons;
- individual wall times, signed deltas, tails and confidence method;
- matching, decision, materialization and fallback overhead;
- learning cost, payback ordinal and cumulative net value; and
- every product failure, invalid observation and exclusion reason.

## 6. Fragment lifecycle

```text
OBSERVED -> SHADOW -> QUALIFIED -> ACTIVE
    |          |          |          |
    +----------+----------+----------+--> SUSPENDED
                                      \
                                       -> EXPIRED
```

- `OBSERVED`: a natural build exposes a potential fragment. It has no
  activation authority.
- `SHADOW`: applicability and predicted value are evaluated while native
  Gradle remains authoritative.
- `QUALIFIED`: correctness authority, repeatability, value and payback gates
  pass for the exact fragment contract.
- `ACTIVE`: the fragment may be composed inside the explicit POC command.
- `SUSPENDED`: compatibility, correctness or value evidence no longer holds.
- `EXPIRED`: evidence aged beyond its preregistered validity horizon.

An observation may create a recommendation without creating an active action.
Official Gradle/plugin contracts, reviewed adapters or source patches remain
the correctness authority for cacheability. Repetition alone is not authority.

## 7. Work breakdown

### Phase A — Contract and cheap applicability

| Order | Block | Goal | State | Depends on |
|---:|---|---|---|---|
| 0 | `AF-000` Tracker and baseline alignment | Freeze the failed whole-profile baseline, define the new hypothesis and reconcile current documentation. | `DONE` | — |
| 1 | `AF-001` Adaptive fragment contract | Define the normative fragment identity, lifecycle, compatibility, invalidation, evidence and conflict model. | `DONE` | AF-000 |
| 2 | `AF-002` Fragment state schemas | Define machine-readable fragment, observation, economic-ledger and portfolio schemas with positive and negative vectors. | `DONE` | AF-001 |
| 3 | `AF-003` Cheap compatibility index | Decide native retention or return compatible fragment candidates without Gradle startup, central synchronization or materialization. | `DONE` | AF-002 |
| 4 | `AF-004` Frozen-history shadow decomposition | Decompose the existing five-repository whole-profile evidence and replay fragment applicability chronologically without making a new timing claim. | `TODO` | AF-002, AF-003 |

### Phase B — Learning and economics

| Order | Block | Goal | State | Depends on |
|---:|---|---|---|---|
| 5 | `AF-005` Fragment economic ledger | Account for signed wall-time value, decision cost, isolated learning cost, recurrence, decay, payback and bounded regret per fragment and composition. | `WAITING` | AF-002 |
| 6 | `AF-006` Ordinary-build online learner | Update fragment evidence from requested builds only and keep native Gradle authoritative until qualification. | `WAITING` | AF-004, AF-005 |
| 7 | `AF-007` Cross-repository hypothesis priors | Use task implementation, plugin, Gradle and structural features to rank hypotheses across repositories without transferring correctness authority or activation. | `WAITING` | AF-006 |
| 8 | `AF-008` Patch-opportunity learning | Turn recurring expensive non-incremental or non-cacheable task evidence into a reviewed Patch Autopilot proposal and measure the accepted patch independently. | `WAITING` | AF-006 |

### Phase C — Composition and activation

| Order | Block | Goal | State | Depends on |
|---:|---|---|---|---|
| 9 | `AF-009` Conflict-aware fragment planner | Compose compatible fragments with explicit dependencies, mutual exclusions and a predicted net-value floor; otherwise select native Gradle. | `WAITING` | AF-003, AF-005, AF-006 |
| 10 | `AF-010` Active Build Impact fragments | Activate producer/subgraph fragments independently, preserve exact required outputs and invalidate only affected fragments across commits. | `WAITING` | AF-009 |
| 11 | `AF-011` Multi-mechanism composition | Directly measure qualified Build Impact, reviewed-task/patch and bounded cache-locality combinations without adding isolated percentages. | `WAITING` | AF-008, AF-010 |
| 12 | `AF-012` Local and central adaptive state | Persist the same typed portfolio and ledger locally and through the existing HTTPS state plane; prove two-machine reuse, offline fallback and no use of Gradle cache objects as policy documents. | `WAITING` | AF-002, AF-005, AF-009 |

### Phase D — Customer path and terminal decision

| Order | Block | Goal | State | Depends on |
|---:|---|---|---|---|
| 13 | `AF-013` Longitudinal five-repository matrix | Run frozen chronological sequences on Spring, OpenTelemetry, Kafka, Micronaut and Groovy against optimized native Gradle with no lookahead. | `WAITING` | AF-010, AF-011 |
| 14 | `AF-014` Installed one-command replay | Exercise the same decisions through a clean published-package installation and `buildopt optimize <workflow>`, including fast native retention and readable value reporting. | `WAITING` | AF-012, AF-013 |
| 15 | `AF-015` Terminal adaptive-fragment decision | Recompute the complete scorecard and choose continue, specialize or stop without changing thresholds. | `WAITING` | AF-013, AF-014 |

## 8. Block definitions and required outcomes

### AF-001 — Adaptive fragment contract

**Deliverables**

- a normative specification for fragment identity, selectors, applicability,
  correctness authority, outputs, dependencies, conflicts and invalidation;
- explicit distinctions among task, producer, project/subgraph, patch and cache
  locality fragments;
- rules for evidence inheritance and decay across commits; and
- executable examples showing partial invalidation instead of whole-portfolio
  invalidation.

**Exit gate**

- the same fragment identity is stable across path-independent checkouts;
- Wrapper/plugin/implementation/output-authority drift changes only the
  fragments it semantically affects;
- missing or ambiguous state yields no active fragment; and
- no field embeds repository-specific behavior.

**Outcome:** `FRAGMENT_CONTRACT_ACCEPTED` or `FRAGMENT_MODEL_REJECTED`.

**Result:** `FRAGMENT_CONTRACT_ACCEPTED` — five fragment classes now have
canonical family/revision identities, explicit correctness authorities,
declared semantic bindings, partial invalidation and a lifecycle that requires
requalification after suspension. The executable proof contains no known
public-repository identity and makes no persistence, activation or timing
claim.

### AF-002 — Fragment state schemas

**Deliverables:** versioned JSON Schemas and fixtures for fragments,
observations, portfolios and economic ledgers; canonical digest rules; state
migration is explicitly out of scope for the POC.

**Exit gate:** valid lifecycle round trips pass; tampering, unknown authority,
cross-repository scope, incompatible generations and impossible transitions
fail closed.

**Outcome:** `TYPED_FRAGMENT_STATE_AVAILABLE`.

**Result:** `TYPED_FRAGMENT_STATE_AVAILABLE` — four closed Draft 2020-12
schemas, storage-neutral Go types and cross-record conformance now cover two
valid lifecycle bundles and seven negative mutations. Canonical JCS digests are
external, generations and repository scope link exactly, unknown versions
remain rejected and state migration is outside the POC.

### AF-003 — Cheap compatibility index

**Deliverables:** a repository-local index derived from Git, Wrapper, workflow,
producer and output fingerprints; a decision-time benchmark; explicit reasons
for compatible, suspended and native-retained outcomes.

**Exit gate:** at least 30 decisions across all five repositories; median below
500 ms, p95 below 1,000 ms; zero Gradle starts, remote calls or output
materialization for a no-value decision.

**Outcome:** `FAST_FRAGMENT_LOOKUP_AVAILABLE` or
`FRAGMENT_LOOKUP_OVERHEAD_REJECTED`.

**Result:** `FAST_FRAGMENT_LOOKUP_AVAILABLE` — one discardable local index
returns exact candidates, binding-drift suspension and explicit native
retention from repository/Wrapper/workflow/producer/output/change fingerprints.
Thirty decisions across the frozen five-repository subjects complete with a
0.025-ms median, 0.039-ms p95 and 0.061-ms maximum, with zero Gradle starts,
remote calls, output materializations or lifecycle mutations. This is lookup
overhead evidence only; it does not activate fragments or claim build savings.

### AF-004 — Frozen-history shadow decomposition

**Deliverables:** deterministic decomposition of retained profiles into
candidate fragments and chronological applicability replay over the existing
five-repository histories.

**Exit gate:** no future information is consumed; whole-profile selections are
reproduced; partial compatibility is reported separately; at least 50% of the
currently eligible descendant builds retain one candidate fragment before
authorizing active implementation.

**Outcome:** `FRAGMENT_COVERAGE_HYPOTHESIS_SUPPORTED` or
`STOP_BEFORE_ACTIVE_FRAGMENT_IMPLEMENTATION`.

### AF-005 — Fragment economic ledger

**Deliverables:** recomputable signed cost/value records, recurrence and decay
rules, payback estimates, cumulative value at fixed horizons and bounded-regret
reporting.

**Exit gate:** synthetic and retained evidence vectors prove that negative
builds reduce value, asynchronous cost is counted exactly once, percentages
are never added and a future projection cannot rewrite an observed result.

**Outcome:** `FRAGMENT_ECONOMICS_RECOMPUTABLE`.

### AF-006 — Ordinary-build online learner

**Deliverables:** observation accumulation, comparable-cohort checks,
qualification/suspension transitions and resumable exact state using only
requested builds.

**Exit gate:** zero measurement-only customer builds; interrupted state resumes
only under exact bindings; insufficient evidence remains `OBSERVED` or
`SHADOW`; value regression suspends only dependent fragments.

**Outcome:** `ONLINE_FRAGMENT_LEARNING_AVAILABLE`.

### AF-007 — Cross-repository hypothesis priors

**Deliverables:** repository-independent features and ranked candidate classes
for task implementations/plugin versions already observed elsewhere.

**Exit gate:** transferred evidence can change exploration priority only;
activation remains impossible until local correctness and value gates pass;
holdout repositories receive the same result after repository names are
replaced.

**Outcome:** `SAFE_HYPOTHESIS_PRIORS_AVAILABLE` or
`CROSS_REPOSITORY_PRIORS_REJECTED`.

### AF-008 — Patch-opportunity learning

**Deliverables:** one generic detector for an expensive task-contract problem,
a reviewable Patch Autopilot proposal, transactional validation and paired
native-before/native-after measurement.

**Exit gate:** accepted patches improve the repository's ordinary Gradle path,
remain reversible, preserve outputs and do not require BuildOpt at execution
time; rejected patches do not change the checkout.

**Outcome:** `DURABLE_PATCH_VALUE_PROVED` or `NO_PATCH_VALUE`.

### AF-009 — Conflict-aware fragment planner

**Deliverables:** dependency/conflict graph, deterministic composition,
predicted net-value floor and native fallback plan.

**Exit gate:** order-independent planning; no incompatible fragments selected;
combined correctness authority is no weaker than every constituent; prediction
failure or ambiguity retains native before Gradle.

**Outcome:** `FRAGMENT_COMPOSITION_PLAN_AVAILABLE`.

### AF-010 — Active Build Impact fragments

**Deliverables:** independent producer/subgraph activation, exact unaffected
output restoration and fragment-specific invalidation.

**Exit gate:** compatible unrelated changes retain unaffected fragments;
changed producers rebuild locally; global/build-logic ambiguity executes the
complete original workflow; exact output and zero-failure gates pass.

**Outcome:** `COMPOSABLE_BUILD_IMPACT_AVAILABLE`.

### AF-011 — Multi-mechanism composition

**Deliverables:** attributable arms for each fragment plus direct measurement
of the final composed path.

**Exit gate:** each active mechanism has independent positive evidence for the
exact scope; the composed path is faster than optimized native Gradle with a
positive lower bound; no percentage is constructed by addition.

**Outcome:** `COMPOSED_VALUE_QUALIFIED` or `RETAIN_BEST_SINGLE_FRAGMENT`.

### AF-012 — Local and central adaptive state

**Deliverables:** local-first portfolio/ledger persistence and reuse through the
existing repository-scoped HTTPS state plane.

**Exit gate:** clean second-machine reuse, optimistic concurrency, corruption
rejection, exact offline snapshot and server-outage native fallback; Gradle
cache objects and BuildOpt control documents retain separate protocols and
retention rules.

**Outcome:** `ADAPTIVE_STATE_PORTABLE`.

### AF-013 — Longitudinal five-repository matrix

**Deliverables:** frozen chronological sequences, raw observations, immutable
JSON results and an independent checker for all terminal quantitative gates.

**Exit gate:** no lookahead, equivalent optimized-native controls, exact
outputs, zero additional failures and complete signed per-build deltas. No
aggregate decision is made until every repository row is closed.

**Outcome:** per-repository `NET_POSITIVE`, `NET_NEGATIVE` or `INCONCLUSIVE`.

### AF-014 — Installed one-command replay

**Deliverables:** clean package installation, one public command, automatic
local/central state use, human/JSON reports and fast bypass/native retention.

**Exit gate:** no hand-authored BuildOpt files in evaluated repositories;
installed decisions equal source-checkout decisions; all product overhead is
inside measured wall time; bypass restores the original command and
environment.

**Outcome:** `ADAPTIVE_ONBOARDING_REPRODUCED`.

### AF-015 — Terminal adaptive-fragment decision

**Deliverables:** a digest-bound scorecard that recomputes every section 4
criterion and a concise decision update for team sharing.

**Exit gate:** all source evidence is immutable and independently checked; no
threshold, repository row, failure or negative build is omitted after results
are known.

**Outcome:** `CONTINUE_ADAPTIVE_FRAGMENT_POC`,
`SPECIALIZE_BOUNDED_FRAGMENT_CLASSES` or `STOP_ADAPTIVE_FRAGMENT_POC`.

## 9. Documentation update matrix

Every block updates this tracker. Other documents are updated in the same
commit whenever the block changes their claims or interfaces.

| Document | Update trigger | Required update | Blocks |
|---|---|---|---|
| This tracker | Every block | State, owner, evidence, outcome, next block, validation and changelog. | AF-000..015 |
| [`implementation-tracker.md`](../../implementation-tracker.md) | Phase or terminal-status change | Active phase, milestone progress and pointer to this detailed tracker. Do not duplicate block evidence. | AF-000, AF-004, AF-010, AF-013, AF-015 |
| [Master RFC](../../gradle-build-optimization-platform.md) | Product invariant or accepted architecture changes | Adaptive fragment model, authority boundary or terminal decision; implementation detail alone does not rewrite the RFC. | AF-001, AF-005, AF-006, AF-015 |
| [`specs/README.md`](../../specs/README.md) and new specs | Executable contract introduced or revised | Contract purpose, authority, checker and explicit POC boundary. | AF-001, AF-002, AF-005, AF-006, AF-009, AF-013, AF-015 |
| [`benchmarks/README.md`](../../benchmarks/README.md) | New measured or shadow evidence | Protocol, runner, controls, result links and non-additive interpretation. | AF-003, AF-004, AF-008, AF-011, AF-013, AF-014 |
| [POC one-pager](../findings/buildopt-poc-handoff.md) | Material customer-value evidence or terminal direction changes | Current idea, mechanisms, latest longitudinal numbers, conclusion and next step only. Remove superseded “current” data. | AF-004, AF-008, AF-011, AF-013, AF-015 |
| [Performance findings](../findings/build-optimization-performance.md) | New attributable timing evidence | Isolated mechanism and composed-path effects, negative evidence and activation decision. | AF-008, AF-010, AF-011, AF-013, AF-015 |
| [Generalization audit](../findings/buildopt-generalization-audit.md) | Compatibility, transfer or breadth evidence changes | Current generic boundary, selection coverage, invalidation granularity and lifetime conclusion. | AF-004, AF-007, AF-010, AF-013, AF-015 |
| [Architecture overview](../architecture/overview.md) | Runtime/data-flow architecture changes | Fragment registry, planner, learner, ledger and local/central boundaries. | AF-002, AF-006, AF-009, AF-012 |
| [Repository map](../architecture/repository-map.md) | Packages, commands or ownership move | Architecture-to-directory mapping and owning validators. | AF-002, AF-003, AF-006, AF-009, AF-012 |
| [Product onboarding](../getting-started/product-onboarding.md) | User-visible behavior changes | First build, learning, active use, native retention and report interpretation. | AF-006, AF-010, AF-012, AF-014 |
| [Product workflows](../guides/product-workflows.md) | New operator/user workflow exists | Observation, qualification, composition, patch review and fallback sequence. | AF-006, AF-008, AF-010, AF-014 |
| [CLI reference](../reference/cli.md) | CLI/options/exit/report changes | Exact syntax, outputs, exit behavior and examples. | AF-006, AF-009, AF-012, AF-014 |
| [Configuration reference](../reference/configuration.md) | New state, policy or server input exists | Defaults, scope, secrets, invalidation and bypass. | AF-003, AF-006, AF-012, AF-014 |
| [Validation reference](../reference/validation.md) | Any checker is added or changed | Command, covered contract, expected result and whether it is static, synthetic or timed. | AF-001..015 |
| [Central cache/state roadmap](./centralized-cache-and-state-roadmap.md) | Shared state representation or synchronization changes | Fragment/ledger state kinds and two-machine behavior; never merge Gradle data and BuildOpt control planes. | AF-002, AF-012 |
| Root [`README.md`](../../README.md) | Public current status, result or onboarding changes | Short current claim and links; do not copy the full tracker. | AF-000, AF-010, AF-013, AF-014, AF-015 |
| [`docs/README.md`](../README.md) | Documentation entry point changes | Link the active tracker and current finding/decision documents. | AF-000 and whenever a document is added/renamed |

## 10. Block closure checklist

A block is complete only when all applicable items are satisfied:

1. The block's source, tests, contracts and documentation are complete.
2. Targeted validation and every new independent checker pass.
3. Timed work retains raw observations and immutable machine-readable results.
4. Negative, invalid and fallback observations remain visible.
5. This tracker records the status, outcome, evidence and next unblocked block.
6. Every documentation row triggered by section 9 is updated in the same
   change.
7. `./dev/check-documentation`, layout and affected code/test gates pass.
8. The diff contains no unrelated changes or generated test residue.
9. The block is committed and pushed independently.
10. Local `HEAD`, `origin/main`, remote `main` and the clean working tree are
    verified before the block is reported complete.

## 11. Evidence register

Evidence IDs in this tracker use `AF-E###` and never reuse historical `E-###`
IDs from the implementation tracker.

| Evidence | Block | Description | State |
|---|---|---|---|
| `AF-E001` | AF-000 | New adaptive-fragment tracker, terminal-baseline reconciliation and documentation navigation. | `DONE` |
| `AF-E002` | AF-001 | [Adaptive fragment contract v1](../../specs/poc-adaptive-fragment-contract-v1.md), exact [machine policy](../../specs/poc-adaptive-fragment-contract-v1.json), dependency-free Go identity/compatibility implementation and [`dev/check-adaptive-fragment-contract`](../../dev/check-adaptive-fragment-contract). The proof covers canonical path-independent identity, five fragment classes, authority rejection, partial binding invalidation, native retention on missing/ambiguous context and mandatory requalification after suspension. | `DONE` |
| `AF-E003` | AF-002 | [Adaptive fragment state v1](../../specs/poc-adaptive-fragment-state-v1.md), exact [machine policy](../../specs/poc-adaptive-fragment-state-v1.json), four Draft 2020-12 schemas, two linked lifecycle bundles, seven negative mutations and [`dev/check-adaptive-fragment-state`](../../dev/check-adaptive-fragment-state). Schema and Go conformance prove canonical identity, JCS digest drift, exact repository/generation links, valid requalification and fail-closed unknown version/authority, tampering and impossible transitions. | `DONE` |
| `AF-E004` | AF-003 | [Compatibility-index contract](../../specs/poc-adaptive-fragment-index-v1.md), exact [machine policy](../../specs/poc-adaptive-fragment-index-v1.json), checked [30-decision report](../../benchmarks/results/adaptive-fragment-lookup-v1-local.json) and [`dev/check-adaptive-fragment-index`](../../dev/check-adaptive-fragment-index). The bounded 12-CPU run records 0.025-ms median/0.039-ms p95/0.061-ms maximum lookup, all three dispositions, five frozen repository families, recomputable summaries, tamper rejection and zero external side effects. | `DONE` |

## 12. Decision log

| Date | Decision | Reason |
|---|---|---|
| 2026-08-25 | Stop treating a complete structural profile as the generic unit of reuse. | The terminal result reached only 1/5 net-positive families and 1/6 eligible descendant selections. |
| 2026-08-25 | Use independently invalidated fragments and cumulative longitudinal value as the new POC hypothesis. | Isolated mechanisms produce strong value, but recurrence and invalidation erase whole-profile economics. |
| 2026-08-25 | Keep Gradle-native cache/configuration/incremental behavior as the substrate. | Cache parity is not differentiation; BuildOpt's value is selection, validated composition, durable fixes and learning. |
| 2026-08-25 | Permit cross-repository evidence only as a hypothesis prior. | Evidence from another repository cannot authorize correctness or activation locally. |
| 2026-08-25 | Separate stable fragment families from evidence-bound revisions. | Repository opportunity identity can survive ordinary commits without allowing changed authority, bindings or stale bytes to inherit qualification. |
| 2026-08-25 | Keep canonical document digests outside adaptive state documents. | RFC 8785 JCS plus an external SHA-256 avoids self-reference and permits the existing local/central content-addressed envelope to carry identical bytes later. |
| 2026-08-25 | Treat the compatibility index as derived and discardable. | Immutable fragment generations retain authority; a corrupt index fails closed and can be rebuilt without migrating or mutating lifecycle state. |
| 2026-08-25 | Keep Git revision as lookup provenance rather than a compatibility binding. | Cross-commit reuse is possible only when every semantic binding actually declared by a fragment remains equal. |

## 13. Change log

| Date | Change |
|---|---|
| 2026-08-25 | Closed AF-003 as `FAST_FRAGMENT_LOOKUP_AVAILABLE`: 30 decisions across five frozen repository families meet the sub-second gate with explicit compatible/suspended/native reasons and no Gradle, remote, materialization or mutation side effects; opened AF-004 shadow decomposition. |
| 2026-08-25 | Closed AF-002 as `TYPED_FRAGMENT_STATE_AVAILABLE`: four immutable record schemas, exact cross-record generations, two valid lifecycle bundles, seven negative mutations and canonical JCS digest rules now unblock the cheap compatibility index in AF-003. |
| 2026-08-25 | Closed AF-001 as `FRAGMENT_CONTRACT_ACCEPTED`: five generic fragment classes now have canonical family/revision identity, explicit authority, selective invalidation, fail-closed compatibility and mandatory requalification; opened AF-002 for typed persistence state. |
| 2026-08-25 | Created the tracker, froze the prior terminal result, defined AF-001..AF-015, the terminal scorecard and mandatory documentation update matrix. |
