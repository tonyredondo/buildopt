# Balanced Public-Repository Qualification

This directory is the terminal evidence for the preregistered
[balanced statistical qualification v2](../../../specs/poc-statistical-qualification-v2.md).
It asks one narrow POC question: does the unchanged generic Structural Build
Impact path materially beat optimized native Gradle on the five established
public repositories when repeatability, tail latency, outputs, task shape, and
fallback are evaluated together?

## Result

All five subjects qualify independently. Percentages are deliberately not
averaged across repositories.

| Repository | Full -> selected projects | Native mean | BuildOpt mean | Mean saving | Balanced blocks | p95 native -> BuildOpt |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Spring Framework | 27 -> 10 | 13.311 s | 11.183 s | **2.128 s / 15.99%** | 8/8 positive | 15.711 -> 13.318 s |
| OpenTelemetry Java Instrumentation | 1,024 -> 34 | 87.869 s | 74.713 s | **13.156 s / 14.97%** | 8/8 positive | 100.229 -> 79.004 s |
| Apache Kafka | 64 -> 3 | 113.381 s | 14.341 s | **99.040 s / 87.35%** | 8/8 positive | 145.478 -> 15.339 s |
| Micronaut Core | 75 -> 22 | 30.411 s | 18.418 s | **11.993 s / 39.44%** | 8/8 positive | 34.006 -> 20.448 s |
| Apache Groovy | 37 -> 2 | 79.868 s | 20.767 s | **59.101 s / 74.00%** | 8/8 positive | 85.338 -> 24.183 s |

Each row contains two independent captures of eight alternating pairs. The 16
pairs form eight opposite-order AB/BA blocks. Qualification requires at least
500 ms and 2% mean saving, positive median block saving, a deterministic
bootstrap lower bound above zero, at least 6/8 positive blocks, candidate p95
no worse than control, identical required outputs, stable measured task
shapes, two successful native full-graph fallbacks, and zero
product-attributable failures.

The terminal captures all use BuildOpt revision
`b8fd0f67598af50ee52eb8b11645a877836416db` and executable SHA-256
`565e1787ddcb27e81f5408158818b2e2445af38d0f083251415fb0a022640e87`.
The installed launcher, proposal validation, and Gradle execution are inside
the candidate timing. Distribution preparation, structural discovery, and
warm-up are outside the stable-state timing and remain onboarding costs.

## Interpretation

This result demonstrates that the generic graph-reduction hypothesis transfers
across five substantial Gradle repositories and beats their optimized native
full-graph controls for the exact measured changes, workflows, revisions, and
declared outputs. It does not prove universal savings for every change or
workflow, and it does not authorize automatic or production activation.

Spring now qualifies under the preregistered balanced criterion instead of
being rejected because one raw observation was negative in the historical
strict 8/8 protocol. No historical result was rewritten. The new decision is
based on 16 fresh pairs, 8/8 positive reciprocal blocks, a positive
1.803..2.386 s block-bootstrap interval, and improved p95.

Groovy capture 1 was conservatively marked inconclusive by the legacy
single-capture evaluator because its final candidate warm-up used a 13/9
executed/from-cache task split while all eight measured pairs stabilized at
14/8. Capture 2 reached 14/8 during its final warm-up and throughout all pairs.
The v2 aggregator checks the measured shapes across both independent captures,
where they are stable; the warm-up transition is preserved in the raw evidence
rather than hidden.

## Layout and validation

- `summary.json` is the five-repository matrix.
- `<repository>/qualification.json` is independently recomputed from its two
  capture directories.
- `<repository>/capture-{1,2}/` contains proposal, graph, raw observations,
  logs, result, fallback, and evaluation artifacts.
- `incidents/` preserves excluded infrastructure and orchestration failures;
  see its [incident index](./incidents/README.md).

Recompute every qualification and compare the result byte for byte:

```bash
./dev/check-statistical-qualification-v2-result \
  "$PWD/benchmarks/results/poc-statistical-qualification-v2"
```

This is bounded POC evidence. It is not a soak, design-partner result,
production-readiness claim, or Test Optimization evidence.
