# Central HTTPS and scoped access POC

## Decision

BuildOpt now exposes the existing Gradle-object and typed-state storage planes
through one optional owner-operated HTTPS listener. This is a proof-of-concept
trust boundary, not a production identity platform: an owner supplies a real
certificate and manually issues short-lived opaque tokens. The local Gradle
gateway remains the only component that will receive the upstream token in the
next block; Gradle itself must never receive it.

## Transport boundary

Central routes require TLS 1.3. A listener may bind outside loopback only when
a certificate and private key are configured together. Plain HTTP remains
limited to the invocation-local `127.0.0.1` gateway and existing local test
fixtures. Clients must trust the configured certificate; disabling certificate
verification is not a supported POC path.

The server accepts the following direct configuration:

```bash
buildopt-server serve \
  --listen 0.0.0.0:8042 \
  --state-dir /var/lib/buildopt \
  --tls-cert /etc/buildopt/tls/server-chain.pem \
  --tls-key /etc/buildopt/tls/server-key.pem \
  --central-auth
```

The private key must be a bounded, regular private file. This block does not
add automatic certificate issuance, reverse-proxy configuration or a hosted
service.

## Credentials and capabilities

An owner issues one opaque token with at least 256 bits of entropy and a
maximum lifetime of 30 days:

```bash
buildopt-server central-token issue \
  --state-dir /var/lib/buildopt \
  --repository-scope-sha256 REPOSITORY_SCOPE_SHA256 \
  --tenant TEAM \
  --repository REPOSITORY \
  --trust-domain OWNER_DOMAIN \
  --namespace GRADLE_COMPATIBILITY_NAMESPACE \
  --namespace-generation 1 \
  --capabilities cache-read,state-read \
  --expires-at 2026-09-01T12:00:00Z
```

The raw token is returned only in that command's JSON output. Durable storage
contains a domain-separated SHA-256 digest, token identifier, exact scope,
capabilities and lifecycle timestamps. The independent capabilities are
`CACHE_READ`, `CACHE_WRITE`, `STATE_READ` and `STATE_WRITE`; read permission
never implies write permission. Revocation by token identifier affects the
next request without restarting the server.

## HTTP semantics

- `GET|PUT /cache/{cacheKey}` serves the opaque Gradle object plane.
- Every cache request requires `X-BuildOpt-Cache-Namespace` equal to the exact
  namespace in its token. The local gateway sets this upstream-only header;
  Gradle never receives it.
- Cache writes require `X-BuildOpt-Cache-Attempt` and an existing pending
  attempt with the exact token repository, trust domain and namespace
  generation. Uploaded bytes remain invisible until the existing commit
  protocol succeeds.
- State objects and manifests use immutable digest-addressed `GET|PUT` routes.
- `GET .../head` returns the current verified state generation.
- `POST .../head:cas` uses `If-None-Match: *` for generation one and an exact
  `If-Match` head digest for later generations.

The repository scope appears in both the token and state URL. A mismatch is a
not-found result rather than namespace disclosure. Missing credentials return
`401`, missing capabilities return `403`, stale head CAS returns `412`, changed
idempotency replays return `409`, and storage failures return `503`.

All central responses prohibit caching and MIME sniffing. Tokens are never
returned by the server, written to logs or persisted in raw form.

## Executable evidence

Run:

```bash
./dev/check-central-https-auth
```

The checker validates the machine contract and executes race-enabled token,
HTTP and real TLS integration tests. It proves a non-loopback bind with a
trusted test certificate, rejects an untrusted client, exercises capability
separation and repository isolation, publishes typed state through immutable
objects/manifests plus head CAS, binds a cache write to a pending attempt and
revokes a live credential without server restart.

## POC boundary and next block

This block proves that independent machines can be given a real encrypted and
scoped server boundary. It does not yet make `buildopt gradle` consume that
boundary. Automatic connection, gateway forwarding, state synchronization,
remote profile selection, multi-machine value measurement and production
HA/RBAC/KMS/backup design remain deferred.

The next block is `POC-CENTRAL-GRADLE-CACHE-001`: forward Gradle's existing
remote-cache protocol through the invocation-local gateway, prove one clean
producer and one read-only consumer, and retain ordinary Gradle execution when
the central service is unavailable.
