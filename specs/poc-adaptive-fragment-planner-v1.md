# Conflict-aware adaptive fragment planner v1

Status: accepted POC contract for `AF-009`.

Machine policy:
[`poc-adaptive-fragment-planner-v1.json`](./poc-adaptive-fragment-planner-v1.json).

## Purpose

This contract defines the pure pre-Gradle planner that turns already compatible,
qualified fragment revisions into either one exact composition or an explicit
native Gradle plan. It does not activate any fragment. `AF-010` owns the first
runtime activation and `AF-011` owns direct measurement of a composed path.

The planner is repository-name independent. Repository scope isolates evidence,
but repository identity is never interpreted as behavior.

## Inputs and authority

Every candidate must be an unexpired `QUALIFIED` or `ACTIVE` generation from
the same repository scope. The planner revalidates the canonical fragment,
revision, lifecycle, dependency graph and authority even when an earlier lookup
reported it compatible.

Each economic prediction names one exact canonical set of family IDs, a fixed
horizon, a signed predicted net value in milliseconds and the SHA-256 of the
external economic evidence. The planner does not add isolated mechanism
percentages or infer a joint effect from individual fragments. A composition
without its own exact prediction is ineligible.

## Deterministic selection

An alternative is eligible only when:

1. every selected fragment's transitive requirements are also selected;
2. no selected family conflicts with another family in either direction;
3. every selected generation is current and retains its own correctness
   authority; and
4. the exact composition prediction meets the fixed positive net-value floor.

The eligible alternative with the greatest predicted net value is selected.
Equal predictions use the lexicographically canonical family set as the stable
tie-break. Reordering candidates or predictions cannot change the resulting
plan ID.

The composed correctness mode is a conjunction: every constituent authority
remains present in the plan. Composition never invents a weaker aggregate
authority.

## Native fallback

Native Gradle is selected before Gradle starts when candidate identity,
revision, lifecycle, expiry, repository scope, dependency closure or prediction
is missing, invalid or ambiguous. It is also selected when every safe exact
alternative is below the predicted net-value floor. A native plan contains no
selected fragments and no predicted value.

Mutually exclusive or dependency-incomplete alternatives are rejected visibly.
Their existence does not prevent a different independently safe exact
alternative from being selected.

## Executable proof

The implementation in [`internal/adaptivefragment`](../internal/adaptivefragment)
proves order-independent selection, dependency closure, bidirectional conflict
handling, canonical tie-breaking, retained constituent authorities and native
fallback. The recomputable report uses synthetic economic values deliberately:
it proves planner behavior and makes no build-time claim.

Run:

```bash
./dev/check-adaptive-fragment-planner
```

The accepted outcome is `FRAGMENT_COMPOSITION_PLAN_AVAILABLE`. It authorizes
`AF-010` to activate independently invalidated Build Impact fragments through
this planner. It does not authorize automatic customer activation, production
rollout, soak testing, design-partner work or Test Optimization behavior.
