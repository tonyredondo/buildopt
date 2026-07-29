# Gradle Optimization Plugin

Init/settings/project plugin and adapters built on public Gradle APIs.

`WS-003` packages the project plugin as ID `dev.buildopt`. A
Configuration-Cache-safe shared build service reads invocation-only launcher
context, connects to its Unix socket, sends the v1 `ProducerHello`, validates
the matching acknowledgement, and closes the channel. Registering the service
with Gradle's build-events listener registry realizes it once even when every
task is up-to-date or Configuration Cache is reused.

The plugin logs and disables its handshake when context is incomplete, the
receiver is missing, or the acknowledgement is invalid. These failures never
fail the baseline Gradle build. The plugin does not modify task inputs,
outputs, dependencies, cache policy, or execution, and it does not emit later
task-event payloads.

`WS-004` adds the authenticated local rendezvous without placing secrets in
Gradle configuration. On every service realization, including Configuration
Cache reuse, the plugin reads fresh invocation context, authenticates an HTTP
readiness probe against the loopback gateway, verifies its connection
generation, and writes the separate event-channel authentication preface before
`ProducerHello`. Incomplete or rejected context remains fail-open for the
baseline build. The gateway exposes no cache data routes yet.

Validate the packaged Java 17 artifact, gateway, and real Wrapper handshake
with:

```bash
./dev/run -- ./dev/check-jvm-release
./dev/check-local-gateway
./dev/check-gradle-plugin-handshake
```

Deep instrumentation and output-semantics changes do not belong in this module
without an explicit contract and gate.
