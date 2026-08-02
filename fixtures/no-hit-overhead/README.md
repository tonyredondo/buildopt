# No-hit overhead fixture

This deterministic Java 17 fixture provides one Tier 1 `JavaCompile` cache
lookup plus long and short session tasks. The long task adds only a declared
fixed delay after producing the reproducible JAR. The short task is used when
pre-outcome policy omits L2 entirely.
