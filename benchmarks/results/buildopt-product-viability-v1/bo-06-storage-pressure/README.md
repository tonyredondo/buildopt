# Why the development replay stopped

Date: 2026-09-15. Status: diagnosis complete; BO-06 remains partial.

The replay stopped because it could not find a sufficiently quiet period before
the next build. The recorded observations reproduce that decision with the same
runner code. They do not establish which process caused the storage pressure.

We also confirmed a problem in how we supervise the experiment: checking the
size of its growing results directory consumes almost a CPU core during builds.
That is work introduced by the measurement system. We should reduce it before
another timing attempt, while preserving the disk limits and output checks.

No Elasticsearch build or output-comparison JVM ran during this investigation.
The previous control still passes, and development still has only 17 of its 42
scheduled builds. This diagnosis establishes no new optimization saving.

## What the retained observations show

| Question | Finding | What it establishes |
| --- | --- | --- |
| Did the quiet-start check reject a usable window? | The frozen code reproduces all 25 admissions and the one refusal. Every one of the refusal's 179 intervals exceeds the 10% storage-pressure limit. | The refusal follows the registered rule. Raising the threshold would change the experiment. |
| Were Gradle or its supervisors still doing substantial work during the failed wait? | Across 179 samples spanning 179.24 seconds, the same 17 processes used 1.18 seconds of CPU, read 24,576 bytes and wrote 73,728 bytes. The outer recorder separately used 0.089 seconds of CPU and recorded 507,904 bytes written during its enclosing phase. | Heavy ongoing work by these observed processes is not supported as the explanation. Earlier writes, short-lived processes and unobserved activity remain possible. |
| Does supervision add substantial work during builds? | Across all 25 completed builds, supervisor CPU time was 95.82%–98.57% of the native command's elapsed time. | The supervisor is busy for roughly one core's worth of work. This is not a measured percentage of build slowdown. |
| Can the directory-size check account for that work? | One call to the unchanged scanner on the retained development directory took 2.500 seconds and 2.540 seconds of process CPU. Its CPU profile places 98.81% of samples beneath the scanner, mostly in filesystem calls. | The scanner is expensive on this directory. Its repeating 100-ms timer can make successive scans effectively continuous. The historical builds were not profiled at this function level. |
| Could capturing results affect the next build? | Capturing the 17 development outputs took 1,096.42 seconds in total, with 344.50 seconds of recorder CPU and 3.947 GB recorded writes. The final capture alone recorded 232.448 MB of writes before the failed wait. | Capture is substantial research work between builds. Its contribution to the later storage pressure is plausible but unmeasured. |

Storage pressure here means the fraction of observed time when at least one
task was stalled waiting for I/O. It is not disk utilization or the percentage
of BuildOpt's time spent waiting. During the failed wait its weighted value was
77.43%; CPU pressure remained below the limit in every interval. Only two
intervals exceeded the memory-pressure limit.

The process totals cover every retained sample inside the wait, with no change
in the observed process identities or missing process fields. Sampling cannot
exclude short-lived activity between observations. The recorded sleeping/running
state describes each process's main thread, so it cannot establish that every
thread was idle. The outer recorder's phase boundaries also differ slightly
from the process-sample boundaries; their counters are reported separately.

The [derived analysis](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-storage-pressure/analysis.json)
contains every admission, completed build and recorded phase, including the
control. It verifies 340 original file bindings before calculating these totals.


## Why the size check is expensive

The worker checks free space and walks the complete experiment directory to
count regular-file sizes. It repeats that walk while a build runs. The directory
contains both live build state and the results retained from previous builds,
so the work grows as the replay progresses.

The existing observer already allows cancellation during a walk and keeps
other supervision checks responsive. It does not wait another 100 ms after a
slow walk: a timer tick can already be waiting. A 2.5-second walk therefore
leaves little pause before the next one.

The diagnostic counted 19,979,202,731 bytes of regular-file sizes. That is a
logical file-size total, not the storage physically allocated on disk. It read
metadata, not file contents. The standalone scan used the final retained tree,
which differs from the trees present during earlier builds. It identifies a
costly operation; it does not predict how many seconds its replacement will
save on a build.

The supervisor used a different CPU from Gradle. That avoids direct competition
for that CPU, but both still use the same filesystem and other shared resources.
The evidence does not quantify any resulting interference.

## What remains unexplained

The size scanner runs during the native command and is joined before that
command's recorded supervision completes. It does not keep scanning throughout
the failed quiet-start wait. We therefore cannot attribute that wait directly
to an actively running size scan.

The retained samples lack storage-device counters, pending-write counters,
per-group storage pressure and resource use from the Python sampling process.
The historical per-group I/O accounting was unavailable. Those gaps prevent us
from separating delayed effects of earlier experiment writes from unrelated
activity on the shared host. Checking the available kernel journal for the
relevant period returned no entries; that does not rule out storage contention.

There is no evidence here for changing the machine's filesystem, stopping other
work, flushing caches, disabling the guard or extending the wait. The useful
next action is to remove the confirmed repeated supervision work and qualify
the resulting measurement method.

## Next step and stop point

Follow the [conditions for the next measurement](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-storage-pressure/next-step.md).
First qualify a smaller amount of disk-accounting work with fixtures. Keep
complete output comparisons, the same limits, cancellation and refusal paths.
Record enough host and observer information to distinguish their activity if
another wait fails, without adding another expensive scan.

A changed runner needs a new frozen identity and a fresh identical-code control.
Only a passing control permits a new complete development sequence from the
cold anchor. The interrupted allocation stays closed; its partial results cannot
be completed by attaching a later suffix. BO-07 and protected changes 21–100
remain deferred.

## Reproduce this diagnosis

From the repository root:

```sh
python3 benchmarks/results/buildopt-product-viability-v1/bo-06-storage-pressure/analyze.py --check
python3 benchmarks/results/buildopt-product-viability-v1/bo-06-storage-pressure/replay-traces.py
```

The first command recomputes the published arithmetic. The second uses the
repository's pinned Go toolchain on Linux amd64, verifies and extracts the
previous runner into a temporary directory, and runs two focused trace checks.
The first deliberately fails to reproduce the refusal; the driver requires
that failure and its specific message. The second checks all 26 decisions and
a synthetic zero-pressure counterpart. Neither command starts a project build
or an output-comparison JVM.

The one live metadata scan is preserved as
[source](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-storage-pressure/storage_profile_test.go.txt),
[result](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-storage-pressure/diagnostic/retained-scan.json)
and [CPU profile summary](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-storage-pressure/diagnostic/retained-scan-profile.txt).
It had a three-second cancellation limit and ran once. Its host-local target
is preserved in the result. Replaying the trace checks does not rerun that scan.

The [file manifest](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-storage-pressure/evidence-manifest.json)
binds this block's analysis, code, probes and receipts. Original build evidence
remains in the [interrupted measurement](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-quiet-measurement/README.md).
The initial unsupported journal command is retained as a setup failure alongside
the successful corrected read. No failed optimization correction or discarded
performance attempt occurred in this block.
