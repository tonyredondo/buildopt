# Private-beta eight-hour soak slice v1

This contract materializes the eight-hour `SOAK` phase from
`benchmarks/beta-v1.yaml` without treating the short CI trial or the separate
Gradle/circuit-breaker matrix as qualification evidence.

## Production execution

The qualifying command runs only inside the pinned
`linux-amd64-4c-16g-v1` cgroup. It divides the manifest's 28,800-second phase
equally across the 1-, 8-, and 32-client strata, so each stratum runs for at
least 9,600 seconds and the complete phase lasts at least eight hours.

One long-lived managed `buildopt` gateway, authenticated single-node Shared
store, and signed authority serve the complete phase. Setup publishes 100
exact-size deterministic objects through real pending HTTP PUTs and one
Ed25519-authorized atomic commit. Each stratum executes the deterministic
10,000-operation cycle with the exact 70/20/8/2 size distribution and 70%
complete hits. Batches remain within the declared concurrency and the
gateway's 200 MiB verified-spool bound.

The soak authority is valid for ten hours so setup, all three 160-minute
strata, validation, and cleanup use one immutable identity without refresh.
The result binds 30,000 mode-`0600` raw observations to the manifest, raw count,
size, and SHA-256. Strict validation reconstructs every operation and requires
zero errors, at least eight hours of measured throughput duration, the exact
golden runner, and the same miss, verified-hit-ready, and downstream
materialization p95 targets as the sustained slice.

## CI trial

The default checker runs 300 scaled operations through the same real gateway,
storage, scheduler, decoder, and tamper checks. Its result is explicitly
`SOAK_TRIAL`, records an unverified runner, and the production validator must
reject it.

Run the short proof:

```bash
./dev/check-beta-soak
```

Run the eight-hour pinned-container qualification:

```bash
./dev/check-beta-soak --qualify
```

A green qualification closes only the exact eight-hour soak slice. The
small/medium/large Gradle fixture matrix, flood/large-object/full-disk circuit
breaker preservation, `A1-G02`, and complete `OPS-001/A1` gate remain
separate.
