# Bounded bandit policy v1

This specification materializes `F0-035` and the Phase 0 contract for
`BANDIT-001`. The beta algorithm is contextual epsilon-greedy over a finite,
prevalidated resource-profile catalog; it is not a continuous optimizer.

## Authority

The bandit may choose only among `STABLE_CONTROL`, `W2_H3G`, `W3_H4G`, and
`W4_H6G` for the exact golden runner/build/compatibility class. It changes only
maximum workers and Gradle daemon heap. It cannot authorize caching, task
qualification, a patch, graph omission, release, or any correctness gate.

An ineligible profile has zero propensity. A signed kill switch, OOM,
sustained swapping, artifact divergence, product-attributable failure, or
queue/p95 guardrail breach suspends the affected arm and rolls back to
`STABLE_CONTROL`.

## Features and assignment

The versioned bucket contains only pre-outcome runner limits,
repository/task-graph/change classes, toolchain, historical hit-rate class,
observable cache/workspace warmth, daemon/JIT state, and contention signals.
Actual hits, final duration/result, and future data are forbidden.

Every assignment persists the policy/catalog/bucket, arm, deterministic seed,
and propensity before Gradle starts. The stable control retains at least 5%.
Exploration begins at no more than 10% and may decline no lower than 2%.
Ties retain control.

## Learning and replay

The beta starts in A/A and fixed cohorts. Every candidate needs 20 valid
outcomes in its exact bucket before bandit entry. Prediction is a trimmed mean
reward with five pseudo-observations shrinking each candidate toward control.
Reward primarily favors lower `customerVisibleBuildMs`; versioned
runner/queue/cost penalties remain explicit components.

An outcome updates exactly once only when it carries the recorded propensity,
arrives no more than 24 hours after assignment, belongs to the same policy
era/bucket, and has complete reward/guardrails. Missing, duplicate, late, or
partial outcomes are `INCONCLUSIVE`. Missing propensity or failed A/A returns
the experiment to fixed assignment.

A measurement epoch, catalog, feature/reward definition, runner, Gradle/JDK,
material build-logic, or drift-threshold change resets only the affected
bucket and requires fresh A/A/fixed cohorts. Samples are never mixed across
eras.

## Conformance

[`bandit-policy-v1.json`](./bandit-policy-v1.json) contains 15 deterministic
replay scenarios. Run:

```bash
./dev/check-bandit-policy
```

The checker binds the four real `RESOURCE_PROFILE` fixtures and executes A/A,
fixed-to-bandit entry, shrinkage/greedy/tie/exploration selection, headroom,
propensity, delayed/duplicate outcome, reset, kill-switch, and guardrail
rollback cases.
