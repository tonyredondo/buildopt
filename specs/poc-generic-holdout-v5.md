# Hibernate ORM reciprocal crossover evaluation

The v4 attribution run preserved a favorable 8.480-second/3.82% mean signal
but did not qualify: only five of eight raw pairs improved and the paired
interval crossed zero. The candidate remained fixed at 32 tasks while the
native control moved among 300, 301 and 302 tasks. Every control-first pair was
positive, compared with one of four candidate-first pairs. Combining adjacent
opposite-order pairs after the result produced four positive crossover blocks,
but that post-hoc observation is diagnostic and cannot qualify the candidate.

Version 5 preregisters the generic correction before collecting new timing:

- two independent batches each execute eight fresh alternating pairs;
- adjacent `CONTROL_FIRST` and `CANDIDATE_FIRST` pairs form one reciprocal
  block, producing eight new blocks in total;
- each private arm performs two excluded target-workload observations from the
  same frozen cache seed before its measured pairs;
- every warm-up and measured invocation records the exact sorted Gradle task
  paths and outcomes as well as their fingerprint, log digest and bounded
  counters;
- the aggregate evidence reports added, removed and outcome-changing task paths
  for every observed target-shape variant;
- target-shape drift, any non-positive block, a non-positive interval, output
  drift, either failed fallback or any product-attributable failure retains
  native Gradle.

The public Hibernate revision, exact comment mutation, root `assemble`
workflow, required core JARs, Temurin 25, 12-worker optimized-native control,
Build-Impact-only candidate, byte-for-byte output comparison, full-graph
fallback, 500-ms/2% minimum effect and positive-bound requirement are
unchanged. The eight-of-eight repeatability requirement now applies to the
eight reciprocal blocks rather than order-sensitive single pairs.

No v2-v4 proposal, warm-up or duration is reused. The new task-path evidence
is diagnostic and must not become a Hibernate-specific product rule. This is a
POC evaluation, not automatic activation, production hardening, Test
Optimization, soak testing or design-partner work.
