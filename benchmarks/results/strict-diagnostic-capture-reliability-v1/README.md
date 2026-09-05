# Strict diagnostic capture reliability v1 evidence

State: `SDCR-000_DONE_SDCR-001_NEXT`.

The [human contract](../../../specs/poc-strict-diagnostic-capture-reliability-v1.md),
[machine contract](../../../specs/poc-strict-diagnostic-capture-reliability-v1.json),
[three-probe manifest](../../../specs/poc-strict-diagnostic-capture-reliability-v1.subjects.json),
and [detailed tracker](../../../docs/plans/strict-diagnostic-capture-reliability-poc.md)
freeze a new harness-reliability experiment after CINC's terminal source gate.

At SDCR-000 there are zero public starts, reports, opportunity rows, source
mutations, candidates, timings, speedup claims, and product failures. CINC
artifacts motivate the three failure classes but are not SDCR evidence inputs.

Validate the freeze with:

```bash
./dev/check-strict-diagnostic-capture-reliability
```
