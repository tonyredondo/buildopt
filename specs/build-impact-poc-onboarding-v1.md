# Build Impact POC onboarding

This contract makes the Build Impact mechanism that cleared the POC value gate
available from an installed `buildopt` package without pretending that the POC
has production promotion authority.

The explicit command reads a repository-committed manifest, its exact reviewed
graph and generated binding, plus a newline-delimited file of changed paths:

```bash
git diff --name-only --diff-filter=ACMR BASE_SHA HEAD_SHA > .buildopt-changes
buildopt impact \
  --repository-id owner/repository \
  --pipeline-class pull-request \
  --changes-file .buildopt-changes \
  --gradle-option=--no-daemon
```

`buildopt impact` is an owner-operated candidate command. It may execute only
an alternative already enumerated by the repository manifest. Global changes,
unknown paths, incomplete relationships, or a candidate that does not cover
every Build Optimization-owned artifact and check retain the manifest's full
entrypoints. Malformed or drifted generated state is rejected before Gradle
starts.

An existing but empty changed-path file is a valid no-change observation and
also retains the full graph. A missing file remains a configuration error.

Repeated `--gradle-option` values are restricted to a bounded allowlist of
execution controls such as offline mode, console mode, daemon/cache flags and
an explicit positive worker count. They cannot add or exclude tasks, select a
different project root, or override the manifest's entrypoints.

`BUILDOPT_BYPASS=1` always restores the original entrypoints and does not
require the generated graph to be healthy, preserving the existing recovery
contract.

The command does not call a production promotion report, weaken `BIA-002`, or
claim production authorization. Test Optimization-owned checks remain outside
the selected build graph and must still be executed by their existing workflow.
The POC checker runs that independent check in both arms and requires identical
bytes.

Run the installed-path proof with:

```bash
./dev/run --toolchain temurin-jdk-21 -- ./dev/check-build-impact-poc-onboarding
```

The checker builds the actual `buildopt` binary, executes the candidate and
full fallback through the repository Gradle Wrapper offline, requires the
candidate to omit `service-b`, requires the fallback to build it, compares the
required `service-a` JAR and independent Test-owned marker byte for byte, and
checks that the source tree remains unchanged.
