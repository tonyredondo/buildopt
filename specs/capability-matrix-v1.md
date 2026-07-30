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
status records; each combination remains explicit. `A0-007` adds
`GRADLE_BOOTSTRAP_CACHE` without promoting unrelated Shared-cache gates.

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

`A0-002` adds exact default-deny policy evidence to all eight JDK 17/21 rows:
core source-set `JavaCompile` is the only task allowlist entry, while `Test`,
custom tasks, modified actions, and transforms fail closed. `A0-003` adds a
native managed L1 on the same rows, including generation rotation,
Configuration Cache reuse, and L2-writer local disablement. `A0-004` adds the
private local blob store and separate WAL-mode cache/control metadata
lifecycles. `A0-005` adds pending/abort, canonical commit CAS, verified HTTP
reads, and startup reconciliation. `A0-006` then proves locally authenticated
policy, cumulative revocation state, routable gateway/server context, and a
golden-row Gradle `HttpBuildCache` PUT/GET. The matrix still reports
`MANAGED_SHARED_CACHE` as `UNAVAILABLE` until the complete A0 HTTP
compatibility/fault matrix and production commit/abort composition close; one
golden integration row does not promote every compatibility row.

`A0-007` makes `GRADLE_BOOTSTRAP_CACHE` exact for all eight JDK 17/21 rows.
Each real Wrapper row consumes an offline read-only dependency snapshot,
retains an independently checksum-verified distribution in a runner-private
writable home, and reuses it after the source archive is removed. The
unexecuted JDK 25 rows remain `UNAVAILABLE` and preserve the Gradle baseline.

`SPK-002` leaves `JVM_AGENT` unavailable in every profile. Its real-daemon
prototype sees class loads rather than method access or task attribution, so
the shared safe fallback is `ABORT_PENDING`; no row may qualify a task from
that evidence.

`SPK-003` leaves `HERMETIC_PRODUCER` unavailable in every profile. The real
Linux probe can create the required namespaces, but clock, randomness,
environment, and kernel-policy gaps prevent complete task-producer coverage.
The helper therefore discards the candidate, aborts pending publication, and
retains the Gradle baseline.

`SPK-004` makes `PATCH_BUNDLE_APPLIER` exact in every executed JDK 17/21
profile because its signed Java/Git boundary is independent of the Gradle/DSL
observation row. The two recipe fixtures pass strict parse, JCS/Ed25519, path
graph, pre/postimage, idempotency, rollback, and recovery checks on both real
runtimes. The unexecuted JDK 25 profile remains `UNAVAILABLE`; every
application failure retains the download-only fallback.
