# Edge Cache two-node proxy v1

This contract closes `C2-005`. It exposes the C2 storage and replication
boundaries through a Gradle-compatible HTTP proxy that can run only on an
explicit IPv4 loopback listener, then proves central collision authority with
two independent Edge state roots.

## Loopback route

The proxy exposes only `GET|PUT /cache/{canonical-key}`. Unsupported methods,
paths, non-loopback listeners, unknown-length writes, oversized writes, and
write attempts in read-only mode fail before object bytes are accepted.

PUT verifies and durably enqueues attempt-private bytes before returning
`201`; exact replay returns `200`, and different bytes for the same local
attempt/key return `409`. Network replication remains asynchronous.

GET always attempts the committed path first. Only a Shared miss or temporary
unavailability may fall back to the exact local write attempt; Shared authority
rejection never falls back. Responses identify `COMMITTED` versus
`PENDING_ATTEMPT`, and pending state never crosses into the committed index.
The authority supplied to the proxy must be a current verified projection and
is rechecked against its expiration on every operation.

## Owner-controlled two-node proof

Two real loopback listeners use independent private Edge roots and the same
signed attempt. Each accepts different opaque bytes for the same key while
Shared is untouched. With Shared unavailable, each returns only its own
attempt-private candidate. On replication, Shared accepts the first bytes and
rejects the second collision; neither Edge resolves it.

A canonical Ed25519 decision then commits the accepted object centrally. Both
proxies fetch and completely verify the Shared winner, never expose the losing
bytes as stable, and continue returning the identical committed winner after
Shared becomes unavailable.

Run:

```bash
./dev/check-edge-cache-two-node-proxy
```

This owner-controlled POC proof does not run the deferred eight-hour soak,
exercise external design partners, install an operating-system service, hot
reload authority documents, or claim production hardening.
