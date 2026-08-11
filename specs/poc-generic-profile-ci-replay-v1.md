# Five-repository review-only CI replay

## Purpose

`POC-GENERIC-PROFILE-CI-REPLAY-001` exercises the committed review-only Action
on clean hosted runners for Spring Framework, OpenTelemetry Java
Instrumentation, Apache Kafka, Micronaut Core, and Apache Groovy. Each job
starts from its frozen public revision, commits the repository-owned profile
input, applies the previously frozen one-file change, and supplies the exact
base and target revisions to the Action.

The replay asks whether the generic proposal remains reproducible in the
environment where a repository owner would consume it. It does not run paired
timing, reuse historical percentages as a new observation, activate a profile,
or change Gradle tasks.

## Two distinct decisions

Every historical row first produced a structural proposal. Wall-time evidence
then decided whether that proposal had value. The replay must keep those
decisions separate:

- all five repositories are expected to reproduce their exact structural
  candidate and complete graph; and
- Spring still retains native Gradle because its historical value row passed
  only seven of eight pairs, while OpenTelemetry, Kafka, Micronaut, and Groovy
  retain their independently qualified historical value decisions.

A matching proposal is therefore not permission to activate it. It is only a
reviewable candidate for the already-defined measurement handoff.

## Repository-owned input

The workflow materializes `.buildopt/profile-ci.json` from the frozen v3 owner
inputs and commits it before creating the source-change commit. The Action sees
only that configuration and the Git diff. Repository URLs, fixed mutations,
JDK selection, and reference evidence belong to the replay harness; they are
not available to product discovery and cannot influence its decision.

The input contains the original Gradle entrypoints, required output globs,
pipeline class, Gradle Wrapper command, bounded options, and timeout. No graph,
candidate task, retained percentage, or repository-specific optimization rule
is provided.

## Drift contract

Each hosted job verifies the complete Action artifact, SHA-256 manifest,
proposal decision, candidate entrypoints, project counts, owner input, changed
path, manifest, generated binding, and declared graph against the corresponding
terminal v3 or v4 reference. The job writes a small replay verdict before it
asserts success.

Any difference is `DRIFT`, not an automatic repair. The workflow retains
optimized native Gradle and requires a new review; it may not relax thresholds,
reuse an old percentage, or add repository-name logic.

## Acceptance

The block closes only when all five clean jobs produce `MATCH`, no active or
qualified profile exists, every proposal remains review-required, and the
workflow-level conclusion is successful. This remains POC evidence only. It
does not authorize production, soak testing, design-partner work, or Test
Optimization.

## Result

Hosted run
[`31467370391`](https://github.com/tonyredondo/buildopt/actions/runs/31467370391)
passed all five independent repository jobs and the aggregate summary from
immutable BuildOpt revision `18caa8f`. Every proposal and reference graph
matched, drift was zero, and no active profile was written. The durable result
is stored under
[`benchmarks/results/poc-generic-profile-ci-replay-v1`](../benchmarks/results/poc-generic-profile-ci-replay-v1/README.md).
