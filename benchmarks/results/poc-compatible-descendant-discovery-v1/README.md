# Compatible descendant discovery POC

This evidence records the first preregistered non-Kafka window after workflow
input ownership became executable. The public OpenTelemetry JMX change mixes
module-owned sources and resources with a root changelog. BuildOpt proves the
changelog is consumed by no requested task, persists only the effective change
set for replay, and discovers an eight-project candidate from a 1,027-project
graph.

The correction is functionally successful: all eight candidate arms execute,
all 50 required outputs match their controls byte for byte, fallback succeeds,
and no product-attributable failure occurs. The performance gate deliberately
rejects the profile:

- control mean: **122.044 seconds**;
- candidate mean: **119.333 seconds**;
- mean saving: **2.711 seconds / 2.22%**;
- positive pairs: **5/8**;
- 95% interval: **-1.124 to +7.330 seconds**; and
- p95: **134.773 to 125.235 seconds**.

The mean and p95 are encouraging, but the interval crosses zero and only five
pairs improve. The terminal decision is therefore
`CALIBRATION_VALUE_NOT_PROVEN`; no profile or materialization pack is
published and no descendant is timed. This is not threshold failure to be
worked around: it demonstrates that a 1,027-to-8 graph reduction can still be
dominated by configuration and build-logic work.

The next experiment must recapture complete producer-bound native observations
for a previously strong public calibration, apply producer-atomic native
volatility quarantine, rebuild quarantined producers locally, and then measure
compatible descendants. The cross-repository value claim remains Kafka-only
until two non-Kafka selected replays are positive.

Validate the compact evidence with:

```bash
./dev/check-compatible-descendant-discovery
```

This is POC evidence only. It adds no repository-specific product rule,
production authority, soak requirement, design-partner dependency, or Test
Optimization behavior.
