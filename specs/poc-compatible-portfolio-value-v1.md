# Compatible Producer Portfolio Value Protocol

## Purpose

This proof-of-concept experiment asks whether producer volatility learned from
two authoritative native observations can remain exact and improve wall time
on a nearby public code revision with the same structural context.

The subject is Micronaut Core `assemble`. A fresh portfolio is learned at
`4dc4299f8dd0faccc0c45c2f83a223b456dc0731` and evaluated at
`a7955f4cc50225044d8eb7c614ba80b607b000dd`. One documentation-only commit lies
between them. The evaluation commit changes production code in `core` plus two
test sources and is the direct child of its declared evaluation base. Learning
and evaluation keep the same frozen Wrapper tree digest.

An earlier preregistered direct child changed only `gradle/libs.versions.toml`.
The ordinary path correctly retained the full graph and therefore produced no
revision-bound materialization to which a producer portfolio could be applied.
That attempt is diagnostic evidence, not a timing result. The replacement
window was selected by change class before observing any performance samples;
the runtime preflight remains mandatory and unchanged.

## Execution order

1. Run the learning revision once through the ordinary optimized-native path.
2. Run one independent native observation at the same revision, learn volatile
   producers and store no historical output bytes.
3. Run the evaluation revision once through the ordinary customer path.
4. Compare exact repository, workflow, Wrapper and observed output-contract
   bindings before cloning or starting its independent native observation.
5. Continue only after `COMPATIBLE / PORTFOLIO_CONTEXT_COMPATIBLE`.
6. Apply the portfolio to two fresh current-revision observations and measure
   eight alternating optimized-native/candidate pairs.

Any context drift returns `NATIVE_RETAINED`, starts no timing and makes no
performance claim.

Compatible context is necessary but not sufficient. When a learned volatile
producer is an intermediate task and the current materialization attributes
only a downstream final producer, BuildOpt must prove the transitive Gradle
producer lineage before transport. Missing lineage returns
`NATIVE_RETAINED / PORTFOLIO_PRODUCER_LINEAGE_UNAVAILABLE`, names the missing
producers, starts no timing and makes no performance claim.

## Value gate

Qualification requires all eight pairs, at least six positive pairs, a
strictly positive saved-time interval, non-regressive candidate p95, at least
500 ms and 2% mean saving, identical required-output digests in every arm, a
successful full-graph fallback and zero product-attributable failures.

Learning cost and evaluation value remain separate. Percentages from this
experiment are never averaged with other repositories or added to other
mechanisms.

## Boundaries

The implementation may use no repository-name branch, relaxed output check or
target-specific threshold. It is POC evidence only and does not authorize
production, soak testing, design-partner work or Test Optimization.

## Observed result

The frozen run binds BuildOpt `7c59e0b` to the preregistered public revisions.
Learning compares 11,187 outputs, quarantines 868 outputs from nine volatile
producers and leaves 10,319 transportable. The evaluation preflight is exactly
compatible, selects 22 of 70 projects and captures 190 outputs/172,543,372
bytes in 2,537 ms after a 623,348-ms ordinary build.

One current output is volatile and 189 are stable, but eight learned
intermediate producers are absent from direct attribution on the final
materialized outputs. The result is therefore
`PORTFOLIO_PRODUCER_LINEAGE_UNAVAILABLE`: native Gradle is retained, zero
timing pairs run and no replay-value percentage is claimed. The next bounded
hypothesis is generic transitive task-producer lineage, not a repository rule
or weaker exact-output gate.
