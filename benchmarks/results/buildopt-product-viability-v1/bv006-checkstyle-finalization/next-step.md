# Next step: four-build measurement control

Status: pending. The finalization comparison is closed. No new builds or watcher have started.

The supported implementation is correctness-qualified, but the whole-request result is mixed: 124.537→83.160s in one replication and 83.003→97.339s in the other. Both first measured requests are near 83s; both second requests are slower despite reversing the implementation order. Preserve that result and its host/paging qualifications. Do not revisit the deprecated callback or seek an exact retrospective cause for the old transient spike.

Run a small A/A control: **identical supported V2 code on both arms**, same frozen 6→7 history and schedule, two cold builds plus two measured builds, **four owner starts total**. Retain the exact qualified owner consumer, native 8-CPU affinity, phase observation, source guards and complete live/independent C5 checks.

The [detailed plan](../../../../docs/plans/buildopt-checkstyle-measurement-control-v1.md) specifies source identity, proof dependencies, measurements, outcomes and proposed limits. Freeze the allocation prospectively: four builds, no timing-driven retry, at most four comparator JVMs, 90 minutes, 32GiB new state and 40GiB minimum free space.

An absolute customer-time difference of at least 1s **and** at least 5% of the faster request is `SPURIOUS_MATERIAL_SIGNAL_IN_AA`. Otherwise report `NO_SPURIOUS_MATERIAL_SIGNAL_IN_THIS_CONTROL`; one control does not establish precision or qualify G0/G3. A source/correctness failure makes the experiment incomplete.

If a false material signal appears, keep value inference closed and design the smallest repair to the timing sequence. Consider moving heavy preservation/checking outside the interval between paired native commands while retaining immutable output sets, full correctness, source guards and recovery proof. Do not remove validation or reset caches to make the result favorable.

The proposed native 17–20 comparison is **deferred behind this measurement control**. Its [static source preflight](proof-v2/analysis/next-window-source-preflight.json) already passes for all four revisions; no native builds ran there. That later window would include inactive original 18 and modeled active opportunity at 19 and 20, at most 16 owner starts. Even a positive selected screen must be followed by a complete chronological denominator and separate adoption/maintenance cost proof.

The adaptive product still needs distinct responses to correctness invalidation, performance drift and host pressure. Invalid context/report identity stops reuse immediately. One slow build with successful reuse on a shared host does not by itself justify an expensive optimization search. Keep all samples and unchanged product thresholds. Formal held-out 21–100, other repositories and adaptive implementation remain deferred.
