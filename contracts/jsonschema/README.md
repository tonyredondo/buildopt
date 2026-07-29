# JSON Schema contracts

Versioned JSON Schema 2020-12 contracts for exportable records and signed commands.

## BUILD_SESSION v1

[`build-session.v1.schema.json`](./build-session.v1.schema.json) is the normative `F0-011` contract. Its canonical identifier is `https://schemas.buildopt.dev/build-session.v1.schema.json`, and its initial `schemaVersion` is `1.0`.

- A compatible addition creates a new minor schema and only adds optional fields or enum-independent definitions. Removing, renaming, requiring, or reinterpreting a field requires a new major schema. The N/N-1 compatibility suite and generated clients remain owned by `F0-022`.
- Every object rejects unknown fields. Optional fields are omitted rather than encoded as `null`, except for process exit codes that may be unknown after an infrastructure failure or cancellation.
- Durations use explicit `{state, unit, method}` observations. `COMPLETE` and `PARTIAL` include a non-negative `valueMs`; `PARTIAL` includes a reason. `UNAVAILABLE` includes a reason and cannot include an invented value.
- `build.startedAt` and `build.completedAt` are the UTC timestamps at the two neutral-envelope boundaries named in `measurementMetadata`; duration values come from the declared monotonic clock.
- Complete records reject recovery metadata. Partial records require non-empty missing sequence ranges. Future aggregate causal effects are rejected because they belong to `EXPERIMENT_RESULT`.
- Exported source-state and work-unit fingerprints use keyed HMAC values and carry a token-key version and trust domain.

JSON Schema cannot compare two timestamps, order interval endpoints, add decomposition components, or compare the first and last values of a missing sequence range. Producers must preserve those RFC invariants; the later `METRICS-001` catalog and compatibility suite will add cross-field conformance without weakening this schema.

Run the pinned Draft 2020-12 conformance test with:

```bash
./dev/check-build-session-schema
```

The test uses the repository-local Go toolchain, asserts date-time formats, validates every fixture under [`testdata/build-session.v1`](./testdata/build-session.v1), and rejects each negative fixture for its documented reason. The validator dependency is isolated and versioned in `dev/schema-validator/go.mod` and authenticated by its `go.sum`; no workstation-global JSON Schema executable is used, and the product module remains dependency-free.

Run the real `WS-006` producer and validate its successful and failed Gradle
documents with the same isolated compiler:

```bash
./dev/check-build-session-export
```

The product-side model remains dependency-free; the checker builds the
isolated validator as a separate executable and injects it into the pinned
JDK-only golden container.

## EXPERIMENT_RESULT and ACTION_RECORD v1

[`experiment-result.v1.schema.json`](./experiment-result.v1.schema.json) and
[`action-record.v1.schema.json`](./action-record.v1.schema.json) are the
normative `F0-012` contracts. Their canonical identifiers end in
`experiment-result.v1.schema.json` and `action-record.v1.schema.json`; both
start at `schemaVersion: 1.0`.

- `EXPERIMENT_RESULT` is an immutable, append-only aggregate over an explicit
  window and population. Every later version names its immediate predecessor.
  It records samples, all assigned outcomes, exclusions, method, intervals,
  metric/policy versions, and one of `PRELIMINARY | FINAL | INVALIDATED`.
- A preliminary result can publish measured effects but cannot claim a
  promotion. A final result publishes gate evaluation. A `PROMOTE` decision
  requires the beta sample floor and affirmative benefit, p95, correctness,
  economics, and comparison gates; insufficient p99 data remains explicit.
- An invalidation is a new result version. It identifies the invalidated
  predecessor and supporting evidence without rewriting that result or
  republishing its effects as current.
- `ACTION_RECORD` is one append-only action-rollout transition with an exact
  prior-state/sequence precondition, source and policy binding, evidence, and
  authorization basis. It never embeds aggregate observed effects.
- `ACTIVATE_IN_CI` and `ACTIVATE_LOCALLY` require a referenced `FINAL` result.
  Cross-record validation additionally requires that exact linked result to
  say `PROMOTE`. A preliminary, inconclusive, invalidated, mismatched, or stale
  result cannot authorize activation.
- A record is audit evidence, not an executable command. Consumers still need
  the separately authenticated policy and state precondition before changing
  runtime behavior.

JSON Schema enforces each document's closed shape and local state conditions.
The lifecycle checker adds invariants that Draft 2020-12 cannot express:
timestamp and interval order, immediate version ancestry, count
reconciliation, state/sequence agreement, authorization timing, action
membership, and exact result-reference linkage.

Run all individual and cross-record fixtures with:

```bash
./dev/check-experiment-action-schemas
```

The command validates four valid and four invalid aggregate records, six valid
and four invalid transition records, and three valid and three invalid linked
lifecycle vectors. All values are synthetic. The test remains in the isolated
schema-validator Go module and is also executed by the base CI core lane.

Future schemas must retain the same explicit identifier, compatibility, required-field, unknown-field, format, bound, and positive/negative fixture policy. Signed commands additionally fail closed on unknown fields.

## Evidence, policy, and resource profile v1

[`evidence-record.v1.schema.json`](./evidence-record.v1.schema.json),
[`optimization-policy.v1.schema.json`](./optimization-policy.v1.schema.json),
and [`resource-profile.v1.schema.json`](./resource-profile.v1.schema.json) are
the normative `F0-013` contracts.

- `EVIDENCE_RECORD` binds source state, implementation, inputs, output,
  policy, namespace, semantic cache contract, redacted observations, complete
  coverage, and separate repeatability/relocatability gates. Observation alone
  is never a qualification source.
- Incomplete tracing can retain only `OBSERVING` or `SUSPENDED`. A discrepancy
  requires `SUSPENDED`; `QUARANTINE_VALIDATED` requires complete tracing and
  both independent gates to pass.
- `OPTIMIZATION_POLICY` is an immutable invocation decision with separate
  complete/configuration digests, monotonic security generations, explicit
  actions, cache and Configuration Cache decisions, a finite resource-profile
  reference, bounded budgets, qualified task contracts, and expiry.
- `BYPASS` and `KILL_SWITCH` policies cannot retain actions, cache access, or
  build enablement. The actual Ed25519 canonicalization/signature vectors
  remain owned by `F0-020`.
- `RESOURCE_PROFILE` defines one prevalidated finite catalog arm. Eligibility
  requires startup, memory, and rollback gates. The golden 4-vCPU/16-GiB
  catalog contains exactly `STABLE_CONTROL`, `W2_H3G`, `W3_H4G`, and `W4_H6G`;
  only workers and Gradle heap vary outside identity/evidence fields.

Run schema, negative, and cross-record golden checks with:

```bash
./dev/check-foundation-contract-schemas
```

The command uses the isolated pinned validator. Cross-record checks enforce
policy/evidence identity, policy validity at observation time, exact selected
profile binding, the fixed four-arm catalog, cgroup headroom, and the absence
of undeclared treatment differences. Simulator/replay, A/A, propensity, drift,
and rollback behavior still belong to the wider `BANDIT-001` gate.
