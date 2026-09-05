# Strict diagnostic capture reliability v1 evidence

State: `SDCR-001_DONE_SDCR-002_NEXT`.

The [human contract](../../../specs/poc-strict-diagnostic-capture-reliability-v1.md),
[machine contract](../../../specs/poc-strict-diagnostic-capture-reliability-v1.json),
[three-probe manifest](../../../specs/poc-strict-diagnostic-capture-reliability-v1.subjects.json),
and [detailed tracker](../../../docs/plans/strict-diagnostic-capture-reliability-poc.md)
freeze a new harness-reliability experiment after CINC's terminal source gate.

At SDCR-000 there are zero public starts, reports, opportunity rows, source
mutations, candidates, timings, speedup claims, and product failures. CINC
artifacts motivate the three failure classes but are not SDCR evidence inputs.

The [SDCR-001 runner proof](./sdcr-e001-runner.json) reconstructs sixteen
fixture cases through the pure root-report selector, CLI, and versioned capture
runner. It covers unique/missing/ambiguous/outside/symlinked report references,
all owner project-directory forms, Git/revision/archive rejection, nested
inventory isolation, atomic success, and forced post-start failure publication.
Public Gradle starts remain zero.

Validate the freeze with:

```bash
./dev/check-strict-diagnostic-capture-reliability
./dev/check-strict-diagnostic-capture-reliability-runner
```
