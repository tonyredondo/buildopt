# Current longitudinal attribution v1

## Purpose

This contract closes `AF-014D` by deriving mechanism value and fallback cost
from the current installed-package campaign. It does not reinterpret the
historical `AF-013` evidence and does not authorize production use.

The attribution is intentionally conservative. A favorable wall-time delta is
not credited to BuildOpt unless the same observation records an activated
profile or fragment. Native retention can therefore have measured cost while
having zero attributable mechanism saving.

## Inputs and identity

The derivation consumes the committed `AF-014C` raw evidence, its deterministic
aggregate report and this machine contract. The output records the SHA-256 of
all three inputs. The checker regenerates the complete document and requires
semantic equality; hand-edited summaries are rejected.

Workflow classes are derived from the Gradle task expression using the ordered
generic rules in the JSON contract. Repository identity never changes a class
or mechanism result.

## Equations

For every comparable pair:

```text
signed delta = control wall time - candidate wall time
recorded BuildOpt cost = candidate wall time - recorded Gradle execution time
residual Gradle/runner delta = signed delta + recorded BuildOpt cost
```

Recorded cost is split further using the nested diagnostics. A positive gap
between external candidate wall time and BuildOpt's internal total is reported
as `externalGapNs`. Any remaining positive recorded time is reported as
`otherRecordedBuildOptNs`. These labels describe where time was observed; they
do not prove that removing the interval would produce the same wall-time gain.

Cold-start value is the first comparable pair in each repository. Continuing
value is the sum of every later pair. Native-retention p50 and p95 use
nearest-rank over the per-pair recorded BuildOpt cost.

## Attribution rules

- `CURRENT_VALUE_ATTRIBUTED` requires at least one selected profile or
  activated fragment in the current campaign.
- `CURRENT_VALUE_NOT_ATTRIBUTABLE` is mandatory when all observations retain
  native Gradle before activation.
- An inactive mechanism has zero attributable savings, even when individual
  pairs are faster.
- Percentages from different repositories or workloads are never added.
- Dependency preparation remains outside both measured arms and is reported
  separately.
- Calibration cost and payback are reported only when actual calibration was
  performed.

## POC boundary

This is short, current, public-repository POC evidence. It requires neither a
soak nor a design partner, creates no universal improvement claim, and leaves
Test Optimization out of scope. `AF-015` owns the terminal adaptive-fragment
decision; this contract only makes its inputs recomputable.
