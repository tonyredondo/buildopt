# Ordinary-build learning economics v1

## Purpose

This POC may learn only from builds the user already requested. It must not run
extra customer builds solely to improve its own evidence. Before continuing
paired calibration, it uses structural recurrence in bounded first-parent Git
history to estimate whether the candidate is likely to remain compatible long
enough to repay its learning cost.

The recurrence estimate is a planning signal, not performance evidence. Direct
ordinary builds provide duration, graph reduction, success, exact-output,
portability and native-volatility observations. Every observation is bound to
the same structural profile fingerprint.

## Decisions

1. Fewer than five compatible historical matches retains native Gradle before
   paired calibration.
2. The first complete ordinary control/candidate pair must project positive
   value and repay all learning cost within five compatible matches.
3. Non-positive savings, payback beyond five matches, output drift, structural
   drift or a product-attributable failure retains native Gradle.
4. A positive economic probe authorizes only continued ordinary-build
   learning. It does not qualify the candidate.
5. Qualification still requires eight alternating pairs and the unchanged
   robust value, lower-bound and p95 gates.

## POC boundary

This contract proves bounded learning economics and fail-closed integration.
It makes no new wall-time claim, creates no measurement-only builds, grants no
production authority and requires neither soak nor design-partner evidence.
Test Optimization remains outside scope.

Validate the contract and committed evidence with:

```bash
./dev/check-ordinary-learning-economics
```
