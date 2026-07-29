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

Future schemas must retain the same explicit identifier, compatibility, required-field, unknown-field, format, bound, and positive/negative fixture policy. Signed commands additionally fail closed on unknown fields.
