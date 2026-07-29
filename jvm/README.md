# JVM components

Multi-project build for the beta's four Java artifacts:

- `gradle-plugin/`
- `jvm-agent/`
- `generated-client/`
- `patcher/`

The golden lane pins the Gradle Wrapper and shared configuration. The artifacts produce Java 17-compatible bytecode and maintain separate lifecycles and versioning.

`ENV-004` materializes the Gradle plugin and JVM agent as separate reproducible
JARs. `F0-022` adds the generated single-attempt control-plane client, and
`SPK-004` adds the strict PatchBundle verifier/applier. All four compile on the
locked JDK 21 with `--release 17`, UTF-8 source encoding, all Java lint warnings
enabled, and warnings treated as errors. `WS-003` activates only the Gradle
plugin's neutral one-frame launcher handshake, and `WS-004` authenticates its
loopback/event rendezvous without configuring cache behavior. The bounded JVM
agent remains unavailable for qualification; the patcher is exact only for its
two signed recipes and never performs remote PR orchestration itself.
