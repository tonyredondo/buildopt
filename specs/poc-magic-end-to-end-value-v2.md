# One-command terminal POC value evidence

## Decision

The one-command POC gate is complete. An immutable public `v0.6.1` package was
installed into a fresh prefix and executed from fresh Ktor and Apache Beam
checkouts with fresh BuildOpt state:

```text
install BuildOpt -> open repository -> buildopt optimize <existing workflow>
```

Neither repository contained a BuildOpt manifest, graph, output contract or
profile. The same generic command discovered the Git comparison, Gradle-owned
outputs and structural candidate; ran eight alternating pairs against the
optimized native Wrapper; verified exact required outputs and full-graph
fallback; and applied the owner-declared 30-build payback limit.

| Repository | Graph | Native mean | BuildOpt mean | Saving | Payback |
| --- | ---: | ---: | ---: | ---: | ---: |
| Ktor `jvmJar` | 133 -> 10 | 38.810 s | 7.830 s | **30.979 s / 79.82%** | 26 builds |
| Beam `classes` | 316 -> 6 | 65.081 s | 24.958 s | **40.123 s / 61.65%** | 28 builds |

Both rows improve in 8/8 pairs, have a positive paired 95% lower bound, lower
candidate p95, byte-identical required outputs, stable task shapes, successful
full-graph fallback and zero product-attributable failures. Percentages remain
repository-specific and are not averaged or added.

A separate Ktor root build-logic change ran the complete native `jvmJar`
workflow successfully and retained native with
`GLOBAL_CHANGE_REQUIRES_FULL_GRAPH`. It produced no calibration or performance
claim. This is a required success condition, not a failed positive case.

## Fair-comparison boundary

The package prefix, repository checkout and BuildOpt state were fresh. Gradle
dependencies and the native build-cache seed were deliberately prebound by
content and supplied identically to both isolated arms. Daemon/configuration
warmup was unmeasured. This compares BuildOpt with an already optimized native
Gradle opportunity instead of crediting BuildOpt for downloads or unequal
cache state.

## Rejected attempts

The terminal evidence preserves two rejected `v0.6.0` Ktor attempts. The first
ran inside a network-isolated sandbox where Gradle could not select a wildcard
address. The second exposed a real Configuration Cache incompatibility in
automatic output discovery after the native build succeeded. BuildOpt retained
native in both cases. The output-discovery implementation was corrected and
validated in Base CI and Native Platform CI before `v0.6.1` was published; no
timing observation from either rejected attempt enters the terminal result.

## POC boundary

This closes feasibility for two substantial public repository families and an
honest negative. It does not claim universal Gradle improvement, production
readiness, autonomous promotion, observed cross-commit profile lifetime, soak
qualification or Test Optimization. A new repository still has to prove its
own output semantics, wall-time value and economics; otherwise native Gradle
remains authoritative.

The recomputable summary and retained observations are in
[`poc-magic-end-to-end-value-v2`](../benchmarks/results/poc-magic-end-to-end-value-v2/README.md).
The earlier v1 matrix remains immutable diagnostic history.
