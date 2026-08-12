# Public Gradle Workflow Value

This immutable POC evidence applies the installed BuildOpt path to four
substantial public workflow families at the revisions frozen in
[`poc-generic-workflow-value-v1.json`](../../../specs/poc-generic-workflow-value-v1.json).
It compares the generic structural candidate with the same optimized native
Gradle workflow, requires exact declared outputs and a successful full-graph
fallback, and keeps every family independent.

## Result

| Public workflow | Project graph | Result | Decision |
| --- | ---: | --- | --- |
| Apache Groovy root `jar` | 37 -> 2 | Measurement stopped. The immutable run saw ambiguous console task evidence; a diagnostic rerun reached output validation and found one time-bearing release-info payload difference. | Retain native. |
| Apache Kafka root `checkstyleMain` | 64 -> 2 | Measurement stopped because the reports embed absolute isolated-workspace paths. | Retain native. |
| Apache Kafka root `shadowJar` | 64 -> 2 | Measurement stopped because the upstream archive preserves timestamps and order. Two native rebuilds had different JAR hashes despite 4,378 identical payloads. | Retain native. |
| Spring Framework root `testClasses` | 27 -> 10 | Native 14.588 s; BuildOpt 11.893 s; **2.695 s / 18.47%** mean saving; positive 95% interval; exact class outputs; 7/8 positive pairs. | Retain native under the frozen 8/8 gate; investigate repeatability. |

The terminal decision is **0/4 qualified**, not “no value.” It demonstrates one
material installed-path signal and three fail-closed output-contract barriers.
No failed observation was removed, no output was normalized after measurement,
and no threshold was relaxed.

## Root-cause findings

[`diagnostics.json`](./diagnostics.json) records the bounded follow-up for every
non-winning family:

- Groovy's reproducible ZIP metadata still contains a generated
  `BuildTime`, so identical task semantics do not imply identical required
  bytes.
- Kafka Checkstyle reports expose the temporary checkout path.
- Kafka explicitly opts out of reproducible archive order and timestamps for
  the measured fat JAR.
- Spring has a credible **18.47%** signal, but one negative pair prevents
  qualification under the preregistered repeatability rule.

These are recoverable POC gaps. The next work must make task identity
structural, define explicit owner-reviewed equivalence only where exact bytes
are not the repository's real semantic contract, and rerun Spring with a new
order-aware protocol. Until then optimized native Gradle remains authoritative.

## Reproduction

```bash
./dev/check-generic-workflow-value
```

The raw proposal, measurement, evaluation and diagnostic logs are retained in
this directory. Online dependency preparation and the first diagnostic
reproductions are not timing observations.

This evidence is for idea validation only. It does not authorize production or
automatic activation, require a soak or design partner, or modify Test
Optimization.
