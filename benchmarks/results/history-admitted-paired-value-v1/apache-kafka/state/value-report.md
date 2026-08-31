# BuildOpt value report

**Outcome:** `LEARNING` — `QUALIFIED_PROFILE_STORED`

This report explains what the explicitly invoked BuildOpt POC observed. It is not a production authorization or a promise that the same percentage transfers to another repository.

## Work reduction

BuildOpt compared the full **66-project** graph with a **3-project** selected graph: **63 projects omitted (95.5%)**. Graph reduction alone is not counted as value.

## Measured wall-time value

Across **8 balanced pairs**, optimized native Gradle averaged **35962 ms** and BuildOpt averaged **29096 ms**. The observed installed-path saving was **6866 ms/build (19.1%)**, with a paired 95% interval of **5717 to 8204 ms** and **8/8** positive pairs.
Tail check: native p95 **38695 ms**, BuildOpt p95 **31310 ms** (delta **-7385 ms**). Required outputs were equivalent and launcher overhead was included.

## Learning cost and payback

Calibration cost **7242 ms**. At the observed mean saving, break-even is **2 matching builds** (owner limit: **5**). Successful exact replays counted: **0**. Projected cumulative net value so far: **-7242 ms**, after calibration and **0.000 ms** of selection overhead; projected payback remaining: **7242 ms**.
Expected useful lifetime: **unavailable** — `EXACT_REVISION_REPLAY_HAS_NO_OBSERVED_FUTURE_MATCH_COUNT`. BuildOpt therefore reports payback as a projection, not as realized customer value across future commits.

## Fallback

The current build used **optimized native Gradle**. Exact reason: `QUALIFIED_PROFILE_STORED`. Native result authoritative: **true**; build successful: **true**.

The sibling `value-report.json` contains the same source metrics and every derived value needed to recompute this report. Mechanism percentages are never added together.
