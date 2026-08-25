# Adaptive fragment activation fixture

This repository-scoped fixture proves `AF-010` without encoding a real
repository name or task implementation rule. Two independent producers emit
deterministic files and `packageAll` assembles them into one reproducible ZIP.

The complete `fullBuild` workflow runs both producers. A partial activation
may instead restore an unaffected producer output whose content and output
revision are exact, rebuild only changed producers, and then run `packageAll`.
Global, build-logic, ambiguous or unsafe state always executes `fullBuild`.

The fixture contains no Test Optimization behavior and no timed performance
claim. It exists only to prove independent invalidation, real Gradle execution
and exact-output equivalence.
