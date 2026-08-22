# Automatic POC qualification

This contract defines the value gate used by the zero-input
`buildopt optimize` learning path. It exists to decide whether a measured
candidate is useful enough to retain inside the proof of concept. It does not
authorize production activation.

## Observation design

The automatic path records eight fresh control/candidate pairs. Pair order
alternates between `CONTROL_FIRST` and `CANDIDATE_FIRST`; no failed, timed-out,
or unfavorable pair may be discarded. Every pair must preserve the same
required-output digest and stable execution shape, and the full native graph
must succeed as a fallback.

This is a single alternating capture. It is not the historical
[`Balanced Statistical Qualification v2`](./poc-statistical-qualification-v2.md),
which combines two independent captures into eight reciprocal blocks. Evidence
and policy identifiers keep the two methods distinct.

## Qualification gate

The automatic POC may retain a profile only when all of these are true:

- mean wall-time saving is at least 500 ms and 2%;
- the deterministic paired-bootstrap 95% lower bound is strictly positive;
- at least six of the eight alternating pairs are positive;
- candidate p95 wall time does not exceed optimized native Gradle p95;
- required outputs and execution shape are stable;
- full native fallback succeeds;
- projected qualification payback is within the configured maximum; and
- no product-attributable failure occurs.

Six positive observations alone are insufficient. A non-positive lower bound,
regressive p95, output drift, shape drift, failed fallback, or uneconomic
payback still retains optimized native Gradle.

## Evidence boundary

The policy identifier is
`ROBUST_6_OF_8_ALTERNATING_PAIRS_INTERVAL_P95_V1`. Historical evidence using
`ROBUST_7_OF_8_POSITIVE_INTERVAL_P95_V1` remains valid under its original
contract and is never reinterpreted.

A diagnostic result captured before this contract may motivate the method but
cannot qualify a profile. The first accepted evidence must be a fresh run from
an exact committed BuildOpt executable. Repository percentages are never
averaged and mechanism percentages are never added.
