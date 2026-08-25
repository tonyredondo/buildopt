# Adaptive fragment longitudinal matrix v1

This protocol closes `AF-013` by normalizing the already measured chronological
control/candidate arms for the five frozen public repository families. It does
not rerun or improve those observations and therefore makes no fresh timing
claim. Reusing the immutable raw measurements avoids selecting a favorable
rerun after seeing earlier results.

This is historical audit evidence. It does not represent the current adaptive
implementation and cannot satisfy the AF-014A..D current installed-package
campaign or the AF-015 terminal scorecard.

The machine contract is
[`poc-adaptive-fragment-longitudinal-v1.json`](./poc-adaptive-fragment-longitudinal-v1.json).
Each row names one source document and source schema. The assembler accepts
only those declared evidence schemas, verifies the source digest, preserves
every signed build delta and charges the recorded qualification/publication
cost once.

## Decision order

Rows are chronological and prequential. Observation `N` records
`maxSourceSequence = N - 1`; no later observation may authorize an earlier
decision. The optimized-native control and BuildOpt candidate use the same
requested workflow and the raw evidence must report exact required outputs and
zero product-attributable failures.

The normalized cumulative value is:

```text
cumulativeNetMs(N) =
  sum(controlWallMs[i] - candidateWallMs[i], i = 1..N)
  - qualificationAndPublicationCostMs
```

The paired wall-time delta already contains synchronous BuildOpt overhead and
must not be charged a second time. Percentages from calibration or isolated
fragment experiments are never added to the longitudinal result.

## Row outcomes

- `NET_POSITIVE`: the complete comparable sequence has positive cumulative
  net value;
- `NET_NEGATIVE`: the complete comparable sequence has non-positive cumulative
  net value; and
- `INCONCLUSIVE`: at least one attempted descendant lacks a comparable direct
  delta. The reason remains visible and no value is inferred.

An inconclusive row is closed evidence, not a pass. In particular, Micronaut's
native-output nondeterminism preserved the exact-output safety gate but did not
produce an attributable performance delta.

## Reproduction

```bash
./dev/check-adaptive-fragment-longitudinal
```

The checker regenerates the normalized report from the declared raw sources,
compares canonical JSON bytes with the checked-in result and then applies
independent semantic and mutation checks.

This remains POC evidence. It does not authorize production use, Test
Optimization, a soak or a design-partner gate.
