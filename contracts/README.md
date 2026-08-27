# Contracts

Normative source for versioned schemas, interfaces, and cross-language conformance vectors.

`F0-010` owns the namespace structure defined in RFC §29.2. Each later item materializes and tests only the artifacts it owns; `F0-011` materialized the immutable build-session schema, `F0-012` materialized the aggregate experiment and action-transition lifecycles, `F0-013` materialized evidence, policy, and finite resource-profile contracts, `F0-014` materialized durable attempt and atomic commit authorization contracts, `F0-015` materialized signed Test Optimization grant/result contracts, `F0-016` materialized the declarative PatchBundle envelope and bundle vectors, `F0-017` materialized the BuildOpt control and internal cache-control HTTP boundaries, `F0-018` materialized the Test Optimization HTTP boundary, `F0-019` materialized the local task-event IDL, `F0-024` materialized the first metric catalog, `SWL-007` materialized the sticky-wrapper decision/state union and conformance vectors, `SWL-010` materialized the bounded paired-trial schema and fixtures, and `POC-CENTRAL-STORAGE-CONTRACT-001` materialized the optional cross-machine state boundary.

| Contract path | Owning item |
|---|---|
| `jsonschema/adaptive-fragment.v1.schema.json` | `AF-002` |
| `jsonschema/adaptive-fragment-observation.v1.schema.json` | `AF-002` |
| `jsonschema/adaptive-fragment-portfolio.v1.schema.json` | `AF-002` |
| `jsonschema/adaptive-fragment-economic-ledger.v1.schema.json` | `AF-002` |
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
| `jsonschema/central-state-manifest.v1.schema.json` | `POC-CENTRAL-STORAGE-CONTRACT-001` |
| `jsonschema/central-state-head.v1.schema.json` | `POC-CENTRAL-STORAGE-CONTRACT-001` |
| `jsonschema/central-state-cas.v1.schema.json` | `POC-CENTRAL-STORAGE-CONTRACT-001` |
| `jsonschema/sticky-wrapper-decision-store.v1.schema.json` | `SWL-007` |
| `jsonschema/sticky-wrapper-observation.v1.schema.json` | `SWL-009` |
| `jsonschema/sticky-wrapper-trial.v1.schema.json` | `SWL-010` |
| `openapi/buildopt-control.v1.yaml` | `F0-017` |
| `openapi/buildopt-cache-control.v1.yaml` | `F0-017` |
| `openapi/test-optimization.v1.yaml` | `F0-018` |
| `proto/local-events/v1/task_events.proto` | `F0-019` |
| `metrics/build-impact-v1.json` | `F0-024` |
| `test-vectors/central-storage/central-storage.v1.json` | `POC-CENTRAL-STORAGE-CONTRACT-001` |

The subdirectories reserve JSON Schema 2020-12, OpenAPI 3.1, Protobuf v3, metrics, and shared test-vector boundaries. RFC examples are explanatory and must not be copied into implementation types. See the [JSON Schema index](./jsonschema/README.md) for `BUILD_SESSION`, lifecycle, evidence/policy, and finite-resource contracts; the [OpenAPI index](./openapi/README.md) for the control-plane HTTP boundaries; the [metric index](./metrics/README.md) for `build-impact-v1`; the [Protobuf index](./proto/README.md) for the local channel and conformance command; and the [test-vector index](./test-vectors/README.md) for the shared JCS, digest, timestamp, Ed25519, compatibility, and state-machine corpora. The repository [generated-code policy](../GENERATED_CODE.md) covers the reviewable descriptor plus the `F0-022` Go and Java OpenAPI clients and rejects drift from any normative source.
