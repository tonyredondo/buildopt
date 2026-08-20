# BuildOpt value report

**Outcome:** `LEARNING` — `QUALIFIED_PROFILE_STORED`

This report explains what the explicitly invoked BuildOpt POC observed. It is not a production authorization or a promise that the same percentage transfers to another repository.

## Work reduction

BuildOpt compared the full **64-project** graph with a **3-project** selected graph: **61 projects omitted (95.3%)**. Graph reduction alone is not counted as value.

## Measured wall-time value

Across **8 balanced pairs**, optimized native Gradle averaged **8246 ms** and BuildOpt averaged **5036 ms**. The observed installed-path saving was **3210 ms/build (38.9%)**, with a paired 95% interval of **2727 to 3738 ms** and **8/8** positive pairs.
Tail check: native p95 **9504 ms**, BuildOpt p95 **6393 ms** (delta **-3111 ms**). Required outputs were equivalent and launcher overhead was included.

## Learning cost and payback

Calibration cost **3436 ms**. At the observed mean saving, break-even is **2 matching builds** (owner limit: **30**). Successful exact replays counted: **0**. Projected cumulative net value so far: **-3436 ms**, after calibration and **0.000 ms** of selection overhead; projected payback remaining: **3436 ms**.
Expected useful lifetime: **unavailable** — `EXACT_REVISION_REPLAY_HAS_NO_OBSERVED_FUTURE_MATCH_COUNT`. BuildOpt therefore reports payback as a projection, not as realized customer value across future commits.

## Fallback

The current build used **optimized native Gradle**. Exact reason: `QUALIFIED_PROFILE_STORED`. Native result authoritative: **true**; build successful: **true**.

The sibling `value-report.json` contains the same source metrics and every derived value needed to recompute this report. Mechanism percentages are never added together.
