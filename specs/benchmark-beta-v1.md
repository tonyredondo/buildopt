# Private-beta benchmark v1

This specification materializes `F0-032` and refines the `OPS-001` benchmark
contract in RFC section 21.5. The machine-readable manifest is
[`benchmarks/beta-v1.yaml`](../benchmarks/beta-v1.yaml). It intentionally uses
the JSON-compatible subset of YAML 1.2 so every consumer observes identical
numbers and duplicate or unknown fields can be rejected.

## Identity and reproducibility

Each run records:

- the exact SHA-256 of `beta-v1.yaml`;
- deterministic seed `2026072901`;
- golden runner specification and immutable image platform digest;
- toolchain/component versions and their source digests;
- actual host/cgroup limits, start/end timestamps, and deviations;
- raw observations before summaries.

Results with different manifest digests are separate benchmark versions and
must not be combined. A smoke subset may validate a harness, but only the full
declared phases can support an `OPS-001/A1` result.

## Workload

The load matrix uses 1, 8, and 32 concurrent clients. Each deterministic
10,000-object cycle contains:

| Share | Object size |
|---:|---:|
| 70% | 64 KiB |
| 20% | 1 MiB |
| 8% | 10 MiB |
| 2% | 100 MiB |

The random stream selects operations and payload bytes from the declared seed;
it does not substitute compression shortcuts or sparse files. Cold and
warm-at-70%-hit phases run for every client count. Sustained load lasts 60
minutes and soak lasts eight hours. Small, medium, and large Tier 1 Gradle
builds retain known deliverables and critical-path markers; `F0-040`
materializes their repositories.

## Fault matrix

Each applicable load/fixture stratum exercises gateway restart, server
restart, mid-PUT cancellation, mid-GET cancellation, truncated blob, corrupt
blob, network latency, network loss, SQLite busy, expired lease, high
watermark, out of space, revoked policy, revoked grant, and process death
between pending and commit.

Fault injection is deterministic and records its trigger observation. A fault
passes only when the expected safety state is observed: partial/corrupt data is
a miss, invalid authority aborts pending, readiness remains false during
reconciliation, and a normal build keeps its contractual result where the
fallback permits.

## Output

Every full result exports hardware/cgroup identity, actual object distribution,
p50/p95/p99, throughput, error classes, bytes, recovery duration, readiness
transitions, fault outcomes, and deviations. Percentiles use raw eligible
observations in each stratum; error/cancel outcomes are not silently removed.

Minimum alerts cover disk/quota, corruption, stuck attempts or leases,
revocation lag, policy freshness, circuit breaker, SQLite contention, export
backlog, and acceptance latency/error budgets. Liveness never implies safe
readiness.

## Validation

Run:

```bash
./dev/check-beta-benchmark
```

The checker strictly decodes the manifest, verifies its pinned runner digest
against the repository source, requires the exact client/object/phase/fault
sets, checks all percentages and durations, and prints the benchmark SHA-256.
It validates the benchmark contract; it does not execute the 60-minute or
eight-hour profiles or create performance evidence.
