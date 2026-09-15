# Control stopped before the second build

Date: 2026-09-15. BO-06 remains partial.

The new control could not finish. Its first Elasticsearch build passed, but the
second could not find 30 continuous seconds of low resource pressure within the
three-minute limit. Seven of the eight scheduled builds remain unrun. There is
no complete pair, no output comparison and no measured saving. The conditional
42-build development replay did not start.

This allocation is closed as `INCOMPLETE_CONTROL_QUIET_TIMEOUT`. The
[governing tracker](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-research-execution-plan-2026-09-14.md#execution-tracker)
keeps BO-07 and protected changes 21–100 deferred.

## What happened

The control used identical corrected code on both sides, with separate build
state, eight CPU assignments and the frozen output recorder. It was scheduled
for development changes 17–20. The command, `:server:precommit`, compiles required
code and runs checks before a change is committed. The first pair was cold
preparation; the following three pairs would have supplied the timing comparison.

| Scheduled work | Result |
| --- | --- |
| Change 17, side N | Build passed; whole request took 206.016 seconds. |
| Change 17, side I | Launch refused after 180 seconds of waiting. |
| Changes 18–20, both sides | Did not start because the first pair was incomplete. |
| Development, anchor through change 20 | Did not start because the control was incomplete. |

The full
[scheduled outcomes](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-disk-measurement/analysis/control-retained-summary.json)
retain every missing result. A successful unpaired build cannot establish
equivalent outputs or a performance difference.

During the refused launch, **11 of 179 intervals** exceeded the 10% disk-pressure
threshold. Average disk pressure was **1.78%**, but the bursts repeatedly
interrupted the required continuous window. CPU and memory pressure stayed below
their thresholds in every interval. Reconstructing the frozen rule over all
samples confirmed that no qualifying window was missed. Low average pressure
does not satisfy this rule.

The new storage samples include device activity and pending writes. Group-level
I/O observations were unavailable and are labelled accordingly. The retained
data do not establish which activity caused each burst. No threshold was changed,
sample removed or build retried.

## How the changed disk guard behaved

The first build changed the directory structure, triggering the guard's fallback
to full scans. It completed seven cached checks and 207 full scans. Its supervisor
used 201.760 seconds of CPU during 205.826 seconds of native execution, almost one
core throughout. Setup is included in that CPU total and was not timed separately.

The earlier fixture improvement therefore did not persist through this cold
build. Later incremental builds were not reached, so their supervision cost
remains unmeasured. These observations also cannot determine how much supervision
changed the build's elapsed time.

Copying and recording the first build's outputs took another 565.115 seconds,
recorded as research work outside the build request. That is a substantial cost
of the experiment. Its contribution to the later pressure bursts is unproven.
The outer sampler used 4.955 seconds of CPU over the complete run; its storage
counter reads accounted for 0.504 seconds of that total. The
[disk observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-disk-measurement/analysis/disk-observations.json)
and [recording costs](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-disk-measurement/analysis/quiet-start-and-recorder.json)
retain those separate measurements.

## Decision and next step

This attempt does not resolve whether the Checkstyle correction saves time.
Another identical run on this workstation would leave the measurement problem
unresolved. The agreed
[stopping conditions](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-storage-pressure/next-step.md)
require a separate decision about the environment or observation policy before
another allocation.

The preferred next step is to prepare a concrete measurement proposal: establish
whether a machine can be reserved for the experiment, and assess the remaining
recording cost using the retained evidence. If we must keep using this shared
machine, any proposed observation rule needs its own justification and control,
fixed before timing. Relaxing the current threshold after this result would not
qualify it. No new timing allocation or recording change is authorized by this
report.

The allocation used **one project build and zero comparison JVMs**, within its
50/50 ceilings. Both worker sessions closed. At closure it retained 9.33 GiB of
state and 23.53 MiB of diagnostic samples, within the ten-hour, 80 GiB and
256 MiB limits. The raw runner preserves one unknown launch reservation; the
refusal receipt records that the second native command never launched. The
[closeout](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-disk-measurement/receipts/allocation-closeout.json)
preserves both counts.

The export includes timings, launch decisions, disk receipts, compressed samples
and runner sources. Large output trees and inventories remain local. Its audit
checks the exported records and arithmetic; it performs no output comparison:

```sh
python3 -B benchmarks/results/buildopt-product-viability-v1/bo-06-disk-measurement/verify-evidence.py
```

Local state remains under
`.tools/state/buildopt-product-viability-v1/bo-06-disk-measurement-2026-09-15`.
