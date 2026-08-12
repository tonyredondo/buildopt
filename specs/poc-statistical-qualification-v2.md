# Balanced Statistical Qualification v2

This POC contract measures whether a repository-neutral Build Impact candidate
materially reduces wall time compared with the same optimized native Gradle
workflow. It replaces the v1 requirement that every individual pair win with
an order-balanced, independently repeated decision. It does not weaken output,
execution-shape, failure, or fallback guarantees.

The five public subjects, revisions, mutations, toolchains, entrypoints and
required outputs remain frozen by
[`poc-generic-profile-matrix-v4.json`](./poc-generic-profile-matrix-v4.json).
The v2 result must be produced by one exact BuildOpt revision and executable.

## Capture design

Each public repository receives two fresh, independent captures. Each capture
contains eight paired observations in alternating order:

```text
control -> candidate
candidate -> control
```

One balanced block is the mean saving of those adjacent observations. Two
captures therefore produce 16 raw pairs and eight balanced blocks. Warm-ups,
dependency preparation and full-graph fallback validation remain outside the
timed effect. No failed or timed-out observation may be discarded.

The v1 capture documents stay immutable, including their original decisions.
The v2 aggregate binds both capture SHA-256 digests and independently
revalidates subject, revision, plan, binary, options, exact outputs, task shape,
host diagnostics and full-graph fallback before calculating value.

## Qualification

A repository qualifies for a reviewable POC profile only when all of these are
true:

- mean saving is at least 500 ms and 2%;
- median balanced-block saving is positive;
- the deterministic 95% bootstrap lower bound over blocks is positive;
- at least six of eight balanced blocks are positive;
- candidate p95 wall time does not exceed optimized native Gradle p95;
- required outputs remain byte-identical across every observation;
- exact task shapes remain stable;
- both full-graph fallbacks succeed; and
- no product-attributable failure occurs.

The result also reports the difference between control-first and
candidate-first mean savings. It is diagnostic rather than a qualification
shortcut. Percentages are never averaged across repositories or added across
mechanisms.

## POC boundary

The result can validate the idea and materialize reviewable evidence. It does
not authorize production, automatic activation, a soak, a design partner, or
changes to Test Optimization. A non-winning or unavailable repository remains
on optimized native Gradle.
