# BuildOpt POC: One-Page Handoff

## The idea

BuildOpt tests whether a generic layer can make substantial Gradle workflows
faster than an already optimized native Gradle baseline. Gradle remains the
execution engine and safe fallback. BuildOpt derives a smaller sufficient task
graph from the exact Git change and requested workflow, materializes verified
unaffected outputs, and enables the candidate only when measured value,
correctness, portability and compatibility all hold.

The intended experience is one command and no repository-specific BuildOpt
files:

```text
install BuildOpt
cd <a Gradle repository>
buildopt optimize <the existing Gradle workflow>
```

This is an owner-operated proof of concept, not a production product. Soak,
design-partner evidence, production SLOs, autonomous promotion and Test
Optimization are outside the current scope.

## Components and current status

| Component | What it does | Current conclusion |
| --- | --- | --- |
| **Structural Build Impact** | Selects only the changed producers and tasks needed by the requested workflow. | The primary accelerator: current isolated calibrations reduce Spring 27→10, OpenTelemetry 1,008→34, Kafka 66→3, Micronaut 70→21 and Groovy 37→2 projects. |
| **Automatic discovery** | Derives Git ownership, Gradle task/output relationships and candidate graphs without repository-name rules. | Works across `classes`, `testClasses`, `assemble` and the five unrelated public repositories. |
| **Incremental learning and value gate** | Accumulates useful control/candidate observations and checks repeatability, uncertainty, p95, outputs, fallback and payback. | Prevents weak current evidence from being promoted: Groovy retains native at 6/8 and a regressive p95. |
| **Verified output materialization** | Restores exact unaffected outputs before their producers are omitted. | Fast and fail-closed, but only portable output sets may move across roots or machines. |
| **Profile portfolio and central state** | Carries verified profiles and packs over HTTP/HTTPS between builds and machines. | Transport and safe cross-commit refresh now work; one Kafka descendant reused a verified local refresh and saved 104.975 seconds. Broader lifetime transfer remains unproven. |
| **Gradle-compatible cache** | Reuses verified task outputs locally or through optional HTTP/HTTPS storage. | Supporting infrastructure near native-cache parity, not the principal speed claim. |
| **Runtime Tuning, Hot State and standard Copy** | Earlier broad resource/state hypotheses. | Retired after neutral, unstable or regressive end-to-end evidence. |

## Latest public-repository evidence

One exact installed executable was used for all five subjects. The current run
requalified each frozen public change, checked native output portability across
independent roots and then followed every preregistered public descendant when
the profile was portable.

| Repository / workflow | Current calibration | Portability | Later builds | Cumulative conclusion |
| --- | ---: | --- | ---: | --- |
| Spring `testClasses` | **18.98% faster**, 8/8 | Rejected: 2 AspectJ classes differ | 0 | **-7.592 s net**; non-portable. |
| OpenTelemetry Spring family | **11.88% faster**, 8/8 | 269 exact outputs | 1 | 0 selections, 1 native fallback; **-13.255 s net**. |
| Kafka `testClasses` | **18.02% faster**, 8/8 | 4,440 exact outputs | 6 | 0 selections, 6 native fallbacks; **-39.961 s net**. |
| Micronaut `assemble` | **13.67% faster**, 8/8 | Rejected: 1 JAR differs | 0 | **-15.457 s net**; non-portable. |
| Groovy `classes` | 6.82% faster, 6/8; p95 worse | Not evaluated | 0 | **-1.835 s net**; current value not proven. |

That frozen V2 baseline was **4/5 qualified, 2/4 portable, 0/7 selected
replays, 7/7 native fallbacks, and 0/5 paid back**. The recovery experiment
then reran the same Kafka qualifier and six descendants after generic product
changes:

| Recovery measurement | Before | After |
| --- | ---: | ---: |
| Selected descendant candidate | 166.299 s | **42.577 s** |
| Attributable selected-replay saving | -5.404 s | **+104.975 s / 71.14%** |
| Six-build cumulative net after learning/publication | -22.040 s | **+66.772 s** |

The selected candidate became 123.722 seconds faster and preserved all 4,449
required outputs exactly. The five native-retained after observations total
-31.441 seconds of uncontrolled arm variation; that value remains visible in
window economics but is not attributed to BuildOpt. Repository percentages are
not averaged and mechanism percentages are not added. These runs used the
12-CPU development host with a common 12-worker cap.

## What this proves

- Generic graph reduction can beat optimized native Gradle on isolated,
  substantial workflows without repository-specific product branches.
- Calibration speed is not customer value. A profile must remain portable,
  refreshable, and eligible on later commits before its learning cost can repay.
- Fail-open behavior is working: every uncertain descendant ran optimized
  native Gradle and preserved exact outputs.
- Cross-commit reuse can now create attributable value: the one selected Kafka
  descendant saves 104.975 seconds and the six-build window finishes 66.772
  seconds net positive after learning and publication.
- The current POC is not yet general. This is one workflow in one public
  repository; it proves recovery is possible, not that every Gradle repository
  will produce the same saving or selection frequency.

## Next steps

1. Replicate the unchanged recovery mechanism on at least two additional public
   repository/workflow families with compatible descendant windows. Do not add
   repository-name rules or weaken exact-output and fallback gates.
2. Measure selection frequency, attributable selected-replay value, rejection
   cost, and cumulative payback separately. A positive native-arm delta cannot
   rescue a regressive selected replay.
3. Improve portability only by narrowing to customer-required Gradle-owned
   outputs. Never normalize, rewrite, or silently drop nondeterministic customer
   artifacts to manufacture a match.
4. Keep the one-command path automatic: learn during useful builds, refresh
   compatible profiles, select only after complete binding validation, and
   fall back near native cost on every uncertain revision.

## Evidence

- [Latest recovery result](../../benchmarks/results/poc-cross-commit-value-recovery-v1/README.md)
- [Machine-readable recovery summary](../../benchmarks/results/poc-cross-commit-value-recovery-v1/summary.json)
- [Recovery protocol](../../specs/poc-cross-commit-value-recovery-v1.md)
- [Five-repository lifetime baseline](../../benchmarks/results/poc-qualified-lifetime-v2/README.md)
- [Detailed performance findings](./build-optimization-performance.md)
- [Generalization audit](./buildopt-generalization-audit.md)
- [One-command roadmap](../plans/one-command-onboarding-roadmap.md)
- [Implementation tracker](../../implementation-tracker.md)
