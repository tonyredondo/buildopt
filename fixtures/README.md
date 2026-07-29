# Fixtures

Reproducible Gradle repositories and scenarios for the golden lane, TestKit, cache conformance, failures, cancellation, and compatibility.

Fixtures must declare their wrapper, JDK, plugins, seed, and expected result; they do not depend on accidental workstation state.

`golden-lane/` is the first fixture: a minimal Java project using Kotlin DSL that runs Gradle with JDK 21, compiles with `--release 17`, generates a reproducible JAR, and exposes an executable marker.

`gradle-correlation/` is the first `F0-040` slice and the completed `SPK-001` harness. Its independent multi-project build covers parallel equivalent tasks, Worker API isolation modes, a real child JVM, remote-cache miss/hit behavior, failure, cancellation, and Configuration Cache reuse on Gradle 9.6.1 and 8.14.3. The spike proves the `UNATTRIBUTED` whole-attempt fallback because cold Kotlin DSL work also emits non-task remote stores.
