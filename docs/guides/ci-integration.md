# CI integration

BuildOpt integrates with GitHub Actions and GitLab CI through immutable source
and release identities. Both paths currently target Linux AMD64 runners and
run the repository's own Gradle Wrapper.

## Required release inputs

Obtain these values from an authenticated release publication:

- exact BuildOpt version;
- HTTPS archive URL;
- lowercase SHA-256 of the complete archive;
- complete commit SHA for the Action or GitLab component.

The installer verifies the archive digest and exact Release Bundle v1 layout
before exposing any binary. A checksum downloaded from the same untrusted
location as the archive is not an independent trust root.

## GitHub Actions

Pin the Action to a 40-character commit SHA:

```yaml
- name: Install BuildOpt
  id: buildopt
  uses: tonyredondo/buildopt@<40-character-commit-sha>
  with:
    version: <exact-release-version>
    archive-url: https://releases.example/buildopt-<version>-linux-amd64.tar.gz
    archive-sha256: <64-lowercase-hex>

- name: Run the existing Gradle command
  shell: bash
  run: >-
    buildopt run --
    ./gradlew
    --init-script "$BUILDOPT_GRADLE_INIT_SCRIPT"
    build
```

The Action installs below `runner.temp`, adds only the packaged `bin/` to
subsequent `PATH`, and exposes non-secret paths to the release root, server,
plugin, agent, and init script. It supports GitHub-hosted and self-hosted Linux
x64 runners.

Do not pass cache or server credentials to untrusted forks. The launcher keeps
its own secrets out of Gradle, but the workflow still owns event and fork
permissions. Validate the maintained Action fixture with:

```bash
./dev/check-github-action
```

See [`action.yml`](../../action.yml) for inputs/outputs and
[the fixture README](../../fixtures/github-actions/README.md) for the exact
immutable test pins.

## GitLab CI

Include the component from an immutable BuildOpt revision and pass the same
release identity:

```yaml
include:
  - project: tonyredondo/buildopt
    ref: <40-character-commit-sha>
    file: /.gitlab/buildopt-component.yml
    inputs:
      version: <exact-release-version>
      archive-url: https://releases.example/buildopt-<version>-linux-amd64.tar.gz
      archive-sha256: <64-lowercase-hex>
      gradle-tasks: build
```

The component invokes the repository Wrapper, publishes a private normalized
job-event artifact and redacted BuildOpt exports for seven days, and records
failure, cancellation, and unavailable outcomes explicitly. Cross-project
merge requests force remote behavior off. The component does not request a
GitLab access, deploy, or job token for BuildOpt.

Validate the complete synthetic GitLab path with:

```bash
./dev/check-gitlab-ci
```

See [`.gitlab/README.md`](../../.gitlab/README.md) and
[`buildopt-component.yml`](../../.gitlab/buildopt-component.yml) for the exact
input names supported by the current revision.

## Bypass and rollback

Define the launcher bypass at the invocation when BuildOpt must be removed
without changing the original Gradle command:

```yaml
env:
  BUILDOPT_BYPASS: '1'
```

Only `1` activates bypass. For a persistent rollback, pin the previously
verified Action/component SHA and release identity together. Do not mix a new
installer revision with an unrelated old archive unless that exact combination
is part of a validated release contract.

The [base recovery runbook](../../runbooks/base-recovery.md) covers kill
switch, rollback, removal, state preservation/purge, and partial patch-delivery
recovery.

## CI result handling

- The Gradle command's exit status remains authoritative once it starts.
- Missing telemetry or history export is not rewritten as a successful
  observation; it is unavailable or diagnostic according to the contract.
- Build Impact must use `check` against committed generated state and retains
  full original entrypoints on drift.
- Test-owned checks remain outside BuildOpt selection and must still be run by
  their owning CI workflow.
- Redacted export artifacts are still private operational data; apply the
  repository's retention and access policy.

## Validate a CI change locally

Provision the locked linters and run the narrow gates:

```bash
./dev/bootstrap --toolchain shellcheck
./dev/bootstrap --toolchain actionlint
./dev/check-lint-toolchains
./dev/check-github-action
./dev/check-gitlab-ci
./dev/check-ci-orchestration
./dev/check-base-ci --static
```

For source changes to installers or package layout, also run
`./dev/check-release-package` and the relevant platform gate.
