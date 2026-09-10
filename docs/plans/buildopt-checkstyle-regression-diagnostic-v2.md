# Checkstyle worker observation correction

Status: frozen for local diagnostic execution, 2026-09-10. Amendment to the
[v1 diagnostic](buildopt-checkstyle-regression-diagnostic-v1.md); its old inputs,
failed probes, one successful native warmup and incomplete capture remain retained.

## Why the capture changed

Actual Checkstyle workers omit JAVA_TOOL_OPTIONS. The preliminary native ordinal6
build succeeded, but its checking phases were not captured. The controller stopped
before the candidate arm, and its exact native scope was closed. One actual native
start and the original eight reservations remain charged; no pair was verified.

The task-owned v2 agent adds its explicit JVM argument at Gradle9.7.1
AbstractCodeQualityTask.configureForkOptions, inherited by Checkstyle. This covers
code-quality workers symmetrically. Actual Gradle API and real native/adapter cold,
warm and changed-failure probes pass, preserving events and original class bytes.
Failed API-fixture classpath/method/code-source probes remain evidence. Explicit
process exit closes ProjectBuilder services after assertions. Native worker phase
capture is an integration requirement, checked after the first native requests.

## Sequence and limits

Fresh paired original6,7,8: six native starts, warm6, observe7 and following8.
Combined with the preliminary start, seven actual builds are planned, within the
unchanged maximum eight actual starts. Retain fourteen charged reservations across
both designs. No retry and no fresh ordinal9; original8–14 retained evidence still
answers later scheduling. Candidate, baseline, workflow and complete live/independent
C5 comparisons are unchanged. Fresh6 does not reconstruct original0–6 cache age.

Overall deadline remains 2026-09-10T09:24:42Z; native window four hours, run64GiB,
free floor40GiB, external diagnostics512MiB. At most15 actual standalone helper
JVMs and six compiler commands across preparation and final analysis; charged
reservations remain separate. Exact counts live in the allocation ledger.

## Required outcome

Verify actual processed/reused counts, phase and asynchronous wait timing, process
CPU/I/O and JFR coverage; distinguish engine behavior from whole-workflow walltime.
All original uncertainty and diagnostic limitations remain. Stop on native/C5/
capture failure or bounds, close owned scopes, report proven findings and the next
step. No candidate repair, qualified product result, publication or broad CPU grid.
