# Source-bound configuration-input corrections v1 evidence

State: `SBIC-001_COMPLETE`.

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

The [SBIC-001 result](./sbic-e001-source-detector/result.json) is independently
rebuilt from retained byte-exact sources and semantic facts. It contains five
pure configuration-process rows across 3/3 selected families and one
side-effect rejection for BlueMap. The versioned detector, CLI, negative tests,
source/call-site hashes, summary counts, unknown-field rejection, and source
drift are all reconstructed. Public Gradle starts remain zero.

The SBIC-002 capture adapter is fixture-proven before public execution. It maps
the frozen wrapper, working directory, JDK, owner arguments/environment and
required outputs into the immutable SDCR runner, and rejects a runner override
for the canonical manifest. Public Gradle starts remain zero at this boundary.

Validate the freeze with:

```bash
./dev/check-source-bound-configuration-input-corrections
./dev/check-source-bound-configuration-input-detector
./dev/check-source-bound-configuration-input-diagnostic-runner
```
