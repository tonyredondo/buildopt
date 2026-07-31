# Private-beta disk fault slice v1

This contract composes the pinned private-beta benchmark with the Shared
capacity substrate. It executes the `DISK_HIGH_WATERMARK` and
`DISK_OUT_OF_SPACE` rows rather than promoting storage unit tests or invented
fault statuses into benchmark evidence.

## Execution

`beta-benchmark disk-faults` uses two fresh reduced-capacity Shared roots and
the real loopback HTTP data plane. The output directory contains a mode-`0600`
`disk-fault-observations.jsonl` stream followed by a mode-`0600`
`disk-fault-result.json`. The summary binds the exact benchmark-manifest
digest, raw count, byte size, and SHA-256. Strict validation rejects unknown
fields, trailing JSON, changed outcomes, missing trigger sequences, and raw
tampering.

The high-watermark row commits eight 100-byte objects below the 850-byte high
mark, then publishes a ninth object. Shared removes two oldest probation
authorities before collecting their physical blobs, reaches 700 bytes below
the 750-byte low mark, returns misses for both evicted keys, and preserves a
complete hit for the new key.

The out-of-space row reports 50 available bytes before a declared 60-byte PUT.
The real HTTP client uses `Expect: 100-continue`; Shared returns `413` before
the transport reads any request-body byte and leaves zero stable, pending, or
reserved bytes. After the deterministic disk probe recovers, the same
attempt accepts the complete 60-byte PUT with `201`, then aborts and collects
the unreferenced blob.

Run:

```bash
./dev/check-beta-disk-faults
```

This exact slice closes the quota/TTL/watermark/byte-SLRU deliverable and gate,
`A1-003` and `A1-G03`. It does not qualify the remaining 13 faults, the
60-minute sustained profile, the eight-hour soak, the golden runner, or the
complete `OPS-001/A1` entry gate.
