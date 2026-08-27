# Sticky wrapper longitudinal campaign v2

Status: preregistered POC measurement contract; blocked by `SWL-014A..D`.

## Why v2 exists

The retained v1 diagnostic proved that the committed wrapper could execute five
substantial public builds and preserve their required outputs. It did not test
the sticky-learning hypothesis: the runner added `--build-cache` only to the
control, configured no server or project identity, set the trial budget to zero
and therefore exercised the candidate's no-op/light-observation route.

That evidence remains immutable and useful for compatibility. It is
`DIAGNOSTIC_ONLY` and cannot contribute to this protocol or the terminal
`SWL-016` decision.

## Preconditions

Timing cannot start until all four gates pass:

1. `SWL-014A` proves arm symmetry and a checker that rejects timing-only
   readiness.
2. `SWL-014B` connects observation, proposal, trial, signed decision, active
   execution, suspension and economics to the real `./buildoptw` path.
3. `SWL-014C` finds independently testable generic actions in at least three of
   the five frozen public families.
4. `SWL-014D` proves positive conservative value for those actions through the
   installed wrapper with exact outputs and complete cost attribution.

Failure of the breadth or installed-value pre-gate stops the experiment before
the longitudinal compute budget is spent.

## Fair arms

For a given repository revision, control and candidate receive the same frozen
Gradle argument vector. Cache enablement is part of that vector and is never
injected into only one arm. Both arms receive separate but equivalent local
Gradle state and the same optional remote-cache availability, namespace,
network profile and read/write policy. Dependency preparation remains outside
pair wall time and copies no task outputs or BuildOpt decision state.

The arms are:

```text
control:   ./gradlew  <identical frozen arguments>
candidate: ./buildoptw <identical frozen arguments>
```

Order alternates by comparable pair. Checkouts, homes, Gradle state and
BuildOpt state remain isolated. Candidate state persists chronologically only
inside the candidate arm and is bound to the exact repository, workflow,
Wrapper, toolchain and action generations.

## Required candidate evidence

Every candidate row records:

- selected lifecycle state and execution decision;
- decision generation, action identifier and binding digest;
- whether no-op, observation, shadow, trial, runtime action or reviewed durable
  action actually executed;
- complete wrapper, decision, observation, trial, cache, fallback, action,
  verification and Gradle phase costs;
- gross saving, total cost and signed net value recomputable from immutable
  records;
- output equivalence and product-attributable outcome; and
- native counterfactual or the exact reason it was not due.

Inactive, shadow, rejected or suspended actions receive zero saving. A cache hit
is an observed mechanism fact and never action authority.

## Campaign and terminal boundary

The five families, current package, chronological primary/reserve commits,
workflows, outputs, toolchains and confidence method are frozen before timing.
The target is 20 comparable requested builds per family and the minimum is 15.
Every positive, negative, no-action and excluded row remains visible.

Sample count alone cannot produce `READY_FOR_SWL_016`. Readiness additionally
requires completed preconditions, symmetric arms, exact/reviewed outputs, zero
product-attributable failures, complete lifecycle evidence, deterministic
checkpoints and a signed economic ledger.

The immutable terminal scorecard remains the one in the
[Sticky Wrapper Learning POC Tracker](../docs/plans/sticky-wrapper-learning-poc-tracker.md).
No threshold is changed by this correction. Soak testing, design-partner work,
production authority, automatic patch merge and Test Optimization remain out
of scope.
