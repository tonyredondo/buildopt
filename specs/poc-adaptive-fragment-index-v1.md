# Adaptive fragment compatibility index v1

Status: accepted POC lookup contract for `AF-003`.

Machine policy: [`poc-adaptive-fragment-index-v1.json`](./poc-adaptive-fragment-index-v1.json).

## Purpose

The compatibility index answers one deliberately cheap question before Gradle
starts: which persisted fragment revisions still match the current repository
facts? It does not decide whether a fragment is valuable, conflict-free or
authorized to execute. Those responsibilities remain with later shadow,
economic and planning blocks.

The index is a repository-local derived cache over the immutable fragment
generations from `AF-002`. It is not authority, may be deleted and rebuilt, and
never repairs corrupt or incompatible state.

## Fingerprints and cross-commit reuse

Each snapshot carries:

- the Git revision as provenance;
- the Gradle Wrapper;
- requested workflow;
- producer lineage;
- required output contract; and
- current change family.

Optional task, platform, network, cache-namespace and patch-base bindings are
included only when a fragment declares them. Git revision is intentionally not
a compatibility binding. A descendant commit can therefore reuse a fragment
when every semantic fact it actually consumes remains equal.

The lookup compares only declared bindings. Unrelated drift cannot invalidate
a fragment. Declared drift returns `SUSPENDED`; missing, ambiguous, expired,
cross-repository or corrupt state returns `NATIVE_RETAINED`. An exact match
returns `COMPATIBLE`, which means candidate only—not activation.

## Bounded evidence

The checked-in [local report](../benchmarks/results/adaptive-fragment-lookup-v1-local.json)
runs the same six decisions on the frozen Spring Framework, OpenTelemetry Java
Instrumentation, Apache Kafka, Micronaut Core and Apache Groovy subjects. The
repository names select evidence rows only; no product branch depends on them.

The 12-CPU Linux run produced 30 decisions:

| Disposition | Count |
|---|---:|
| `COMPATIBLE` | 10 |
| `SUSPENDED` | 5 |
| `NATIVE_RETAINED` | 15 |

Median lookup was **0.025 ms**, p95 was **0.039 ms**, and maximum was
**0.061 ms**, well below the 500/1,000-ms gate. The decision loop performed
zero Gradle starts, remote calls, output materializations and lifecycle
mutations. This is a lookup-overhead result, not a build-time improvement.

Run:

```bash
./dev/check-adaptive-fragment-index
```

The accepted outcome is `FAST_FRAGMENT_LOOKUP_AVAILABLE`. Central state,
online learning, economic selection, fragment composition, installed replay,
production rollout, soak work, design-partner work and Test Optimization remain
outside this block.
