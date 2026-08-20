# BuildOpt value report

**Outcome:** `LEARNING` — `QUALIFIED_PROFILE_STORED`

This report explains what the explicitly invoked BuildOpt POC observed. It is not a production authorization or a promise that the same percentage transfers to another repository.

## Work reduction

BuildOpt compared the full **1024-project** graph with a **34-project** selected graph: **990 projects omitted (96.7%)**. Graph reduction alone is not counted as value.

## Measured wall-time value

Across **8 balanced pairs**, optimized native Gradle averaged **75861 ms** and BuildOpt averaged **64534 ms**. The observed installed-path saving was **11328 ms/build (14.9%)**, with a paired 95% interval of **8866 to 14535 ms** and **8/8** positive pairs.
Tail check: native p95 **84503 ms**, BuildOpt p95 **69015 ms** (delta **-15488 ms**). Required outputs were equivalent and launcher overhead was included.

## Learning cost and payback

Calibration cost **7357 ms**. At the observed mean saving, break-even is **1 matching builds** (owner limit: **30**). Successful exact replays counted: **0**. Projected cumulative net value so far: **-7357 ms**, after calibration and **0.000 ms** of selection overhead; projected payback remaining: **7357 ms**.
Expected useful lifetime: **unavailable** — `EXACT_REVISION_REPLAY_HAS_NO_OBSERVED_FUTURE_MATCH_COUNT`. BuildOpt therefore reports payback as a projection, not as realized customer value across future commits.

## Fallback

The current build used **optimized native Gradle**. Exact reason: `QUALIFIED_PROFILE_STORED`. Native result authoritative: **true**; build successful: **true**.

The sibling `value-report.json` contains the same source metrics and every derived value needed to recompute this report. Mechanism percentages are never added together.
