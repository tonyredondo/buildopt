# Memory-bounded five-repository generic structural profile matrix

This POC reruns the complete Spring Framework, OpenTelemetry Java
Instrumentation, Apache Kafka, Micronaut Core, and Apache Groovy matrix after
two OpenTelemetry v2 attempts exhausted the host memory envelope during the
correctness-only full-graph fallback.

Version 3 preserves every timed condition from v2:

- the same five public revisions, fixed changes, workflows, and outputs;
- optimized native Gradle versus the installed Build Impact candidate;
- 12 workers, parallel execution, daemon use, native build cache, and eight
  alternating measured pairs;
- both arms prepared before either measured process;
- process-only inter-arm gap bounded at five seconds;
- byte-identical required outputs and the unchanged 500-ms/2%/positive-bound
  qualification gates.

Only the untimed safety fallback changes. After all eight pairs and output
checks finish, the harness must stop both isolated Gradle daemons successfully.
It then validates the candidate's native full-graph fallback with
`--no-daemon`, `--no-parallel`, and `--max-workers=4`. Fallback duration never
enters the measured effect. This prevents the fallback JVM from overlapping
the two hot measurement heaps while retaining the same tasks, sources, and
required outputs.

The v2 attempts are failure evidence only. One attached invocation was
externally interrupted after eight pairs; one transient service was selected
by `systemd-oomd` after eight pairs; and one service protected from `oomd` was
terminated by the kernel OOM killer during fallback. None produced a terminal
`result.json`, and no pair from them is reusable.

Each repository remains an atomic capture unit. An interrupted or failed row
is deleted and rerun from zero; completed terminal rows may be retained while
later repositories run. The final summary is created only when all five rows
have terminal `result.json` files.

A row qualifies only when all eight pairs are positive, mean savings exceed
500 ms and 2%, the paired lower bound is positive, required outputs are stable
and identical, and the memory-bounded full-graph fallback succeeds.
Repository percentages are never averaged and mechanism percentages are never
added.

This is POC evidence only. It does not authorize automatic activation,
production rollout, Test Optimization, soak testing, or design-partner work.
