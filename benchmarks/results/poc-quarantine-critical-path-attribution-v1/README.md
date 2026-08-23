# Quarantine critical-path attribution

This directory closes the frozen Micronaut quarantine-frontier investigation.
It combines the unchanged eight-pair causal result with diagnostic Gradle task
operations and resolved dependency graphs. The trace does not replace or
recalculate the wall-time evidence.

The capture used public Micronaut revisions `4dc4299f` and `a7955f4c`, BuildOpt
capture revision `d33c457`, Gradle's wrapper, 12 workers, 58 graph-proven
candidate entrypoints, the same 52-of-70-project frontier, 101 transported
outputs and 89 locally rebuilt outputs from seven quarantined producers. The
required output digest is identical in every observation.

## Results

| Metric | Optimized native | BuildOpt candidate | Candidate delta |
|---|---:|---:|---:|
| Causal wall time | 13,077.125 ms | 12,880.125 ms | **-197 ms (-1.51%)** |
| Executed tasks | 678 | 568 | **-110** |
| Cumulative task duration | 43,357.125 ms | 38,626.125 ms | **-4,731 ms** |
| Main-build task span | 5,617.500 ms | 5,865.875 ms | **+248.375 ms** |
| Longest hard-dependency chain | 5,183.500 ms | 5,362.375 ms | **+178.875 ms** |

The wall-time result is not qualified: 5/8 pairs improve, the 95% saved-time
interval is -703.5..+951.125 ms, and p95 changes only from 14,216 to 14,149 ms.
Fallback succeeds and the output digest remains exact.

The candidate eliminates 110 tasks, but none appears on any control critical
path. Their mean control duration totals 1,706.375 ms: 20 `Jar` tasks account
for 1,299.750 ms and three `Javadoc` tasks for 280.375 ms; the remainder is
small compile, resource and lifecycle work. The larger 4,731-ms cumulative
reduction includes timing variation in tasks present in both arms. It does not
shorten the main task span or hard-dependency chain.

## Decision

`STOP_MICRONAUT_QUARANTINE_LINE`.

The exact graph-proven candidate removes substantial parallel/off-critical
work, but the owner-visible wall-time gate remains negative and the remaining
critical work is required by the exact output boundary. A theoretically
smaller entrypoint or task count is not sufficient reason for another
Micronaut-specific optimization. BuildOpt continues to retain optimized native
Gradle for this profile.

This is bounded POC evidence. It does not authorize repository-specific
product rules, production activation, weaker output checks, Test Optimization,
soak testing or design-partner work.

## Files and validation

- [`source-summary.json`](./source-summary.json) is the unchanged compatible
  portfolio wall-time result.
- [`attribution.json`](./attribution.json) contains all 16 trace digests,
  paired wall times, per-build critical chains and the 678-task aggregate.
- Raw operation logs are not versioned; their SHA-256 digests are retained in
  `attribution.json` and the checked aggregate can be regenerated from a fresh
  capture.

Validate the checked evidence without rerunning Micronaut:

```bash
./dev/check-quarantine-critical-path-attribution
```

Capture a new frozen run, or reanalyze an existing raw capture, with:

```bash
./dev/run-quarantine-critical-path-attribution /new/absolute/evidence/directory
./dev/run-quarantine-critical-path-attribution \
  /new/absolute/analysis/directory /absolute/existing/raw/directory
```
