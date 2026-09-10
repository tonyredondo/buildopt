# Historical screen observation closeout v4

Date: 2026-09-10. This is an analysis-only closeout of completed execution.
No native request, comparator JVM, CPU profile, commit, candidate, cost formula,
materiality threshold or previous qualification changes.

The frozen history run and its independent C5 reconstruction pass all 30 native
requests and 15 pairs. The v3 controller then stops in the original history
reader. That failure and all original frozen files remain unchanged.

One observation of the bound N supervisor has seven threads on observer CPU8
and one thread on native CPUs0..7. It occurs 29.774–34.451 ms after the start of
original ordinal8 N, not during initial systemd startup. The pinned
`affinityScope.start` changes its locked launch thread to the native mask around
`exec.Cmd.Start`, then restores and verifies the observer mask. Neighboring
observations show observer CPU8. The record is consistent with this source path;
it does not measure the exact excursion duration.

**The worker-exclusive-affinity criterion failed and stays failed.** The v4
history reader retains the observation as `FAILED_WORKER_EXCLUSIVE_AFFINITY`,
sets all-time and combined native/worker qualification false, and keeps strict
rejection of incorrect native masks, unknown worker scope and missing coverage.
It can reconstruct exploratory economics and output correctness without claiming
that isolation, G0, diagnostic quality, precision, G3 or product value passed.
An economic point signal cannot bypass this failed qualification.

The regression reproduces the original failure using all 11,390 real samples,
retains the negative worker qualification and rejects native-mask, missing-mask
and wrong-scope mutations. Another worker-mask violation remains unqualified.
The initial test fixture lacked any clean observer sample and correctly failed
the minimum-coverage assertion; the fixture was corrected to include its clean
control. Production reader logic did not change for that fixture correction.
Zero native/JVM starts were added.

Raw state is `.tools/state/buildopt-product-viability-v1/bv006-screen-completion-fixed`.
Use `analysis/history-worker-affinity-observation.json`,
`logs/history-reader-v4-test.log`, `analysis/reader-v4-fixture-correction.json`
and `inputs/history-observation-v4-freeze.json`. The superseded controller receipt
remains `receipts/controller-v3.json`; a separate closeout receipt will bind the
completed negative qualification and report. No frozen receipt is rewritten.
