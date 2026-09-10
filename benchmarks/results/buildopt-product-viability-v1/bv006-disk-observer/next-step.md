# Next BV-006 step: isolate research CPU without weakening live guards

Status: **pending**. The completed [fixed control](./README.md) establishes
cheaper scans and correct independent liveness checks, but its CPU and native
latency intervals cross zero. Repeating this old/new control or tuning small
scanner details is not the next investigation. Keep the verified code and all
negative/uncertain timing evidence.

Hypothesis: the research supervisor competes with Gradle for the same eight
CPUs. Isolating research observation on a spare physical core may improve timing
fidelity while retaining the same complete live disk polling. This is a
measurement-control hypothesis, not a new customer worker/heap optimization.
The retired product-tuning routes remain closed.

The current host reports CPUs 0-31, with native CPUs 0-7 on physical cores 0-7
and their SMT siblings 16-23. CPU 8 is on physical core 8 (sibling 24). Recheck
allowed masks and topology before selecting anything; availability is not a
reservation or permission to move unrelated workloads.

| Step | State | Work and proof | Required outcome |
|---|---|---|---|
| I1: recover and review constraints | pending | Verify this seal, main source and owned worktrees; inspect manifest/worker launch validation, the pinned Go process-inheritance contract, allowed CPU masks and thread topology | An explicit CPU allocation for native versus research work; unchanged native source, argv, flags and eight-CPU budget |
| I2: reviewable isolation mechanism | pending | Prototype only in a task-owned bound source version; consider a locked OS thread with saved/restored affinity around child start, keeping the supervisor on a spare physical core. Review Go/runtime inheritance and all restoration/error paths before use | No leaked affinity, no silent fallback, no weakening of live disk/ownership/cancellation guards |
| I3: focused qualification | pending | Prove actual parent/child/all-thread affinity, daemon reuse, normal output/cache behavior, launch failure, failed affinity change, restoration, cancellation and driver death | Actual observed masks and all affected native/lifecycle contracts pass before owner timings |
| I4: register one fixed control | pending | Compare the current corrected observer sharing CPUs 0-7 with the same observer isolated outside that physical-core set. Keep native CPUs 0-7, argv, state, source, capture policy and order the same; fix eight warmups and eight measured builds per version | Source-bound 32-request control with complete counts, all failures retained, declared uncertainty and no optional stopping |
| I5: reconstruct the outcome | pending | Measure native wall, daemon phases, CPU masks, observer CPU and quality/variation; charge all research work even on the spare core | Supported improvement or explicit negative/inconclusive result; isolation itself is not C5 saving or G0 readiness |
| I6: owner readiness | blocked | After a supported instrument change, prospectively register fresh C5 N/I plain/capture qualification under the original 10-ms/100-ms gates and interval/sizing rules | G0 prerequisites pass or remain insufficient; no reuse/accumulation of these control rows |
| I7: chronological product evidence | blocked | After all G0 prerequisites pass, complete engineering 0..20 and freeze untouched confirmation inputs | Resume the agreed lifecycle-value program; validation and H3 remain untouched until then |

The existing GRADLE manifest validator accepts native launchers named `gradle`
or `gradlew`. Do not rename another executable or inject an unbound command to
bypass that restriction. Any required launch-contract change must be explicit,
source-bound and qualified; otherwise stop with the incompatibility. Native
argv and runtime behavior must remain equivalent across the two versions.

Proposed ceiling for I1-I5: three hours, at most eight fixture Gradle starts and
32 owner starts (40 total), zero standalone JVM/JFR helpers, at most 128
non-Gradle fixture requests, 60 GiB new state and at least 40 GiB free. Register
exact commands, tested versions, CPU masks and costs before execution. No more
native work belongs to the closed disk-observer allocation. A failed prerequisite,
fixed control failure or exhausted bound closes the attempt with its evidence.

Use only task-owned process settings. No host configuration, package install,
privilege change, persistent service, heap tuning or unrelated-process change
belongs here. The original timing gates and live resource limits remain intact.
If isolation proves insufficient, record that result before selecting another
instrument approach; do not repeat until green or claim an optimizer win from
research resource allocation.
