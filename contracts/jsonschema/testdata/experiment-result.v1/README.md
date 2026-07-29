# EXPERIMENT_RESULT v1 fixtures

These synthetic fixtures exercise the append-only aggregate lifecycle in
[`experiment-result.v1.schema.json`](../../experiment-result.v1.schema.json).
They contain no real repository, customer, or build data.

## Valid

| Fixture | Contract exercised |
|---|---|
| [`preliminary-learning.json`](./valid/preliminary-learning.json) | A small, explicitly preliminary aggregate that cannot authorize promotion |
| [`final-qualified.json`](./valid/final-qualified.json) | A direct/reversible result that meets the beta sample and recorded gate requirements |
| [`final-inconclusive.json`](./valid/final-inconclusive.json) | A final analysis whose invalid comparison remains `INCONCLUSIVE` |
| [`invalidated-result.json`](./valid/invalidated-result.json) | A new append-only version invalidating the preceding result without republishing effects |

## Invalid

| Fixture | Required rejection |
|---|---|
| [`action-scope-without-action.json`](./invalid/action-scope-without-action.json) | An action-incremental result must identify its action population |
| [`final-without-gates.json`](./invalid/final-without-gates.json) | A final result must publish its gate evaluation |
| [`invalidated-republishes-effects.json`](./invalid/invalidated-republishes-effects.json) | An invalidation version cannot republish aggregate effects as current |
| [`preliminary-promotes.json`](./invalid/preliminary-promotes.json) | A preliminary result cannot claim a promotion decision |

The semantic lifecycle vectors additionally check timestamp order, interval
order, version ancestry, exclusion totals, and authorization linkage.
