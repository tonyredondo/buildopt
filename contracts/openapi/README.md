# OpenAPI contracts

OpenAPI 3.1/JSON contracts for BuildOpt control, internal cache control, and
Test Optimization integration.

## BuildOpt control APIs

[`buildopt-control.v1.yaml`](./buildopt-control.v1.yaml) and
[`buildopt-cache-control.v1.yaml`](./buildopt-cache-control.v1.yaml) are the
normative `F0-017` documents. Both use the JSON Schema 2020-12 dialect,
deployment-scoped HTTP bearer authentication over TLS, JSON request/response
bodies, required contract-version headers, bounded deadlines, cancellation
semantics, stable error codes, explicit retryability/backoff, and defined
unknown-response recovery.

The control API owns:

- immutable policy resolution before Gradle starts;
- append-only, `If-Match`-preconditioned attempt transitions;
- isolated `FULL_RELEVANT_VALIDATION` submission and status reads.

The internal cache-control API owns:

- opening and reading bounded pending attempts;
- atomic commit with the exact authenticated `CommitDecision`;
- explicit abort and cumulative revocation/L1-generation reads.

Every mutation requires an `Idempotency-Key`. Exact replays return the same
result and reuse with different content is `IDEMPOTENCY_CONFLICT`. Stateful
mutations additionally require `If-Match`; an unknown commit/abort response is
resolved by reading attempt state before retry. Deadlines never infer a
positive policy, validation, or cache verdict.

Opaque Gradle `HttpBuildCache` GET/PUT bytes are deliberately absent. That
public protocol remains a separate data-plane boundary, while attempt,
revocation, provenance, and commit metadata use this internal API.

Validate both documents and exercise every operation against the in-process
request/response-validating mock with:

```bash
./dev/check-buildopt-openapi
```

The checker uses the isolated, checksum-authenticated Go module and
`kin-openapi` validator. Mock requests and responses are validated against the
loaded documents and the already-materialized policy, attempt, validation, and
commit JSON Schemas; it also proves exact replay and conflicting-payload
behavior.

## Test Optimization API

[`test-optimization.v1.yaml`](./test-optimization.v1.yaml) is the normative
`F0-018` producer/consumer boundary. It resolves a signed grant before Gradle
configuration, rechecks signed cumulative grant status before commit, submits
one idempotent `FULL_RELEVANT_VALIDATION`, and polls delayed results within the
original deadline. Artifact references are content-addressed and may use only a
customer-owned channel or an authorized ephemeral HTTPS locator.

Missing, expired, revoked, or incompatible grants and incomplete validation
remain fail-closed for the optimization while preserving the baseline build.
Every mutation binds the request ID, and validation additionally binds the
action ID. Exact replays return the same operation; another payload is an
`IDEMPOTENCY_CONFLICT`.

Validate the document and exercise all four operations against its
request/response-validating mock with:

```bash
./dev/check-test-optimization-openapi
```

`F0-020..022` materialize the shared cryptographic and failure vectors,
generated single-attempt Go/Java transport clients, and N/N-1 compatibility.
The generated clients deliberately leave retry and semantic response
validation visible to their callers. Full cross-product missing, expired,
revoked, delayed, and corrupt-artifact fixtures remain with `F0-033`. Queue
execution and the atomic SQLite commit/recovery implementation remain with
`CI-ORCH-001`/`CACHE-008`; these API documents do not close either wider gate.
