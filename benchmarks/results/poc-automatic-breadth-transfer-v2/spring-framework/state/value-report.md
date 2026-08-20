# BuildOpt value report

**Outcome:** `NATIVE_RETAINED` — `CALIBRATION_VALUE_NOT_PROVEN`

This report explains what the explicitly invoked BuildOpt POC observed. It is not a production authorization or a promise that the same percentage transfers to another repository.

## Work reduction

BuildOpt compared the full **27-project** graph with a **10-project** selected graph: **17 projects omitted (63.0%)**. Graph reduction alone is not counted as value.

## Measured wall-time value

Across **8 balanced pairs**, optimized native Gradle averaged **10456 ms** and BuildOpt averaged **9126 ms**. The observed installed-path saving was **1329 ms/build (12.7%)**, with a paired 95% interval of **282 to 2572 ms** and **7/8** positive pairs.
Tail check: native p95 **15411 ms**, BuildOpt p95 **10689 ms** (delta **-4722 ms**). Required outputs were equivalent and launcher overhead was included.

## Learning cost and payback

Calibration cost **88668 ms**. At the observed mean saving, break-even is **67 matching builds** (owner limit: **30**). Successful exact replays counted: **0**. Projected cumulative net value so far: **-88668 ms**, after calibration and **0.000 ms** of selection overhead; projected payback remaining: **88668 ms**.
Expected useful lifetime: **unavailable** — `EXACT_REVISION_REPLAY_HAS_NO_OBSERVED_FUTURE_MATCH_COUNT`. BuildOpt therefore reports payback as a projection, not as realized customer value across future commits.

## Fallback

The current build used **optimized native Gradle**. Exact reason: `CALIBRATION_VALUE_NOT_PROVEN`. Native result authoritative: **true**; build successful: **true**.

The sibling `value-report.json` contains the same source metrics and every derived value needed to recompute this report. Mechanism percentages are never added together.
