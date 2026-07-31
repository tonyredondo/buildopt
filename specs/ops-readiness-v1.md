# Operations readiness and revocation v1

This contract is the first executable slice of `OPS-001/A1`. It separates
process liveness from safe cache readiness and reloads the externally signed
local authority without restarting `buildopt-server`.

## Startup and shutdown

The loopback listener starts with liveness `200`, readiness `503`, and every
product route disabled. Shared opens both SQLite stores, verifies blobs and
metadata, expires leases, repairs the audit index, removes orphans, and loads
the pinned local authority before the application handler is activated.
Readiness then becomes `200`.

Shutdown changes readiness back to `503` before draining requests. Liveness
only proves that the HTTP process can answer; it never means the cache is safe.
Both endpoints allow only `GET` and `HEAD`, emit no body, and disable caching.

## Online authority reload

When authenticated cache routing is configured, the server fingerprints the
authority document, trust root, and credential files every second. A change
first disables readiness and the cache route, then revalidates all three files,
persists monotonic policy/revocation state, and installs a new immutable
handler. Invalid, expired, unsafe, or rolled-back state leaves readiness false.

The fixture atomically publishes a correctly signed higher revocation epoch,
measures propagation below the 60-second contract, proves the old authority
returns `401`, and proves the new read-only authority reaches a safe `404`
miss. The Launcher independently reloads the same signed state before each new
invocation; no raw credential is logged or persisted by Shared.

Run:

```bash
./dev/check-ops-readiness
```

This slice does not invent benchmark evidence. The full load/fault profile,
eight-hour soak, and alert delivery remain open in `OPS-001/A1`.
