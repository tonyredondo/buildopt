# Test Optimization integration v1

This specification materializes `F0-033`, `INT-001`, and the Phase 0
conformance required by `TESTOPT-API-001`. It composes the existing OpenAPI,
signed grant/result schemas, common cryptographic vectors, HTTP failure
semantics, and N/N-1 policy without giving either product database access to
the other.

## Ownership

Test Optimization owns test selection, ordering, sharding, retries, reports,
and eligibility of `Test` task cache entries. Build Optimization owns build
work and may only consume a signed decision.

Before Gradle configures any root, `buildSrc`, composite, included, or plugin
build test task, Build Optimization resolves a current grant. Missing,
expired, untrusted, incompatible, or timed-out authority applies
`doNotCacheIf` to every registered test type. If the adapter cannot apply that
rule before all participating builds configure, Build Cache is disabled for
the invocation. The normal baseline still runs.

Before committing grant-dependent pending entries, the consumer reads signed
status again. A revoked grant, lower/invalid epoch, changed digest, expiration,
or unavailable status aborts all corresponding pending work.

## Validation

A patch that may alter test inputs submits exactly one idempotent
`FULL_RELEVANT_VALIDATION` request. Test Optimization alone resolves the
relevant tests. `202` is polled with bounded backoff and jitter under the
original deadline; no webhook or inbound callback exists.

Candidate and control artifacts are content-addressed and bound to the same
repository, revision, source state, and `actionId`. Only a customer-owned
channel or authorized ephemeral HTTPS object is legal. The producer verifies
size and SHA-256 before testing; a path supplied by the caller is never opened.

Only a trusted, unexpired, schema-valid signed `PASSED` result over the exact
artifact set permits the action to proceed. `FAILED`, `INCONCLUSIVE`, timeout,
corruption, incompatible version, or signature failure blocks the action and
cannot change the authoritative baseline result.

## Compatibility and retry

Both products support the current and previous minor within major version 1.
An incompatible major or a gap larger than one minor returns
`CONTRACT_INCOMPATIBLE` and preserves the baseline. Exact retries retain
`requestId`, `actionId`, payload, and deadline; changed payload reuse returns
`IDEMPOTENCY_CONFLICT`.

## Shared fixtures

[`test-optimization-integration-v1.json`](./test-optimization-integration-v1.json)
defines 16 producer/consumer scenarios and binds exact synthetic artifact
bytes under
[`fixtures/test-optimization`](../fixtures/test-optimization/README.md).
Both roles consume the same expected producer response and consumer safety
outcome.

Run:

```bash
./dev/check-test-optimization-integration
```

The checker verifies the artifact digest, interprets every scenario, and then
runs the existing OpenAPI, signed-schema, cryptographic, failure-semantics, and
client compatibility checkers.
