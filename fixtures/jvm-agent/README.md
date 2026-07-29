# JVM Agent spike fixture

This real-Wrapper repository executes every `SPIKE-AGENT-001` access class:
Java IO/NIO, environment and system properties, a native child process, a
loopback network attempt, clock/locale/timezone, and randomness.

The task writes only fixed `EXECUTED` markers. It intentionally registers none
of those accesses as Gradle inputs because the spike asks what the agent can
observe. The prototype sees only allowlisted class loads, not calls; therefore
the fixture proves the `UNAVAILABLE` outcome rather than manufacturing access
coverage.
