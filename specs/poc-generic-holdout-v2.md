# Corrected Hibernate ORM holdout output contract

The first frozen holdout attempt produced a complete generic 29-to-1 proposal
and completed both excluded warm-ups, then stopped before accepting pair one:
the declared `hibernate-core/build/libs/**` output did not exist.

The frozen Hibernate revision explains the mismatch independently of timing.
`local-build-plugins/src/main/java/org/hibernate/build/aspects/ModuleAspect.java`
line 31 changes every module build directory from Gradle's `build` default to
`target`. The repository-owned output contract was therefore invalid; BuildOpt
correctly refused to measure a workload with zero required outputs.

Version 2 changes only the required-output glob to
`hibernate-core/target/libs/**`. It preserves the public revision, root
`assemble` workflow, exact source mutation, Temurin 25, optimized-native
12-worker options, candidate mechanism, eight-pair order, thresholds and full
fallback. No warm-up, proposal or timing from v1 is reused. The correction is
preregistered before rerunning proposal or target-workflow timing.

This is an owner-contract correction, not repository-specific product logic or
post-result performance tuning. The original attempt remains immutable under
`benchmarks/results/poc-generic-holdout-v1-attempt1`.
