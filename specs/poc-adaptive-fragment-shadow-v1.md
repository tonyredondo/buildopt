# Adaptive fragment frozen-history shadow replay v1

Status: accepted POC evidence contract for `AF-004`.

Machine policy: [`poc-adaptive-fragment-shadow-v1.json`](./poc-adaptive-fragment-shadow-v1.json).

## Purpose

The shadow replay asks whether splitting the historical whole-profile model
could retain useful compatibility across descendant commits. It uses the
already frozen five-repository lifetime evidence and does not run, time or
change a build.

Only Apache Kafka produced a robust qualified profile. Spring Framework,
OpenTelemetry Java Instrumentation, Micronaut Core and Apache Groovy stopped
before calibration because their ordinary histories could not repay learning;
they therefore contribute provenance but no invented descendant fragment
decisions.

## Deterministic decomposition

The qualified Kafka profile is separated into two candidates:

- `SUBGRAPH` retains the structural Gradle selection contract;
- `OUTPUT_MATERIALIZATION` retains the qualified output bytes.

The economic gate remains separate. Structural compatibility cannot authorize
output replay, and compatibility alone cannot authorize execution. An output
refresh suspends only materialization; an economic rejection keeps execution
native without erasing the structural observation.

Each decision reads the qualification plus the facts recorded at its own
sequence. `maxSourceSequence` must equal the decision sequence, so later
observations cannot change an earlier result.

## Frozen result

The six eligible Kafka descendants produce:

| Shadow result | Builds |
|---|---:|
| Whole profile selected and reproduced | 1 |
| Partial compatibility retained | 5 |
| At least one candidate fragment retained | 6 |
| Future observations consumed | 0 |

The fragment-retention ratio is therefore **6/6 (100%)**, above the frozen 50%
gate. Four descendants retain the subgraph while output materialization is
suspended for refreshed bytes; one retains the subgraph while materialization
is not evaluated because the economic gate keeps native execution. The
accepted outcome is `FRAGMENT_COVERAGE_HYPOTHESIS_SUPPORTED`.

This result supports implementing active fragment economics and learning. It
is not evidence that any partial fragment saves wall time, and it authorizes no
fragment activation.

Run:

```bash
./dev/check-adaptive-fragment-shadow
```

The checker recomputes the report from every frozen subject result and capture,
verifies their SHA-256 provenance, reproduces the whole-profile selections and
rejects report tampering. Production rollout, soak/design-partner work and Test
Optimization remain outside this POC block.
