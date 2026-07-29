# OpenAPI contracts

OpenAPI 3.1/JSON contracts for BuildOpt control, internal cache control, and Test Optimization integration.

Each API must define authentication boundaries, idempotency keys, state preconditions, deadlines, cancellation, stable error codes, retryability, maximum backoff, and unknown-response behavior. Public Gradle `HttpBuildCache` payloads remain opaque and separate from internal attempt, provenance, revocation, and commit APIs.

`F0-017` and `F0-018` own the API documents; `F0-010` creates their namespace only.
