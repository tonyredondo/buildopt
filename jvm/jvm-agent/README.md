# JVM Instrumentation Agent

Opt-in Java agent for deep tracing inside the Gradle daemon.

It is an observability backend with published coverage and overhead. It is not a sandbox and does not prove hermeticity.

`SPK-002` closes as `UNAVAILABLE`: the dependency-free prototype installs a
bounded allowlisted `ClassFileTransformer`, but class loading is not method
access or task attribution. Every emitted report therefore has
`traceComplete=false`, aborts pending publication, and leaves qualification in
`OBSERVING`.

The real-daemon fixture covers Configuration Cache reuse, buffer overflow,
transformer conflict, injected `premain` crash, and clean uninstrumented
recovery. Redefinition and retransformation remain disabled. See
[`specs/jvm-agent-spike-v1.md`](../../specs/jvm-agent-spike-v1.md).
