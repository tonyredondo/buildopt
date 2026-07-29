# Gradle Optimization Plugin

Init/settings/project plugin and adapters built on public Gradle APIs.

The first increment is a handshake without optimizations (`WS-003`). Deep instrumentation and output-semantics changes do not belong in this module without an explicit contract and gate.

`ENV-004` adds only a no-op project-plugin class compiled against the public Gradle API. It has no published plugin ID and performs no handshake or optimization. `WS-003` owns the first activation contract.
