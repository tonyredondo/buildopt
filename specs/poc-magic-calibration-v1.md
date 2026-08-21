# Automatic calibration behind the one-command POC

## Purpose

After automatic discovery finds a complete structural candidate,
`buildopt optimize <gradle args...>` now evaluates that candidate without
requiring any BuildOpt-authored input from the repository owner. The original
optimized-native Gradle invocation runs first and remains the authoritative
customer result. Discovery and calibration then share the explicit wall-time
budget.

## Measurement and value decision

The command reuses the frozen installed structural-measurement engine rather
than introducing a second benchmark protocol. It prepares isolated native and
candidate checkouts, stabilizes their Gradle state, runs eight alternating
control/candidate pairs, compares required outputs exactly by default, and
executes the complete original graph as a fallback correctness proof.

A candidate qualifies only when at least seven of eight pairs improve, mean
saving is at least 500 ms and 2%, the paired 95% lower bound is positive,
candidate p95 is no worse than native Gradle, observed task shapes are stable,
outputs are equivalent, fallback succeeds, and:

```text
ceil(total discovery and calibration cost / mean saving per matching build)
    <= --max-break-even-builds
```

`--calibration-pairs` still accepts 2 through 16 so an owner can bound an
exploratory invocation. The current qualification protocol is frozen at eight
pairs: a smaller budget writes the discovery result but returns
`CALIBRATION_PAIR_BUDGET_INSUFFICIENT` without a performance claim. A larger
number does not silently change the protocol.

## Checkpoint reuse

Successful calibration writes one private atomic evidence document under the
selected `.buildopt/optimize/.../calibration/` directory. A later identical
invocation skips discovery and measurement only when all invocation bindings,
generated discovery documents, evidence bytes, recomputed metrics, value
decision, fallback result, executable and Wrapper still match. Missing,
modified or structurally invalid evidence cannot be resumed.

## Current boundary

A positive result is:

```text
LEARNING / CANDIDATE_CALIBRATION_QUALIFIED / QUALIFIED
```

It proves that the generated candidate is worth keeping as POC learning. It
does not materialize a reusable profile, select that candidate for the
authoritative build, or grant production authority. Those responsibilities
belong to the next profile-portfolio and automatic-replay blocks. Failure,
timeout, insufficient evidence, non-positive value or poor payback retains
optimized native Gradle with an explicit reason.

The executable fixture adds work to the omitted project of a generic
two-project Gradle build. It proves an eight-pair qualification, exact output
equivalence, full-graph fallback, private evidence, exact checkpoint reuse and
an under-budget no-claim decision. It is a protocol fixture, not a product
performance percentage or a repository-specific rule.

The exact machine contract is
[`poc-magic-calibration-v1.json`](./poc-magic-calibration-v1.json).
