# New-family calibration economics

## Purpose

`POC-NEW-FAMILY-CALIBRATION-ECONOMICS-001` measures how many repeated Ktor
builds are needed to repay the installed discovery and candidate-stabilization
cost for the three change shapes already qualified by the terminal
change-breadth experiment. It does not rerun or rewrite the terminal build-time
savings.

This is a proof-of-concept economics study. It asks whether the demonstrated
steady-state value is large enough to repay the one-time work needed to find
and stabilize the candidate. It is not a production admission controller,
background calibration service or automatic activation policy.

## Frozen inputs

The study is bound to:

- the public Ktor revision and three selective cells in
  `poc-new-family-change-breadth-v1.json`;
- the terminal BuildOpt revision, executable, two terminal captures and
  qualification for each cell;
- the separate calibration executable built from revision `82c25e3`; and
- the exact Gradle options, 12-worker runner class, changed paths, required
  outputs and candidate entrypoints already frozen for the terminal study.

The global-configuration cell remains native full graph and is not timed.

## Fresh phases

Each selective cell receives two independent captures. Every capture records:

1. a cold installed `profile propose` pass;
2. an exact digest-bound proposal replay, reported separately;
3. a drift probe that must not replay;
4. candidate cache seed and base-daemon stabilization; and
5. two exact matching target-workload fingerprints, with a bounded third
   target run required only when the first two differ.

The calibration command runs no measured control/candidate pairs, makes no
qualification decision and does not time the global fallback. Required outputs
must still equal the terminal output digest and count for the cell.

## Economics

Per cell, and never averaged across change shapes:

```text
installed calibration cost = fresh cold proposal mean
                           + fresh candidate warm-up mean

installed break-even builds = ceil(
  installed calibration cost / unchanged terminal mean saving
)
```

The exact-replay view replaces the cold proposal mean with the exact replay
mean. It represents reevaluating the same immutable inputs, not first-time
onboarding.

## Acceptance

The block closes only when all three cells have two fresh captures, exact cold
and replay artifacts, rejected drift, converged target fingerprints, terminal-
identical required outputs and a recomputable cell-specific break-even result.
No maximum break-even threshold was chosen before measurement; the result is
descriptive and may show that a cell is uneconomic.

## Boundaries

This work does not authorize production use, automatic activation,
repository-name rules, Test Optimization, soak or design-partner operation.
It preserves the terminal performance claims and tests only whether their
installed calibration cost is recoverable through repeated builds.
