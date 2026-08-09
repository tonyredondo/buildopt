# Structural transfer evaluation

This revision-2 protocol asks whether repository-independent structural analysis transfers
to a fourth substantial public Gradle family. It freezes Micronaut Core revision
`8de8f38aceb6239f7df05c92c4eb7a26113e882b` before accepting any timing.

The optimized native control runs the binary `assemble` graph. The installed
BuildOpt candidate runs only `:micronaut-http-client-jdk:assemble` after the
same fixed production-source mutation. Both arms exclude documentation and
source archives, restore the same cache seed, use the same checkout, Gradle
home, daemon and 12-worker budget, and compare every required JAR byte for byte.
Both arms run the complete `assemble` semantics, including documentation and
source archives. Revision 1 attempted common `-x` exclusions, but the installed
CLI correctly rejected graph-changing Gradle options before candidate warm-up;
no timing pair was produced or reused.

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
