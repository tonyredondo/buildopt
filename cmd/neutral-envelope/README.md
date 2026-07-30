# Neutral measurement envelope

`neutral-envelope` is the dependency-free `WS-009` and `A0-009` measurement
helper. It executes one assigned arm under an external monotonic clock,
validates the required deliverable, emits a private raw observation, combines
complete alternating pairs, and validates existing reports.

The causal-pilot commands persist a prescribed assignment before execution,
retain success and failure outcomes, publish a deterministic paired-bootstrap
`PRELIMINARY` `EXPERIMENT_RESULT`, and export its bounded private JSONL stream
byte for byte. They never authorize promotion or mutate a completed
`BUILD_SESSION`.

It is a development/evidence binary, not a customer launcher. The executable
contract, boundaries, qualification rules, and commands are defined in
[`specs/walking-skeleton-overhead-v1.md`](../../specs/walking-skeleton-overhead-v1.md).
The causal extension is defined in
[`specs/causal-pilot-v1.md`](../../specs/causal-pilot-v1.md).
