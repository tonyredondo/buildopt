# Why the control still needs better measurement conditions

15 September 2026. BO-06 remains partial.

The slow build started while the machine was already experiencing substantial
disk and memory waits. We also found that supervising the experiment consumed
almost one CPU core during most builds. These are concrete measurement problems
to address. The retained observations do not identify the cause of the entire
43.26-second difference in the
[previous control](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-qualified-measurement/README.md).

This block analyzed those existing eight builds and tested a small utility that
waits for a quiet start. It launched no Elasticsearch build or comparator JVM
and read no protected history. The previous result, candidate and stopping
rule are unchanged.

## What the records establish

The last pair ran identical corrected code on Elasticsearch change 20. These
are whole-request times, including the wrapper's work inside that boundary:

| Observation | Side I | Side N |
| --- | ---: | ---: |
| Whole request | 68.892 s | 112.505 s |
| Disk-wait pressure in the preceding 30 seconds | 3.20% | 53.40% |
| Memory-wait pressure in the preceding 30 seconds | 0.01% | 32.59% |
| Supervisor CPU time during the native invocation | 64.62 s | 87.00 s |

Linux pressure counters report how often at least some work was waiting for
a resource. These percentages describe waiting time, not disk utilization or
which process caused it. The preceding windows are estimates from recorded
intervals, including partial intervals at their boundaries. They are a
diagnostic reconstruction, not a reason to remove a measured build.
See the [kernel's pressure accounting documentation](https://docs.kernel.org/accounting/psi.html).

The bulk copying of build results happened **after** each native invocation.
There was no overlap between any of the eight native invocations and any
recorded bulk-capture phase. The previous copy finished 14.02 seconds before N
started at change 20. We cannot tell whether that copy left storage or memory
work that affected the following build.

During a build, the recorder writes task information and buffers command logs.
The bound task recorder does not explicitly force each event to disk. Separately,
the supervisor checks processes and repeatedly walks the experiment's complete
file tree to enforce its disk limit. That tree includes retained results and
both copies of the project. The polling interval is 100 ms; a slow walk can
leave the next poll ready as soon as it finishes. This reads file metadata,
not every file's contents.

The supervisor used roughly one core through most invocations, on CPU 8,
separate from the build's CPUs 0–7. Its recorded CPU time includes all its work;
we did not profile how much each check consumed. The longer build also gives
the observer more time to run. The extra CPU total alone cannot explain why
that build was slower.

## What the records cannot settle

The process samples cover the worker supervisor and Gradle processes. They do
not cover the outer process that copies results between builds. Per-thread
waits and historical per-group disk pressure are also missing. The slow build
has an 11.83-second sampling gap. Sampled I/O counters therefore cannot provide
complete attribution or rule out recording work as a contributor.

Changing the Checkstyle correction is not supported by this diagnosis. Its
checks were slightly faster in the slower build, and the required outputs
matched. The remaining question is whether we can measure complete builds
without the measurement process or pre-existing host activity obscuring the
difference we want to test.

## A quiet-start check, tested separately

[quiet_start.py](https://github.com/tonyredondo/buildopt/blob/main/dev/history-replay/quiet_start.py)
reads three small Linux pressure counters once a second. It requires at least
30 continuous seconds in which every observed CPU, disk and memory pressure
interval is at or below 10%. A read taking more than 100 ms, a sampling gap over
three seconds, or a counter reset prevents admission for that window. It waits
at most 180 seconds by default, retains the observations and exits without
launching a build.

Nine tests cover the window, thresholds, bursts, missing observations, failure,
cancellation and timeout behavior. A single live observation found a quiet
window after 30.01 seconds, using 7.53 ms of observer CPU time. All samples and
the [test log](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-recording-pressure/quiet-start-tests.log)
are retained. This establishes the utility's behavior and its cost in that
observation. It does not establish stable build timings.

The utility is **not integrated into the replay runner yet**. A passing result
from running it separately cannot admit a later build.

## Conditions for the next attempt

1. Integrate the quiet window immediately before each build, after expensive
   preparation and copying. Bind the observation to that request and verify
   freshness at launch. Test the actual runner's success, busy-host, observation
   failure, cancellation and deadline paths before using it on Elasticsearch.
2. Retain waiting time as experiment cost within the allocation. A timeout stops
   before launch and leaves the sequence incomplete. Keep every build that does
   start, including later pressure spikes; no retry or filtering based on timing.
3. Record the outer recorder's CPU and I/O separately from the supervisor and
   native build. Account for this observation's own cost. Establish how much the
   full-tree disk guard contributes before changing it; any replacement must
   preserve disk-limit and cancellation checks.
4. Freeze the resulting runner, observation policy and resource limits before
   a fresh control. Recheck complete admission and retain comparator qualification
   against the actual inputs. The correction, output requirements and control
   rule remain unchanged: stop at a difference of at least three seconds and
   5% of the faster measured side. This block allocates no new owner builds.
5. Run the complete development sequence only after a fresh control passes.
   A quiet start cannot guarantee a quiet build. If the control still fails,
   retain it and diagnose the remaining uncertainty; do not repeatedly rerun
   the slow pair until it passes. BO-07 and changes 21–100 remain deferred.

The next deliverable is the runner integration and its bounded verification.
It is a prerequisite for measuring sustained savings, not a new optimization
or evidence that BuildOpt saves time.

## Reproduce this diagnosis

Run from the repository:

```bash
python3 -B benchmarks/results/buildopt-product-viability-v1/bo-06-recording-pressure/analyze.py --check
python3 -B -I dev/history-replay/quiet_start_test.py
```

The analysis checks the published input hashes and rebuilds all eight timing,
capture-overlap and preceding-pressure records. It also reconstructs the last
pair's sampled process counters, retaining coverage gaps. Source checks use
the measured runner archive and confirm the actual task-recorder binding.
The [derived records](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-recording-pressure/analysis.json)
contain those inputs and limits. Neither command starts a build.

The tested quiet-start utility, tests and check scripts are also retained in
`observer-source.tar.gz`, with individual file hashes. The diagnosis can still
be verified after those files change in the working repository; it uses the
archived source for its historical bindings.
