# BuildOpt value report

**Outcome:** `NATIVE_RETAINED` — `CALIBRATION_VALUE_NOT_PROVEN`

This report explains what the explicitly invoked BuildOpt POC observed. It is not a production authorization or a promise that the same percentage transfers to another repository.

## Work reduction

BuildOpt compared the full **27-project** graph with a **10-project** selected graph: **17 projects omitted (63.0%)**. Graph reduction alone is not counted as value.

## Measured wall-time value

Across **8 balanced pairs**, optimized native Gradle averaged **12340 ms** and BuildOpt averaged **9029 ms**. The observed installed-path saving was **3311 ms/build (26.8%)**, with a paired 95% interval of **426 to 8114 ms** and **7/8** positive pairs.
Tail check: native p95 **31122 ms**, BuildOpt p95 **11789 ms** (delta **-19333 ms**). Required outputs were equivalent and launcher overhead was included.

## Learning cost and payback

Calibration cost **339603 ms**. At the observed mean saving, break-even is **103 matching builds** (owner limit: **30**). Successful exact replays counted: **0**. Projected cumulative net value so far: **-339603 ms**, after calibration and **0.000 ms** of selection overhead; projected payback remaining: **339603 ms**.
Expected useful lifetime: **unavailable** — `EXACT_REVISION_REPLAY_HAS_NO_OBSERVED_FUTURE_MATCH_COUNT`. BuildOpt therefore reports payback as a projection, not as realized customer value across future commits.

## Fallback

The current build used **optimized native Gradle**. Exact reason: `CALIBRATION_VALUE_NOT_PROVEN`. Native result authoritative: **true**; build successful: **true**.

The sibling `value-report.json` contains the same source metrics and every derived value needed to recompute this report. Mechanism percentages are never added together.
