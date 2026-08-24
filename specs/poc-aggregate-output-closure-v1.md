# Aggregate output closure POC

## Question

Can BuildOpt derive every output required by a custom aggregate Gradle workflow
without recognizing a repository, plugin, task name, output directory or file
extension?

This is a correctness prerequisite for accelerating aggregate workflows. A
lifecycle task can have no outputs while depending on arbitrary producer tasks.
Running only the changed producer is safe only when every omitted output can be
materialized exactly or its producer is rebuilt.

## Generic closure

BuildOpt already observes two supported Gradle models during the useful native
build:

1. the configured task graph, including exact task dependency relationships;
2. every repository-owned output root and the task and project that produced
   it.

When conventional lifecycle partitioning finds no direct output, BuildOpt now
uses those models to compute a fail-closed closure:

- resolve the exact requested task or task selector;
- traverse every configured dependency reachable from it;
- require every observed output to have one project owner and at least one
  reachable producer;
- rebuild the exact producer tasks owned by the changed project;
- materialize all remaining required outputs from revision-bound verified
  state; and
- retain native Gradle if a dependency, owner, producer or bounded set is
  incomplete.

The producer tasks, required outputs, candidate outputs and closure diagnostic
are carried into the generated manifest and state binding. Existing
conventional workflows retain their established path; this closure is invoked
only when that path cannot identify a candidate.

## Executable proof

The Groovy and Kotlin fixtures request `:changed:bundleAll`. That task has no
actions or outputs and depends on `:changed:emitPayload` and
`:stable:emitPayload`. Both producers write extension-neutral files under
`build/custom-output`, deliberately outside conventional archive, classes,
distribution and publication paths.

The matrix runs Gradle 8.14.3 and 9.6.1 for both DSLs. Each case must:

- close three reachable tasks and two required outputs;
- use only `:changed:emitPayload` as the candidate entrypoint;
- capture and materialize the stable output;
- rebuild no stable producer in the clean candidate workspace;
- reproduce the same two-file aggregate SHA-256 as the full workflow; and
- report zero product failures.

Unit coverage independently rejects a missing dependency, ambiguous output
ownership and a producer outside the requested graph.

## POC boundary

The proof is revision-bound and makes no cross-revision ABI claim. It validates
correctness, not wall-time value, so no performance replay is required in this
block. It adds no production authority, soak, design-partner requirement or
Test Optimization behavior. Structural profile rebinding is the next block
that may generalize this exact closure across compatible commits.
