# Structural transfer evaluation

This revision-4 protocol asks whether repository-independent structural analysis transfers
to a fourth substantial public Gradle family. It freezes Micronaut Core revision
`8de8f38aceb6239f7df05c92c4eb7a26113e882b` before accepting any timing.

The optimized native control runs the binary `assemble` graph. The installed
BuildOpt candidate runs only `:micronaut-http-client-jdk:assemble` after the
same fixed production-source mutation. Both arms restore the same cache seed,
use the same checkout, Gradle home, daemon and 12-worker budget, and compare
every required JAR byte for byte. Both run complete `assemble` semantics,
including documentation and source archives. Revision 1 attempted common `-x`
exclusions, but the installed
CLI correctly rejected graph-changing Gradle options before candidate warm-up;
no timing pair was produced or reused.

Revision 3 changes only the generated graph binding after the generic
`POC-SOURCE-OWNERSHIP-001` correction. The conservative cyclic source boundary
is unchanged, while direct ownership now resolves solely to
`:micronaut-http-client-jdk`, which is inside the 22-project candidate. The
diagnostic regeneration accepted no warm-up or timing observation. Repository,
mutation, tasks, outputs, cache state, pair order and value thresholds remain
unchanged.

Revision 4 clarifies the already intended optimized-native cache symmetry after
a zero-observation replay stopped on the first measured control arm. Both
warm-ups must compile the fixed mutation before the shared measured seed is
captured. Measured control and candidate arms may then restore that same seed
and obtain a native `FROM-CACHE` result for the mutated compile task. Requiring
every measured arm to miss would disable Gradle's optimized native cache after
deliberately seeding it and would bias the comparison. The task must still be
observed in every arm, and both arms retain identical cache, reset, output and
ordering rules. No completed pair from the stopped replay is reusable.

Discovery must remain repository-name independent and complete: 75 projects in
the control reach, 22 in the candidate reach, no Test tasks and no unknown
relationship. A `gradle.properties` change must still restore the full graph.

One unmeasured warm-up per arm precedes eight alternating pairs. Qualification
requires at least 500 ms and 2% mean savings, a positive deterministic-bootstrap
95% lower bound, all eight pairs positive, identical non-empty outputs and zero
product-attributable failures. Thresholds do not move and failed pairs are not
discarded. A failure retains native Gradle as the Micronaut default.

Micronaut documents GraalVM 25.1.3 for contributors. The POC deliberately uses
the project-locked Temurin 25.0.3+9 because the preflight is compatible; this is
a validation boundary, not an upstream support claim.

This is POC evidence only. It does not authorize production activation, a
universal savings claim, Test Optimization, soak testing, a design partner or a
public release.
