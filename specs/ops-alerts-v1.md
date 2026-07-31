# Operations alert surface v1

This contract closes the minimum local alert surface required by RFC section
21.5 without claiming the remaining `OPS-001/A1` benchmark, fault, or soak
evidence.

## Bounded endpoint

`GET|HEAD /ops/v1/alerts` is served by the same canonical IPv4 loopback
listener as liveness and readiness, including while product routes are not
ready. It returns all ten required classes in deterministic order with only
`OK` or `FIRING`, the firing timestamp, the observation timestamp, and current
readiness. It never includes tenant, repository, cache key, digest, path,
credential, policy contents, error text, or object metadata. Responses use
`Cache-Control: no-store`; mutation methods are rejected.

## Runtime signals

The server evaluates:

- filesystem availability, persistent quarantine records, expired pending
  attempts, SQLite quick/foreign-key checks, and bounded probe latency;
- duration of a signed authority change, policy expiration, and fail-closed
  route disablement;
- in-flight or failed immutable `BUILD_SESSION` export;
- server error rate and nearest-rank p95 for valid build-session acceptance
  requests over the latest bounded window.

`REVOCATION_LAG` fires only after the same authority reload has exceeded 60
seconds. `POLICY_FRESHNESS` fires when the active signed authority has at most
60 seconds left. `CIRCUIT_BREAKER` fires whenever authority change handling has
disabled the cache route and clears only after a verified authority is active.

Storage probes run outside request handling every 30 seconds with a two-second
context. Their result is aggregate state only. A probe failure fires
`SQLITE_CONTENTION`; a failed integrity check or retained quarantine record
fires `CORRUPTION`.

Run:

```bash
./dev/check-ops-alerts
```

The race-enabled fixture activates every required class, verifies the exact
read-only JSON surface, then supplies healthy signals and proves every class
recovers. A real server test also observes the alert endpoint while startup
reconciliation keeps readiness false. External paging integration is not
claimed.
