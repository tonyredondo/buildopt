# JVM Agent spike v1

This bounded `SPIKE-AGENT-001` result answers `SPK-002` for Gradle 9.6.1,
JDK 21, Linux x86-64, and the Kotlin DSL golden lane.

## Result

The prototype is `UNAVAILABLE` for access tracing. A Java
`ClassFileTransformer` can observe that an allowlisted API class loaded, but
that does not prove which task called a method, which operation occurred, or
which path/value was accessed. Treating class loading as I/O, environment,
process, network, clock, or randomness evidence would violate `TASK-001`.

The agent therefore emits:

- `traceComplete=false`;
- `qualification=UNAVAILABLE`;
- `pendingPublication=ABORTED`;
- `taskQualificationState=OBSERVING`;
- fallback `ABORT_PENDING`.

No task can qualify from this report. The implementation remains useful only
as an executable boundary showing where bytecode rewriting or an official
adapter would be required later.

## Executed coverage vector

The real Gradle fixture performs IO and NIO reads/writes, environment and
system-property queries, a native child process, a loopback network attempt,
clock/locale/timezone access, and deterministic plus secure/thread-local
randomness calls. The agent labels each dimension only `LOAD_ONLY` or
`UNOBSERVED`; neither is exact access coverage.

The real Wrapper starts a dedicated instrumented daemon, reuses Configuration
Cache on the second invocation, and stops the daemon before reading the atomic
report. The check also measures one warm baseline invocation and one warm
instrumented invocation. That single order-sensitive sample is descriptive,
not a promotion or overhead budget.

## Failure semantics

- A capacity-two run must drop events, emit `BUFFER_OVERFLOW`, and abort
  pending publication while the Gradle task succeeds.
- An injected transform conflict must emit `TRANSFORMER_CONFLICT` while the JVM
  ignores the failed transform and the baseline task succeeds.
- An injected `premain` crash runs only in an isolated diagnostic daemon,
  records that the compatibility class must be disabled, and fails that
  invocation. A fresh uninstrumented daemon must then reproduce the baseline
  output. The product must never retry a post-task customer build merely
  because an agent failed.

Run:

```bash
./dev/check-jvm-agent-spike
```

The result does not activate the agent, prove hermeticity, or close any C1
qualification gate. Future exact instrumentation needs a separate dependency,
allowlist, method-level mutation suite, overhead/soak evidence, and the same
fault matrix.
