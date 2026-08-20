# BuildOpt value report

**Outcome:** `LEARNING` — `QUALIFIED_PROFILE_STORED`

This report explains what the explicitly invoked BuildOpt POC observed. It is not a production authorization or a promise that the same percentage transfers to another repository.

## Work reduction

BuildOpt compared the full **37-project** graph with a **2-project** selected graph: **35 projects omitted (94.6%)**. Graph reduction alone is not counted as value.

## Measured wall-time value

Across **8 balanced pairs**, optimized native Gradle averaged **61855 ms** and BuildOpt averaged **15398 ms**. The observed installed-path saving was **46456 ms/build (75.1%)**, with a paired 95% interval of **41735 to 50552 ms** and **8/8** positive pairs.
Tail check: native p95 **67440 ms**, BuildOpt p95 **23088 ms** (delta **-44352 ms**). Required outputs were equivalent and launcher overhead was included.

## Learning cost and payback

Calibration cost **2170 ms**. At the observed mean saving, break-even is **1 matching builds** (owner limit: **30**). Successful exact replays counted: **0**. Projected cumulative net value so far: **-2170 ms**, after calibration and **0.000 ms** of selection overhead; projected payback remaining: **2170 ms**.
Expected useful lifetime: **unavailable** — `EXACT_REVISION_REPLAY_HAS_NO_OBSERVED_FUTURE_MATCH_COUNT`. BuildOpt therefore reports payback as a projection, not as realized customer value across future commits.

## Fallback

The current build used **optimized native Gradle**. Exact reason: `QUALIFIED_PROFILE_STORED`. Native result authoritative: **true**; build successful: **true**.

The sibling `value-report.json` contains the same source metrics and every derived value needed to recompute this report. Mechanism percentages are never added together.
