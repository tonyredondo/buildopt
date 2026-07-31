# Private-beta system fault slice v1

This contract executes the final six pinned private-beta fault rows against
real `buildopt` managed-gateway and `buildopt-server` processes. It covers
gateway and server restart, upstream latency and connection loss, and signed
policy and grant revocation. Existing process, readiness, and revocation unit
tests are supporting evidence, not substitutes for benchmark-bound execution.

## Execution

`beta-benchmark system-faults` receives absolute paths to the two built
executables and creates isolated private state per fault. The output directory
contains a mode-`0600` `system-fault-observations.jsonl` stream followed by a
mode-`0600` `system-fault-result.json`. The summary binds the exact benchmark
manifest digest, raw count, byte size, and SHA-256. Strict validation rejects
unknown fields, trailing JSON, changed outcomes, missing trigger sequences,
and raw tampering.

The gateway row serves a complete verified object through the real managed
gateway, stops its owning invocation, and starts a second invocation in the
same slot. Endpoint, credential, and generation remain stable and the second
GET is the same complete hit. The server row stops a ready real server,
introduces 256 valid-shaped orphan blobs, restarts it, and observes liveness
`200` while readiness and a product route remain `503`. Readiness becomes
`200` only after reconciliation deletes every orphan.

The network rows send requests through the real managed gateway to a loopback
fault controller. A 250 ms upstream delay crosses a 100 ms caller deadline
without returning bytes or hiding the error, then recovers to a complete hit.
A dropped upstream connection becomes a byte-free fail-open `404`, followed by
a complete hit after connectivity returns.

Each revocation row creates a real pending object and verifies its old-epoch
commit decision. A signed local-authority update advances policy, revocation,
L1, gateway, and namespace generations; the grant case also replaces the
signed grant digest. The old authenticated route returns a byte-free `401`,
the late commit is rejected, the attempt becomes `ABORTED`, pending bytes reach
zero, and the rotated namespace contains no hit.

Run:

```bash
./dev/check-beta-system-faults
```

Together with the disk and Shared slices, this qualifies all 15 pinned fault
rows and closes the fail-closed restart/recovery and rotation gate `A1-G04`.
It does not qualify the 60-minute sustained profile, eight-hour soak, golden
runner result, `A1-G02`, or the complete `OPS-001/A1` operational gate.
