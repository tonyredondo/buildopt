# Source-bound configuration-input corrections v1 evidence

State: `STOP_INSUFFICIENT_DIAGNOSTIC_BINDING`.

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

The SBIC-002 capture adapter was fixture-proven before public execution. It maps
the frozen wrapper, working directory, JDK, owner arguments/environment and
required outputs into the immutable SDCR runner, and rejects a runner override
for the canonical manifest. Public Gradle starts remain zero at this boundary.

The first [SBIC-002 capture](./sbic-e002-strict-diagnostics/suwayomi-server/capture.json)
is retained as a harness-bound result. Gradle failed strictly after 3m19s and
reported one unique external process plus one report file, but `--stacktrace`
printed the identical report URI twice. The original selector therefore
recorded `ROOT_REPORT_REFERENCE_AMBIGUOUS` and did not retain the report. A
versioned v2 selector now deduplicates identical URI text while leaving two
distinct paths ambiguous. Suwayomi is conclusive but supplies no bound row and
was not replayed.

The terminal [SBIC-002 result](./sbic-e002-strict-diagnostics/result.json)
reconstructs all three fresh starts. QuickCarpet retained a 26-problem report,
but every problem belongs to unsupported Fabric Loom behavior and none reports
the three source-bound Git reads. LSSS retained two problems and binds its one
external-process problem exactly to `git rev-parse HEAD`. Together with the
unretained Suwayomi report, this yields 3/3 conclusive families, zero lost
starts, two retained reports, 28 parsed problems, and only 1/3
diagnostic-bound families versus the required 2/3.

SBIC therefore stops before controlled materiality. `SBIC-003..005` are not
authorized; `SBIC-006` records the terminal decision only. Public source
mutations, candidate builds, timing samples, speedup claims, and product
failures remain zero. This source-enriched study proves neither prevalence nor
value.

Validate the freeze with:

```bash
./dev/check-source-bound-configuration-input-corrections
./dev/check-source-bound-configuration-input-detector
./dev/check-source-bound-configuration-input-diagnostic-runner
./dev/check-source-bound-configuration-input-strict-diagnostics
```
