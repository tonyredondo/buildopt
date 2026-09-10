# Checkstyle finalization v1

Date: 2026-09-10. Status: correctness verified; frozen comparison prepared.
Owner authorization: execute the safe-finalization follow-up, with shared-host
contention recorded. This follows the [accepted investigation](../findings/buildopt-checkstyle-shared-host-disposition-2026-09-10.md).

## Design and acceptance

Replace the history Commit `doLast` action with a task-graph completion callback
scoped to the exact task path. Preserve Prepare invalidation and before/after
context checks. Commit requires an executed, non-skipped, successful native task.
The worker binds its completed XML digest in the proposal; Commit rejects a
missing or subsequently changed report. Preserve native task type, checker,
outputs, task-local state, failure/cancellation handling and atomic publication.

The [Gradle task-graph callback](https://docs.gradle.org/current/javadoc/org/gradle/api/execution/TaskExecutionGraph.html)
is deprecated for Configuration Cache use. This experiment stays within the
already qualified Gradle 9.7.1 owner with Configuration Cache disabled; it does
not qualify a general integration for other Gradle versions or cached configuration.
Verify the consuming callback and worker-lock behavior with actual Gradle.
Publishing directly from the worker loses final task outcome validation; an
explicit worker await inside a trailing action retains the lock-reacquisition
problem. Neither alternative is selected.

| Step | Status | Required proof and outcome |
|---|---|---|
| Design and actual callback | verified | Actual final action releases project locks; completion callback runs after worker success, without adding a task action or global listener requiring reacquisition. |
| Affected correctness | verified | Cold/warm/change XML equivalence; task-local history; native/adapter errors and real cancellation; stale proposals, skipped/no-source work, changed context and changed/incomplete report cannot publish success. Existing unaffected proof remains tied to unchanged inputs. |
| Frozen comparison | pending | Current optimized versus revised optimized on original ordinals 6→7; two independent balanced replays, eight owner builds total at eight CPUs. Warmup and measured requests kept separate. Complete C5 output equivalence required. |
| Shared-host observation | pending | Once-per-second host CPU busy/idle, run queue and CPU/I/O pressure, plus owned-process CPU. Retain both arms and predeclare symmetric contention interpretation before launch; no post-hoc exclusion of slow samples. |
| Decision | pending | Report all whole-request times and finalization phases. An improvement in task span alone cannot admit the revision. Preserve negative history and distinguish exploration from product qualification. |

## Bounds and recovery

Four-hour phase ceiling; 64 GiB new state and at least 40 GiB free. Up to nine
small Gradle correctness invocations (including two retained setup/reader mistakes),
one native formatting invocation, eight owner comparison invocations,
20 standalone helper JVMs and ten compiler commands. Record actual starts,
failures and any prospective amendments. No 80-request pilot, extra CPU profile,
historical-spike reproduction, held-out ordinal 21–100, other owner or publication.

Raw state: `.tools/state/buildopt-product-viability-v1/bv006-checkstyle-finalization`.
Locator: `engineeringPrefix.checkstyleFinalization` in the existing
`task-state.json`. Candidate postimages and exact current-to-revised patch live
under that phase; existing frozen experiments and dirty checkout work are preserved.
The local BuildOpt target remains `main` at
`b76ded08c952ebb386576fafce4ae2d8fdcc09f1`; subject replays use new shared worktrees.
No native comparison may start before its correctness prerequisites pass.

Comparison design: `inputs/comparison-design.json` under the raw state freezes the
whole-request criterion (both measured pairs save at least 1s and pooled saving at
least 5%) and symmetric host-pressure flags before launch. Exact current and
revised sources use a qualification-only `controlBaseline` in the replay tool;
this arm labeling cannot establish a native-versus-product claim.
