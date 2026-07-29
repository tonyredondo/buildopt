# BUILD_SESSION v1 fixtures

These fixtures are executable examples for [`build-session.v1.schema.json`](../../build-session.v1.schema.json). They contain synthetic identifiers, digests, timestamps, and prices only.

## Valid

| Fixture | Contract exercised |
|---|---|
| [`complete-build-failure.json`](./valid/complete-build-failure.json) | Complete candidate failure with a non-zero exit code and exact time to first actionable build failure |
| [`complete-ci-candidate.json`](./valid/complete-ci-candidate.json) | Complete candidate assignment with authenticated queue timing, optional model/resource/cost/cache facts, and an approximated critical path |
| [`complete-local-passthrough.json`](./valid/complete-local-passthrough.json) | Complete local passthrough that declares CI timing, resource, cost, and critical-path capabilities unavailable without values |
| [`partial-recovery.json`](./valid/partial-recovery.json) | Cancelled recovery with `complete: false`, partial envelope timing, no Gradle process, and an explicit missing sequence range |

## Invalid

| Fixture | Required rejection |
|---|---|
| [`complete-with-partial-metadata.json`](./invalid/complete-with-partial-metadata.json) | A complete record cannot carry partial assembly metadata |
| [`future-aggregate-effect.json`](./invalid/future-aggregate-effect.json) | A build session cannot contain a future aggregate causal effect |
| [`impossible-timestamp.json`](./invalid/impossible-timestamp.json) | RFC 3339 format assertions reject an impossible calendar date even when its lexical shape is valid |
| [`negative-duration.json`](./invalid/negative-duration.json) | Durations cannot be negative |
| [`partial-without-recovery.json`](./invalid/partial-without-recovery.json) | A partial record must identify missing sequence ranges |
| [`success-with-nonzero-exit.json`](./invalid/success-with-nonzero-exit.json) | A successful build must have exit code zero |
| [`unavailable-with-value.json`](./invalid/unavailable-with-value.json) | An unavailable metric cannot invent a value |

`dev/schema-validator` discovers every JSON file in both directories. Adding an invalid fixture also requires an expected diagnostic in the test so an unrelated validation failure cannot make the case appear to pass.
