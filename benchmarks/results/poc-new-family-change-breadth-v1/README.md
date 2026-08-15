# Ktor change-breadth evidence

This directory contains the terminal evidence for the preregistered
`POC-NEW-FAMILY-CHANGE-BREADTH-001` study. It tests whether BuildOpt's generic
structural Build Impact path continues to provide value when Ktor's public
`jvmJar` workflow receives dependency-source, JVM-resource, multi-module, or
global-configuration changes.

The accepted evidence binds:

- Ktor revision `bc7de799f4eb997a63250f2f70492d85f5c92f50`;
- Gradle Wrapper 9.5.1 and Temurin 21;
- BuildOpt revision `35065d30ed066af3f9dea75382b2b9b9af66301a`;
- optimized native Gradle as the control;
- only structural Build Impact as the candidate mechanism;
- the complete preregistered Gradle option list, including
  `-Pktor.develocity.skipBuildScans=true`;
- exact bytes for every required JAR.

## Terminal results

Each selective row contains two independent captures, 16 timed pairs and eight
reciprocal order blocks. Percentages are cell-specific and are not averaged.

| Change | Full -> selected projects | Native mean | BuildOpt mean | Mean saving | Block interval | p95 native -> BuildOpt | Decision |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Upstream dependency source | 133 -> 3 | 98.269 s | 13.954 s | **84.314 s / 85.80%** | +76.348..+91.332 s | 135.716 -> 19.202 s | `QUALIFIED` |
| JVM service resource | 133 -> 9 | 57.238 s | 7.720 s | **49.517 s / 86.51%** | +40.513..+59.160 s | 91.795 -> 11.345 s | `QUALIFIED` |
| Mixed production source, two modules | 133 -> 9 | 104.877 s | 23.096 s | **81.781 s / 77.98%** | +72.094..+89.813 s | 146.338 -> 34.848 s | `QUALIFIED` |
| Root Gradle configuration | full graph | not timed | not timed | no timing claim | not applicable | not applicable | `FULL_GRAPH_FALLBACK_VERIFIED` |

All 48 timed pairs and all 24 reciprocal blocks save time. Required JARs are
byte-identical, task shapes are stable, candidate p95 is lower in every timed
cell, all six selective full-graph fallbacks succeed, and no observation has a
product-attributable failure. The two independent root-configuration proposals
both return `NATIVE_FULL_GRAPH / GLOBAL_CHANGE_REQUIRES_FULL_GRAPH` and emit no
candidate timing evidence.

## Rejected diagnostic run

The `incidents/980d021-omitted-preregistered-gradle-option/` directory retains
the first complete diagnostic matrix. It is not terminal evidence: the generic
runner did not propagate one Gradle option frozen in the preregistration. Both
arms within that run were internally comparable, but accepting it would have
violated the experiment contract.

The runner was corrected and committed before the terminal rerun. All eight
accepted captures restarted from zero on BuildOpt `35065d3`; no timing,
warm-up, profile, or summary from the incident was reused.

## Reproduce the checks

The terminal measurements are intentionally not run on every CI push. CI
validates the captured evidence and recomputes each qualification:

```bash
./dev/check-new-family-change-breadth-result
./dev/test-new-family-change-breadth-result
```

The checker binds source and product revisions, changed paths, required
outputs, proposals, exact Gradle options, mechanisms, raw observation counts,
fallbacks and every preregistered qualification threshold. The negative test
proves that summary or qualification tampering fails closed.

## Boundaries

This is POC evidence for Ktor's public JVM JAR selector, not a claim about its
complete multiplatform release graph or every Gradle repository. It adds no
repository-name product rule and grants no automatic or production activation.
The next study must measure discovery and stabilization cost against these
immutable per-invocation savings before making a broader onboarding claim.
