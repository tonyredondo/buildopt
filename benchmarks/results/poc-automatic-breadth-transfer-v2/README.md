# Automatic Breadth Transfer V2

This bundle is the terminal five-repository value result for
`POC-AUTOMATIC-BREADTH-TRANSFER-V2-001`. One exact BuildOpt binary ran the
same frozen public repositories and customer workflows as V1 after three
generic changes: ordinary-invocation incremental learning, verified output
materialization and aggregate-workflow partitioning.

No target repository contains a BuildOpt file or repository-name product
rule. Every ordinary invocation first removes ignored build outputs while
preserving the private `.buildopt` learning state. Candidate and optimized
native arms therefore start from the same clean-workspace condition. The
candidate must reproduce the exact required-output digest and retain working
full-graph fallback.

## Result

| Repository / workflow | Graph | Native mean | Candidate mean | Mean saving | 95% saving interval | Positive pairs | Learning / payback | Decision |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Spring Framework `testClasses` | 27 -> 10 | 10.456 s | 9.127 s | **1.329 s / 12.71%** | +0.282..+2.572 s | 7/8 | 88.668 s / 67 builds | Native retained. |
| OpenTelemetry Spring family | 1,024 -> 34 | 74.529 s | 63.376 s | **11.154 s / 14.97%** | +7.903..+14.223 s | 8/8 | 201.913 s / 19 builds | Qualified. |
| Apache Kafka `testClasses` | 64 -> 3 | 9.774 s | 4.406 s | **5.368 s / 54.92%** | +3.513..+7.337 s | 8/8 | 70.808 s / 14 builds | Qualified. |
| Micronaut Core `assemble` | 75 -> 22 | 23.108 s | 7.802 s | **15.306 s / 66.24%** | +14.245..+16.498 s | 8/8 | 114.284 s / 8 builds | Qualified. |
| Apache Groovy `classes` | 37 -> 2 | 65.652 s | 15.776 s | **49.876 s / 75.97%** | +45.842..+53.680 s | 8/8 | 73.857 s / 2 builds | Qualified. |

All 85 ordinary invocations completed with zero product-attributable failures.
All five discovery, incremental-learning, aggregate-partition and
materialization records are complete. Output equivalence is exact in every
observation. Four repositories pass the unchanged gates: at least 500 ms and
2% mean saving, a positive paired interval, 8/8 positive pairs, working
fallback and at most 30 matching builds to repay learning.

Spring is intentionally not promoted. Its candidate is faster on average and
its interval is positive, but only 7/8 pairs improve and the observed learning
cost needs 67 matching builds to repay. The correct decision is therefore
`NATIVE_RETAINED / CALIBRATION_VALUE_NOT_PROVEN`.

Repository percentages are not averaged and mechanism percentages are not
added. The run used the 12-CPU development host with a common 12-worker cap;
it is bounded POC evidence, not production or contractual golden-runner
evidence.

## Materialization overhead finding

The first Spring diagnostic exposed a generic implementation defect: syncing
every blob and destination directory for every restored file made 14,445
small files add about 629 seconds of wrapper overhead. The retained
implementation keeps content hashes and atomic renames, but batches directory
durability after all writes. The first post-fix Spring pair reduced candidate
wrapper overhead to about 2.7 seconds, more than 99% lower. A crash may make a
payload unavailable, but the next digest check rejects it and runs native;
unverified output can never be accepted.

## Recompute

The strict checker validates all 85 invocation documents, repository and
executable bindings, state-tree hashes, materialization manifests, exact
outputs, pair order, means, p95 values, intervals, payback and terminal
decisions:

```bash
./dev/check-automatic-breadth-transfer-v2 \
  benchmarks/results/poc-automatic-breadth-transfer-v2/summary.json
```

The expensive public-repository capture is intentionally not part of normal
CI. Base CI validates the checked-in evidence and focused implementation tests.
The underlying protocol is defined in
[`poc-automatic-breadth-transfer-v2.md`](../../../specs/poc-automatic-breadth-transfer-v2.md).

## Scope

- POC only; no production authority, SLO, soak or design-partner requirement.
- Test Optimization remains out of scope.
- No repository-specific product branch, threshold movement or hidden manual
  profile is introduced.
- V1 remains immutable diagnostic evidence for the before/after comparison.
