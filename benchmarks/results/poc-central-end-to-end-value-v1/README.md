# Central End-to-End Value Result

This terminal POC result compares the complete installed BuildOpt central path
with optimized native Gradle under the same already-committed remote-cache
opportunity. The control executes the full requested graph. The candidate uses
the automatically discovered structural scope and the same central objects.

| Public repository / workflow | Graph | Central objects | Native mean | BuildOpt mean | Direct result | 95% saving interval | p95 | Payback |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Ktor `jvmJar` | 133 -> 10 | 6 / 474,183 B | 215.506 s | 37.828 s | **177.678 s / 82.45% faster** | 156.555..198.802 s | 243.986 -> 55.139 s | 28 builds |
| Beam `classes` | 316 -> 6 | 155 / 46,565,962 B | 48.475 s | 21.130 s | **27.345 s / 56.41% faster** | 22.407..32.283 s | 53.468 -> 28.319 s | 29 builds |

Both repository families improved in all eight alternating pairs and produced
the exact required outputs. The comparison includes installed-package,
selection, launcher, gateway, TLS, central synchronization and Gradle wall
time. Dependency/bootstrap preparation is outside the measured pairs and each
arm has an isolated workspace, Gradle User Home and BuildOpt cache.

The honest negative changes Ktor root build logic. BuildOpt records
`GLOBAL_CHANGE_REQUIRES_FULL_GRAPH`, retains the native workflow, obtains 13
central cache hits and makes no performance claim. This proves the central
connection does not turn an unsafe structural decision into a selective one.

The run used the 12-CPU Linux host with a common eight-worker limit. It is
bounded POC evidence, not the contractual 4-CPU golden-runner class, a soak,
production authorization or a universal claim. Repository percentages are not
averaged and mechanism percentages are not added.

The machine-readable source of truth is [`summary.json`](./summary.json).
Validate it with:

```bash
./dev/check-central-end-to-end-value \
  benchmarks/results/poc-central-end-to-end-value-v1/summary.json
```
