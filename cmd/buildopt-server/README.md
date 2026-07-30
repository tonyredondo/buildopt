# `buildopt-server`

Go modular monolith for the private beta.

It will host the Shared Cache, Policy API, experiment/evidence state, and
export. Internal boundaries follow versioned contracts without prematurely
splitting the private beta into microservices.

`WS-005` activates the first session-ingest boundary:

```bash
BUILDOPT_SERVER_INGEST_TOKEN=<opaque-token> \
    buildopt-server serve --listen 127.0.0.1:8042
```

The walking-skeleton server binds canonical IPv4 loopback, requires a 32-512
byte Bearer token, accepts strict JSON only at
`POST /internal/v1/build-sessions`, limits bodies to 64 KiB, and requires an
`Idempotency-Key` matching the session ID. A first accepted record returns
`202`; an identical retry returns `204`; conflicting content for an existing
ID returns `409`. Authentication failures and malformed input expose neither
the credential nor submitted content.

The in-memory ingest store deliberately retains no state after shutdown.

`WS-006` optionally activates the first local-file export:

```bash
BUILDOPT_SERVER_INGEST_TOKEN=<opaque-token> \
    buildopt-server serve \
    --listen 127.0.0.1:8042 \
    --export-dir /absolute/or/relative/export/path
```

After accepting a complete authenticated Gradle handoff, the server converts
it to the normative `BUILD_SESSION v1` producer model. Each immutable document
is formatted JSON ending in one newline, mode `0600`, and atomically linked
under a filename derived from the SHA-256 of its session ID. An identical
replay retains the existing bytes; different bytes for the same ID are a
conflict. Export failure is diagnostic and fail-open for the Gradle result.

The document contains an exact neutral-envelope duration, an explicitly
approximated launcher-observed Gradle-process interval, pre-outcome passthrough
assignment, predeclared tokenized workload identity, and explicit
`UNAVAILABLE` observations for CI timing, critical path, task outcomes, cache
causes, resources, overhead decomposition, and cost. It never invents values
or includes the ingest credential.

`A0-004`/`A0-005` optionally activate the single-node Shared storage and
publication substrate:

```bash
BUILDOPT_SERVER_INGEST_TOKEN=<opaque-token> \
    buildopt-server serve \
    --listen 127.0.0.1:8042 \
    --state-dir /absolute/private/buildopt-state
```

The server takes one process-lifetime writer lease, rejects network/clustered
filesystems, and owns private content-addressed blobs plus separately migrated
WAL-mode `cache.sqlite` and `control.sqlite`. Schema v2 adds the A0-005 durable
pending/abort lifecycle, canonical Ed25519 `CommitDecision`, atomic commit CAS,
quarantine, and startup-blocking reconciliation.
The state path must be a dedicated empty or already BuildOpt-owned root;
unrelated existing entries are rejected without adding storage files.

`A0-006` activates the cache route only from a complete private authority:

```bash
BUILDOPT_SERVER_INGEST_TOKEN=<opaque-token> \
    buildopt-server serve \
    --listen 127.0.0.1:8042 \
    --state-dir /absolute/private/buildopt-state \
    --cache-authority /run/buildopt/authority.json \
    --cache-trust-root /run/buildopt/trust-root.json \
    --cache-credential /run/buildopt/cache-credential
```

Schema v3 persists the highest signed policy/revocation generations, canonical
authority documents, and attempt bindings without persisting the raw
credential. `/cache/{key}` requires both the exact Bearer credential and
`X-BuildOpt-Authority-Digest`, rechecks current unexpired state on every
request, reads only verified committed objects, and writes only to the signed
pending attempt. Omitting all three authority flags leaves the route absent;
partial or invalid configuration prevents server startup.

JSONL, bounded delivery retry, remote TLS, cache/policy APIs, hardened
identity, and non-local workload profiles remain owned by later tracker items.

Validate the handler, concurrency, real launcher/server binaries, graceful
shutdown, credential isolation, child outcomes, fail-open delivery, and local
bypass with:

```bash
./dev/check-session-ingest
./dev/check-build-session-export
./dev/check-shared-storage
./dev/check-pending-commit
./dev/check-local-authority
```
