# Private-beta Shared fault slice v1

This contract executes seven pinned private-beta fault rows against the real
single-node Shared data plane. It covers cancellation, blob integrity,
SQLite write contention, pending-lease expiry, and process death between
pending publication and authorized commit. Existing storage unit tests are
supporting evidence, not substitutes for the benchmark-bound execution.

## Execution

`beta-benchmark shared-faults` creates a fresh Shared root per fault and sends
all data-plane traffic through real loopback HTTP. The output directory
contains a mode-`0600` `shared-fault-observations.jsonl` stream followed by a
mode-`0600` `shared-fault-result.json`. The summary binds the exact benchmark
manifest digest, raw count, byte size, and SHA-256. Strict validation rejects
unknown fields, trailing JSON, changed outcomes, missing trigger sequences,
and raw tampering.

The PUT cancellation row starts a declared 1,000-byte streaming request,
observes a live spool after 100 bytes, cancels the request, and proves that no
pending, stable, or reserved authority survives. A complete retry returns
`201`. The GET cancellation row reads 4,096 bytes of a verified 1 MiB object
without accepting a hit, then proves a complete digest-matching retry.

The integrity rows truncate a 1,000-byte blob to 999 bytes or corrupt it
without changing its length. Each real GET returns a byte-free `404`, removes
stable authority, and quarantines the physical blob. The SQLite row holds an
external `BEGIN IMMEDIATE` write lock, observes `SQLITE_BUSY` with zero
committed objects, releases the lock, and proves that an exact retry publishes
one complete hit.

The lease row waits past a real 250 ms pending expiry, reconciles the attempt
to `ABORTED`, releases its owner, and collects its orphan. The process-death
row closes and reopens Shared with a pending object, proves it remains a miss,
then applies the already verified authorized decision and serves the complete
100-byte hit.

Run:

```bash
./dev/check-beta-shared-faults
```

This slice qualifies seven additional manifest faults. It does not qualify the
remaining gateway/server restart, network latency/loss, or policy/grant
revocation rows; the 60-minute sustained profile, eight-hour soak, golden
runner, `A1-G02`, `A1-G04`, and complete `OPS-001/A1` gate also remain open.
