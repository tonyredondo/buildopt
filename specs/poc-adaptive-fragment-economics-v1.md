# Adaptive fragment economic ledger v1

Status: accepted POC economic contract for `AF-005`.

Machine policy: [`poc-adaptive-fragment-economics-v1.json`](./poc-adaptive-fragment-economics-v1.json).

## Purpose

The economic ledger keeps observed value immutable and signed before any
fragment can be activated. A slower candidate is negative evidence; a native
retention path contributes synchronous cost; learning and publication costs
are charged once by stable event identity even when retries repeat the event.

For one fragment or inseparable composition:

```text
observed net = sum(activated signed gross savings)
             - sum(synchronous overhead)
             - sum(unique asynchronous cost events)
```

Compatibility, activation and requested-build counts remain an exact fraction.
The ledger never adds percentages across builds, fragments or mechanisms.

## Recurrence, decay and horizons

Projection is derived and cannot edit the observed ledger entry. Expected
gross value per requested build uses the complete signed observed gross divided
by requested builds, so historical activation recurrence is already present.
The value decays by 900/1,000 per future build. Fixed 1-, 5- and 10-build
horizons report projected gross, projected synchronous cost, projected net and
observed-plus-projected net separately.

Observed payback is the first requested build whose cumulative value becomes
non-negative after positive gross value exists. Projected payback is reported
only when the fixed future horizon crosses zero. Regret reports the complete
observed downside and whether it exceeds the policy budget; it never clips the
ledger value.

## Evidence

The checked report contains three recomputable assessments:

| Vector | Result |
|---|---:|
| Synthetic signed fragment | 600 ms gross - 60 ms synchronous - 500 ms unique asynchronous = **+40 ms** |
| Synthetic negative fragment | -300 ms gross - 50 ms synchronous = **-350 ms**, outside its 100-ms regret budget |
| Retained Kafka composition | 135,127 ms gross - 42,040 ms native-retention wrapper cost - 10,560 ms qualification/publication = **+82,527 ms** |

The synthetic signed vector includes one -100-ms build, reducing cumulative
value by 120 ms after its 20-ms synchronous cost. Its asynchronous cost appears
twice but is charged once. Changing projection policy changes future rows while
leaving the observed entry and recurrence identical.

Kafka remains a `COMPOSITION`: the retained evidence cannot attribute its
135,127-ms selected replay safely between `SUBGRAPH` and
`OUTPUT_MATERIALIZATION`. Its observed payback remains build 2. Projections are
POC estimates, not new wall-time measurements or activation authority.

Run:

```bash
./dev/check-adaptive-fragment-economics
```

The checker recomputes the report from frozen evidence, runs signed/negative,
deduplication, immutability and regret tests, and rejects tampering. Production
rollout, soak/design-partner work and Test Optimization remain outside this
block.
