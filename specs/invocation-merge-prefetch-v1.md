# Invocation merging and policy prefetch v1

This POC contract closes `B-005` and, together with `B-004`, closes `B-G04`.
Invocation merging belongs to the CI integration because a Gradle plugin cannot
merge processes that have not started. The engine accepts exactly two immutable
Wrapper invocations and removes the first only after a versioned model proves
that the second transitively contains its work.

Repository, revision, working directory, Wrapper/JDK, JVM arguments,
`GRADLE_USER_HOME`, Gradle/system properties, environment, credentials, init
scripts, and cache policy must match. Intermediate consumers, external effects,
CI barriers, releases, changed failure/retry/continue/exclusion/finalizer/order
semantics, or a diverging isolated control preserve both original invocations.

Policy prefetch is a non-authoritative in-memory latency cache. Concurrent
requests for the same repository/revision/pipeline/compatibility key share one
fetch. The complete payload digest is checked and a caller-supplied authority
verifier must succeed before storage. Exact binding, expiry, and minimum
revocation epoch are rechecked before use; invalid entries are discarded.
Callers wait with their own deadline, and losing prefetched state always falls
back to the normal authenticated policy path.

Run `./dev/check-invocation-merge-prefetch` for the executable contract.
