# Installed Spring Build Impact breadth experiment

This preregistered POC asks whether the installed Build Impact result survives
two materially different scopes inside the same pinned Spring Framework
revision. It does not tune the mechanism after seeing timings and does not
widen production selection authority.

The selective cells are:

- a leaf production change in `spring-webmvc`, requiring that module's complete
  main and test class outputs; and
- a shared production change in `spring-core`, requiring the downstream
  `spring-jms` class outputs.

Each cell compares four alternating pairs of optimized native Gradle
`testClasses` against the installed `buildopt impact` candidate. Both arms use
all 12 CPUs, the same offline dependency state and native-cache seed, clean
outputs, the same fixed mutation and the same build-logic test guard. Every
declared output must be non-empty and byte-identical. The unchanged gate is at
least 500 ms and 2% mean saving, a positive paired-bootstrap lower bound, four
positive pairs and zero product failures per cell.

A third cell changes `gradle.properties`. It is not a performance comparison:
the installed command must classify it as `IMPACT_GLOBAL_CHANGE` and execute
the original full `testClasses` graph. This proves that breadth does not come
from bypassing the conservative fallback.

If both selective cells qualify and the fallback passes, the terminal decision
is `BROADEN_INSTALLED_SPRING_IMPACT`. Otherwise it is
`RETAIN_SINGLE_INSTALLED_SPRING_SCOPE`; failed or unfavorable observations are
retained and thresholds cannot move.

This experiment covers build-owned test preparation only. It executes no
root-build Gradle `Test`, changes no test selection, adds no Spring-specific
product heuristic, and makes no production, release, soak or design-partner
claim.
