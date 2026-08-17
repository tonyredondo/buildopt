# Automatic one-command public-repository matrix

This directory preserves the first customer-shaped `buildopt optimize` matrix
with zero hand-authored BuildOpt files. Every row uses a substantial public
Gradle repository and an installed Linux AMD64 development package. The raw
technical results are retained under [`raw`](./raw); `summary.json` carries the
recomputable comparison and terminal acceptance decision.

| Repository / workflow | Projects | Native mean | Candidate mean | Direct effect | Payback / decision |
| --- | ---: | ---: | ---: | ---: | --- |
| Ktor `jvmJar` | 133 -> 10 | 33.595 s | 6.049 s | **27.546 s / 82.00% faster** | **27 builds; qualified** |
| Spring `classes` | 27 -> 21 | 10.135 s | 9.072 s | **1.063 s / 10.49% faster** | 328 builds; native retained |
| Beam `classes` | 316 -> 6 | 20.396 s | 4.901 s | **15.495 s / 75.97% faster** | 37 builds; native retained |
| Groovy `classes` | 37 -> 30 | 61.497 s | 62.038 s | **0.542 s / 0.88% slower** | native retained |
| Kafka `testClasses` | 66 -> 36 | 8.921 s | 11.671 s | **2.750 s / 30.83% slower** | native retained |
| Micronaut `assemble` | unavailable | unavailable | unavailable | no timing claim | output semantics ambiguous; native retained |

Ktor, Spring, Groovy, Kafka and Beam each preserve eight paired observations
when calibration ran. Ktor, Spring and Beam have 8/8 positive pairs and exact
fallback; only Ktor repays within the declared maximum of 30 builds. Beam's
candidate is a particularly strong structural result (316 -> 6 projects and
75.97% lower replay wall time), but its complete 558.913-second first-decision
cost requires 37 matching builds.

This does **not** close the work item. The terminal gate requires two different
economically qualified families from a published package and fresh
install-to-decision state. Current count: **1/2**. The development-package
matrix also includes diagnostic retries for Kafka and Beam, which are retained
under `raw` but excluded from accepted timing.

The results demonstrate that the automatic POC can discover value and reject
unsafe or uneconomic candidates without repository-name rules. They also show
the next bottleneck: reduce generic calibration cost rather than weakening the
payback gate or searching randomly for favorable repositories.

Validate the contract and every raw-result binding with:

```bash
./dev/check-magic-end-to-end-value
```
