# Supported Checkstyle finalization: bounded comparison

Frozen before the new owner comparison, 2026-09-10. This supersedes V1 admission, not its retained evidence.

V1's `TaskExecutionGraph.afterTask` registration fails Elasticsearch's native warning-as-error policy. The retained `corrected` trial has two actual owner starts: current succeeds, V1 fails. Neither is a measured comparison. The earlier `fixed` controller failed before any native start. Do not suppress warnings or reuse these failed trials as successful performance evidence.

The revised candidate registers a public shared `BuildService` and declares every adapted task with `usesService`. Preparation registers a task-local commit operation. Gradle closes the service after its last task finishes and before the build ends. The finalizer still requires native task success, matching before/after context, and the exact closed XML report digest before atomic history publication. It adds no trailing task action and no execution listener. Concurrent registration is thread-safe and no maximum parallel usage is imposed. Configuration Cache support remains unqualified; the existing owner disables it.

API basis: [Gradle 9.7.1 shared build service lifecycle](https://docs.gradle.org/current/userguide/build_services.html). Validate this lifecycle on the actual pinned owner; documentation alone is insufficient proof.

| Step | Required outcome | Evidence |
| --- | --- | --- |
| Reject V1 integration | Preserve failed owner exit and warning; leave original freeze unchanged | `bv006-checkstyle-finalization/profiles/corrected`, end receipt |
| Supported lifecycle | Strict `--warning-mode=fail` cold/warm, late failure, native violation, report/context tampering, no-source, worker cancellation, recovery, up-to-date all satisfy guards | `analysis/correctness-v2.json`, six `supported-final-*` fixture records |
| Source quality and observation | Native formatter passes; actual worker-agent boundary and service-driven commit/context instrumentation observed | `formatter-v2`, `receipts/agent-proof-v2.json` |
| Fresh balanced comparison | Eight successful owner requests, all complete output comparisons pass, no extra history/CPU profiles | `profiles/supported`; exact current consumer and source bindings |
| Decision | Whole request improvement, phase costs and all host-pressure observations reconciled; no filtering or reruns selected by timing | Existing frozen `inputs/comparison-design.json`; profile/phase/host analyses |
| Historical relevance | Separate current-versus-revised improvement from native product value and sparse activity | `analysis/history-reconciliation.json`; current tracker |

The comparison replays original commits 6→7 twice, with private current/revised state, eight native CPUs, balanced cold and measured order, and full C5 output checking. Both arms already contain the Checkstyle optimization. This isolates the finalization change; it cannot establish an improvement over native builds or product viability.

Keep the original success criterion: each measured pair must save at least one second and pooled whole-request time at least five percent. Retain every pair and the predeclared CPU/IO pressure flags. Pressure can include the experiment itself; it is not attribution to a particular external process. Small sample size remains explicit.

Amended phase allowance: ten actual owner starts including the rejected trial, 24 controller reservations including the pre-native failure, 17 small fixture starts including all setup/formatting iterations, two formatter starts, 20 standalone helpers, ten compiler commands, original four-hour deadline (10:56:31 UTC), 64 GiB new artifacts and 40 GiB minimum free disk. Stop at these limits and retain partial evidence. No 80-request pilot, held-out 21–100 replay, other repository build, publication or unrelated-process changes are authorized by this phase.

Exact local state: `/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/buildopt-product-viability-v1/task-state.json`, `engineeringPrefix.checkstyleFinalization`; phase directory `bv006-checkstyle-finalization`, candidate `candidate-v2`. Repository remains main at `b76ded08c952ebb386576fafce4ae2d8fdcc09f1`; subject shared Git is `.tools/state/eic-native-v1/repos/elasticsearch.git`. Result publication belongs in `benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-finalization`; old V1 patches must remain visibly rejected.
