# Terminal five-repository structural matrix

This directory preserves the complete terminal evidence for
`POC-GENERIC-PROFILE-MATRIX-002`. The same installed, repository-independent
`profile propose -> profile measure -> profile evaluate` path was applied to
five pinned public Gradle repositories. Every timed row uses Build Impact alone
against that repository's declared optimized-native Gradle workflow, eight
alternating isolated pairs, 12 workers, byte-identical required outputs, and a
fail-closed full-graph fallback.

| Repository | Graph | Native mean | BuildOpt mean | Direct result | Terminal v3 decision |
| --- | ---: | ---: | ---: | ---: | --- |
| Spring Framework | 27 -> 10 projects | 13.940 s | 11.438 s | **17.94% faster**, 2.501 s, 7/8 positive, interval +1.479..+3.422 s | Retain native under the frozen 8-of-8 gate |
| OpenTelemetry Java Instrumentation | 1,024 -> 34 projects | 87.242 s | 73.759 s | Eight positive timed pairs, but no accepted percentage | Retain native because the reduced-concurrency fallback changed required output bytes |
| Apache Kafka | 64 -> 3 projects | 82.498 s | 13.113 s | **84.11% faster**, 69.385 s, 8/8 positive | Qualify exact structural profile |
| Micronaut Core | 75 -> 22 projects | 27.407 s | 15.968 s | **41.74% faster**, 11.439 s, 8/8 positive | Qualify exact structural profile |
| Apache Groovy | 37 -> 2 projects | 75.064 s | 19.629 s | **73.85% faster**, 55.434 s, 8/8 positive | Qualify exact structural profile |

OpenTelemetry's v3 timings are retained as rejected evidence, not reused as a
result. Its measured arms were byte-identical, but the correctness-only
fallback changed scheduling from parallel 12-worker execution to non-parallel
four-worker execution and produced different required bytes. The independently
preregistered v4 correction reruns only that row from zero and is stored in the
adjacent [`poc-generic-profile-matrix-v4`](../poc-generic-profile-matrix-v4/README.md)
bundle.

No repository percentages are averaged and no mechanism percentages are
added. The result is POC evidence for exact scopes, not production or automatic
activation authority.

Validate this bundle without network access:

```bash
./dev/check-generic-profile-matrix \
  benchmarks/results/poc-generic-profile-matrix-v3 \
  specs/poc-generic-profile-matrix-v3.json
```
