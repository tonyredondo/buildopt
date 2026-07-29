# JVM components

Multi-project build reserved for the beta's three Java artifacts:

- `gradle-plugin/`
- `jvm-agent/`
- `patcher/`

The golden lane pins the Gradle Wrapper and shared configuration. The artifacts produce Java 17-compatible bytecode and maintain separate lifecycles and versioning.

`ENV-004` materializes the Gradle plugin and JVM agent as separate reproducible JARs. Both compile on the locked JDK 21 with `--release 17`, UTF-8 source encoding, all Java lint warnings enabled, and warnings treated as errors. The compatibility entrypoints are deliberately neutral: behavior remains behind each component's later tracker gate.
