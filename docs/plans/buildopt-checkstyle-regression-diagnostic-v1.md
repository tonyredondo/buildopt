# Checkstyle regression and recovery diagnostic

Status: frozen design for local execution, 2026-09-10. User request: explain the
ordinal 7 regression, including cache misses and subsequent return to normal.
Program: BUILDOPT-VIABILITY-V1. This is a diagnostic, not formal value confirmation.

## Inputs and scope

Use the corrected private-task-map candidate unchanged and the same native
baseline, complete C5 output checks, pinned toolchains, shared subject Git and
8 CPUs / 8 Gradle workers. Neither prior frozen result nor validation 21–100
is modified. A task-owned Java agent instruments the original class bytes in
memory; original class resources and source remain unchanged. The agent applies
to both arms, records real engine file counts, adapter hashing/process phases,
Prepare/context/Commit durations and bounded JFR evidence. No agent is installed
in a user/global environment or retained in a product patch.

## Questions and falsifiable distinctions

1. Native task/cache scheduling: compare exact action identities and task outcomes.
   Original ordinal 7 has identical N/I actions; later Main tasks are UP-TO-DATE.
2. Candidate reuse miss: actual adapter counts must distinguish full processing
   from reuse. Prior success records alone do not prove which branch executed.
3. Time outside checking: relate engine and prepare/commit intervals to the whole
   Main task and asynchronous Gradle waits. Task CPU is not walltime.
4. Filesystem/cache/GC/contention: correlate JFR blocking and file events, native
   process CPU/I/O/fault counts and host pressure. Shared-host counters alone
   cannot attribute exclusive causality.

## Exact bounded sequence

Run one persistent-state N/I replay of original ordinals 6, 7, 8 and 9: warm
6, reproduce 7, observe the two following commits. Four pairs/eight native
starts total, including warmups and failures, no retries. This supersedes the
previous proposal of two separate 6-to-7 repetitions to answer the explicit
recovery question without increasing the eight-start limit. The runner requires
consecutive Git edges, so no duplicate revision is represented as new history.

All four pairs require live and independent complete C5 output comparison and
all three candidate histories. Fresh state at 6 does not recreate daemon/cache
age from original 0–6. Later skipped checking proves avoidance, not fast repeated
engine execution; direct engine proof covers warm replay semantics separately.

Native window: four hours. Overall preparation/run deadline and exact counters
are in `.tools/state/buildopt-product-viability-v1/bv006-checkstyle-regression/allocation.json`.
Run storage: 64 GiB; free floor: 40 GiB; external diagnostics: 512 MiB. JFR targets
8 MiB per relevant JVM; final dumps/coverage must be checked, especially at daemon
shutdown. Helpers: at most 12 JVMs (three observed preparation probes, eight
comparators, one combined JFR analysis); compiler commands: at most four.

The initial observed agent probe failed bytecode verification before any native
run. Its error/source/jar remain retained. The corrected timer-local setup passes
actual native/adapter cold, warm and failing cases, preserves events/error counts
and original resource bytes, and verifies the installer classes can load.

## Required decision and stop

Return proven cache/scheduling behavior, phase attribution, recovery observations
and the causal evidence or explicit remaining uncertainty. No non-reproduction
is called a repair and no qualified product claim follows from this diagnostic.
Stop on native/correctness failure or bounds; preserve all charges and close owned
processes. No broad CPU grid, implicit retry, unrelated source change, publication
or reopening of retired routes. Recovery locator: `engineeringPrefix.checkstyleRegression`
in the existing program task-state file.
