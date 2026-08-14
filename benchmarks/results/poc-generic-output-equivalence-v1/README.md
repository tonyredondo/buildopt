# Generic Output-Equivalence Qualification

This directory contains the terminal evidence for the preregistered
[generic output-equivalence POC](../../../specs/poc-generic-output-equivalence-v1.md).
It asks whether generic Structural Build Impact can reduce wall time for
build-owned reports and archives whose correct outputs are not necessarily
byte-reproducible, without treating real payload drift as equivalent.

## Result

All three workflow subjects qualify independently. Percentages are not
averaged across workflows.

| Repository and workflow | Full -> selected projects | Native mean | BuildOpt mean | Mean saving | Balanced blocks | p95 native -> BuildOpt |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Apache Groovy `jar` | 37 -> 2 | 72.319 s | 19.455 s | **52.864 s / 73.10%** | 8/8 positive | 75.725 -> 21.341 s |
| Apache Kafka `checkstyleMain` | 64 -> 2 | 82.835 s | 58.209 s | **24.627 s / 29.73%** | 8/8 positive | 88.980 -> 61.402 s |
| Apache Kafka `shadowJar` | 64 -> 2 | 40.728 s | 13.625 s | **27.103 s / 66.55%** | 8/8 positive | 44.147 -> 15.583 s |

Each subject contains two independent captures of eight alternating pairs.
The 16 raw pairs form eight reciprocal AB/BA blocks. All **48/48 raw pairs**
improved, every candidate p95 is lower, all measured and final warm-up task
shapes are stable, both native full-graph fallbacks pass per subject, and no
product-attributable failure occurred.

## Correctness boundary

Byte identity remains the default. The reviewed exceptions are narrow and
bound to the captured owner contract:

- Groovy ignores only the value of `BuildTime` inside the exact
  `META-INF/groovy-release-info.properties` entry. Every other archive entry,
  property, payload, mode, and size remains bound.
- Kafka Checkstyle keeps `main.html` byte-exact and replaces only the isolated
  repository root in the declared UTF-8 `main.xml` report.
- Kafka `shadowJar` compares canonical ZIP entry names, modes, directory
  flags, uncompressed sizes, and payload digests. Archive order, timestamps,
  compression representation, comments, and extra fields are not semantic
  payload.

The conformance suite mutates undeclared report content, archive payloads, and
properties and requires rejection. Output rules are selected by relative
paths and modes; the product contains no branch for Groovy, Kafka, or a
repository name.

## Interpretation

The earlier public-workflow run retained native Gradle because byte comparison
could not distinguish payload drift from build-owned nondeterminism. This
result demonstrates that explicit owner-reviewed semantic contracts recover
those workflows while preserving the same material-value, tail, repeatability,
shape, fallback, and zero-failure gates.

The evidence does **not** prove every JAR, report, change, or Gradle repository
is safe to reduce. It authorizes review of these exact profiles only. New or
drifted outputs remain byte-exact or fall back to the original optimized-native
workflow.

## Layout and validation

- `summary.json` is the terminal three-subject matrix.
- `<workflow>/qualification.json` is independently recomputed from its two
  capture directories.
- `<workflow>/capture-{1,2}/` contains the owner contract, proposal, graph,
  raw observations, output digests, fallback, and evaluation artifacts.
- [`incidents/`](./incidents/README.md) preserves every excluded collection or
  convergence attempt and the correction made before terminal timing.

Recompute every aggregate and compare each qualification byte for byte:

```bash
./dev/check-generic-output-equivalence-result \
  "$PWD/benchmarks/results/poc-generic-output-equivalence-v1"
```

This is bounded POC evidence. It creates no production authority, automatic
activation, soak requirement, design-partner claim, or Test Optimization
scope.
