# Economics-Gated Reviewed Native Patch v1

`ECONOMICS_GATED_REVIEWED_NATIVE_PATCH_V1` tests whether BuildOpt can find a
third valuable reviewed-native correction without spending correctness and
paired-value campaigns on source-safe but economically immaterial tasks.

The experiment reverses the RNCE selection order. Source safety remains a
mandatory first gate, but it does not authorize patch compilation. A source
candidate must first bind to one real owner workflow and one optimized-native
diagnostic showing that the task executes on the hard-dependency critical path
and contains at least 500 ms and 2% of addressable wall time. The diagnostic is
discovery evidence, never a timing sample or speedup claim.

## Ordered gates

1. Audit ten exact public revisions outside the NAC, RNPP and RNCE source
   cohorts. Every row is reconstructed from frozen Git source with the
   unchanged normalization-aware v2 classifier. Earlier result rows may not
   supply a candidate, decision, duration or output.
2. Bind source-safe candidates to real task registrations and owner workflows.
   At most six optimized-native diagnostics across at most three families may
   capture operation traces, hard-dependency task graphs and exact outputs.
3. Admit a proposal only when the selected task executes on the critical path,
   its observed addressable duration is at least 500 ms and 2% of the complete
   invocation, its outputs and effects are bounded, and conservative machine
   work can repay within 300 compatible builds. At most two proposals from two
   families may continue.
4. Each admitted proposal receives fresh patch compilation and at most six
   correctness starts. Complete same-root and cross-root outputs, cache
   restoration, required invalidation, exact revert and zero product failures
   are mandatory.
5. Each correctness-qualified proposal receives one excluded stabilization
   and eight balanced optimized-native/candidate pairs. Qualification requires
   8/8 positive pairs, at least 500 ms and 2% mean saving, a positive paired
   95% interval, non-regressive p95 and finite payback.
6. Only a value-qualified proposal may enter a controlled first-exposure draft.
   The owner reports active review seconds, understands the proposal, accepts
   it with at most one clarification, and the combined machine plus review-time
   equivalent must repay within 300 compatible builds.

## Budgets and stops

The source cohort is exactly ten families. Source acquisition and diagnostics
may consume at most 8 GiB additional disk. No public Gradle build starts below
8 GiB free, and the route stops before another start below 6 GiB. A failed gate
stops its dependent blocks without relabelling diagnostics as timing or moving
thresholds after observation.

The route does not authorize upstream pull requests, merges, default-branch
mutation, automatic application, production, soak, design-partner work or Test
Optimization behavior.
