# Gradle bootstrap cache v1

This specification materializes `A0-007`. It consumes an orchestrator-built
dependency snapshot through Gradle's supported incubating
`GRADLE_RO_DEP_CACHE` mechanism and retains verified Wrapper distributions in
one private writable home per runner and signed cache scope. It does not share
a writable `GRADLE_USER_HOME`, transport Configuration Cache entries, or
invent a dependency-cache format.

## Signed activation

The launcher enables the managed path only when the current local authority
contains `DEPENDENCY_CACHE` and
`BUILDOPT_GRADLE_BOOTSTRAP_CONFIG_PATH` names an exact canonical-JCS,
current-user-owned mode-`0600` document. The document binds an absolute private
state root, runner slot, compatibility class, dependency snapshot, repository
Wrapper properties, trusted distribution archive, and Wrapper JAR digest.

Only the repository's regular executable `gradlew` is accepted. The Wrapper
properties must use its normal `GRADLE_USER_HOME` paths, enable URL validation,
provide a SHA-256 distribution checksum, use a canonical HTTPS URL, and select
the tested Gradle 8.14.3 or 9.6.1 `bin`/`all` distribution.

## Read-only dependency snapshot

The dependency root contains exactly:

```text
<dependencyCacheRoot>/
  buildopt-dependency-cache.json
  modules-2/
```

The root, manifest, and complete `modules-2` tree are trusted regular
files/directories without writable permission bits or symlinks. Lock files are
rejected. The canonical manifest binds `gradleVersion`,
`compatibilityClass`, the signed `configurationPolicyDigest`, and an immutable
`snapshotId`. Repository dependency verification and locking are therefore
part of the signed compatibility policy rather than mutable launcher state.

Snapshot publication belongs to orchestration outside an active Gradle
invocation. The launcher never mutates or repairs the shared tree and exposes
it only as `GRADLE_RO_DEP_CACHE`.

## Private writable layer and Wrapper reuse

The launcher hashes the authority scope, signed configuration policy, runner
slot, compatibility class, dependency root and snapshot, distribution URL and
checksum, and Wrapper JAR checksum into an opaque scope. That scope owns one
mode-`0700` `GRADLE_USER_HOME` and one non-blocking exclusive lease for the
complete child lifetime. Different runners or incompatible policy/snapshot
inputs cannot share writable metadata, locks, daemons, or Configuration Cache.

Before first use, the launcher checks the repository Wrapper JAR, reads a
trusted read-only distribution archive, verifies its SHA-256 independently
from task-output cache authority, and copies it into Gradle Wrapper's native
URL-hash layout. Gradle performs its normal checksum validation and extraction.
After the child, the launcher retains a private canonical marker only when the
native `.ok` file and expected executable exist. A later invocation may reuse
that bound installation after the source archive has disappeared.

The child receives only `GRADLE_USER_HOME` and `GRADLE_RO_DEP_CACHE`; the raw
BuildOpt configuration path and all authority material are removed.

## Fallback

Missing configuration preserves the caller's existing Gradle environment.
Invalid authority/configuration, unsupported versions, an unsafe snapshot,
checksum failure, a busy writable scope, or a corrupt retained distribution
emits a diagnostic and runs without managed Gradle overrides. BuildOpt does not
convert a bootstrap failure into a build failure or delete ambiguous state.

## Executable evidence

Run:

```bash
./dev/check-gradle-bootstrap-cache
```

The checker validates the exact machine contract, race-enabled launcher
security/lifecycle tests, vet, and a static binary build. It then exercises
Gradle 8.14.3 and 9.6.1 with JDK 17 and 21. Every row seeds a real dependency,
publishes only `modules-2` as a recursive read-only snapshot, installs the
checksum-pinned distribution through the Wrapper, resolves offline without
copying the artifact into the writable layer, and reuses the installation
after its source archive is removed.
