# Strict diagnostic capture reliability v1 evidence

State: `CAPTURE_RELIABILITY_PROVEN`.

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

The terminal [public-probe result](./sdcr-e002-public-probes/result.json) and
independent checker reconstruct three fresh starts. OpenAPI Generator is a
typed cold incompatibility; Licensee completes with 237 inventory reports and
Gradle Profiler completes with one, but neither root child log references a
Configuration Cache report. The runner therefore selects zero files instead of
guessing. The gate passes at 3/3 conclusive and zero silently lost starts.
Durations are functional only and no opportunity/value claim is made.

Validate the freeze with:

```bash
./dev/check-strict-diagnostic-capture-reliability
./dev/check-strict-diagnostic-capture-reliability-runner
./dev/check-strict-diagnostic-capture-reliability-public
```
