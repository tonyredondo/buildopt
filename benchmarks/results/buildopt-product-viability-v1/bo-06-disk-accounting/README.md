# Reducing the cost of supervising disk use

The changed disk guard is qualified on small fixtures and the next measurement
inputs are frozen. In the fixture with 32,768 retained files, twenty checks
used about **74% less CPU**, including initial setup and closing the observer.
This is a reduction in measurement work. It is not an Elasticsearch speedup.

The [previous diagnosis](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-storage-pressure/README.md)
found that the supervisor repeatedly recounted earlier results. The replacement
counts the whole tree once per request, then recounts directories whose files
changed. Every regular-file entry still counts. Missing notifications or an
unsupported change return that request to the complete scanner. Limits, output
capture, the quiet-start rule and the savings criteria are unchanged.

## Measured cost

Each size was tested twice with both versions, reversing their order. Files
were added, grown and removed during twenty checks, including temporary
directories. There were eight measured cases in total.

| Retained files | Repetition | Previous CPU | New CPU | Previous elapsed | New elapsed |
| --- | --- | --- | --- | --- | --- |
| 16 | 1 | 1.664 ms | 3.979 ms | 1.616 ms | 8.745 ms |
| 16 | 2 | 1.656 ms | 3.363 ms | 1.578 ms | 8.427 ms |
| 32,768 | 1 | 691.407 ms | 176.474 ms | 684.952 ms | 162.498 ms |
| 32,768 | 2 | 700.910 ms | 182.436 ms | 695.079 ms | 169.184 ms |

The small case became slower by about seven milliseconds overall. The large
case reduced total CPU in both repetitions, including the cost of watching the
files. These fixtures do not establish how often the fast path will remain
usable during a real Gradle build. Directory moves, watch exhaustion and other
invalidations can still restore the old cost. The
[accounting contract](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-disk-accounting/accounting-contract.md)
and [individual measurements](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-disk-accounting/receipts/disk-performance.jsonl)
retain the conditions and setup costs.

## What passed

The filesystem tests compare the live count with a complete scan after changes
to files, directories, hardlinks, symlinks and sparse files. They also exercise
lost notifications, missing permissions, low free space and cancellation.
The initial regression test observed the old scanner reopening an unchanged
directory; the replacement avoids that repeated walk.

The final runner completed 24 native fixture starts. The checks cover matching
outputs, a wrong result, native fallback, source drift, missing evidence,
quiet-start refusal and process closure after driver loss. A live size breach
cancelled its child. The independent result checker rejects a missing disk
observer receipt. Unit checks, race checks and Go vet passed. No project build
or output-comparison JVM ran in this block.

The storage sampler now records host device activity, pending writes, its own
CPU/I/O, the recorder's CPU/I/O and group I/O pressure where readable. In the
small integration fixture, 23 samples averaged 0.499 ms each and used 8.515 ms
of CPU in total. The sampler used 21.560 ms of CPU over its whole 23-second run,
including its other work and writing samples. Optional group counters were
unavailable in those observations and remain labelled as such. The new data
cannot explain the cause of an earlier wait retrospectively.

The [verification record](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-disk-accounting/verification.json)
links the checks, retained fixture records, limits and source hashes. The
[runner archive](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-disk-accounting/runner-source.tar.gz)
can be checked with `./dev/check-disk-accounting --unit`. Base CI also runs this
check and the storage-observation tests.

## Next step

The [new input freeze](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-disk-accounting/next-control-admission.json)
binds the runner, sampler, unchanged comparator, policy and protocol. Full
validation admits the control proposal and refuses development until a genuine
passing control exists for this identity. An earlier control cannot qualify
this changed implementation. No timing allocation was activated here.

Follow the existing
[measurement conditions](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-storage-pressure/next-step.md):
prepare a fresh bounded allocation, run eight identical-code control builds,
and run the complete 42-build development sequence only if the control passes.
Check actual observer fallback and setup cost, and retain all storage samples,
including explicit unavailable fields. The limits remain 50 project builds,
50 comparator starts, ten hours and 80 GiB for that separate block.

BO-06 remains partial until the actual-owner measurement requirements pass.
The old interrupted allocation stays closed. Changes 21–100 and BO-07 remain
deferred, and ordinary-workflow value still needs BO-09.
