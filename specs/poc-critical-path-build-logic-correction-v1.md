# Critical-Path Build-Logic Correction v1

`CRITICAL_PATH_BUILD_LOGIC_CORRECTION_V1` tests a different reviewed-native
patch seam after `CPFRNP`: owner build logic that explicitly disables Gradle
state tracking, caching, or incremental execution for an economically material
standard task.

## Ordered gates

1. Freeze five public families with complete, independently checked native
   critical-path analyses. Retained analyses are discovery inputs, never new
   timing samples.
2. Reconstruct tasks on the hard-dependency critical path that contribute at
   least 500 ms and 2% of their build span.
3. Scan the exact source revision for explicit `cacheIf { false }`,
   `upToDateWhen { false }`, `doNotCacheIf`, `doNotTrackState`, or incremental
   mode set to false. Repository and task labels cannot affect classification.
4. Bind an opt-out only through a literal task name or task type. Dynamic or
   unresolved bindings are incomplete; an opt-out on a non-material task is a
   conclusive no-action.
5. Require 5/5 conclusive families and at least 3/5 proposal families before a
   public source patch or candidate build.
6. Any admitted correction must then prove owner semantics, exact complete
   outputs, same-root and cross-root behavior, invalidation, exact revert, and
   zero product failures before paired timing.
7. Value requires eight balanced positive pairs, at least 500 ms and 2% mean
   saving, a positive paired 95% interval, non-regressive p95, owner acceptance,
   and combined payback within 300 compatible builds.

## Resource envelope

- Source classification performs zero Gradle starts.
- At most five source trees and five retained native analyses are consumed.
- Four GiB maximum additional source materialization disk.
- Later correctness and value blocks remain closed unless the source gate
  passes.

## Boundaries

Absence of an explicit state opt-out does not prove that all build logic is
optimal. It proves only that this bounded, mechanically reviewable correction
class is absent from the material tasks. Historical durations identify the
tasks to inspect but create no fresh speedup claim. No upstream pull request,
default-branch mutation, automatic apply or merge, production, soak, design
partner, threshold change, or Test Optimization behavior is authorized.
