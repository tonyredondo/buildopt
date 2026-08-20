# BuildOpt value report

**Outcome:** `LEARNING` — `QUALIFIED_PROFILE_STORED`

This report explains what the explicitly invoked BuildOpt POC observed. It is not a production authorization or a promise that the same percentage transfers to another repository.

## Work reduction

BuildOpt compared the full **1024-project** graph with a **34-project** selected graph: **990 projects omitted (96.7%)**. Graph reduction alone is not counted as value.

## Measured wall-time value

Across **8 balanced pairs**, optimized native Gradle averaged **74529 ms** and BuildOpt averaged **63376 ms**. The observed installed-path saving was **11154 ms/build (15.0%)**, with a paired 95% interval of **7903 to 14223 ms** and **8/8** positive pairs.
Tail check: native p95 **86925 ms**, BuildOpt p95 **67348 ms** (delta **-19577 ms**). Required outputs were equivalent and launcher overhead was included.

## Learning cost and payback

Calibration cost **201913 ms**. At the observed mean saving, break-even is **19 matching builds** (owner limit: **30**). Successful exact replays counted: **0**. Projected cumulative net value so far: **-201913 ms**, after calibration and **0.000 ms** of selection overhead; projected payback remaining: **201913 ms**.
Expected useful lifetime: **unavailable** — `EXACT_REVISION_REPLAY_HAS_NO_OBSERVED_FUTURE_MATCH_COUNT`. BuildOpt therefore reports payback as a projection, not as realized customer value across future commits.

## Fallback

The current build used **optimized native Gradle**. Exact reason: `QUALIFIED_PROFILE_STORED`. Native result authoritative: **true**; build successful: **true**.

The sibling `value-report.json` contains the same source metrics and every derived value needed to recompute this report. Mechanism percentages are never added together.
