# JVM components

Multi-project build reserved for the beta's three Java artifacts:

- `gradle-plugin/`
- `jvm-agent/`
- `patcher/`

The golden lane pins the Gradle Wrapper and shared configuration. The artifacts produce Java 17-compatible bytecode and maintain separate lifecycles and versioning.
