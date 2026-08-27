# Sticky-wrapper longitudinal sample

This directory contains the retained bounded diagnostic sample from
`SWL-015 v1`.
It runs the repository-committed `./buildoptw` command against the same frozen
Gradle workflow as optimized native Gradle on one current first-parent revision
per public repository family.

The sample is permanently `DIAGNOSTIC_ONLY`. In addition to using one
control-first revision per family, the v1 runner injected `--build-cache` only
into control and configured candidate with no server identity and zero trial
budget. Candidate therefore exercised no-op/light observation rather than a
trial or active optimization. It is not eligible for `SWL-016`; the corrected
v2 protocol supersedes it without rewriting these observations. All accepted
pairs still required successful arms, byte-identical required outputs and a
signed wall-time delta.

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

This sample proves that the wrapper can run substantial public Gradle builds
and preserve exact outputs. Its signed deltas are not attributable product
value because the arms had different cache activation and no BuildOpt action
executed. The next measurement does not expand this v1 runner. It first closes
`SWL-014A..D`, then uses the cache-symmetric, lifecycle-aware v2 protocol over
the same frozen public-family breadth.
