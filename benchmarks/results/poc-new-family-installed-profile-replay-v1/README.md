# Ktor public installed qualified-profile replay

This bundle proves that public BuildOpt `v0.3.2` replays the three reviewed
Ktor structural profiles through the customer-facing `buildopt poc` command.
Each clean external checkout reconstructs the exact qualified change and uses
only repository-committed profile state.

| Change | Installed candidate | Option-drift fallback | Required outputs |
| --- | --- | --- | ---: |
| Upstream dependency source | `:ktor-http:jvmJar`, `:ktor-utils:jvmJar` | `jvmJar` | 2 exact JARs |
| JVM service resource | `:ktor-server-core:jvmJar` | `jvmJar` | 1 exact JAR |
| Mixed production source | `:ktor-http:jvmJar`, `:ktor-server-core:jvmJar` | `jvmJar` | 2 exact JARs |

All three exact profiles select `POC_CANDIDATE`. Repeating each invocation
with the complete qualified Gradle option list plus `--stacktrace` retains
native Gradle before execution as
`FULL_GRAPH / PROFILE_GRADLE_OPTIONS_DRIFT`. Every profile digest remains
satisfied, every candidate/native output matches by exact bytes, all three
historical output digests also match as a diagnostic, and product-attributable
failures are zero.

The bundle publishes structured plans, output manifests, results and the
normalized SHA-256 bindings of the candidate and fallback logs. The raw logs
were validated during capture but are intentionally not published.

This is installed-path correctness and adoption evidence. It creates no new
timing percentage and does not rewrite the terminal Ktor qualifications:
85.80% for dependency source, 86.51% for a JVM resource and 77.98% for the
mixed-source edit. Those values remain independent and are not averaged.

Revalidate the committed evidence and its negative fixtures without network
access or Gradle execution:

```bash
./dev/check-new-family-installed-profile-replay-result
./dev/test-new-family-installed-profile-replay-result
```

The preregistered method is
[`poc-new-family-installed-profile-replay-v1.md`](../../../specs/poc-new-family-installed-profile-replay-v1.md).
