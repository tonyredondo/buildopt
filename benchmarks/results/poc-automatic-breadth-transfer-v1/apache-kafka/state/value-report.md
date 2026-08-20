# BuildOpt value report

**Outcome:** `NATIVE_RETAINED` — `CALIBRATION_VALUE_NOT_PROVEN`

This report explains what the explicitly invoked BuildOpt POC observed. It is not a production authorization or a promise that the same percentage transfers to another repository.

## Work reduction

BuildOpt compared the full **64-project** graph with a **36-project** selected graph: **28 projects omitted (43.8%)**. Graph reduction alone is not counted as value.

## Measured wall-time value

Across **8 balanced pairs**, optimized native Gradle averaged **14766 ms** and BuildOpt averaged **12785 ms**. The observed installed-path saving was **1982 ms/build (13.4%)**, with a paired 95% interval of **-91 to 5183 ms** and **3/8** positive pairs.
Tail check: native p95 **29018 ms**, BuildOpt p95 **16664 ms** (delta **-12354 ms**). Required outputs were equivalent and launcher overhead was included.

## Learning cost and payback

Calibration cost **374762 ms**. At the observed mean saving, break-even is **190 matching builds** (owner limit: **30**). Successful exact replays counted: **0**. Projected cumulative net value so far: **-374762 ms**, after calibration and **0.000 ms** of selection overhead; projected payback remaining: **374762 ms**.
Expected useful lifetime: **unavailable** — `EXACT_REVISION_REPLAY_HAS_NO_OBSERVED_FUTURE_MATCH_COUNT`. BuildOpt therefore reports payback as a projection, not as realized customer value across future commits.

## Fallback

The current build used **optimized native Gradle**. Exact reason: `CALIBRATION_VALUE_NOT_PROVEN`. Native result authoritative: **true**; build successful: **true**.

The sibling `value-report.json` contains the same source metrics and every derived value needed to recompute this report. Mechanism percentages are never added together.
