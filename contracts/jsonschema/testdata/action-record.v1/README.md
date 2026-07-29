# ACTION_RECORD v1 fixtures

These synthetic fixtures exercise the append-only transition contract in
[`action-record.v1.schema.json`](../../action-record.v1.schema.json). A valid
record audits authorization; merely receiving the document does not authorize
or execute an optimization.

## Valid schema records

| Fixture | Contract exercised |
|---|---|
| [`begin-shadow.json`](./valid/begin-shadow.json) | Policy-authorized `PROPOSED → SHADOW` transition |
| [`activate-in-ci.json`](./valid/activate-in-ci.json) | Final-result-authorized `CI_CANARY → ACTIVE_IN_CI` transition |
| [`activate-in-ci-claimed-final.json`](./valid/activate-in-ci-claimed-final.json) | Structurally valid activation whose claimed result status is checked against the linked lifecycle fixture |
| [`activate-in-ci-inconclusive.json`](./valid/activate-in-ci-inconclusive.json) | Structurally valid activation whose linked final decision is checked semantically |
| [`activate-in-ci-stale-result.json`](./valid/activate-in-ci-stale-result.json) | Structurally valid activation whose linked version is checked semantically |
| [`rollback-invalidated.json`](./valid/rollback-invalidated.json) | Safety rollback linked to an invalidation result |

The three “structurally valid” negative-link cases are deliberately accepted by
the standalone schema and rejected by the cross-record lifecycle validator.

## Invalid schema records

| Fixture | Required rejection |
|---|---|
| [`activate-with-preliminary-reference.json`](./invalid/activate-with-preliminary-reference.json) | Activation cannot explicitly cite a preliminary result |
| [`empty-evidence.json`](./invalid/empty-evidence.json) | Every transition requires at least one immutable evidence reference |
| [`embeds-observed-effect.json`](./invalid/embeds-observed-effect.json) | An action record cannot copy aggregate effects instead of referencing their result |
| [`wrong-transition-state.json`](./invalid/wrong-transition-state.json) | Transition, source state, destination state, and precondition must agree |
