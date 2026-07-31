# Private-beta sustained load slice v1

This contract materializes the 60-minute `SUSTAINED` phase from
`benchmarks/beta-v1.yaml` without allowing a short CI run to become
qualification evidence.

## Production execution

The qualifying command runs only inside the pinned
`linux-amd64-4c-16g-v1` cgroup. It divides the manifest's 3,600-second phase
equally across the 1-, 8-, and 32-client strata, so each stratum runs for at
least 1,200 seconds and the complete phase lasts at least one hour.

Setup publishes 100 exact-size, deterministic, non-sparse objects through real
pending HTTP PUTs and one Ed25519-authorized atomic commit. Each stratum then
executes a deterministic 10,000-operation logical cycle: the exact 70/20/8/2
size distribution is repeated 100 times with 70% complete hits. Requests pass
through a real managed `buildopt` gateway to authenticated single-node Shared
storage. Batches never exceed either the declared client count or the
gateway's 200 MiB process-wide verified-spool bound.

The signed authority used by this one-hour slice has an explicit two-hour
validity horizon, covering object publication, all three strata, and result
sealing without refreshing identity or policy during the measurement.

The result contains 30,000 mode-`0600` raw observations plus a mode-`0600`
summary bound to the manifest and raw SHA-256. The validator reconstructs
every expected operation, client stratum, size, hit/miss result, byte count,
percentile, and throughput calculation. It requires zero errors and checks
the RFC p95 targets using their distinct measurement boundaries:
request-to-response-headers is at most 50 ms for misses and, after the gateway
has fully spooled and verified a hit, at most 150 ms through 1 MiB, 400 ms
through 10 MiB, and 2.5 seconds through 100 MiB. Response-header-to-complete
body materialization is checked separately against
`150 ms + payloadBytes / 200 MiB/s`. The raw row retains both header-ready and
complete-body durations.

## CI trial

The default checker uses the same real gateway, storage, scheduler, strict
decoder, and tamper checks with 300 scaled operations. Its result is
unambiguously `TRIAL`, records the runner as unverified, and is rejected by
the production validator.

Run the short proof:

```bash
./dev/check-beta-sustained
```

Run the one-hour pinned-container qualification:

```bash
./dev/check-beta-sustained --qualify
```

A green qualifying run closes only the 60-minute sustained slice. The
eight-hour soak, small/medium/large Gradle fixture preservation, `A1-G02`, and
the complete `OPS-001/A1` gate remain separate.
