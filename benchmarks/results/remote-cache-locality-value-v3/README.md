# Remote Cache Locality Value v3 evidence

`RCL3-001` freezes the corrected experiment. No v3 public build, cache seed or
timing sample exists. V2 evidence supplies no v3 row.

- [`specs/poc-remote-cache-locality-value-v3.md`](../../../specs/poc-remote-cache-locality-value-v3.md)
- [`specs/poc-remote-cache-locality-value-v3.json`](../../../specs/poc-remote-cache-locality-value-v3.json)
- [`specs/poc-remote-cache-locality-value-v3.subjects.json`](../../../specs/poc-remote-cache-locality-value-v3.subjects.json)
- [`docs/plans/remote-cache-locality-value-v3-poc-tracker.md`](../../../docs/plans/remote-cache-locality-value-v3-poc-tracker.md)

[`harness-proof.json`](./harness-proof.json) closes `RCL3-002`: production Edge
committed reads, offline restart, corruption-as-miss and unsafe-use negatives
pass freshly. Five 1-MiB calibration GETs land at 41.045–41.668 ms under the
frozen 30-ms plus 100-MiB/s profile. Public builds and value timings remain zero.

[`public-correctness.json`](./public-correctness.json) closes `RCL3-003` with
all five rows complete. Groovy, OpenTelemetry and Spring are exact and eligible;
Kafka is natively unstable and Micronaut exposes zero cache hits. There are zero
product-attributable failures, so the frozen 3/5 gate opens `RCL3-004` timing.

[`paired-value.json`](./paired-value.json) completes all 24 balanced pairs.
None of the three eligible families passes every frozen value criterion:
Groovy misses 2%, OpenTelemetry is negative, and Spring's bootstrap lower 95%
crosses zero. [`terminal-decision.json`](./terminal-decision.json) therefore
records `STOP_REMOTE_CACHE_LOCALITY_VALUE_V3`. `RCL3-005` is not run because
its 3/5 prerequisite failed; no installed-economics or real-path viability
claim exists.
