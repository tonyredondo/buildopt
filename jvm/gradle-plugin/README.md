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
task-event payloads. Authenticated rendezvous and a retained multi-producer
channel remain owned by `WS-004`.

Validate the packaged Java 17 artifact and the real Wrapper handshake with:

```bash
./dev/run -- ./dev/check-jvm-release
./dev/check-gradle-plugin-handshake
```

Deep instrumentation and output-semantics changes do not belong in this module
without an explicit contract and gate.
