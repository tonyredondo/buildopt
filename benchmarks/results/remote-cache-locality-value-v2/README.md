# Remote Cache Locality Value v2 evidence

`RCL-001` freezes planning authority only. No public build, cache seed, timing
sample or speedup claim exists yet. Historical Edge/locality reports are design
inputs and cannot satisfy an RCL row.

Authoritative contracts:

- [`specs/poc-remote-cache-locality-value-v2.md`](../../../specs/poc-remote-cache-locality-value-v2.md)
- [`specs/poc-remote-cache-locality-value-v2.json`](../../../specs/poc-remote-cache-locality-value-v2.json)
- [`specs/poc-remote-cache-locality-value-v2.subjects.json`](../../../specs/poc-remote-cache-locality-value-v2.subjects.json)
- [`docs/plans/remote-cache-locality-value-poc-tracker.md`](../../../docs/plans/remote-cache-locality-value-poc-tracker.md)

[`harness-proof.json`](./harness-proof.json) closes `RCL-002`. Its independent
checker binds six source files and reruns native Gradle key parity, verified
committed read-through, offline restart, corruption-as-miss and unsafe-proxy
negatives. It records zero public builds and zero timing samples. `RCL-003` is
next and may collect only fresh untimed public producer/consumer correctness.

[`public-correctness.json`](./public-correctness.json) stops `RCL-003`. Groovy
restores 39 objects and preserves its JAR but supplies only 6.90 MB versus the
8-MiB gate. Kafka supplies 170.91 MB and 58 hits, but the frozen required
`test-fixtures.jar` digest differs between producer and consumer. The remaining
three families are `NOT_RUN_GATE_CLOSED`; `RCL-004/005` are not authorized.

[`terminal-decision.json`](./terminal-decision.json) closes `RCL-006` as
`STOP_REMOTE_CACHE_LOCALITY_VALUE_POC`. No warm pair, installed build, signed
portfolio, payback or speedup was measured. The result rejects this frozen
cohort/output contract; it is not evidence that cache locality has no value.
