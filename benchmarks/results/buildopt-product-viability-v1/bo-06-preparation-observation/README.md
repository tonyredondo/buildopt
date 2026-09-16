# Preparation caused heavy disk waits, then recovered

Date: 2026-09-16. BO-06 remains partial.

Preparing the experiment caused heavy disk pressure, but this time it cleared
after preparation finished. The first qualifying quiet window appeared after
47 seconds, within the existing three-minute limit. We did not reproduce the
prolonged wait that prevented the previous control from starting.

Both environments and the first request's input checks completed successfully.
No Gradle build or comparison JVM ran. These results describe the experiment's
setup; they do not measure a BuildOpt saving.

## Before, during and after preparation

We recorded three minutes before preparation, prepared the two environments in
their original order, checked the first environment's inputs, then observed
recovery. The sequence used the same dependency files, source revision and
preparation operations as the
[interrupted control](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-reserved-host/README.md).

| Period | Average disk-wait pressure | Quiet windows under the existing rule |
| --- | --- | --- |
| Before preparation, 180 seconds | 0.80% | Present; brief spikes remained |
| During preparation, about 708 seconds | 84.56% | None |
| First 180 seconds after preparation | 0.88% | First at 47 seconds; the final window also qualified |

Pressure measures time when work waited for a resource, not disk utilization.
Memory pressure also rose during preparation and fell afterward. No `updatedb`
file-indexing process appeared in the 107 process checks. Those checks cannot
rule out short-lived or unrelated activity between observations.

The recovery observation stopped on the next one-second sample, after 181
seconds. We retained that extra sample, but the conclusion above uses only
windows ending within the original 180-second limit. All 1,250 evaluated window
decisions, including that separate budget check, agree with the original
runner's rule. These overlapping windows are not independent trials.

## Where preparation spent its time

The first environment took 352.14 seconds to prepare and the second took
337.51 seconds. The subsequent input check took 16.15 seconds. The detailed
records show that copying files dominated:

| Operation across both environments | Elapsed time |
| --- | --- |
| Copy the six dependency and tool directories | 445.74 seconds |
| Verify those directories before and after copying | 215.89 seconds |

The copies accounted for 63% of the preparation process's elapsed time, while
using only 4.38 seconds of its own CPU time. The copy function asks the
filesystem to finish writing each file before continuing. This identifies
copying as a substantial source of setup delay; this experiment did not isolate
the cost of each filesystem operation or test a replacement.

The observer used 2.38 seconds of CPU over the complete 17-minute-49-second
observation. Its cost is retained separately. The observer did not scan the
growing experiment directory.

## Decision and next step

The preparation can generate substantial disk pressure, and the existing quiet
wait can allow it to subside before a build starts. This pass supports another
bounded control with the unchanged measurement method. It does not explain
every earlier timeout, establish that the file indexer caused them, or guarantee
quiet conditions throughout a future build.

The next block should prepare a fresh eight-build identical-code control using
the existing runner, correction, comparator and thresholds. Only a passing
control permits the complete 42-build development sequence. Keep the existing
limits: one attempt, at most 50 project builds and 50 comparison JVM starts,
ten hours, 80 GiB of new state and at least 40 GiB free disk. Retain every
interruption and unfavorable result. Nothing in this diagnostic activates
that allocation or permits an automatic retry.

There is no need to change the copy implementation to act on this result.
Reducing its setup cost is a separate question. The immediate research question
remains whether the fixed Checkstyle correction saves time across the complete
development sequence. BO-06 remains partial; BO-07 and protected changes 21–100
stay deferred under the
[governing plan](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-research-execution-plan-2026-09-14.md#execution-tracker).

## Evidence and verification

The diagnostic used a separate entry point built from the frozen source. Its
only changes were that entry point and phase observations. The original timing
runner was unchanged. The compiled diagnostic rejected all seven workflow
commands tested and contained no workflow, worker-launch or output-comparison
entry points. Copy preservation, altered-file detection and refusal checks
passed. The observer's success and timeout checks both closed their child
process groups.

The single observation completed within its one-hour limit. All 36 frozen
diagnostic inputs still match. The two retained environments occupy 6.87 GiB
by allocated-file accounting. Available disk fell by at most 0.36 GiB during
sampling; that separate host-level measure includes unrelated activity and
shared filesystem storage. The observer and preparation process group are
closed. No service, threshold or existing experiment was changed.

The [calculation](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-preparation-observation/analysis.json)
contains the pressure windows and operation timings. The
[observation protocol](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-preparation-observation/inputs/observation-protocol.json)
records the limits set before the pass. Compressed raw samples, operation
records, the diagnostic source and validation receipts are retained alongside
them. To verify the published calculation without starting another experiment:

```sh
python3 -B benchmarks/results/buildopt-product-viability-v1/bo-06-preparation-observation/analyse.py --verify
```

Local task record:
`.tools/state/buildopt-product-viability-v1/bo-06-preparation-observation-2026-09-16/task-state.json`.
