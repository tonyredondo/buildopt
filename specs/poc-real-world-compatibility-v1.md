# Public-repository compatibility

This contract is the safety and compatibility gate before BuildOpt uses public
repositories as performance evidence. It does not measure savings.

Three released revisions are fixed by commit, wrapper, settings, and Gradle
distribution hashes. Each repository runs one representative JVM task twice:
once through its Gradle Wrapper and once through the installed public
`buildopt gradle` entry point. Both arms use clean disposable checkouts and
separate empty homes on the 4-CPU/16-GiB runner.

## Repository targets

| Repository | Gradle / DSL | Target | Scope deliberately excluded |
|---|---|---|---|
| Spotless `gradle/8.9.0` | 9.4.1 / Groovy | `:plugin-gradle:testClasses` | Maven plugin, CI remote cache, scans, publication |
| Mockito `v5.23.0` | 8.14.2 / Kotlin | `:mockito-extensions:mockito-junit-jupiter:testClasses` | Android, GraalVM integration, release and publication |
| SpotBugs `4.10.3` | 9.6.1 / mixed | `:spotbugs-tests:testClasses` | Eclipse test/site tasks, signing, publication |

`CI`, `ANDROID_HOME`, cache credentials, and other repository-specific opt-ins
are absent rather than assigned empty strings. This matters because Gradle's
environment provider treats an empty variable as present. Build scans are
disabled explicitly, and any executed publish, upload, release, or signing task
fails the gate.

SpotBugs does not record its Gradle distribution checksum in the selected
revision. The harness verifies the original wrapper files and then injects the
official Gradle 9.6.1 checksum into both disposable checkouts before execution.
That auditable safety patch is the only permitted source-tree modification.

Passing authorizes `POC-REALWORLD-002`, a preregistered paired performance
matrix. It does not authorize a performance, universal-savings, production,
soak, design-partner, or Test Optimization claim.

Validate checked evidence with:

```bash
./dev/check-poc-real-world-compatibility
```
