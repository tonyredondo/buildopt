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
Other target rows remain `UNAVAILABLE` until `F0-040`/module-specific
conformance runs them; a target label is not evidence.

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

[`capability-matrix-v1.json`](./capability-matrix-v1.json) defines three
profiles and all ten Tier 1 rows. Profile reuse only deduplicates identical
status records; each combination remains explicit.

Run:

```bash
./dev/check-capability-matrix
```

The checker requires the exact Tier 1 Cartesian product, validates every
status/method/reason/fallback/evidence combination, prevents untested rows from
claiming a capability, and cross-checks the two tested Gradle versions against
the correlation evidence.
