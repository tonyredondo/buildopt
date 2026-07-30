# Causal pilot harness v1

Status: implemented by `A0-009`.

This contract closes the first internal savings proof without turning a small
local sample into beta promotion evidence. It extends the neutral envelope from
the descriptive `WS-009` overhead report to a pre-outcome paired experiment and
a separate immutable `EXPERIMENT_RESULT`.

## Assigned treatment

Each pair executes the same source, requested work, required JAR, BuildOpt
launcher, Gradle version, JDK, Tier 1 policy, isolated workspace state, and
single-use daemon state.

- `CONTROL` runs through `buildopt run` with Tier 1 active and the build cache
  disabled.
- `CANDIDATE` runs through the same launcher and policy with the native managed
  L1 enabled and warmed before the measurement window.

The difference therefore has `effectScope=ACTION_INCREMENTAL`: it measures the
managed-L1 action, while the common product envelope cancels between arms.
Every measured duration already includes action setup and product overhead.
`incrementalActionOverheadMs` is a non-negative decomposition input; zero means
that the bounded internal pilot makes no separate positive overhead claim, not
that the complete measured envelope was excluded.

Odd pairs execute control first and even pairs execute candidate first. A
mode-`0600` assignment is durably published under a current-user mode-`0700`
directory before its command starts. The assignment fixes the experiment,
epoch, action, baseline and control definitions, cohort, six required strata,
work-units fingerprint, state prescription, command class, propensity, and
exclusion policy. Actual hits, duration, outcome, and output digest cannot enter
that record.

## Neutral observation and analysis

`pilot-observe` owns the clock outside the complete assigned command and ends
only after the process and required deliverable are available. The observation
binds the exact assignment SHA-256. Success, build failure, infrastructure
failure, and cancellation are all retained; a slow or failed candidate cannot
disappear from assigned outcome counts.

The successful-latency estimand analyzes only complete successful pairs while
the result keeps intention-to-treat outcome counts and explicit exclusions.
The producer calculates:

```text
observedNetBuildTimeSavedMs =
  mean(control customerVisibleBuildMs - candidate customerVisibleBuildMs)

observedBuildTimeReductionRatio =
  sum(controlMs - candidateMs) / sum(controlMs)

customerVisibleBuildP95DeltaMs =
  candidate p95(customerVisibleBuildMs) -
  control p95(customerVisibleBuildMs)
```

Intervals come from 4,096 deterministic paired-bootstrap resamples. Negative
pair effects and aggregate regressions are retained. Required-output divergence
and product-attributable failures remain non-compensable.

## Result lifecycle and A0 boundary

The producer emits normative `EXPERIMENT_RESULT v1` JSON only after all
observations close. It never edits a `BUILD_SESSION`. Result JSON and the bare
result JSONL stream are current-user private, bounded to 64 MiB with 1 MiB
lines, append-only by `(experimentId, resultVersion)`, idempotent only for
byte-identical content, and exportable to stdout byte for byte.

The internal A0 proof needs four fully analyzed pairs, a positive mean saving,
a strictly positive lower 95% bound, identical required deliverables, and no
failure regression. A cache hit alone is insufficient.

The output is always `PRELIMINARY`: it has no `gateEvaluation`, cannot authorize
promotion, and does not satisfy the beta sample/window, feedback, queue, p99,
economics, or external-pilot requirements. It closes `A0-G09` only. The
separate no-hit p95 budget in `A0-G06` remains open.

Run the complete real Gradle exercise:

```bash
./dev/check-causal-pilot
```
