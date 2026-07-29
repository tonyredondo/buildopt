# Metric contracts

Machine-readable, versioned definitions for BuildOpt measurements and
experiment decisions.

| Catalog | Owning item | Validation |
|---|---|---|
| [`build-impact-v1.json`](./build-impact-v1.json) | `F0-024` / `METRICS-001` / `MEASURE-001` | `./dev/check-metrics-catalog` |

`metricDefinitionVersion` identifies semantic meaning independently of JSON
schema versions. A semantic change creates a new catalog version and never
rewrites historical records. Every metric declares its owner, purpose, formula,
unit, grain, population and denominator, source and time boundaries, permitted
dimensions, null policy, quality rules, retention owner, caveats, sign, and
observation methods.
