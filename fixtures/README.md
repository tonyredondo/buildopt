# Fixtures

Reproducible Gradle repositories and scenarios for the golden lane, TestKit, cache conformance, failures, cancellation, and compatibility.

Fixtures must declare their wrapper, JDK, plugins, seed, and expected result; they do not depend on accidental workstation state.

`golden-lane/` is the first fixture: a minimal Java project using Kotlin DSL that runs Gradle with JDK 21, compiles with `--release 17`, generates a reproducible JAR, and exposes an executable marker.

`gradle-correlation/` is the first `F0-040` slice. Its independent multi-project build proves parallel execution of two equivalent cacheable tasks, a shared native cache key, local build-cache miss/hit behavior, and Configuration Cache reuse before `SPK-001` adds correlation instrumentation.
