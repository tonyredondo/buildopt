# Change-aware producer closure evidence

This directory contains the fresh evidence produced for `SWL-CHANGE-001`.
It covers five adjacent first-parent transitions in each frozen public Gradle
family, selected without consulting prior BuildOpt outcomes.

| Family | Conclusive transitions | Testable actions | No safe action |
| --- | ---: | ---: | ---: |
| Apache Groovy | 5/5 | 0 | 5 |
| Apache Kafka | 5/5 | 0 | 5 |
| Micronaut Core | 5/5 | 0 | 5 |
| OpenTelemetry Java Instrumentation | 5/5 | 0 | 5 |
| Spring Framework | 5/5 | 1 | 4 |

All 25 transitions are conclusive and all five family inputs are complete.
The only testable action appears in Spring Framework. This block measures
evidence completeness and action construction only: it contains no wall-time
measurement and grants no activation authority. `SWL-CHANGE-002` must
independently recompute whether the preregistered action-breadth gate passes.

`contract.json` freezes the cohort and producer contract, `captures/` contains
the raw Gradle evidence, `reports/` contains the typed analyzer decisions,
`transitions.jsonl` is the immutable transition ledger, and `summary.json`
contains the aggregate result. `sha256sums` binds every evidence file.

Validate the checked evidence with:

```bash
./dev/check-change-aware-public-capture
```
