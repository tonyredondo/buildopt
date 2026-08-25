# Generic POC Functional-Coverage Decision

The frozen terminal decision is **`STOP_GENERIC_POC`**.

The result passes five of eight criteria:

- all five public repository families were observed;
- 27 requested-build/output observations are exact, with zero measurement-only
  builds and zero product failures;
- selection contains no repository-specific product rule;
- Kafka's qualified target is robust at 8/8 positive pairs, a positive interval
  and a lower candidate p95; and
- Kafka projects and observes payback at match two, ending 82.527 seconds net
  positive.

It fails the three criteria that make the claim generic:

| Criterion | Required | Observed |
|---|---:|---:|
| Net-positive repository families | at least 3/5 | 1/5 |
| Eligible descendant selection | at least 50% | 1/6 (16.67%) |
| Pre-Gradle native-retention overhead | median <500 ms, p95 <1,000 ms | one sample: median/p95 4,098 ms |

The single early-decision observation is not presented as a distribution. With
one observed value, its nearest-rank median and p95 are both 4,098 ms, and the
criterion fails. Post-Gradle discovery fallbacks are excluded because the
frozen criterion applies only where a decision is available before Gradle.

This stops the current generic structural-profile hypothesis and withdraws a
broad customer-value claim. It preserves the measured Kafka and Spring wins,
safe fallback/correctness controls, cache/central-state infrastructure and all
negative evidence. It does not authorize repository-specific rules,
production hardening, soak or design-partner work.

Recompute and verify the decision with:

```bash
./dev/check-functional-coverage-decision
```

The machine-readable result is [summary.json](./summary.json), and the frozen
contract is [poc-functional-coverage-decision-v1.md](../../../specs/poc-functional-coverage-decision-v1.md).
