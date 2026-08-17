# Published one-command terminal evidence

This bundle closes the customer-shaped POC gate using the immutable public
[`v0.6.1`](https://github.com/tonyredondo/buildopt/releases/tag/v0.6.1)
package. The package was installed into a fresh prefix and used from fresh
Ktor and Apache Beam checkouts with fresh BuildOpt state and no hand-authored
BuildOpt files.

| Repository / workflow | Graph | Native mean | BuildOpt mean | Saving | 95% saving interval | p95 | Payback |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Ktor `jvmJar --max-workers=12` | 133 -> 10 | 38.810 s | 7.830 s | **30.979 s / 79.82%** | 24.679..38.922 s | 61.575 -> 11.957 s | 26 builds |
| Beam `classes --max-workers=12` | 316 -> 6 | 65.081 s | 24.958 s | **40.123 s / 61.65%** | 33.867..51.470 s | 102.621 -> 25.946 s | 28 builds |

Both rows have 8/8 positive alternating pairs, exact required-output hashes in
every pair, stable task-outcome fingerprints, lower candidate p95, successful
full-graph fallback and zero product-attributable failures. The percentages
are not averaged.

The required negative is a Ktor root `settings.gradle.kts` change. Its full
native `jvmJar` build succeeded, and BuildOpt retained native with
`GLOBAL_CHANGE_REQUIRES_FULL_GRAPH` before calibration. This demonstrates safe
non-activation, not a performance regression.

The package, checkout and BuildOpt state were fresh. Both measurement arms used
the same content-bound Gradle dependencies and the same immutable native-cache
seed, with unmeasured daemon/configuration warmup. The comparison is therefore
against optimized native Gradle rather than a download or cache-state mismatch.

Two rejected `v0.6.0` attempts are retained. One records an unusable sandbox
network namespace; the other records the Configuration Cache output-discovery
defect that led to the `v0.6.1` correction. Neither contributes timing data.

Validate the summary, raw results, pair data, output hashes, p95, economics,
fallback, package binding and negative case with:

```bash
./dev/check-magic-end-to-end-value-v2
```

This is POC evidence, not a production or universal-performance claim. Test
Optimization, soak qualification and design-partner evidence remain outside
scope.
