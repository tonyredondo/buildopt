# Preserve pre-exec affinity observations separately

2026-09-09, after the corrected candidate's first eight native requests and
independent C5 reconstruction completed successfully. No further native profile
has started. This is a post-observation reader correction, not a native retry or
an economic threshold change.

The original reader rejects one sample of the bound N supervisor while systemd
still exposes its single-thread pre-exec `(history-replay)` stub. PID 1448134,
start tick 43512014, belongs to the exact later supervisor and owned cgroup.
The sample ends at boot time 435120149287159 ns with the inherited 32-CPU mask.
The native receipt records observer affinity `8` at 435120461949702 ns; the first
native request starts at 435120462183393 ns. The anomalous sample therefore ends
before the recorded ready and native boundaries. All native and running-worker
thread observations match their required masks. Original raw data and failure
remain immutable.

The new reader retains this row under `preExecObservations` and explicitly sets
`strictAllTimeAffinityQualified=false`. It does not assert an all-time affinity
pass. The narrower native/running-worker check uses bound PID/start time, exact
unit/cgroup, a sole pre-exec thread and the recorded observer-affinity boundary.
A pre-exec exception cannot overlap or follow that boundary, include a native
child, use a running worker command, or excuse a different process. Every other
mask remains strict. CPU ticks are retained as observed; zero quantized ticks
is not a claim of zero startup CPU consumption.

The regression reproduces the original rejection, validates all 3,635 real
samples, and rejects thirteen altered identity, scope, phase, thread, readiness
and native-affinity cases. It launches zero Gradle or helper JVM commands. The
analysis continues under the already authorized exploratory walltime protocol;
formal diagnostic readiness stays unqualified. The 5% / 1-second / two-positive-
pair / actual-reuse conditions and all output gates remain unchanged.

Resume reuses the completed eight builds, reconstructs their report with the
new reader, then executes the unchanged 4/2-CPU manifests and conditional
history. Original reader/controller remain frozen. New reader, test, controller,
failure and this amendment are additionally bound before resumption. Current
locator: `engineeringPrefix.screenCompletionFixed`; raw files are in
`.tools/state/buildopt-product-viability-v1/bv006-screen-completion-fixed`.
