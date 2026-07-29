# Fixtures

Reproducible Gradle repositories and scenarios for the golden lane, TestKit, cache conformance, failures, cancellation, and compatibility.

Fixtures must declare their wrapper, JDK, plugins, seed, and expected result; they do not depend on accidental workstation state.

`golden-lane/` is the first fixture: a minimal Java project using Kotlin DSL that runs Gradle with JDK 21, compiles with `--release 17`, generates a reproducible JAR, and exposes an executable marker.

`gradle-correlation/` is the first `F0-040` slice and the completed `SPK-001` harness. Its independent multi-project build covers parallel equivalent tasks, Worker API isolation modes, a real child JVM, remote-cache miss/hit behavior, failure, cancellation, and Configuration Cache reuse on Gradle 9.6.1 and 8.14.3. The spike proves the `UNATTRIBUTED` whole-attempt fallback because cold Kotlin DSL work also emits non-task remote stores.

`gradle-handshake/` is the independent `WS-003`/`WS-004` fixture. The real
Gradle 9.6.1 Wrapper loads the exact packaged plugin JAR, authenticates both
local rendezvous hops, compares a deterministic task output with the direct
baseline, reuses Configuration Cache with the task up-to-date, exercises a
missing rendezvous, and preserves an intentional Gradle failure.

`github-actions/` is the `WS-007` consumer fixture. It binds the root composite
Action and a synthetic Release Bundle v1 archive to immutable commit/checksum
identities, exercises installation locally without network access, and
provides a manual read-only hosted workflow that proves PATH/output publication
plus argv and exit-code preservation. It does not implement authoritative CI
or the protected validation queue.

`ci-orchestration/` is the inert `F0-030` protected validation-workflow
fixture. Its schedule, manual recovery, repository concurrency, trusted
default-branch boundary, read-only permissions, and single-lease command are
checked together with the executable queue/budget/recovery scenarios.

`test-optimization/` contains the shared `F0-033` producer/consumer artifact
bytes. The integration corpus binds their exact size and digest and proves
that corrupt or caller-selected paths cannot satisfy validation.
