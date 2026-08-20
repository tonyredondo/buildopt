# Aggregate Workflow Partition POC v1

## Question

Can `buildopt optimize` reduce a broad Gradle lifecycle workflow without
raising its 64-entrypoint safety bound, adding repository-specific rules or
losing outputs from a clean workspace?

## Contract

The partition is derived only after one successful full workflow has provided
the configured Gradle graph, exact output ownership and revision-bound output
materialization state. For `assemble`, `classes`, `testClasses` and other
supported task selectors, BuildOpt separates the workflow into:

1. task groups for projects that directly own changed inputs;
2. candidate output patterns produced by those groups; and
3. required outputs from all other projects, restored from the exact full-graph
   observation of the same repository revision.

Each task group records its selector, generic variant, Gradle lifecycle
contract, entrypoints and output patterns. The flat candidate entrypoint set
remains bounded to 64. More than 64 directly changed owners, ambiguous output
ownership, missing producer relationships or an incomplete group retain native
Gradle.

## ABI boundary

The POC makes no cross-revision ABI inference. A downstream output may be
omitted only because its exact bytes were already captured by the full workflow
for the same repository revision, workflow and output contract. Revision,
Wrapper, graph, command, output or materialization drift rejects reuse before
the candidate starts. This is deliberately narrower than assuming that a
source change is ABI compatible.

## Executable proof

The runner creates a generic 66-project Groovy DSL build: one changed `core`
project and 65 consumers. The project dependency graph marks all 66 projects as
transitively affected, so the previous project-lifecycle candidate would need
66 entrypoints and exceed the unchanged cap. The partitioned candidate instead
runs only `:core:assemble` and materializes the 65 consumer JARs into a clean
workspace.

Acceptance requires:

- 66 complete baseline JARs and the same 66 candidate JARs;
- one identical aggregate SHA-256 for both output sets;
- 66 legacy entrypoints reduced to one bounded candidate entrypoint;
- 65 exact files materialized and no consumer producer executed;
- unit coverage for `assemble`, `classes`, `testClasses`, incomplete groups and
  more than 64 directly changed owners; and
- no task-cap increase, repository-name rule, production authority, timing
  claim or Test Optimization behavior.

This block proves correctness and generic breadth. The unchanged five-public-
repository transfer is the next block that determines whether this structural
coverage produces useful wall-time and payback.
