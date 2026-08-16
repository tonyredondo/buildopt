# Customer-readable value report

## Decision

Every completed `buildopt optimize <gradle args...>` invocation writes a short
human report and a strict machine report beside its existing state:

```text
.buildopt/optimize/v1/value-report.md
.buildopt/optimize/v1/value-report.json
```

The Markdown answers what work was removed, whether installed BuildOpt was
actually faster than optimized native Gradle, how uncertain that observation
is, what learning cost must be repaid, and why native Gradle ran when no safe
value claim exists. A reader does not need the implementation tracker.

## Recomputable value

The JSON preserves the source means, paired 95% interval, positive-pair count,
nearest-rank p95 for both arms, graph counts, calibration cost, exact replay
count and measured selection overhead. Every derived number has one fixed
formula in the machine contract. Graph reduction is never itself called a
speedup, unavailable metrics are not replaced with zero, and percentages from
unrelated mechanisms or repositories are never added.

The calibration mean is an observed installed-path effect. Cumulative value is
more limited: it projects that measured mean over successful exact replays in
the current generation, then subtracts calibration and selection costs. Every
selected attempt contributes its selection overhead, including a build that
later fails, while only successful replays receive projected savings. The report
labels that number as a projection rather than observed cumulative wall time.

## Honest useful-lifetime boundary

The current POC replays profiles only for the same exact repository revision
and bindings, so it cannot yet observe cross-commit applicability or estimate
how many future builds a profile will remain useful. The report therefore emits:

```text
UNAVAILABLE / EXACT_REVISION_REPLAY_HAS_NO_OBSERVED_FUTURE_MATCH_COUNT
```

It still reports the measured break-even and the owner-supplied maximum, but it
does not pretend that payback will occur. The end-to-end value block must
collect real lifetime evidence before replacing this state with an estimate.

## Fallback and authority

When native Gradle is authoritative, the report records the exact current
reason and whether the native build succeeded. When a profile is selected, it
states that optimized native Gradle remains available on any binding drift or
uncertainty. Reports remain current-user private, are replaced atomically, and
keep `productionAuthorized=false`; Test Optimization remains out of scope.

GitHub Actions and GitLab CI publish both value files with `result.json`, a
short CI summary and checksums. Raw Gradle arguments, console logs, credentials
and absolute checkout paths remain excluded.

The exact machine contract is
[`poc-magic-wow-report-v1.json`](./poc-magic-wow-report-v1.json).
