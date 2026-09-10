# Next BV-006 diagnostic: daemon work and supervisor interference

Status: **pending; not executed**. This is the successor to the registered
precision stop, not another batch of the same qualification. C5 value,
engineering 0..20, validation 21..100 and H3 remain blocked/unconsumed.

## Question and evidence

Why do unchanged plain owner requests vary by seconds inside the daemon-command
interval? The complete 80-request pilot has one daemon PID per arm, with no restart.
Its largest N paired deviations are also present in daemon-command duration.
The collector attribution showed similar direct work in both arms. These
observations do not yet isolate JVM CPU, GC/JIT, I/O, locks, scheduling or
interference from the owned supervisor's live resource scan.

## Fixed scope to register before execution

Keep the frozen Elasticsearch anchor, baseline/C5 patches, native command,
JDKs, eight-worker limit, affinity, independent state and daemon policy.
Use fresh N/I states. After eight fixed warmup requests per arm, run exactly
eight measured **plain** requests per arm in alternating order. This is
32 owner Gradle starts, plus at most two small-fixture preflight starts.
No timing-based exclusions, early success or continuation until a spike occurs.

Before launching, declare one separate allocation: at most 34 Gradle starts,
16 standalone diagnostic/helper JVM starts, three hours, 60 GiB new artifacts,
and 40 GiB minimum free. Freeze exact tool commands, event settings, code/runtime
bindings and time alignment in a machine-readable manifest. These are proposed
bounds, not evidence that a profiler is already prepared or running.

| Step | Work | Required outcome |
|---|---|---|
| Recover inputs | Verify this seal, current source, owner pins and quiescent units; preserve all failed results | One explicit identity and fresh allocation; zero reused task outputs |
| Qualify observation | Check the pinned JDK's available JVM profiling tools on the small fixture; collect direct owned-worker CPU and scan-duration/count evidence | Valid event coverage and time alignment; unchanged native outputs; explicit observer cost/limitations |
| Capture 32 owner requests | Record owned-daemon CPU samples, compilation, GC/safepoints and blocking/I/O events; record supervisor CPU and resource-scan intervals separately | All requests, warmups, helper starts and raw streams retained; exact per-request/daemon identity; no primary performance or G0 claim |
| Independently reconstruct | Align native/daemon intervals and event data; inspect all requests and compare slow intervals with the other fixed requests | Classify supported causes and retain unknown/overlapping time without adding parallel durations or assigning residuals by subtraction |
| Decide one correction | Require a mechanism supported by the captured event timeline and a focused reproduction/control | One concrete correction with required output/lifecycle/timing proof, or an explicit inconclusive result; no unsupported rewrite |

Only attach to the exact task-owned daemon PIDs. Do not change host permissions,
system services, global JVM settings, native heap/worker policy, or undelegated
I/O accounting. Count any `jcmd`, `jfr` or other helper JVM separately. Profiler
and worker diagnostics can affect scheduling and are never qualification data.
A missing event, unsupported counter or unclear timeline remains missing proof.

The original 10-ms wrapper/100-ms imbalance gates remain unchanged. A supported
correction needs its affected behavior proved and a freshly frozen owner
qualification before engineering can resume. Replaying or trimming the prior
samples, increasing the sample count after seeing success, or moving to the
validation history does not solve this prerequisite.
