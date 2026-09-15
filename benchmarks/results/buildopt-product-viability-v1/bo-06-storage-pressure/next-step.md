# BO-06: conditions for another development measurement

This is a supporting contract for the
[governing BO-06 step](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-research-execution-plan-2026-09-14.md#bo-06-qualify-correctness-and-freeze-the-experiment),
not a new research track or an active timing allocation. The
[storage diagnosis](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-storage-pressure/README.md)
identifies repeated full-directory scans as substantial supervisor work. It
does not establish the cause of the historical storage-pressure timeout.

The next implementation block ends with fixture proof and frozen inputs.
Elasticsearch timing belongs to the subsequent measurement block. Neither block
reopens protected changes 21–100 or changes the savings criteria.

| Order | Work | Required result |
| --- | --- | --- |
| 1 | Trace every writer beneath the experiment directory and define which retained files can still change. Measure the current scanner on a bounded representative fixture. | An explicit accounting boundary, a reproducible cost baseline and a list of invariants the replacement must preserve. No assumed immutability merely because an earlier build finished. |
| 2 | Reduce repeated accounting of retained results while continuing to account for active build state. Prefer a change at the existing disk guard. | Every file still counts toward the same limit. Missing observations or an invalid cached count must prevent admission or fall back to a verified complete count. If this requires a different limit or detection guarantee, record the conflict before implementation. |
| 3 | Exercise size growth, creation/deletion, rename, temporary files, symlinks, count invalidation, read errors, low free space, cancellation, parent loss and completion during a scan. | Correct totals under the supported mutation contract, refusal when limits or observations fail, responsive independent supervision, no escaped owned process and no orphan observer. Compare against a full scan on the same fixture. |
| 4 | Compare the old and new supervision work on the same fixture sizes, with at most two measured repetitions per version and size. Include a small and a retained-results-heavy case. | Lower repeated CPU work on the large case with the same accounting guarantees. Retain elapsed time and cancellation latency. This qualifies the measurement change, not an Elasticsearch speedup. |
| 5 | Add only the missing observations needed to understand another storage wait: host device counters, pending writes, observer CPU/I/O, and group pressure where available. Read counters without elevated permissions or scanning unrelated processes. | Timestamped observations for the experiment and its observers, explicit unsupported fields, and measured collection cost. If group accounting remains unavailable, preserve that attribution limit. Do not make a missing optional diagnostic silently qualify a build. |
| 6 | Run a small integrated fixture through control admission, launch, comparison, refusal and closure with the final runner. Recheck the comparator binding and affected compatibility tests. | Proof for the code that will actually run, including failures. Freeze the runner, policy, comparator and protocol under one new identity. No reuse of the earlier control to qualify changed code. |
| 7 | Prepare the separate timing allocation and recheck readiness immediately before starting. | The unchanged quiet-start rule, exact commands/order, independent arm state, observation policy and outcome retention are recorded before measurement. Keep the existing maximum of 50 project builds, 50 comparator starts, ten hours and 80 GiB, including preparation and waiting. |
| 8 | Run eight identical-code control builds. If that control passes, run all 42 development builds from the cold anchor through change 20. | A complete result under the frozen method, including preparation and comparison costs. A refusal or exhausted limit remains an incomplete result; do not retry selected rows, extend limits or attach a suffix to the old allocation. |

Before the fixture work starts, record its own duration, disk and launch limits
in the task state. Use fixtures to settle the supervision change; do not consume
another development replay to debug it. The eight-build control still applies
the registered combined three-second and 5% stopping rule. All other existing
correctness, contention and performance rules retain their meanings.

If the new control cannot find quiet conditions, close that allocation with the
new observations. Do not enter another cycle of blind retries or relaxed limits.
A dedicated measurement environment or a changed observation policy would then
be a separate methodological decision. If development completes, assess its
whole-sequence result and preparation costs before deciding whether BO-06 can
freeze and BO-07 can begin.

Keep output capture unchanged in the first supervision correction so its effect
can be assessed separately. Consider changing capture only if the next evidence
identifies it as the remaining obstacle and full output reconstruction remains
provable. The ordinary-workflow comparison in BO-09 remains necessary: a saving
inside the research recorder does not by itself prove a usable product saving.
