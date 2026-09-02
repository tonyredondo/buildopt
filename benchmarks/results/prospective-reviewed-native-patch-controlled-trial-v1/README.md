# Prospective Reviewed Native Patch Controlled Trial v1 evidence

The prospective holdout is 10/10 conclusive and contains three action
families. Registration and effect binding admits only Shadow
`generateDocTests` to a native diagnostic. Gradle Versions is rejected because
its task action only writes an already-realized `String`; the expensive
dependency resolution remains necessary to calculate a cache key. Protobuf is
rejected because `ProtobufExtract` is consumer-registered and has no task in
the repository-owned Gradle workflow.

The frozen Shadow `documentTest` diagnostic completes successfully with 15
executed tasks and zero product failures. `:generateDocTests` executes for 80
ms, represents 0.01698% of the 471,064-ms invocation, and is not on the native
hard-dependency critical path. It therefore fails all three economic admission
requirements: 500 ms, 2%, and critical-path membership.

The terminal decision is
`STOP_PROSPECTIVE_REVIEWED_NATIVE_PATCH_NO_ECONOMIC_PROPOSAL`. Correctness,
paired value, owner review, and delivery are not authorized by the failed
diagnostic prerequisite. The campaign contains no public patch, candidate
build, timing sample, upstream mutation, or speedup claim. All source and
diagnostic work is charged conservatively as 549 machine seconds; payback is
unavailable because qualified saving is zero.
