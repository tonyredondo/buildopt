# Reserved-workstation control could not start

Date: 2026-09-16. BO-06 remains partial.

The repeat stopped before its first build. After preparing both environments,
the runner waited three minutes for quiet conditions and refused to launch
Gradle. All eight control builds and the conditional 42-build development
sequence remain unrun. There is no performance comparison or measured saving.

The owner had reserved the workstation after the
[previous interrupted control](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-disk-measurement/README.md).
That justified a separate attempt with fresh state. The runner, correction,
comparator, recording method, thresholds and limits stayed unchanged. The
previous attempt remains part of the evidence.

## What prevented the build

Disk-wait pressure exceeded the 10% threshold in all 179 observed intervals.
Its average was 62.38%, while CPU pressure averaged 0.19%. These figures measure
time when work was waiting for a resource, not disk utilization. Reconstructing
the frozen rule confirms that no 30-second quiet window qualified. Sample gaps
and counter-read times stayed within their limits.

A system file indexer, `updatedb`, was still active during preparation and
appeared blocked waiting for I/O. It could have contributed to the contention,
or been affected by the same problem. Its I/O counters were unavailable under
the current permissions, so this observation does not establish the cause.
No service or unrelated process was changed.

The experiment itself had already done substantial preparation:

| Recorded phase | Elapsed time |
| --- | --- |
| Prepare the first environment | 440.495 seconds |
| Prepare the second environment | 611.329 seconds |
| Verify the first request's source and state | 228.898 seconds |
| Wait for quiet conditions | 180.126 seconds, including receipt work |

During the refused wait, the runner used about 0.055 seconds of CPU. Sampled
runner counters showed no new physical disk reads or writes across 179.45
seconds. The outer sampler used about 0.301 seconds of CPU over that interval.
Earlier preparation may still have left pending writes; these observations
cannot separate them from other storage activity.

Neither native build supervision nor post-build output capture ran. This
attempt therefore adds no timing evidence about Checkstyle, and cannot judge
whether the disk-accounting change helps during a build.

[Scheduled outcomes and pressure calculation](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-reserved-host/analysis/control-retained-summary.json),
[runner and sampler observations during the wait](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-reserved-host/analysis/refused-wait-observers.json),
and [system indexer observation](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-reserved-host/receipts/system-indexer-observation.json).

## Decision and next step

Close this allocation as `INCOMPLETE_CONTROL_QUIET_TIMEOUT`. BO-06 remains
partial; BO-07 and protected changes 21–100 stay deferred. Reserving the
workstation did not establish quiet storage conditions. It also did not turn
this into a negative result for the optimization.

Before proposing another build sequence, establish a usable measurement window
with the recorder stopped and the system indexer finished. A bounded host-only
observation should retain CPU, disk and memory pressure and the indexer's state.
If disk pressure persists then, investigate the host storage separately. If it
appears only when preparation resumes, investigate that preparation and any
pending writes. Neither observation alone permits a retrospective threshold
change or proves the exact cause. Do not launch another control automatically.

This follows the
[governing execution plan](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-research-execution-plan-2026-09-14.md#bo-06-qualify-correctness-and-freeze-the-experiment).
The next technical question is whether the existing measurement method can run
under demonstrably quiet conditions, before spending another build allocation.

## Retained evidence

The single worker session and all controllers closed. The attempt retained
6.89 GiB of state, within its 80-GiB limit. It used no project build or comparison
JVM; eight of each had been reserved for the control. The raw result preserves
one unknown launch reservation. The refusal receipt and absence of any native
launch record explain that reservation without rewriting the raw result.

The portable audit checks exported records, pressure calculations, source
bindings and the absence of comparisons. It does not run a build or reconstruct
build outputs:

```sh
python3 -B benchmarks/results/buildopt-product-viability-v1/bo-06-reserved-host/verify-evidence.py
```

Local state:
`.tools/state/buildopt-product-viability-v1/bo-06-reserved-host-2026-09-16/`.
