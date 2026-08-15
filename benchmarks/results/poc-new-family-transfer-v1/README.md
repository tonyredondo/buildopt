# Ktor new-family transfer

This terminal bundle tests the unchanged generic Structural Build Impact path
on Ktor, a substantial Kotlin Multiplatform Gradle family that was not used to
develop the existing reviewed profiles. The subject, public revision, JVM JAR
workflow, source mutation, required output, optimized-native control and value
gates were preregistered before any accepted proposal or timing.

## Result

The generic proposal independently selected `:ktor-http:jvmJar` in both
captures. It reduced the declared `jvmJar` workflow from 133 projects to three,
omitting 130 projects (97.74%) without a Ktor or repository-name branch.

| Evidence | Native mean | BuildOpt mean | Mean saving | Positive pairs | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| Capture 1 | 106.975 s | 14.847 s | **92.128 s / 86.12%** | 8/8 | qualify |
| Capture 2 | 100.474 s | 13.769 s | **86.705 s / 86.30%** | 8/8 | qualify |
| Balanced aggregate | 103.724 s | 14.308 s | **89.416 s / 86.21%** | 8/8 reciprocal blocks | `QUALIFY_BALANCED_STRUCTURAL_PROFILE` |

Across both captures, all 16 raw pairs improved. The deterministic block
bootstrap interval is **+79.451..+98.422 seconds**, the median block saving is
94.907 seconds, and p95 improves from 153.818 seconds to 20.397 seconds. The
required `ktor-http` JVM JAR is byte-identical in every arm, measured task
shapes are stable, both global-change fallbacks restore the complete native
graph, and product-attributable failures are zero.

## What this demonstrates

The result extends the generic POC beyond the repository families used to
develop its profiles. For this fixed Ktor change and public JVM JAR workflow,
reducing the graph from 1,141 observed tasks to 58 removes configuration,
scheduling, cache lookup, compilation and packaging work that optimized native
Gradle still performs for the unqualified selector. The gain is therefore a
complete workflow effect, not a task-duration microbenchmark.

The result does not claim that every Ktor workflow or every Gradle repository
will improve. The selector covers Ktor's public JVM JAR tasks, not the complete
multiplatform release build. Review remains explicit, exact output semantics
remain authoritative, and uncertain/global changes retain native Gradle.

## Preserved incidents

Five failed or invalid attempts are retained under [`incidents/`](./incidents/):

1. generic Gradle project properties were initially rejected;
2. a duplicate console option was passed to Gradle;
3. configuration-on-demand hid the synthetic discovery entrypoint;
4. Ktor's target loader ignored CLI `-Ptarget.*` properties, so the workflow
   was corrected before timing to the public `jvmJar` selector; and
5. an otherwise complete capture was rejected because the evidence renderer
   did not accept the reviewed `-P` property already accepted by the launcher.

No observation from those incidents is included in the terminal result. Each
generic defect was corrected and both accepted captures restarted from zero on
immutable BuildOpt revision `d658a23c20a6dca603eefdad5394a9b076cfde93`.

## Revalidation

```bash
./dev/check-new-family-transfer
./dev/test-new-family-transfer-result
```

The first command validates both source captures and recomputes the balanced
qualification. The second proves that summary and qualification tampering are
rejected. This remains POC evidence only: it grants no automatic or production
activation, requires no soak or design partner, and does not modify Test
Optimization.
