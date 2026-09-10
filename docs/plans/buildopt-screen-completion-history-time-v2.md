# Disjoint history: time-only budget amendment

2026-09-09, registered before any disjoint-history start. The owner authorized
increasing the required budget and completing the experiment autonomously.

The corrected 8/4-CPU profiles are verified. The 2-CPU profile continues with
its original inputs and bounds. One preflight in that profile took 680.692 s;
other completed preflights took 5.078–32.237 s. Small-file reads and elevated
I/O pressure were observed on the rotational filesystem. These are research
preparation costs, not customer build durations or a quantified causal account
of the entire delay.

To give the unchanged historical test adequate wall-clock headroom, its future
manifest is versioned separately with only these changes:

| Limit | Original | Future history only |
| --- | --- | --- |
| Deadline | 2026-09-10 02:00:01.505713 UTC | 2026-09-10 07:00:00 UTC |
| Maximum elapsed run | 4 hours | 6 hours |

All 30 historical Gradle starts, 15-minute individual request bounds, the
85-GiB per-run limit, the shared 160-GiB allocation, 40-GiB free-space floor,
candidate, baseline, commits, task/output checks and economic conditions stay
unchanged. CPU counts remain 24 new / 37 including earlier attempts. Combined
new native execution remains at most 54 for this round and 65 including the
superseded candidate round. No retry or additional CPU sample is authorized by
this amendment. Formal validation 21..100 remains untouched.

The old coordinator is paused while its existing 2-CPU runner continues. The
replacement waits for that runner and its observer to finish, then closes the
old coordinator, analyzes the same eight builds and starts history at the first
qualifying profile, 8 CPUs. It changes no running native process or CPU allocation.
The original manifests, scripts, freezes and results remain immutable; the new
manifest, path-selecting wrappers, coordinator and this amendment are frozen
separately. Static comparison permits exactly the two limit changes above. The
same qualified CLI must validate the manifest as part of `run` before launching
any workflow; completed runtime evidence will close that integration proof.
A separate read-only validation scan was stopped during CPU preparation and
retained as a zero-start probe. It will not be duplicated while native timing
is active.

State locator: `engineeringPrefix.screenCompletionFixed` in
`.tools/state/buildopt-product-viability-v1/task-state.json`.
New controller: `bv006-screen-completion-fixed/complete-v3.py`.
New manifest: `profiles/history-8/manifest-time-v2.json` below that phase root.
