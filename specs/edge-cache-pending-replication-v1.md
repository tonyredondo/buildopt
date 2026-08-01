# Edge Cache pending replication v1

This contract closes `C2-004`. Edge may durably buffer a trusted writer's
opaque object only inside its exact signed attempt, then replicate it
asynchronously into the corresponding Shared pending attempt. Neither local
durability nor a successful upstream PUT creates committed authority.

## Attempt-private admission

`WriteAuthority` can be projected only from a current verified authority that
grants `TRUSTED_CI_ONLY` writes, names an attempt, and has an unexpired attempt
lease. Its lifetime is the earlier of the authority and lease expirations.

Before reading bytes, Edge reserves the exact declared size against committed
plus pending logical bytes and all outstanding reservations. It verifies the
complete length and SHA-256 before linking an immutable content-addressed blob
and publishing metadata. Exact retry of the same attempt/key/bytes is
idempotent; different bytes for that attempt/key are rejected.

Pending bytes can be reopened only with the exact still-current
`WriteAuthority`. A different attempt, rejected replication, expiration,
metadata drift, or blob corruption is a byte-free miss. The committed read path
does not consult pending metadata.

## Durable asynchronous replication

Schema v3 adds durable `QUEUED`, `REPLICATING`, `REPLICATED`, and
`REJECTED` pending states. A bounded worker claims at most 64 due entries per
pass, verifies the local blob again, and sends an authenticated redirect-free
`PUT /cache/{key}` with exact `Content-Length` and signed authority digest.
Only `200` or `201` with the exact blob-digest acknowledgement marks the
entry replicated.

Unavailable or malformed upstream responses return the entry to `QUEUED`
with durable exponential retry from one second up to five minutes. Authority,
attempt, conflict, and capacity rejections become `REJECTED`. A process
restart conservatively recovers every interrupted `REPLICATING` claim as
immediately due `QUEUED` work. Pending TTL removes metadata before physical
orphan cleanup, and blobs shared with committed or other pending references are
preserved.

Run:

```bash
./dev/check-edge-cache-pending-replication
```

The checker includes a real Shared attempt and read-write beta token. It proves
that locally durable bytes are invisible to Shared before replication, arrive
as one Shared pending object afterward, remain absent from both stable paths,
and survive Edge restart as attempt-private replicated state. This slice does
not implement the loopback operator proxy, the two-node collision proof, the
final C2 gate, soak, or production hardening.
