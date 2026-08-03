# CI integration

BuildOpt has one CI entrypoint: install the package, then replace
`./gradlew ...` with `buildopt gradle ...`. The target repository keeps its own
Wrapper, tasks and exit semantics.

## GitHub Actions

Pin the Action source to a reviewed full commit SHA:

```yaml
- name: Install BuildOpt
  uses: tonyredondo/buildopt@<40-character-commit-sha>

- name: Run the existing Gradle build
  run: buildopt gradle --no-daemon build
```

The Action resolves the latest published native release, verifies its SHA-256
and internal manifest, installs below `runner.temp`, and adds its `bin/` to
subsequent `PATH`. To keep native bits fixed while updating workflow code:

```yaml
with:
  version: 0.1.1
```

`archive-url` and `archive-sha256` remain a paired compatibility override for
the older signed Release Bundle v1 installer. Ordinary users should not need
them.

Do not expose cache/server credentials to untrusted forks. The Action does not
request them and its default local path works without secrets.

## GitLab CI

Include the component from an immutable BuildOpt revision:

```yaml
include:
  - project: tonyredondo/buildopt
    ref: <40-character-commit-sha>
    file: /.gitlab/buildopt-component.yml
    inputs:
      gradle-tasks: build
```

The component is self-contained: it resolves and verifies the native package,
installs below `.buildopt/runtime`, runs the repository Wrapper through
`buildopt gradle`, and publishes a normalized job event plus redacted exports.
Set `version: 0.1.1` to pin native bits. Cross-project merge requests force
remote behavior off, and no GitLab or BuildOpt credential is requested.

## Bypass and rollback

Use the same workflow with the optimization path removed:

```yaml
env:
  BUILDOPT_BYPASS: '1'
```

Only `1` activates bypass. For persistent rollback, restore the previous
Action/component SHA and native version together.

## Build Impact

Adopt Build Impact as a separate CI step after the base build is stable. Run
`generate` locally or in an explicit update job, review and commit the graph,
then use `check` in ordinary pull requests. Drift or uncertainty falls back to
the full repository-authorized entrypoint; it never selects tests outside the
owning workflow.

## Validate an integration change

```bash
./dev/check-github-action
./dev/check-gitlab-ci
./dev/check-ci-orchestration
./dev/check-base-ci --static
```

Platform package changes are additionally exercised by Native Platform CI.
See [product onboarding](../getting-started/product-onboarding.md) for the
component rollout order and [configuration](../reference/configuration.md) for
credential boundaries.
