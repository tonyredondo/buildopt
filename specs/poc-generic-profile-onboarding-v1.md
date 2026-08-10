# Generic structural profile onboarding

## Purpose

`buildopt profile propose` removes the hand-authored JSON step from the
structural Build Impact POC. It turns one original Gradle task selector, one
exact Git change and repository-declared required outputs into a reviewable
measurement proposal. It does not predict savings, write an active profile or
authorize production use.

## Inputs

The command requires:

- a repository and pipeline identity;
- one unqualified Gradle task selector such as `classes` or `assemble`;
- a lowercase 40-character base revision and a newline-delimited changes file
  that exactly equals `git diff --name-only --no-renames BASE HEAD`;
- one or more repository-relative output globs whose byte equality defines
  correctness; and
- optional Gradle discovery options and global-change globs.

The tracked target tree must be clean. The target is always the current `HEAD`.
Common Gradle build-logic paths are global fallbacks by default and can be
replaced explicitly.

## Two-pass discovery

1. BuildOpt runs the configured Gradle model for the original selector without
   executing its tasks. It records typed project dependencies, project source
   boundaries, included builds, task reach, `Test` presence and unknown work.
2. Each exact changed path must have one most-specific project owner. BuildOpt
   proposes the same lifecycle task on those owners, creates a draft manifest,
   and runs discovery again for both the original and candidate workflows.
3. The existing opportunity analyzer validates the resulting manifest, graph
   and generated-state binding. Only a complete smaller graph becomes
   `MEASURE_STRUCTURAL_CANDIDATE`.

There are no repository-name branches or output inferences. Required outputs
remain an owner declaration because only the repository owner can define the
semantic result that must be preserved.

## Outputs and handoff

An accepted proposal writes:

- `buildopt-impact-manifest.json`;
- `buildopt-impact-graph.generated.json`;
- `buildopt-impact.generated.json`;
- `buildopt-fallback-changes.txt`; and
- `buildopt-profile-proposal.json`.

The proposal contains the selected entrypoints, project reduction, required
outputs, fallback, unknown-relationship state and the exact `profile measure`
argument vector. A packaged BuildOpt revision may be supplied to make that
vector immediately executable; otherwise the report marks the immutable
revision as the only remaining value to fill.

`profile measure` and `profile evaluate` remain unchanged. The former owns the
eight-pair optimized-native comparison and exact-output checks; the latter may
materialize a review-required profile only from qualifying evidence.

## Fail-closed boundary

The proposal retains `NATIVE_FULL_GRAPH` for global or unowned changes,
ambiguous project ownership, `Test` execution, custom executable workflows,
external included builds, incomplete relationships, no graph reduction, or
manifest/graph/Git drift.

Rejected proposals write no manifest, graph, generated state, fallback file or
active profile. Every outcome keeps `reviewRequired=true`,
`activationAutomatic=false`, `productionAuthorized=false` and Test
Optimization out of scope.

## Conformance

```bash
./dev/run --toolchain temurin-jdk-21 -- ./dev/check-generic-profile-onboarding
```

The checker creates a clean external two-project Git repository with no
BuildOpt JSON, proposes a one-project `classes` workflow, reproduces all output
documents byte for byte, feeds them to `profile analyze` and no-evidence
`profile evaluate`, and rejects a custom workflow without creating candidate
state. It adds no performance percentage.
