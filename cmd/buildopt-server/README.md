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

`A0-008` adds a bounded private `buildopt-events.jsonl` stream. Every completed
session appends deterministic observed and published events with per-build
sequences `1` and `2`; identical delivery remains byte-identical and changed
identity or sequence reuse is a conflict. The stream is capped at 64 MiB with
1 MiB lines. Startup repairs only an unterminated final line and otherwise
fails closed on malformed durable content.

If sequence 1 survives without its complete document or sequence 2, startup
publishes an immutable schema-valid `complete:false` recovery with the exact
missing range. If the complete document survived, startup verifies it and
replays the deterministic publication event instead. Existing complete and
partial files are never overwritten.

`UX-F1-001` optionally exposes the immutable redacted exports to a local user
interface through a separate read credential:

```bash
BUILDOPT_SERVER_INGEST_TOKEN=<write-only-ingest-token> \
BUILDOPT_HISTORY_API_TOKEN=<independent-read-token> \
    buildopt-server serve \
        --listen 127.0.0.1:8042 \
        --export-dir /private/buildopt-exports
```

The existing canonical IPv4 loopback listener then serves:

```text
GET /api/v1/build-sessions?repository=TOKEN&outcome=SUCCESS&limit=25&cursor=OPAQUE
GET /api/v1/build-session?id=SESSION_ID
```

Both routes authenticate before routing, emit `Cache-Control: no-store`, read
only mode-`0600` filename-bound `BUILD_SESSION v1` documents from the private
mode-`0700` export root, and return only the identities already HMAC-redacted
by the exporter. The collection is newest-first and capped at 100 items per
page; the opaque cursor binds the last completed timestamp and session ID.
Unsafe or malformed history fails closed without exposing paths. Omitting the
history token leaves both routes absent, and the ingest token is not accepted
for reads.

Copy the validated stream directly to stdout for a CI artifact:

```bash
buildopt-server export \
    --export-dir /private/buildopt-exports \
    --format jsonl
```

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

`A1-002` adds schema v5 and the manual scoped-token registry. Provision a
token while the server remains active (SQLite WAL serializes the update):

```bash
buildopt-server token issue \
    --state-dir /absolute/private/buildopt-state \
    --tenant tenant-7 \
    --repository owner/repository \
    --trust-domain private-beta \
    --namespace stable \
    --namespace-generation 12 \
    --plane stable \
    --access read-write \
    --expires-at <RFC3339-within-30-days>
```

The JSON response exposes the opaque token once; only its domain-separated
SHA-256 digest, exact scope, expiry, and revocation state persist. Enable the
stable data-plane authenticator with `serve --cache-token-auth` plus the normal
authority flags. Revoke by the non-secret ID:

```bash
buildopt-server token revoke \
    --state-dir /absolute/private/buildopt-state \
    --token-id <32-lowercase-hex-id>
```

Each request consults `control.sqlite`, so revocation is effective before the
next request/build. Stable, quarantine, and control tokens are not
interchangeable; a read token cannot `PUT`.

`A1-004` makes SUMMARY the default export profile and HMAC-tokenizes
repository, trust-domain, and task identities before JSON or JSONL reaches
disk. TASKS/EVIDENCE require `--authorize-expanded-export`; DIAGNOSTIC also
requires `--diagnostic-until <UTC-RFC3339>` within seven days. The same
authorization must be supplied when copying a wider existing stream.

Delete every managed copy beneath one marked isolated deployment root with:

```bash
buildopt-server data delete \
    --data-root /absolute/private/deployment-data \
    --deletion-id delete-001 \
    --tenant tenant-7 \
    --repository owner/repository \
    --trust-domain private-beta \
    --next-namespace-generation 13 \
    --next-l1-security-generation 51 \
    --token-key /absolute/private/deletion-hmac.key \
    --token-key-version deletion-key-v1
```

The command refuses active Shared/export/L1 leases, writes logical revocation
before removing known managed components, emits only a tokenized tombstone,
and makes exact retries idempotent. Customer-controlled copies can be listed
with repeated `--external-destination` flags; they retain their own
tombstone-consumption obligation.

Encrypted delivery retry/DLQ, deployed TLS termination and sinks, cache/policy
APIs, hardened identity, and non-local profiles remain owned by later items.

Validate the handler, concurrency, real launcher/server binaries, graceful
shutdown, credential isolation, child outcomes, fail-open delivery, and local
bypass with:

```bash
./dev/check-session-ingest
./dev/check-build-session-export
./dev/check-shared-storage
./dev/check-pending-commit
./dev/check-local-authority
./dev/check-private-beta-data-lifecycle
```

`DEPLOY-001` additionally packages the real server, launcher, and Gradle plugin
into two signed versions, starts each selected deployment against the same
Shared/export roots, and proves upgrade, rollback, preserve/reinstall, and
explicit purge with:

```bash
./dev/check-deployment-lifecycle
```

`A2-002` composes that signed installation with the strict self-hosted
configuration, external mode-`0600` secret references, persistent private
state/export roots, and a deterministic systemd unit:

```bash
./dev/check-self-hosted-service-install
```

The installer never enables or starts the unit. Follow
[`runbooks/self-hosted-single-node.md`](../../runbooks/self-hosted-single-node.md)
for explicit host activation and readiness admission.

`A2-003` adds serialized signed upgrade and explicit restart validation without
changing configuration or persistent data:

```bash
./dev/check-self-hosted-upgrade-restart
```

The old process keeps serving its already-open immutable release while v2 is
selected for the next restart. Descriptor-composition failure rolls selection
back to v1, and pending bytes remain invisible before, during, and after a
successful v2 restart.

`A2-004` exposes a local inspection command that returns only generation and
scope metadata after verifying a current private authority against its pinned
trust root and credential:

```bash
buildopt-server authority inspect \
  --authority /etc/buildopt/authority.json \
  --trust-root /etc/buildopt/trust-root.json \
  --credential /etc/buildopt/cache-credential
```

`dev/manage-self-hosted restore` composes that proof with an offline private
snapshot, strict generation rotation, absent-target publication, and an
explicit post-restore server start. See the self-hosted runbook; no credential
or signing material appears in the inspection output.

`OPS-001/A1` separates process health from safe serving:

- `GET|HEAD /livez` returns `200` while the HTTP process is responsive;
- `GET|HEAD /readyz` returns `503` until Shared reconciliation and authority
  loading complete, then `200`;
- product routes return `503` whenever readiness is false;
- a changed signed local authority is revalidated every second, with old or
  invalid authority disabling cache routing and readiness.

Validate the live-before-ready lifecycle and measured revocation propagation:

```bash
./dev/check-ops-readiness
```

The same loopback listener exposes `GET|HEAD /ops/v1/alerts`. Its bounded JSON
contains every required RFC alert class, current `OK|FIRING` state, firing
time, observation time, and readiness without including tenant, repository,
cache-key, digest, path, credential, policy, or error details. Storage is
sampled every 30 seconds; authority, export, and valid session-acceptance
signals are updated on their runtime paths.

Validate exact alert activation, recovery, and the read-only endpoint:

```bash
./dev/check-ops-alerts
```
