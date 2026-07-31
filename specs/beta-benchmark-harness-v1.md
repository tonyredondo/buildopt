# Private-beta benchmark harness v1

This contract turns `benchmarks/beta-v1.yaml` into an executable result path.
Its first profile is deliberately `SMOKE`: it proves scheduling, the data
plane, raw evidence, summaries, and validation without shortening or
simulating the qualifying durations.

## Smoke execution

The harness runs one deterministic 100-object cycle in every phase/client
stratum: `COLD`, `WARM_70`, `SUSTAINED`, and `SOAK`, each with 1, 8, and 32
workers. It retains the exact 70/20/8/2 mix while scaling object sizes to 4
KiB, 64 KiB, 512 KiB, and 1 MiB.

Cold operations use real loopback HTTP PUTs into real Shared pending attempts.
An ephemeral benchmark-only Ed25519 key signs exact complete decisions before
atomic commit. Later phases issue 70% complete verified hits and 30% byte-free
misses through the real Shared HTTP handler. Payloads come from a fixed
xorshift64* byte stream keyed by the manifest seed and object identity; they
are neither sparse nor compression shortcuts.

## Evidence and validation

`observations.jsonl` records all 1,200 operation outcomes before the summary.
`result.json` binds the exact benchmark digest, host/cgroup identity,
components, actual distribution, nearest-rank p50/p95/p99, throughput, errors,
bytes, recovery/readiness/fault fields, deviations, and the raw file's
SHA-256/count/size. Both files are mode `0600` below an empty mode `0700`
directory.

The validator rejects unknown result/observation fields, manifest drift,
missing strata, wrong mix/hit/status/byte facts, invented fault outcomes,
trailing JSON, raw count/size/digest drift, and tampering.

Run:

```bash
./dev/check-beta-benchmark-harness
```

This smoke profile does not run the declared wall-clock durations, Gradle
fixtures, or fault matrix and cannot close `OPS-001/A1`.
