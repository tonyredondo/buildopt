# ADR 0001: Initial golden lane

- Status: accepted
- Date: 2026-07-29
- Revision: 2026-07-29 — development runner adjusted to 4 vCPU/16 GiB
- Items: `F0-002`, `GOLDEN-LANE-001`

## Context

The walking skeleton and initial spikes need one reproducible combination before expanding the capability matrix. Without a stable runner class, we cannot attribute duration differences to the product or distinguish a regression from an infrastructure change.

The RFC pins Gradle 9.6.1, JDK 21, Linux x86-64, Kotlin DSL, and a 4 vCPU/16 GiB development runner. This class matches the workstation used for the initial implementation and makes the gate reproducible without requiring additional infrastructure. This ADR materializes versions, digests, checksums, and validation commands without depending on a global Gradle installation.

## Decision

Golden lane v1 is defined by:

- Gradle Wrapper 9.6.1 with the `bin` distribution.
- Distribution SHA-256 `9c0f7faeeb306cb14e4279a3e084ca6b596894089a0638e68a07c945a32c9e14`.
- Wrapper JAR SHA-256 `497c8c2a7e5031f6aa847f88104aa80a93532ec32ee17bdb8d1d2f67a194a9c7`.
- Official Eclipse Temurin image `21.0.11_10-jdk-jammy`.
- Linux amd64 image manifest `sha256:a5418a1fcf440bb273e1db3bce5b0794eb78bfc9d044ba740de76dcbe6075f50`.
- Source manifest list `sha256:9d8dcf999b0bce2453e913823595a5ff2a4e8e9e5d5241b45280d0ff069818ec`.
- Java 21.0.11+10 to run Gradle and `--release 17` for product bytecode.
- Linux amd64, 4 vCPU, 16 GiB nominal memory, `C.UTF-8` locale, and UTC timezone.
- Kotlin DSL for the initial fixture.

The platform digest is the executable identity. The tag is retained as readable provenance but never authorizes an implicit update. The workstation may use another JDK 21 patch release for smoke tests; only the pinned image and runner class produce golden lane evidence.

This class serves MVP conformance, integration, and development. Performance results are not extrapolated from it to customer runners: later benchmarks and experiments declare their own `runnerClass`, baseline, and resource catalog.

The wrapper enables checksum verification, URL validation, a 30-second timeout, and three retries with an initial one-second backoff. Configuration Cache and Build Cache are enabled in the fixture so the first vertical slices use the same path the product will evaluate.

The machine-readable specification is [`specs/golden-lane-runner-v1.json`](../specs/golden-lane-runner-v1.json).

## Validation

Four levels retain distinct results:

1. `./dev/check-golden-lane --static` validates local contracts, properties, and checksums without running a build.
2. `buildopt-with-jdk21 ./dev/check-golden-lane --smoke` compiles and packages on a non-contractual host, then checks Gradle, JDK, Java 17 bytecode, and the deliverable.
3. `buildopt-with-jdk21 ./dev/check-golden-lane --require-runner-class` verifies the workstation and build directly.
4. `./dev/run-golden-lane-container --require-runner-class` enforces 4 CPU/16 GiB, runs the image by digest, and verifies effective cgroup limits.

`--smoke` is never recorded as runner-class evidence. A host with insufficient resources can validate the content but not the contractual gate. `GOLDEN-LANE-001` requires levels 3 and 4 to complete successfully.

Direct validation requires exactly 4 logical CPUs and observable memory between 15 and 16 GiB, allowing for operating-system reservation. Container validation accepts an equal or larger host, applies 4 CPU/16 GiB limits, and checks effective cgroup v2 values from inside the container. The preflight uses the 15 GiB observable minimum rather than the nominal 16 GiB so reserved system memory is not mistaken for missing capacity.

## Rejected alternatives

- Global `gradle`: introduces unversioned state and does not verify the distribution.
- Moving aliases such as `latest` or `21-jdk-jammy`: may change without a repository change.
- Building a custom image around local JDK 21.0.12 now: adds publishing, signing, and lifecycle work before `F0-038`. The MVP uses an exact official image and platform digest instead.
- Treating the unconstrained local host as the golden runner: its capacity alone does not define a reproducible runner class.

## Consequences

- Changing Gradle, JDK, image, CPU, memory, locale, or timezone creates another runner-class version and requires baseline revalidation.
- The first build needs access to the Gradle distribution; later executions verify and reuse the local copy.
- `GOLDEN-LANE-001` does not close until strict mode runs on both host and container over a compatible runner; the `F0-002` definition artifacts may be versioned first.

## Sources

- [Gradle 9.6.1 distribution](https://services.gradle.org/distributions/gradle-9.6.1-bin.zip)
- [Gradle 9.6.1 distribution checksum](https://services.gradle.org/distributions/gradle-9.6.1-bin.zip.sha256)
- [Gradle 9.6.1 wrapper checksum](https://services.gradle.org/distributions/gradle-9.6.1-wrapper.jar.sha256)
- [Gradle Wrapper verification](https://docs.gradle.org/9.6.1/userguide/gradle_wrapper.html#sec:verification)
- [Gradle Java compatibility](https://docs.gradle.org/9.6.1/userguide/compatibility.html)
- [Eclipse Temurin official image](https://hub.docker.com/_/eclipse-temurin)
