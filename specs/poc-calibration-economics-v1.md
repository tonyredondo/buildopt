# Calibration economics

## Purpose

`POC-CALIBRATION-ECONOMICS-001` measures how much work is required before a
reviewed Structural Build Impact profile reaches the steady state qualified by
the generic change-breadth experiment. It does not move setup work into the
timed build pairs or present steady-state savings as first-build savings.

The experiment covers all six qualified source-change cells from the terminal
change-breadth matrix. Each cell receives two fresh phase captures from the
same immutable BuildOpt revision and executable used by the terminal value
evidence.

## Frozen phases

The phase clock is external wall time and each value remains visible:

1. repository checkout;
2. offline Gradle distribution preparation;
3. output preflight plus structural discovery through the real
   `buildopt profile propose` command;
4. control and candidate cache seeding;
5. control and candidate base-daemon stabilization;
6. three target-workload stabilization executions per arm; and
7. steady-state native and BuildOpt wall time.

The public proposal command intentionally combines output preflight and graph
discovery, so the POC reports that phase as one observed cost rather than
inventing an internal split. Checkout is measured but excluded from every
break-even calculation because both native Gradle and BuildOpt require the
same source revision.

If the host does not already contain the exact Wrapper distribution, the
runner downloads it once with up to three bounded attempts before any phase
clock starts. The measured distribution phase is then the reproducible copy of
that exact prepared distribution into the private capture home plus one
offline Wrapper version check. Network latency is deliberately not presented
as BuildOpt calibration cost.

Warm-up costs and steady-state value are recomputed from the two immutable
terminal captures already used to qualify each change-breadth cell. They are
not re-created from logs, and no failed or excluded attempt contributes data.
The three target executions are the terminal evidence phases
`TARGET_WORKLOAD_STABILIZATION`, `TARGET_WORKLOAD_STABILITY_CONFIRMATION`, and
`TARGET_WORKLOAD_STABILITY_RECONFIRMATION`; all three count toward calibration.

## Three economic views

- **Installed workflow** includes proposal discovery and candidate warm-ups.
  It assumes the repository and Gradle distribution already exist and excludes
  control-arm work that is needed only to validate the POC.
- **POC validation** includes proposal discovery plus both control and
  candidate warm-ups. It shows the cost of producing comparative evidence.
- **Cold single workflow** adds offline Gradle distribution preparation to the
  complete POC validation cost. It assigns that shared environment cost to one
  workflow conservatively.

For each cell and view:

```text
break-even builds = ceil(calibration cost / terminal mean saving per build)
```

The unit is repeated qualifying builds after calibration. Results remain
cell-specific; percentages and costs are never averaged across repositories,
workflows, or changes.

## Boundaries

This is a POC economics study, not an installation SLA or production rollout.
It does not measure public-release download latency, online dependency
download, repository clone cost as product overhead, automatic activation,
soak, design-partner operation, or Test Optimization. A new, ambiguous,
global, or drifted input still retains optimized native Gradle.

## Validation

```bash
./dev/check-calibration-economics
```

Terminal evidence is assembled and independently recomputed only after all
twelve phase captures exist.
