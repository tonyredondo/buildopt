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
