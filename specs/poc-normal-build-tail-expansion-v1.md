# Normal-build task-tail expansion POC

## Question

After qualifying the exact standard `Jar` adapter and rejecting the exact
standard `Copy` adapter for unstable incremental value, does the retained real
Spring/OpenTelemetry task evidence justify implementing another standard-task
adapter?

This is an evidence-synthesis gate, not a new performance experiment. It uses
only the fixed task ranking that selected `Copy`, the complete retained Copy
cascade result, and the Spring generalization result. No previous percentages
are added and no new timing claim is made.

## Selection gate

A new candidate must satisfy every condition:

- at least 500 ms of observed task duration in the retained real trace;
- an exact, unmodified standard Gradle task implementation;
- bounded inputs, outputs, and side effects;
- no existing native-cache replay for the same work;
- no Test-owned execution behavior; and
- stable incremental evidence if the candidate has already been implemented.

Repository-specific task paths may identify evidence but cannot enter product
matching. A candidate below the value floor, already handled by Gradle, custom
or effect-ambiguous, or incrementally unstable remains native.

## Retained review

| Tail | Observed evidence | Decision |
|---|---:|---|
| Standard `Jar` | 3,624 ms | Already qualified and active for the exact shape. |
| Standard `Copy` | 2,807-ms source trace; 24.90% favorable incremental mean | Rejected: only 3/4 incremental pairs are positive and the interval crosses zero. |
| Custom `ShadowJar` | 497 ms | Rejected: below the floor, custom plugin task, and already restored from native cache. |
| Configured `JavaExec` (`generateJflex`) | 427 ms | Rejected: below the floor and process effects are not bounded by a generic standard-task adapter. |
| Spring trace | No new standard task selected | No candidate. |

The directly measured Impact + Jar + Copy profile remains valid whole-build
cascade evidence at 52.89%, but it cannot overrule the failed direct
incremental authorization for Copy.

## Decision

`STOP_NORMAL_BUILD_TASK_ADAPTER_EXPANSION_NO_ACTIONABLE_TAIL`.

The POC has exhausted the current retained real traces for another exact
normal-build task adapter. This is not a universal statement that no other
optimization exists. It means the next adapter must wait for a new dominant
tail from a materially different workload rather than being selected from a
generic catalog. The next bounded experiment is build-owned test compilation,
preparation, and packaging; Test-owned selection and execution remain outside
scope.
