# Producer-Bound Quarantine Lifetime Protocol

## Purpose

This protocol tests whether a structurally qualified profile can create value
on later public commits after volatile native outputs are converted into a
producer-atomic local-rebuild boundary. It is a bounded POC experiment, not a
production activation policy.

## Qualification

The automatic POC path measures eight alternating control/candidate pairs and
qualifies only when all of the following hold:

- mean saving is at least 500 ms and 2%;
- the deterministic paired 95% lower bound is strictly positive;
- at least six of eight pairs improve;
- candidate p95 is no worse than optimized native Gradle;
- required outputs, fallback and task shape are exact;
- expected payback is no more than 30 matching builds; and
- no product-attributable failure occurs.

This is the explicit
`ROBUST_6_OF_8_ALTERNATING_PAIRS_INTERVAL_P95_V1` automatic POC policy. It does
not modify the historical qualified-lifetime V2 contract, whose immutable
subjects retain their 7/8 policy and original specification SHA.

## Producer-atomic portability

Two independent native observations bind each output path to its executed
Gradle producer. If any output differs, every output of that producer is
quarantined. Stable outputs may be transported only when their complete path
inventory and SHA-256 match exactly. Quarantined paths must exist after both
arms and must be rebuilt locally; their bytes are not asserted across roots.
Missing attribution, paths or stable bytes retain native Gradle.

## Lifetime observations

The frozen public Spring subject qualifies on one `spring-jms` change and then
observes two first-parent descendants. Each observation alternates arm order,
uses isolated Gradle state, verifies the stable and quarantined output sets,
and records profile selection or optimized-native retention before reporting
cumulative economics.

The result is successful when compatible descendants may select a profile,
incompatible descendants retain native Gradle, every output boundary holds,
and product-attributable failures remain zero. A negative native-retained arm
difference remains visible but is not attributed as an optimization saving.

## Boundaries

The result does not authorize production activation, average repository
percentages, add mechanism percentages, require soak or a design partner, or
include Test Optimization.
