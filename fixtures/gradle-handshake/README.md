# Gradle plugin handshake fixture

`WS-003` and `WS-004` use this independent Gradle build to prove that the
packaged `dev.buildopt` plugin authenticates the loopback gateway and event
socket, then sends one `ProducerHello` to `buildopt run` without changing the
baseline task result.

The fixture is executed with the repository's Gradle 9.6.1 Wrapper and locked
JDK 21. `neutralProbe` writes one deterministic output; `intentionalFailure`
verifies that an accepted handshake does not replace Gradle's exit status. The
init script loads the exact locally built plugin JAR supplied by the checker.
The second invocation reuses Configuration Cache while receiving a fresh local
credential that is not stored in the entry.
