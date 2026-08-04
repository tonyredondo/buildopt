# Fixtures

[`poc-breadth`](./poc-breadth/README.md) is the realistic five-project
Kotlin/Groovy POC generalization fixture. It tests no-change, leaf-source,
shared-source, and global build-logic changes against optimized native Gradle,
with exact selection/fallback counts and byte-identical required outputs.

[`poc-value`](./poc-value/README.md) contains the bounded Kotlin/Groovy
workloads used by the strict accelerator-coverage matrix. They compare against
optimized native Gradle and are synthetic POC evidence, not production or
universal-performance claims.

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

`data-lifecycle/` contains the `F0-037` raw-to-redacted JSON golden profiles,
at-least-once JSONL delivery cases, and changed-duplicate negative. All
sensitive values are synthetic; managed golden outputs retain only keyed
tokens.

`tier1/` materializes the `F0-040` Kotlin and Groovy consumer repositories.
Its TestKit and real-Wrapper matrix exercises the packaged product plugin, a
Java 17 cacheable custom task, an artifact transform, build-cache replay, and
Configuration Cache reuse on all Gradle 8.14.3/9.6.1 JDK 17/21 rows. JDK 25
remains an explicit, unproven target.

`jvm-agent/` is the real-daemon `SPK-002` access fixture. Its untracked custom
task executes all six required access dimensions while the bounded agent,
overflow, conflict, crash, Configuration Cache, and baseline-recovery
scenarios run only through a real Wrapper.

`hermetic-helper/` documents the synthetic task-specific producer used by
`SPK-003`. The real Rust checker probes the host, rejects candidate execution
when coverage is incomplete, and verifies the full producer only through the
uninstrumented Gradle-baseline fallback.

`patcher/` documents the `SPK-004` real-Git fixture generator. Each acceptance
case receives a private repository, signed bundle, blob directory, staging
root, and in-memory draft-PR adapter; all of them are deleted after the case.

`no-hit-overhead/` is the deterministic `A0-G06` Java 17 workload. It exposes
long and short session tasks around one Tier 1 `JavaCompile` cache lookup,
declares no external repositories or dependencies, and produces the required
reproducible JAR used by the paired overhead gate.

`test-cache-isolation/` is the `A0-G08` root/composite boundary. Its root
build, actual `buildSrc`, and included plugin each expose a cacheable `Test`
task so the control can replay from an authenticated remote cache and the
managed no-grant path can prove zero `GET`/`PUT` requests.

`build-impact/` contains the C3-001 repository-committed manifest fixture. It
binds the repository and pull-request pipeline, enumerates one complete
customer-owned JVM entrypoint alternative and required outputs, retains Test
Optimization ownership of its check, and forces `FULL_GRAPH` for unknown
changes without authorizing an omission.
