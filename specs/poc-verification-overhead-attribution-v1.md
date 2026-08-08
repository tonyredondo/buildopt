# Verification overhead attribution POC

The complete Spring verification graph reduced the selected project closure
from 23 projects to 12, but its uninstrumented result saved only 103.75 ms
(0.31%) with two favorable pairs out of four. This protocol diagnoses that
neutral result without treating diagnostic durations as a new performance
sample or changing the retained full-graph decision.

## Fixed comparison

The repository revision, source mutation, Gradle options, cache state,
required Checkstyle report, native `checkstyleMain` control and
`:spring-webmvc:checkstyleMain` candidate remain identical to the completed
verification experiment. Each arm receives one unmeasured warm-up followed by
one diagnostic observation. The runner also requires the exact native-cache
seed SHA-256 recorded by that source experiment.

Gradle 9.6.1 operation traces provide non-overlapping external phases and task
rows. The BuildOpt candidate also emits its existing phase-timing report. The
result must compare:

- launcher and Gradle-client startup;
- configuration before the main task interval;
- the temporal task interval;
- Gradle finalization and launcher teardown;
- every task path, duration and outcome;
- tasks present only in the native control; and
- BuildOpt planning, Gradle setup, runtime setup and teardown.

Tracing perturbs execution and the task durations can overlap under parallel
execution. These values are attribution evidence only; they cannot qualify the
scope, replace the four accepted pairs or be added together as independent
savings.

## Action rule

At most one generic correction is allowed, and only when a named
candidate-specific phase exposes at least 500 ms of recoverable critical-path
cost while selection, outputs and fallback remain unchanged. A difference in
Gradle startup, daemon scheduling, task overlap or ordinary sample noise is not
enough.

If no such phase exists, the terminal decision is
`STOP_VERIFICATION_OPTIMIZATION_NO_ACTIONABLE_BOTTLENECK`. Verification remains
on optimized native Gradle and this line stops rather than rerunning the same
comparison.

This is owner-controlled POC diagnosis. It changes no production authority,
requires no soak or design partner, and does not modify Test Optimization.

The machine-readable contract is
[`poc-verification-overhead-attribution-v1.json`](./poc-verification-overhead-attribution-v1.json).
