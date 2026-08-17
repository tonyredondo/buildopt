# Generic calibration economics on Apache Beam

This bundle preserves the POC experiment that made the automatically discovered
Apache Beam `classes` profile economically usable without weakening the value
gate. The implementation binds the existing Gradle dependency cache by content
instead of copying 2.63 GB into every arm, snapshots the native build cache once
and hard-links it into private arms, and uses the first measured pair to freeze
the control and candidate task shapes.

The accepted run used eight alternating pairs. Every pair produced the same 12
required outputs, the same control/candidate task fingerprints and a positive
wall-time saving. Full-graph fallback also passed.

| Measurement | Comparable baseline | Optimized protocol | Change |
| --- | ---: | ---: | ---: |
| Calibration cost | 1,097.547 s | 988.145 s | **-109.402 s / -9.97%** |
| Mean native wall time | 58.682 s | 61.916 s | not interpreted across runs |
| Mean BuildOpt wall time | 24.642 s | 23.754 s | not interpreted across runs |
| Mean saving | 34.040 s | **38.162 s** | not added to cost reduction |
| Break-even | 33 builds | **26 builds** | **7 builds sooner** |

The earlier automatic matrix reported 558.913 seconds of learning and a
15.495-second saving. It did not use the later authoritative dependency binding
and measured task-shape protocol, so it remains historical evidence rather than
the cost baseline for this comparison. The current absolute learning cost is
988.145 seconds; BuildOpt qualifies because the same rigorous run measures a
38.162-second saving and therefore repays within the declared 30 builds.

This is a local development-package POC result, not the terminal published-
package onboarding proof. `POC-MAGIC-END-TO-END-VALUE-001` remains open until a
fresh public package repeats the Ktor and Beam decisions from clean state.

Validate the complete dataset with:

```bash
./dev/check-magic-calibration-cost
```
