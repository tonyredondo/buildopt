# `buildopt-server`

Go modular monolith for the private beta.

It will host the Shared Cache, Policy API, experiment/evidence state, and
export. Internal boundaries follow versioned contracts without prematurely
splitting the private beta into microservices.

`WS-005` activates only the first session-ingest boundary:

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

The in-memory store deliberately retains no state after shutdown. The accepted
record is a provisional internal handoff, not the normative export.
`WS-006` owns conversion to and validation of `BUILD_SESSION v1`; later
contracts own durable SQLite state, JSONL, bounded retry/spooling, remote TLS,
cache/policy APIs, and hardened identity.

Validate the handler, concurrency, real launcher/server binaries, graceful
shutdown, credential isolation, child outcomes, fail-open delivery, and local
bypass with:

```bash
./dev/check-session-ingest
```
