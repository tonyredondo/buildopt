# Compatible Producer Portfolio Value Protocol

## Purpose

This proof-of-concept experiment asks whether producer volatility learned from
two authoritative native observations can remain exact and improve wall time
on a directly adjacent public revision with the same structural context.

The subject is Micronaut Core `assemble`. A fresh portfolio is learned at
`8e418f75dd7a3aa66bc94786101bc8a2005cb5e2` and evaluated only at its direct
child `4dc4299f8dd0faccc0c45c2f83a223b456dc0731`. The child changes only
`gradle/libs.versions.toml`; both revisions have the same frozen Wrapper tree
digest.

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
