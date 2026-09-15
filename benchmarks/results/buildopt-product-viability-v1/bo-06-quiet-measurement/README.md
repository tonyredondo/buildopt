# A passing control and an interrupted development replay

Date: 2026-09-15. BO-06 remains partial.

The identical-code control passed. Development then stopped after 17 of its
42 scheduled builds because the next build could not obtain low storage pressure
within the allowed three minutes. All executed builds succeeded, and the eight
completed development pairs produced matching results. The sequence is incomplete,
so it does not establish a saving across everyday code changes.

This closes one allocation. The earlier
[negative control](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-qualified-measurement/README.md)
remains negative; this run does not replace it or prove what caused its timing
spike. The [governing tracker](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-research-execution-plan-2026-09-14.md#execution-tracker)
keeps BO-07 and protected changes 21–100 deferred.

## What ran

The subject was Elasticsearch's `:server:precommit` command, which compiles
required code and runs checks before a change is committed. Both sides used
the same retained build correction, Gradle build cache, eight CPU assignments
and output recorder. Configuration Cache and the diagnostic Java phase agent
were disabled. No additional CPU profile or retry was used.

The control ran identical corrected code on both sides over development changes
17–20. After it passed, the development replay compared the baseline with the
Checkstyle correction: a change that lets the source-code rules checker reuse
work on unchanged files. That replay was scheduled for the cold anchor and all
20 development changes, keeping each side's build state between commits.

The [verified runner and waiting policy](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-quiet-start-integration/README.md)
require 30 seconds with low CPU, storage and memory pressure before each launch.
Waiting consumes the research allocation and remains outside the measured build.
Both sides use the same rule. Passing it does not isolate the machine or guarantee
that conditions stay quiet while the build runs.

## Control result

| Development change | Side N | Side I | N minus I |
| --- | --- | --- | --- |
| 17, cold preparation | 219.965 s | 209.787 s | 10.177 s |
| 18 | 28.557 s | 29.302 s | −0.745 s |
| 19 | 35.313 s | 34.879 s | 0.434 s |
| 20 | 64.624 s | 64.181 s | 0.443 s |

The registered rule uses changes 18–20. Their totals were 128.494 and 128.363
seconds: an absolute difference of **0.132 seconds, or 0.103% of the faster side**.
The rule stops the trial only when the difference reaches both three seconds
and 5%. All eight builds, four live output comparisons and four independent
reconstructions passed. The cold pair remains recorded and charged.

This is `CONTROL_PASSED`, not an optimization saving or a general precision
claim. Four control builds retain host-pressure flags; none was excluded.

## Development result

| Change | Baseline | Checkstyle correction | Result |
| --- | --- | --- | --- |
| 0, cold anchor | 207.950 s | 209.710 s | Outputs match |
| 1 | 7.367 s | 7.634 s | Outputs match |
| 2 | 6.190 s | 6.157 s | Outputs match |
| 3 | 19.015 s | 18.323 s | Outputs match |
| 4 | 17.912 s | 17.590 s | Outputs match |
| 5 | 5.890 s | 5.989 s | Outputs match |
| 6 | 5.970 s | 5.845 s | Outputs match |
| 7 | 125.947 s | 87.815 s | Outputs match |
| 8 | 7.290 s | Did not start | Waiting limit reached |
| 9–20 | Did not start | Did not start | Earlier interruption |

All eight completed pairs passed both live and independent comparison. The
baseline build at change 8 also succeeded, but has no counterpart. Its time
cannot be treated as a saving. Nine of the 17 completed builds retain the
registered host-pressure flags, including both sides at change 7. Its favorable
difference therefore cannot by itself establish the correction's effect or
justify a result for the missing sequence.

The [42 scheduled outcomes](./analysis/prefix-retained-summary.json) preserve
25 unrun builds explicitly. The unchanged runner result is `INCOMPLETE_EVIDENCE`:
change 8 is a partial pair and changes 9–20 depend on incomplete preceding work.
Zero durations in those raw slots represent missing evidence, not zero-cost
builds. The runner conservatively records one unknown launch reservation. Its
quiet-start receipt and the frozen code's return before native launch explain
that reservation; this report does not rewrite the raw counter.

## Why the next build did not start

The candidate side at change 8 waited 180 seconds. Every one of the 179 measured
intervals exceeded the 10% storage-pressure threshold. The average was 77.43%;
CPU pressure averaged 0.093%. These are Linux pressure measurements: the fraction
of time with at least one task waiting for a resource, not disk utilization.

The outer recorder used 0.089 seconds of CPU during this wait and recorded no
storage reads and about 0.5 MB of writes. Earlier capture work had written about
232 MB. These observations do not identify the cause of the sustained pressure;
earlier writes and other activity can continue to affect storage afterward.
No particular process is blamed.

The waiting rule prevented a launch under those conditions, as intended. It did
not solve the wider measurement problem: pressure also rose during some builds
that began with a qualifying quiet window. The next investigation should use
these retained observations to understand sustained storage pressure and define
conditions for completing development. It should not raise the threshold,
discard this attempt or resume this closed allocation.

## Accounting and evidence

The allocation reserved 50 owner builds and 50 comparison JVMs. It used
**25 owner builds and 24 comparison JVMs**: eight of each in the control, then
17 builds and 16 comparisons in development. No retry or protected build ran.
All four worker sessions closed. The initial ten-hour, 80 GiB and diagnostic
limits were respected at closure; exact usage is in the
[allocation record](./receipts/allocation-closeout.json).

Quiet waiting took 1,396.531 seconds across 26 attempts, including the refusal.
It remains research cost. Recorder CPU and I/O counters are separate observations,
not additional build elapsed time. Any future saving measured in this mode would
apply to the recorded workflow; BO-09 still requires comparison with ordinary
Gradle before a product-performance claim.

The export includes all available attempt receipts, timing, quiet observations,
recorder costs, compressed process samples, qualified comparator and frozen
runner sources. Large output trees, caches and state inventories remain local.
The portable check verifies exported records and arithmetic; it does not rerun
the output comparisons or independently inspect those retained output trees.

Run it from the repository root:

```sh
python3 -B benchmarks/results/buildopt-product-viability-v1/bo-06-quiet-measurement/verify-evidence.py
```

The audit also rejected [four altered closure claims](./validation/audit-cases.json),
including a false passing result for the incomplete development sequence.

The [evidence index](./evidence-manifest.json) binds every exported record and
lists the refused attempt's absent files. The [control summary](./analysis/control-summary-v4.json),
[development accounting](./analysis/prefix-retained-summary.json),
[waiting and recorder observations](./analysis/quiet-start-and-recorder.json) and
[development pressure flags](./analysis/prefix-host-pressure.json) support
this report. Local state is retained under
`.tools/state/buildopt-product-viability-v1/bo-06-quiet-measurement-2026-09-15`.
