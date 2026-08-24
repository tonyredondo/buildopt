# Cross-commit breadth V2 result

This bundle is the terminal evidence for the preregistered
[cross-commit breadth V2 protocol](../../../specs/poc-cross-commit-breadth-v2.md).
It asks whether the unchanged generic producer-bound profile can create
cumulative value on three later public-commit windows, not merely accelerate
the commit on which it was calibrated.

## Result

All three target changes qualify with exact portable outputs, but none of the
three subjects finishes its descendant window claim-eligible.

| Repository / workflow | Target calibration | Portable outputs | Later replay | Three-build net |
| --- | ---: | ---: | --- | ---: |
| OpenTelemetry Spring family | **8.53%**, 8/8 positive | 248 | 0 selected / 3 native | **-168.751 s** |
| Ktor `jvmJar` | **56.33%**, 8/8 positive | 116 | 1 selected / 2 native | **-52.237 s** |
| Groovy JSON `classes` | **10.56%**, 8/8 positive | 3,823 | 0 selected / 3 native | **-37.684 s** |

Ktor's first descendant is a real success in isolation: it selects the
profile, preserves all required outputs and changes 216.096 seconds of
optimized native Gradle into 100.066 seconds, saving **116.030 seconds / 53.69%**.
The next two commits retain native and their paired arm deltas are -93.421 and
-59.263 seconds. Those fallbacks plus 15.583 seconds of qualification and
publication cost make the complete window negative.

Across the matrix, the three qualifications are positive, all three output
sets are portable, one of nine descendants selects a replay, zero of three
subjects pays back, and product failures remain zero. The signed
subject-window deltas total -258.672 seconds, but repository percentages are
not averaged and the total is not a universal performance estimate.

## Interpretation

The experiment separates target-build potential from customer value across
commits. Structural graph reduction is still material on each calibrated
change. The missing value is breadth and fallback economics: eight of nine
later revisions cannot safely select the learned profile, and executing the
BuildOpt path before native retention is too expensive in these observations.

The cross-commit value claim therefore remains bounded to the previously
proven Kafka and Spring windows. The next generic hypothesis should make a
native-retained decision close to native cost and improve compatibility
without weakening exact outputs, rather than adding another isolated
calibration or a repository-specific rule.

## Validate

```bash
./dev/check-cross-commit-breadth-v2 \
  benchmarks/results/poc-cross-commit-breadth-v2
```

`summary.json` is deterministically assembled from `raw/summary.json`. Each
subject directory retains the qualification capture, paired calibration,
terminal result and exact transported-output list. Large source checkouts,
Gradle caches and logs are excluded.

This is bounded 12-CPU POC evidence. It is not a production, soak,
design-partner or Test Optimization claim.
