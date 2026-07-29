# Contracts

Normative source for versioned schemas, interfaces, and cross-language conformance vectors.

`F0-010` owns the namespace structure defined in RFC §29.2. Each later item materializes and tests only the artifacts it owns; `F0-011` materialized the immutable build-session schema, `F0-012` materialized the aggregate experiment and action-transition lifecycles, `F0-013` materialized evidence, policy, and finite resource-profile contracts, `F0-014` materialized durable attempt and atomic commit authorization contracts, `F0-015` materialized signed Test Optimization grant/result contracts, `F0-016` materialized the declarative PatchBundle envelope and bundle vectors, `F0-017` materialized the BuildOpt control and internal cache-control HTTP boundaries, `F0-018` materialized the Test Optimization HTTP boundary, `F0-019` materialized the local task-event IDL, and `F0-024` materialized the first metric catalog.

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
| `metrics/build-impact-v1.json` | `F0-024` |

The subdirectories reserve JSON Schema 2020-12, OpenAPI 3.1, Protobuf v3, metrics, and shared test-vector boundaries. RFC examples are explanatory and must not be copied into implementation types. See the [JSON Schema index](./jsonschema/README.md) for `BUILD_SESSION`, lifecycle, evidence/policy, and finite-resource contracts; the [OpenAPI index](./openapi/README.md) for the control-plane HTTP boundaries; the [metric index](./metrics/README.md) for `build-impact-v1`; and the [Protobuf index](./proto/README.md) for the local channel and conformance command. `F0-005` tracks a reviewable generated descriptor snapshot and rejects drift under the repository [generated-code policy](../GENERATED_CODE.md); generated Go and Java clients remain owned by `F0-022`.
