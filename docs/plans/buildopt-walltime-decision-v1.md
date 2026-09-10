# BuildOpt walltime decision: close measurement work and test product value

Date: 2026-09-09. Authorized by the owner's “ok, haz esos 3 sin interrupción”.
This amendment implements the three agreed steps without another open-ended
instrumentation campaign. Prior failures and formal G0/G3/G4/G5 gates are unchanged.

## W1: bounded instrument closeout

Run at most two cold actual-owner requests with the sealed v3 observed replay.
Stop this diagnostic on the first failed observation or request. Preserve the
500-ms daemon coverage criterion and every startup/end gap, error and cost.
Do not fix or retry sampling here. A failure is a diagnostic result; it does not
prove that independently stamped whole-request walltime cannot be explored.

For W2, use the already qualified isolated v2 replay executable from
`bv006-cpu-isolation/control/history-replay`: its native inheritance, supervisor
separation, output checking, cancellation and exact request stamps have existing
source-bound proof. It does not include the new buffered observer. The optional
external process samples describe CPU placement and gaps; they are not the clock
used for whole-request duration. This is an explicit instrument selection before
new timings, not a claim that the previous observation-quality failure passed.
No production observer source change is needed for this exploratory comparison.

## W2: fixed baseline/candidate comparison

Retain the existing CPU-screen design: profiles 8, 4, 2 in that order; two warmup
builds per arm and four measured pairs per profile; 12 starts per profile and
36 total, including failures. Six consecutive engineering revisions 15..20 are
frozen; 15/16 warm up, 17..20 are measured. Affinity and worker counts vary
together, so conclusions concern resource profiles. No repetitions or replacement
rows beyond the cap. Both arms retain independent native state through history.

Source, output equivalence, exact request start/end, cost capture, process ownership,
and CPU placement remain required. Diagnostic sampling gaps are reported separately
and cannot be called zero or qualified. Sampling itself runs outside native CPU
masks. Its cost and remaining contamination uncertainty must be disclosed; this
screen cannot qualify statistical overhead balance or product value.

Report every paired walltime, mean and median delta, aggregate percentage, worst
regression, Checkstyle action frequency, candidate state reuse and maintenance
cost. Charge candidate inverse work outside the request. Report warmups separately
and include them in execution costs. Keep correct no-action transitions in the
measured denominator. Do not substitute task-duration savings for workflow saving.

The useful exploratory signal is prospectively defined as both at least 5%
aggregate net customer wall reduction and at least one second per measured
transition, with at least two positive pairs and no correctness/ownership failure.
Four selected pairs are not a confidence interval, representative lifecycle
estimate or G3 pass. A small positive mean alone does not trigger more investment.

## W3: conditional independent engineering history and decision

If no profile meets that signal, close this campaign with no justified progression
to adaptation/replication. Retain whether the result is negative, insufficient
opportunity or inconclusive; do not turn failure to establish benefit into proof
that all build optimization is impossible.

If a profile meets it, select the first qualifying profile in the fixed order
8, 4, 2. Register a separate maximum of 30 workflow starts: one independent N/I
replay of original engineering ordinals 0..14, with 0/1 as warmup and 2..14 measured.
These revisions do not overlap W2's selected 15..20 window. Keep the same candidate,
workflow, output rules and selected resource profile. Retain all changes and
no-action rows. Do not tune the implementation between the two windows.

Evaluate net saving, action frequency, regressions and state survival over those
13 measured transitions. This provides an independent engineering check of the
selected signal; it is still not the untouched formal confirmation cohort. If
benefit fails to survive, stop investment in this candidate for the tested setup.
If it survives, record the exact prerequisite for formal seed confirmation and
then the pre-existing multi-repository/adaptive stages. Do not build an adaptive
controller simply because a selected short screen is positive.

## Bounds, identity and durable state

One six-hour task allocation: at most 2 cold + 36 screen + 30 conditional history
Gradle starts, 70 metadata helper JVM starts, 160 GiB new task artifacts and
40 GiB minimum free. Cold requests have a 900-second timeout; each screen profile
has a two-hour ceiling inside its three-hour screen deadline. Independent-history
bounds must be frozen before its first request. No unlisted native preparation,
automatic retries, customer contact, publication or host configuration changes.

All 80 formal validation ordinals 21..100 remain untouched. Earlier retired
mechanisms remain closed. All native reservations, actual starts and unknown
launches are separate; prior program totals are 646 reserved / 547 actual Gradle,
plus the already retained uncertainties. Existing local implementation and
experiment authorization covers the declared work without intermediate approval.

Repository: `/home/tonyredondo/repos/github/tonyredondo/buildopt`, local `main`,
HEAD `b76ded08c952ebb386576fafce4ae2d8fdcc09f1`, common Git at that path plus `.git`.
Preserve existing dirty work. Task locator:
`/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/buildopt-product-viability-v1/task-state.json`,
key `engineeringPrefix.walltimeDecision`. Raw state: sibling directory
`bv006-walltime-decision`; W2 retains the existing `bv006-cpu-screen` root.
