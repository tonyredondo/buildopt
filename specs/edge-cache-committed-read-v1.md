# Edge Cache committed read-through v1

This contract closes `C2-002`. The owner-operated Edge may persist an object
only after the authenticated Shared stable route returns a complete committed
hit, and every later online or offline read rechecks exact current signed
authority before returning any byte.

## Shared committed source

Edge sends `GET /cache/{key}` with its scoped bearer token and the exact signed
authority digest. A cacheable response is one `200` with known bounded
`Content-Length`, no content encoding, canonical SHA-256 `ETag`, exact
`X-BuildOpt-Commit-State: COMMITTED`, and one canonical
`X-BuildOpt-Decision-Digest`. Redirects are never followed.

The body is streamed into a private spool, length and SHA-256 are checked over
all bytes, and the file is synced before a content-addressed immutable blob is
linked. Only then does SQLite WAL `synchronous=FULL` publish the metadata. A
miss, rejected token, malformed header, truncated/oversized/corrupt body,
cancellation, or storage failure publishes no readable entry.

## Offline read authority

Metadata binds tenant, repository, trust domain, namespace and generation,
Shared authority digest, cumulative revocation epoch/digest, L1 security
generation, decision digest, object digest/size, and local TTL. Every read
requires a `localauthority.Verified` projection that is still current and
matches all those fields exactly. Missing/expired authority, any policy or
revocation advance, TTL expiry, invalid metadata, symlinks, size drift, or
digest corruption is a byte-free miss.

The single-node state root, blob/spool directories, database, and writer lock
are private. One non-blocking process lease owns metadata mutation; restart
reopens the same verified committed entry without contacting Shared.

Run:

```bash
./dev/check-edge-cache-committed-read
```

The checker includes a real Shared pending PUT, Ed25519 commit decision,
atomic commit, scoped-token GET, Edge publication, Shared shutdown, Edge
restart, and offline read. This slice does not implement SLRU/pressure
maintenance, pending replication, an operator server, the final C2 gate, soak,
or production hardening.
