# Source-bound configuration-input corrections v1 evidence

State: `SBIC-000_COMPLETE`.

The [human contract](../../../specs/poc-source-bound-configuration-input-corrections-v1.md),
[machine contract](../../../specs/poc-source-bound-configuration-input-corrections-v1.json),
[three-family manifest](../../../specs/poc-source-bound-configuration-input-corrections-v1.subjects.json),
and [detailed tracker](../../../docs/plans/source-bound-configuration-input-corrections-poc.md)
freeze a source-enriched mechanism study.

The exact selected families are Suwayomi Server, QuickCarpet, and LSSS.
BlueMap is retained as a prospective source-only exclusion because its version
path can update Git index metadata. Selection occurred before every SBIC Gradle
start and used no CINC or SDCR report as evidence.

At SBIC-000 there are zero public Gradle starts, strict diagnostics, public
source mutations, candidate builds, paired timing samples, speedup claims, and
product failures. This experiment can establish a correction mechanism across
at least two families, but it cannot claim arbitrary-repository prevalence.

Validate the freeze with:

```bash
./dev/check-source-bound-configuration-input-corrections
```
