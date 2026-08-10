# Five-repository generic structural profile matrix

This directory contains the terminal evidence for
`POC-GENERIC-PROFILE-MATRIX-001`. The same installed, repository-independent
`profile propose -> profile measure -> profile evaluate` path was applied to
five pinned public Gradle repositories. Each accepted measurement compares
Build Impact alone with that repository's declared optimized-native Gradle
workflow over eight alternating isolated pairs, byte-identical required
outputs, and a successful native full-graph fallback.

| Repository | Graph | Native mean | Candidate mean | Direct result | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| Spring Framework | 27 -> 10 projects | 12.338 s | 11.495 s | **6.83% faster**, 843 ms, 5/8 positive, interval -738..+2,256 ms | Retain native; uncertainty crosses zero |
| OpenTelemetry Java Instrumentation | 1,024 -> 34 projects | — | — | No accepted result | Retain native; pair 6 exceeded the preregistered 5-second inter-arm gap by 444 ms |
| Apache Kafka | 64 -> 3 projects | 87.873 s | 13.080 s | **85.12% faster**, 74.793 s, 8/8 positive, interval +62.183..+90.347 s | Qualify exact structural profile |
| Micronaut Core | 75 -> 22 projects | 25.699 s | 14.848 s | **42.22% faster**, 10.851 s, 8/8 positive, interval +10.251..+11.458 s | Qualify exact structural profile |
| Apache Groovy | 37 -> 2 projects | 69.446 s | 19.455 s | **71.99% faster**, 49.991 s, 8/8 positive, interval +47.969..+51.873 s | Qualify exact structural profile |

OpenTelemetry's first five pairs were directionally favorable, but they are
not a result: the sixth pair violated the frozen timing boundary, so the whole
cell is `MEASUREMENT_UNAVAILABLE`. The raw log is retained to make that
rejection auditable rather than selectively reporting favorable observations.

The three qualifying rows prove that structural graph reduction can create
large end-to-end cascade value across different repository and workload
families. Spring proves the equally important opposite: a smaller graph does
not automatically justify activation when the end-to-end interval remains
uncertain. No repository percentages are averaged, no mechanism percentages
are added, and the older Jar/Edge compositions in `summary.json` remain
separate historical context.

Validate this bundle without network access:

```bash
./dev/check-generic-profile-matrix \
  benchmarks/results/poc-generic-profile-matrix-v1
```

The frozen protocol and exact public revisions are documented in
[`specs/poc-generic-profile-matrix-v1.md`](../../../specs/poc-generic-profile-matrix-v1.md).
