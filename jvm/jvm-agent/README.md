# JVM Instrumentation Agent

Opt-in Java agent for deep tracing inside the Gradle daemon.

It is an observability backend with published coverage and overhead. It is not a sandbox and does not prove hermeticity. Its first work is bounded by `SPK-002`.

`ENV-004` adds only the packaged `Premain-Class` and verifies that the JAR loads without installing transformers. Redefinition and retransformation remain disabled until the bounded spike defines and validates that behavior.
