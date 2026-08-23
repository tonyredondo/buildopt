# Transitive Producer Lineage Protocol

## Purpose

This proof-of-concept experiment tests whether Gradle's observed task graph can
propagate volatile intermediate producers to every downstream materialized
output, rebuild the resulting exact producer frontier and safely rerun the
frozen Micronaut compatible-portfolio window.

The subject, revisions, `assemble` workflow, Wrapper binding, output contract,
eight alternating pairs and value gate remain unchanged from
[`poc-compatible-portfolio-value-v1.md`](./poc-compatible-portfolio-value-v1.md).

## Lineage contract

For each materialized output, BuildOpt records:

1. the direct Gradle producer task;
2. the complete transitive task dependency lineage observed in that build;
3. the owning project for every recorded task; and
4. the lineage-bound manifest digest used by transport and replay.

A producer learned as volatile quarantines an output when it is either the
direct producer or appears anywhere in that output's lineage. Incomplete,
ambiguous, cyclic or contradictory task graphs retain native Gradle before
transport.

Quarantine changes execution as well as transport. Every quarantined output's
direct producer enters the rebuild frontier. BuildOpt may replace a set of
direct producers with the owning project's lifecycle entrypoint only when the
same observed task graph proves that entrypoint transitively covers every
producer in that set. Otherwise the direct tasks remain explicit. The
implementation may not branch on repository identity or known Micronaut task
names.

## Acceptance

Correctness requires:

- every transported output to exclude all volatile direct and transitive
  producers;
- every quarantined output to be reproduced locally;
- the exact required-output digest to match optimized native Gradle in every
  pair;
- a successful full-graph fallback; and
- zero product-attributable failures.

Performance qualification remains unchanged: eight pairs, at least six
positive pairs, a strictly positive saved-time interval, non-regressive p95,
at least 500 ms and 2% mean saving.

## Observed result

The first lineage-aware candidate quarantined the correct 89 outputs but
rebuilt only 11 entrypoints across eight projects. It returned
`REQUIRED_OUTPUT_DRIFT` and recovered through the full graph. The corrected
frontier uses 58 entrypoints across 52 projects and completes all eight pairs
with an identical required-output digest and zero product failures.

The final candidate averages 13,253.25 ms against 13,318.25 ms optimized
native, a 65-ms/0.49% saving. Only five pairs improve, the 95% interval is
-1,114.375..+1,166.75 ms and candidate p95 regresses from 14,267 to 16,967 ms.
The terminal decision is therefore
`COMPATIBLE_PORTFOLIO_VALUE_NOT_PROVEN`.

The next bounded hypothesis is a smaller exact rebuild frontier and lower
verification overhead, not a weaker output or statistical gate.

## Boundaries

This protocol is POC-only. It adds no production authority, soak or
design-partner requirement, repository-specific product rule or Test
Optimization behavior. Percentages remain repository- and experiment-specific
and are never added across mechanisms.
