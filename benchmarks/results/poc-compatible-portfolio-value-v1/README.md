# Compatible Producer Portfolio Value Result

This proof-of-concept result evaluates whether a producer-volatility portfolio
learned on one Micronaut Core `assemble` revision can safely accelerate a
nearby source-code revision with the same repository, workflow, Gradle Wrapper
and runtime output-contract bindings.

## Result

The structural opportunity is large, but replay value is not yet measurable.

- Learning compares 11,187 outputs. It quarantines 868 outputs from nine
  volatile producers and leaves 10,319 outputs transportable.
- The evaluation change selects 22 of 70 projects and omits 48.
- The first evaluation build captures 190 unaffected outputs totaling
  172,543,372 bytes in 2,537 ms. The ordinary build takes 623,348 ms.
- The runtime preflight is exactly `COMPATIBLE` across repository, workflow,
  Wrapper and output-contract bindings.
- Independent evaluation observes 189 stable outputs and one volatile
  `micronaut-jackson-databind` JAR.
- Eight learned intermediate producers do not appear in the direct producer
  attribution of the final materialized outputs. BuildOpt therefore returns
  `NATIVE_RETAINED / PORTFOLIO_PRODUCER_LINEAGE_UNAVAILABLE` and starts no
  timing pairs.

This is not a performance failure and not a performance success. It proves
that compatibility alone is insufficient: safe cross-revision transport also
needs transitive Gradle producer lineage from volatile intermediate tasks to
the final outputs that BuildOpt would restore. Ignoring the missing producers
could transport JARs derived from volatile compilation, so the POC retains
native Gradle.

An earlier global version-catalog window also retained the full graph and had
no revision-bound materialization. It is diagnostic evidence only and is not a
timing result.

## Revalidation

```bash
./dev/check-compatible-portfolio-value \
  benchmarks/results/poc-compatible-portfolio-value-v1/summary.json
```

The machine-readable [summary](./summary.json) binds the public revisions,
exact executable, compatibility preflight, native observations, missing
producer list and POC boundaries. Percentages are neither averaged across
repositories nor added across mechanisms. No production, soak,
design-partner or Test Optimization authority follows.
