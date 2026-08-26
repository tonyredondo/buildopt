# Adaptive fragment terminal decision v1

Status: accepted terminal POC decision contract for `AF-015`.

This contract closes the adaptive-fragment generalization hypothesis against
the current installed-package campaign. It does not rerun builds, merge older
source-checkout measurements into the current scorecard or move a threshold
after seeing the result.

The checker binds the immutable AF-014C raw observations, deterministic report,
AF-014D attribution and campaign protocol by SHA-256. It then recomputes all 15
criteria from section 4 of the adaptive-fragment tracker. Per-family confidence
uses a deterministic 10,000-sample paired bootstrap with a fixed seed and the
one-sided fifth-percentile lower bound. Cumulative values at builds 1, 5, 10,
15 and 20 remain visible for every family.

The outcomes are deliberately asymmetric:

- `CONTINUE_ADAPTIVE_FRAGMENT_POC` requires every criterion to pass.
- `SPECIALIZE_BOUNDED_FRAGMENT_CLASSES` requires correctness plus observed
  bounded value: at least one real fragment activation, positive attributable
  saving and at least one positive family.
- `STOP_ADAPTIVE_FRAGMENT_POC` applies when the new unit of learning has no
  defensible generic or bounded current value.

Passing correctness, cohort integrity or fail-safe native retention cannot by
itself rescue the value hypothesis. Historical AF-013 observations are retained
as context only and are explicitly marked `decisionInput=false`.

This is a POC decision. It authorizes neither production rollout nor Test
Optimization, soak testing or design-partner work.
