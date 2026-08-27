# Sticky-wrapper longitudinal sample

This directory contains the first bounded diagnostic sample for `SWL-015`.
It runs the repository-committed `./buildoptw` command against the same frozen
Gradle workflow as optimized native Gradle on one current first-parent revision
per public repository family.

The sample is intentionally incomplete: `SWL_MAX_PRIMARY_PER_FAMILY=1` was
used to obtain a fast signal while preserving the frozen cohort and protocol.
It is not eligible for the `SWL-016` terminal decision, which requires at
least 15 comparable pairs in every family. All accepted pairs still require
successful arms, byte-identical required outputs, and a signed wall-time delta.

## Observed result

| Repository | Control | BuildOpt wrapper | Signed delta | Relative result | Output | Pair |
| --- | ---: | ---: | ---: | ---: | --- | --- |
| Spring Framework | 283.944 s | 289.226 s | -5.282 s | -1.86% | Exact | Negative |
| OpenTelemetry Java Instrumentation | 512.194 s | 517.979 s | -5.786 s | -1.13% | Exact | Negative |
| Apache Kafka | 210.889 s | 205.213 s | +5.676 s | +2.69% | Exact | Positive |
| Micronaut Core | 494.033 s | 511.159 s | -17.126 s | -3.47% | Exact | Negative |
| Apache Groovy | 128.134 s | 127.765 s | +0.370 s | +0.29% | Exact | Positive |

The signed delta is `control wall time - BuildOpt wall time`; positive means
that BuildOpt was faster. The portfolio total is **-22.149 s**, with **2/5
positive pairs**, **3/5 negative pairs**, exact required outputs in **5/5**
pairs, and **0 product failures**. Every pair ran `CONTROL_FIRST`, so this
bounded sample is diagnostic rather than a balanced estimate.

Dependency preparation was recorded separately and excluded from pair wall
time. Each arm used a separate worktree, Gradle home, cache root and BuildOpt
state. The archive was `BuildOpt 0.15.0` with SHA-256
`d69302cad3d37116e59a32417f8acaec64685df0c27546ab699a0ef6a74a5e28`.

Validate the machine-readable evidence with:

```bash
./dev/check-sticky-wrapper-longitudinal \
  benchmarks/results/poc-sticky-wrapper-longitudinal-sample-v1
```

The raw and derived files are retained separately so later work can reproduce
the arithmetic without rerunning the repositories:

- [`raw.json`](./raw.json)
- [`report.json`](./report.json)

## Interpretation

This sample proves that the wrapper can run substantial public Gradle builds,
preserve exact outputs and expose a measurable comparison. It does **not** yet
show a reliable general improvement: the net result is negative and the
sample has only one order, one revision per family and no longitudinal reuse.
The next measurement must expand the same frozen cohort to at least 15
comparable pairs per family, alternate arm order, and retain every positive
and negative observation before deciding whether the idea creates durable
customer value.
