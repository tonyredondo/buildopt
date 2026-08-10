# OpenTelemetry scheduling-equivalent fallback correction

This POC correction reruns only the OpenTelemetry Java Instrumentation row
after the terminal five-repository v3 matrix proved that the timed structural
candidate is stable and faster, but the reduced-concurrency correctness
fallback changes required output bytes.

The v3 OpenTelemetry row remains immutable failure evidence. Its eight timed
pairs were all positive and byte-identical between optimized native Gradle and
BuildOpt, but no evidence document was emitted because the full-graph fallback
ran with `--no-parallel --max-workers=4` while the measured arms used
`--parallel --max-workers=12`. The other four v3 rows are not rerun or changed.

Version 4 freezes the following before any new OpenTelemetry observation:

- the same public revision, fixed source change, 53-entrypoint Spring-family
  workflow, required outputs, JDK, cache state, and installed BuildOpt path;
- eight fresh alternating pairs with both arms prepared before either process;
- daemon use, parallel execution, 12 workers, native build cache, and disabled
  Configuration Cache for every timed arm;
- the process-only five-second inter-arm gap and post-pair byte comparison;
- the unchanged 500-ms, 2%, positive-bound, and 8-of-8 qualification gates;
- successful full-graph fallback with byte-identical required outputs.

The only correction is outside the measured effect. Both hot measurement
daemons are stopped first. The fallback then uses `--no-daemon` while retaining
the measured `--parallel --max-workers=12` scheduling. This avoids overlapping
three Gradle heaps without changing the scheduling mode that produced the
stable measured outputs. If outputs still differ, the collector must fail
closed and report the first bounded set of missing, unexpected, or changed
paths.

No v1, v2, or v3 OpenTelemetry timing is reused. A terminal v4 result must be
captured from zero and checked as one atomic subject. Final reporting may place
that corrected OpenTelemetry row beside the immutable v3 results for Spring,
Kafka, Micronaut, and Groovy, but must label the fallback-proof revision and
must never average repository percentages or add mechanism percentages.

This is POC evidence only. It does not authorize automatic activation,
production rollout, Test Optimization, soak testing, or design-partner work.
