# One-command end-to-end value evidence

## Purpose

This contract evaluates the current customer-shaped POC path:

```text
install package -> open a Gradle repository -> buildopt optimize <workflow>
```

The repository supplies no BuildOpt manifest, graph, profile, output contract,
or repository-specific rule. BuildOpt must derive a candidate from Git and
Gradle facts, compare the complete installed path with optimized native Gradle,
verify the required outputs and fallback, and keep native Gradle when value or
semantics are not proven.

## Terminal gate

Completion requires a published package and fresh install-to-decision captures
where at least two different Gradle repository families produce exact outputs,
pass all eight balanced pairs, improve mean and p95, and repay calibration
within the owner-declared maximum. At least one honest unsupported or negative
case must retain native Gradle. Mechanism percentages and repository results
are never averaged or added.

## Current evidence decision

The first automatic matrix is diagnostic rather than terminal. Ktor qualifies
at 81.995% lower measured wall time and a 27-build break-even. Spring and Beam
both pass the direct value gate, but their 328- and 37-build payback exceeds the
declared maximum of 30. Groovy and Kafka are slower, and Micronaut retains
native before timing because root `assemble` output semantics remain too broad.

The matrix therefore proves safe automatic discovery, exact calibration and
honest fallback across substantial public repositories, but it does not yet
prove a generally valuable zero-configuration customer experience. Only one
of the required two families is economically qualified. The next experiment
must reduce generic first-decision cost, especially immutable dependency
snapshotting and base-cache warming, before publishing a release and repeating
the terminal captures from fresh state.

The source results and recomputable summary are stored under
[`benchmarks/results/poc-magic-end-to-end-value-v1`](../benchmarks/results/poc-magic-end-to-end-value-v1/README.md).
The exact machine contract is
[`poc-magic-end-to-end-value-v1.json`](./poc-magic-end-to-end-value-v1.json).

## POC boundary

This evidence does not authorize production activation, cross-revision
inference, repository-name rules, Test Optimization, soak testing, or a design
partner dependency. It does authorize continued POC work on generic automatic
discovery, calibration economics, graph precision and fail-closed replay.
