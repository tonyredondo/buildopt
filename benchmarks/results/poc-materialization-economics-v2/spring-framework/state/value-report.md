# BuildOpt value report

**Outcome:** `LEARNING` — `QUALIFIED_PROFILE_STORED`

This report explains what the explicitly invoked BuildOpt POC observed. It is not a production authorization or a promise that the same percentage transfers to another repository.

## Work reduction

BuildOpt compared the full **27-project** graph with a **10-project** selected graph: **17 projects omitted (63.0%)**. Graph reduction alone is not counted as value.

## Measured wall-time value

Across **8 balanced pairs**, optimized native Gradle averaged **11062 ms** and BuildOpt averaged **9960 ms**. The observed installed-path saving was **1102 ms/build (10.0%)**, with a paired 95% interval of **369 to 2166 ms** and **8/8** positive pairs.
Tail check: native p95 **16081 ms**, BuildOpt p95 **11546 ms** (delta **-4535 ms**). Required outputs were equivalent and launcher overhead was included.

## Learning cost and payback

Calibration cost **4070 ms**. At the observed mean saving, break-even is **4 matching builds** (owner limit: **30**). Successful exact replays counted: **0**. Projected cumulative net value so far: **-4070 ms**, after calibration and **0.000 ms** of selection overhead; projected payback remaining: **4070 ms**.
Expected useful lifetime: **unavailable** — `EXACT_REVISION_REPLAY_HAS_NO_OBSERVED_FUTURE_MATCH_COUNT`. BuildOpt therefore reports payback as a projection, not as realized customer value across future commits.

## Fallback

The current build used **optimized native Gradle**. Exact reason: `QUALIFIED_PROFILE_STORED`. Native result authoritative: **true**; build successful: **true**.

The sibling `value-report.json` contains the same source metrics and every derived value needed to recompute this report. Mechanism percentages are never added together.
