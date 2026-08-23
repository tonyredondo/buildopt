# Transitive Producer Lineage Result

This proof-of-concept result closes the correctness gap found when a volatile
Gradle task is upstream of a final output that BuildOpt would otherwise
transport. It uses the same frozen Micronaut Core `assemble` window, value
gate, exact-output contract and optimized-native baseline as the preceding
compatible-portfolio experiment.

## What changed

BuildOpt now records the exact Gradle task dependency graph alongside every
materialized output. A volatile producer quarantines all downstream outputs
whose lineage contains that task. Missing, ambiguous, cyclic or contradictory
lineage retains native Gradle.

The first implementation correctly quarantined 89 outputs and left 101
transportable, but rebuilt only the directly volatile tasks. That eight-project
candidate could not reproduce the required output set, so the exact-output
gate returned `REQUIRED_OUTPUT_DRIFT` and a full-graph recovery produced the
expected digest.

The correction derives a generic rebuild frontier from every quarantined
output's direct producer. It uses a project lifecycle task only when the
observed Gradle graph proves that task covers every required direct producer;
otherwise it keeps the direct producer explicit. No repository name, task
name allowlist or Micronaut-specific path is used.

## Final result

The corrected replay completes all eight alternating pairs with one identical
required-output digest and zero product-attributable failures:

| Measure | Optimized native | BuildOpt candidate |
| --- | ---: | ---: |
| Mean wall time | 13,318.25 ms | 13,253.25 ms |
| p95 wall time | 14,267 ms | 16,967 ms |
| Graph | 70 projects | 52 selected / 18 omitted |

- Mean saving: **65 ms / 0.49%**.
- Positive pairs: **5/8**.
- 95% saved-time interval: **-1,114.375..+1,166.75 ms**.
- Candidate rebuild frontier: **58 entrypoints**, down from 68 before exact
  materialization, but much wider than the incorrect 11-entrypoint frontier.
- Transport partition: **101 outputs transported, 89 quarantined**.
- Calibration cost: **1,853 ms**; projected break-even: **29 builds**.
- Full-graph fallback: successful, with required-output SHA-256
  `c737a1b9c094e10672dc51c65cad73df02d238882c8a72b5c87f235c8ae8a8d4`.

The terminal decision is
`COMPATIBLE_PORTFOLIO_VALUE_NOT_PROVEN`. Producer lineage and rebuild safety
are proven for this frozen window; repeatable wall-time value is not. The p95
regression and interval crossing zero prevent qualification.

## Evidence and revalidation

- [`pre-fix-summary.json`](./pre-fix-summary.json) preserves the first
  lineage-aware run.
- [`diagnostic.json`](./diagnostic.json) binds the exact failed recovery, final
  frontier and all 16 terminal timing observations.
- [`summary.json`](./summary.json) is the terminal aggregate.

```bash
./dev/check-transitive-producer-lineage
```

This is bounded POC evidence. It does not authorize production activation,
soak or design-partner work, repository-specific rules, averaged percentages,
added mechanism percentages or Test Optimization.
