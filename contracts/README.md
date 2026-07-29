# Contracts

Normative source for versioned schemas, interfaces, and cross-language conformance vectors.

`F0-010` owns the namespace structure defined in RFC §29.2. Each later item materializes and tests only the artifacts it owns; `F0-011` has materialized the first schema and its fixtures.

| Contract path | Owning item |
|---|---|
| `jsonschema/build-session.v1.schema.json` | `F0-011` |
| `jsonschema/experiment-result.v1.schema.json` | `F0-012` |
| `jsonschema/action-record.v1.schema.json` | `F0-012` |
| `jsonschema/evidence-record.v1.schema.json` | `F0-013` |
| `jsonschema/optimization-policy.v1.schema.json` | `F0-013` |
| `jsonschema/attempt-state.v1.schema.json` | `F0-014` |
| `jsonschema/ci-validation-request.v1.schema.json` | `F0-014` |
| `jsonschema/commit-decision.v1.schema.json` | `F0-014` |
| `jsonschema/resource-profile.v1.schema.json` | `F0-013` |
| `jsonschema/test-cache-grant.v1.schema.json` | `F0-015` |
| `jsonschema/test-validation-result.v1.schema.json` | `F0-015` |
| `jsonschema/patch-bundle.v1.schema.json` | `F0-016` |
| `openapi/buildopt-control.v1.yaml` | `F0-017` |
| `openapi/buildopt-cache-control.v1.yaml` | `F0-017` |
| `openapi/test-optimization.v1.yaml` | `F0-018` |
| `proto/local-events/v1/task_events.proto` | `F0-019` |

The subdirectories reserve JSON Schema 2020-12, OpenAPI 3.1, Protobuf v3, and shared test-vector boundaries. RFC examples are explanatory and must not be copied into implementation types. See the [JSON Schema index](./jsonschema/README.md) for the normative `BUILD_SESSION v1` contract and conformance command. Generated Go and Java clients remain owned by `F0-022`.
