# What would count as a successful Build Optimization experiment?

A useful correction must produce the same required build results and save time
as the code changes. The saving must cover the work needed to find, check and
maintain it. These are the research requirements used in the quarterly review;
the detailed experiment rules remain in the linked plan and replay specification.

## Start with a measurable opportunity

First, the ordinary Gradle build must run reliably with fixed source versions,
commands and toolchains. Its existing cache features stay enabled wherever
the workflow supports them. Recording the experiment must not introduce an
unexplained difference between the versions being compared.

Observed builds must then show enough avoidable work to reach the savings
requirements below. A costly-looking function is insufficient if Gradle rarely
runs it or if shortening it would not make the complete build finish sooner.

## Preserve the result

The correction must keep all required files, diagnostics and failure behavior.
Checks cover changed and deleted files, changed rules, failed builds,
cancellation, recovery and removal of the patch. When the code or tool version
makes previous results unsafe to reuse, the optimization must stop that reuse.

One missing required result is a correctness failure, however much time the
build saves. Builds that decline the optimization still count in the timing.

## Measure a complete sequence of changes

Each repository uses 100 consecutive changes from Git history. The first twenty
are for developing the correction and checking the measurement method. The
remaining eighty are reserved for validation. They must not be used to tune
the implementation before reporting its result.

Run the complete sequence twice, each time from separate build state. Both
runs must meet the requirements:

| Requirement | What it means |
| --- | --- |
| At least 5% saved overall and an average of at least one second per scheduled validation build | Count the complete requested build and all required extra work. Include builds where the correction cannot help. |
| Statistical support for a positive saving | Account for the relationship between consecutive commits. Treating them as unrelated samples would overstate confidence. |
| Preparation recovered within the 100-change sequence | Accumulated savings must cover preparation and stay ahead afterward. An estimate based only on selected fast builds does not pass. |
| A limit on slower builds | The optimized p95 may exceed Gradle's p95 by no more than the larger of one second or 5% of Gradle's p95. Retain every individual slowdown and the worst case. |
| Complete results and no correctness failures | Keep every scheduled outcome. Missing builds, declined optimizations and unsuccessful attempts must remain visible. |

The p95 describes the slower end of the build times: 95% of observations are
at or below it. Passing this timing limit cannot excuse a wrong build result.
An incomplete or unreliable experiment leaves the saving unproven.

## Show that the approach works elsewhere

Choose two additional independent repositories before seeing their performance.
Use the same approach and unchanged history requirements. At least two of the
three projects must pass. Report all three, including places where the
correction cannot apply.

This distinguishes a useful individual patch from an approach that works
across different build structures. A later, longer history must also check
less frequent changes such as tool upgrades; a short successful window cannot
establish long-term maintenance.

## Test whether adaptation adds a saving

After the fixed correction passes, compare ordinary Gradle, that fixed correction
and a version that can replace corrections over time. Use history that was not
already used to develop the replacement policy.

The adaptive version must respond to confirmed loss of benefit, check a
replacement before using it and stop unsafe reuse immediately. One slow build
on a busy machine should not trigger repeated searches. At each commit, it may
use only information available by then.

Count monitoring, searching, checking replacements and any background work.
Adaptation must still save more time than both ordinary Gradle and the fixed
correction. Working replacement logic alone does not demonstrate a benefit.

The [technical acceptance criteria](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-product-viability-v1.md#6-evidence-ladder-and-decisions)
and [history measurement rules](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-product-viability-v1-replay.md)
specify the calculations and checks. The [research review](https://github.com/tonyredondo/buildopt/blob/main/docs/findings/buildopt-next-quarter-investment-review-2026-09-10.md#proposed-research-okrs)
sets out the proposed next steps. These requirements explain how to judge the
research; they do not report experiments that have yet to run.
