# Checkstyle control with the qualified comparator

15 September 2026. Decision: `CONTROL_MATERIAL_DIFFERENCE`.

The same corrected build produced a 43.26-second difference across the three
measured changes. That exceeds the control's stopping rule, so the planned
42-build development sequence did not start. This leaves the correction's
saving across the complete development history unmeasured.

All eight builds succeeded. Their four output comparisons passed, and a
separate reconstruction confirmed all four. The problem in this attempt was
the timing difference, not a mismatch in the required build results.

## What ran

We replayed Elasticsearch changes 17–20 on two separate copies, keeping each
copy's build state between changes. Both used the same supported Checkstyle
correction. Checkstyle checks Java source against coding rules; our correction
lets it reuse previous checks while preserving the complete reports.

The command was `:server:precommit --continue`, with eight workers, native build
caching enabled and Configuration Cache disabled. Both copies used the frozen
output recorder and qualified comparator. Execution order alternated between
the copies. The first pair started fresh and was excluded from the timing
decision, as specified before the run.

| Original change | Side N | Side I | Included in the timing decision |
| --- | ---: | ---: | --- |
| 17 | 217.366 s | 230.011 s | No, initial pair |
| 18 | 29.324 s | 28.947 s | Yes |
| 19 | 35.404 s | 36.135 s | Yes |
| 20 | 112.505 s | 68.892 s | Yes |
| **Measured total** | **177.233 s** | **133.974 s** | Three changes |

The absolute difference was 43.259 seconds, or 32.29% of the faster side.
The registered rule stops at a difference of at least three seconds **and**
5%. Both conditions were met. The generic saving fields in the
[machine-readable result](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-qualified-measurement/analysis/control-summary-v4.json)
describe the difference between these identical-code runs; they are not an
optimization benefit.

## Where the difference appeared

Change 20 accounts for nearly all the difference. Its 1,326 recorded tasks had
the same identities and outcomes on both sides. Several compilation tasks
took much longer in N: compiling the security module took 29.99 seconds versus
4.21 seconds in I. These tasks overlap, so their durations cannot be added to
explain the total build time.

The main Checkstyle task took 10.01 seconds in N and 10.73 seconds in I.
Its other two source checks were also slightly faster in N. The large slowdown
therefore appeared elsewhere in the build.

The host recorded substantially more disk-wait pressure during N at change 20:
38.80% versus 4.25% in I. N also had an 11.83-second gap between process samples.
This is consistent with a temporary execution stall. The observations do not
establish which process or storage activity caused it, or whether recording
the experiment contributed. Four of the eight requests retain pressure flags;
none was removed from the result. See the
[task comparison and observation limits](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-qualified-measurement/analysis/control-variation.json).

## Completion and evidence

The allocation is closed. It used eight owner builds and eight comparator JVMs:
four live comparisons and four independent reconstructions. There were no
retries or protected-history builds. Both owned service sessions and their
process groups are closed. The runner, candidate and qualified policy retained
their frozen identities.

Measurement and closure took 62.44 minutes, including preparation and output
checks. Retained local state used 13.02 GiB and diagnostic samples used
53.42 MiB, within the approved limits. Native-state preparation is recorded as
experiment work; this control does not measure the cost of adopting the product.
The [allocation closeout](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-qualified-measurement/receipts/allocation-closeout.json)
binds the counts, input checks and process closure.

Run `python3 -B benchmarks/results/buildopt-product-viability-v1/bo-06-qualified-measurement/verify-evidence.py`
from the repository to verify the 291 exported evidence files, source archive,
build records and timing arithmetic. This starts no build or comparator JVM.
Complete captured output trees and caches remain in the local task state;
the portable audit does not repeat the full output comparisons.

## Next step

BO-06 remains partial. Before another control, we need to address the disk
pressure observed in this attempt and define the measurement conditions before
launch. The retained task records and process samples are the starting point;
they do not justify changing the correction or weakening the stopping rule.

A fresh passing control under the qualified policy is still required before
the complete development sequence and final BO-06 freeze. This allocation
cannot be reopened to retry the slow pair. The earlier selected savings remain
evidence for their original comparisons. BO-07 and protected changes 21–100
remain deferred under the
[governing plan](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-research-execution-plan-2026-09-14.md).
