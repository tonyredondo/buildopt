# Capability matrix v1

This specification materializes `F0-036` and the reporting contract for
`COMPAT-001`. It separates the Tier 1 target from combinations actually proven
at the current repository revision.

## Target combinations

Tier 1 is Linux x86-64 with:

- Gradle 8.14.3 on JDK 17 or 21;
- Gradle 9.6.1 on JDK 17, 21, or 25;
- Groovy DSL and Kotlin DSL for each runtime pair.

The implementation golden lane remains Gradle 9.6.1/JDK 21/Kotlin DSL. The
correlation spike additionally exercised Gradle 8.14.3/JDK 21/Kotlin DSL.
`F0-040` executes all eight JDK 17/21 rows with the packaged product plugin,
custom task, artifact transform, build cache, and Configuration Cache through
both TestKit and a real Wrapper. The two JDK 25 rows remain `UNTESTED`; a
target label is not evidence.

## Status contract

Every capability reports exactly one of:

- `EXACT`: the adapter/fixture proves the stated observation or behavior;
- `APPROXIMATED`: a named method produces a bounded estimate;
- `UNAVAILABLE`: the combination cannot currently provide it and declares a
  safe fallback and reason.

`UNAVAILABLE` is not zero or false. Unknown Gradle/JDK/DSL/plugin combinations
use the implicit `UNKNOWN` profile: observe or preserve private L1 where safe,
otherwise run the Gradle baseline or abort pending publication. A policy from
another combination is never reused.

The matrix distinguishes task outcomes from exact task-to-native-cache-PUT
correlation. Both tested Kotlin rows have exact task-owned observations but
retain `UNAVAILABLE` correlation because cold build-logic stores are
unattributed; the fallback aborts the whole pending attempt.

## Machine-readable matrix

[`capability-matrix-v1.json`](./capability-matrix-v1.json) defines four
profiles and all ten Tier 1 rows. Profile reuse only deduplicates identical
status records; each combination remains explicit.

Run:

```bash
./dev/check-capability-matrix
```

The checker requires the exact Tier 1 Cartesian product, validates every
status/method/reason/fallback/evidence combination, prevents the unexecuted
JDK 25 rows from claiming a capability, and cross-checks fixture, golden-lane,
and correlation evidence. `F0-040` itself is rerun with:

```bash
./dev/check-tier1-fixtures
```

Loading the product plugin without a launcher does not claim an authenticated
handshake. Only the golden lane retains that stronger capability.

`SPK-002` leaves `JVM_AGENT` unavailable in every profile. Its real-daemon
prototype sees class loads rather than method access or task attribution, so
the shared safe fallback is `ABORT_PENDING`; no row may qualify a task from
that evidence.
