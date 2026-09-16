# Quiet periods returned while the experiment was stopped

Date: 2026-09-16. BO-06 remains partial.

The workstation met the existing quiet-window rule during part of a three-minute
observation. The first qualifying window ended after 58 seconds. The earlier
experiment recorder was stopped, its worker group was gone, and no system file
indexer appeared in the 19 process checks. No project build or comparison JVM ran.

This resolves one question: the sustained disk pressure that blocked the
[previous control](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-reserved-host/README.md)
was no longer present throughout the observation. We still need to find out
whether preparing the experiment brings that pressure back.

## What we observed

The observation ran from 04:45:41 to 04:48:41 UTC, reading CPU, disk and memory
pressure once a second. We kept all 180 samples and evaluated the same rule:
at least 30 seconds with every complete observed interval at or below 10%
pressure, with the existing limits on read time and gaps between samples.

| Observation | Result |
| --- | --- |
| Average disk-wait pressure | 0.71%, compared with 62.38% during the previous refused wait |
| Intervals above the disk threshold | 3 of 179; the largest reached 27.34% |
| First qualifying window | 58 seconds after observation began |
| Qualifying window evaluations | 94; these overlap and are not independent trials |
| Final window | Did not qualify, because it contained a recent spike |
| CPU and memory intervals above the threshold | None |

Pressure measures time when work waited for a resource. It does not measure
disk utilization. The two averages describe separate observations under different
conditions; their difference cannot be credited to a particular process or fix.

The three disk spikes occurred near seconds 27, 152 and 174. Brief background
pressure therefore remained even with the earlier recorder and indexer absent
at our checks. Sampling process state cannot rule out short-lived activity
between those checks. The lightweight observer used about 0.267 seconds of CPU
over three minutes; it also has a measurement cost.

## What follows from this

The existing rule can find a quiet window on this machine. A quiet workstation
at one moment does not ensure a quiet build later. This host-only observation
does not qualify a worker launch, replace the identical-code control, or
establish an optimization saving. The previous incomplete results stand.

The next useful check is preparation alone: observe storage before, during and
after creating the experiment's environments and verifying their inputs, with
no Gradle launch. Keep the same pressure rule and retain any pending writes.
That would show whether quiet windows survive preparation and how long recovery
takes. If pressure remains high, use those phase records to locate the work
responsible before attempting another build sequence. This would still be a
prospective diagnosis, not proof of what caused the earlier failure.

This block ends with the host observation. It starts no new control allocation,
changes no service or threshold, and leaves BO-07 and protected changes 21–100
deferred. The
[governing tracker](https://github.com/tonyredondo/buildopt/blob/main/docs/plans/buildopt-research-execution-plan-2026-09-14.md#execution-tracker)
remains the work queue.

## Evidence and checks

The [raw observations](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-host-observation/observation.json)
include all pressure counters, process checks and storage snapshots. The
[calculation](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-host-observation/analysis.json)
retains every interval and window decision. The
[prospective limits](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-host-observation/allocation.json)
allowed one 180-second observation, a 210-second ceiling, at most 4 MiB of
diagnostics and no retries or project builds. The observer exited normally.

An independent check using the original runner's window function agreed on
all 180 decisions. The extracted source was checked against the previously
published frozen archive; its provenance and test result are in `validation/`.
The collection script and the unchanged policy are retained alongside the data.
To verify the published calculation without collecting another sample:

```sh
python3 -B benchmarks/results/buildopt-product-viability-v1/bo-06-host-observation/analyse.py --verify
```

Local task record:
`.tools/state/buildopt-product-viability-v1/bo-06-host-observation-2026-09-16/task-state.json`.
