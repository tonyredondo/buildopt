# BuildOpt value report

**Outcome:** `NATIVE_RETAINED` — `CALIBRATION_VALUE_NOT_PROVEN`

This report explains what the explicitly invoked BuildOpt POC observed. It is not a production authorization or a promise that the same percentage transfers to another repository.

## Work reduction

BuildOpt compared the full **37-project** graph with a **30-project** selected graph: **7 projects omitted (18.9%)**. Graph reduction alone is not counted as value.

## Measured wall-time value

Across **8 balanced pairs**, optimized native Gradle averaged **71480 ms** and BuildOpt averaged **69472 ms**. The observed installed-path saving was **2008 ms/build (2.8%)**, with a paired 95% interval of **498 to 3634 ms** and **7/8** positive pairs.
Tail check: native p95 **78708 ms**, BuildOpt p95 **72610 ms** (delta **-6098 ms**). Required outputs were equivalent and launcher overhead was included.

## Learning cost and payback

Calibration cost **1423987 ms**. At the observed mean saving, break-even is **710 matching builds** (owner limit: **30**). Successful exact replays counted: **0**. Projected cumulative net value so far: **-1423987 ms**, after calibration and **0.000 ms** of selection overhead; projected payback remaining: **1423987 ms**.
Expected useful lifetime: **unavailable** — `EXACT_REVISION_REPLAY_HAS_NO_OBSERVED_FUTURE_MATCH_COUNT`. BuildOpt therefore reports payback as a projection, not as realized customer value across future commits.

## Fallback

The current build used **optimized native Gradle**. Exact reason: `CALIBRATION_VALUE_NOT_PROVEN`. Native result authoritative: **true**; build successful: **true**.

The sibling `value-report.json` contains the same source metrics and every derived value needed to recompute this report. Mechanism percentages are never added together.
