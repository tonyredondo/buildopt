# Critical-Path-First Reviewed Native Patch v1

`CRITICAL_PATH_FIRST_REVIEWED_NATIVE_PATCH_V1` tests whether BuildOpt can find
reviewable native Gradle corrections prospectively when selection starts from
measured owner-workflow economics rather than source annotations.

## Ordered gates

1. Freeze ten previously unused public Gradle repository revisions and one
   repository-owned workflow per family before any diagnostic result exists.
2. Run at most one optimized-native diagnostic per family. The diagnostic may
   collect an operation trace and task graph, but it is not a timing sample and
   makes no speedup claim.
3. Inspect source only for bounded repository-owned tasks that execute on the
   hard-dependency critical path and contribute at least 500 ms and 2% of the
   complete invocation. Repository and task names are labels only.
4. Require 10/10 conclusive diagnostic rows and at least 3/10 families with a
   source-bounded economic proposal before any candidate build. Otherwise stop
   the dependent correctness, value, review, and delivery blocks.
5. Admit at most two proposals. A proposal may correct cacheability,
   incremental inputs, output declarations, or demonstrably avoidable task
   work, but it must be a small reversible native Gradle change with bounded
   effects and no inferred semantics.
6. For every admitted proposal, prove exact outputs, same-root and cross-root
   behavior, required invalidation, exact revert, and zero product failures.
7. Only correctness-qualified proposals receive one excluded stabilization per
   root and eight balanced optimized-native/candidate pairs. Qualification
   requires 8/8 positive pairs, at least 500 ms and 2% mean saving, a positive
   paired 95% interval, non-regressive p95, owner acceptance, and combined
   payback within 300 compatible builds.

## Resource envelope

- Ten optimized-native diagnostics maximum, one per family.
- Twenty minutes maximum per diagnostic and two charged machine hours for the
  diagnostic campaign.
- Eight GiB maximum additional disk, at least eight GiB free before a public
  build, and a hard stop before another start below six GiB.
- Twelve Gradle workers maximum.

## Boundaries

No source inspection is permitted before the diagnostic gate identifies an
economic repository-owned task. A source-safe but sub-threshold task supplies
no proposal. The route does not authorize upstream pull requests, public
default-branch mutation, automatic apply or merge, production, soak, design
partners, moved thresholds, or Test Optimization behavior.
