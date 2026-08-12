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
  version: 0.2.0
```

`archive-url` and `archive-sha256` remain a paired compatibility override for
the older signed Release Bundle v1 installer. Ordinary users should not need
them.

Do not expose cache/server credentials to untrusted forks. The Action does not
request them and its default local path works without secrets.

### Review a structural proposal

Before adopting Build Impact, a repository can ask CI for a proposal without
changing or timing its build. Generate and check in `.buildopt/profile.json`
using the [owner-input contract](../../specs/poc-generic-owner-input-v1.md),
then use the strict [review-only CI contract](../../specs/poc-generic-profile-ci-v1.md),
check out the exact target with its base available, and use the same Action at
a full commit:

```yaml
- name: Publish the BuildOpt proposal
  uses: tonyredondo/buildopt@<40-character-commit-sha>
  with:
    mode: profile-proposal
    profile-base-revision: ${{ github.event.pull_request.base.sha }}
    profile-target-revision: ${{ github.event.pull_request.head.sha }}
```

The `buildopt-profile-proposal` artifact contains normalized inputs, the exact
change set, proposal or native decision, a readable summary and checksums. It
does not run `profile measure`, write `buildopt-qualified-profile.json`, edit
the workflow or activate anything. Treat `NATIVE_FULL_GRAPH` as the safe and
expected outcome whenever discovery is incomplete or the change is global.

BuildOpt's own manual five-repository replay runs this exact surface on clean
Spring, OpenTelemetry, Kafka, Micronaut, and Groovy checkouts. It verifies that
the owner inputs still reproduce the reviewed terminal graphs and records any
difference as drift. It creates no new timing result: a matching proposal must
still pass the separate paired measurement and review gate before activation.
See the [replay specification](../../specs/poc-generic-profile-ci-replay-v1.md).

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
Set `version: 0.2.0` to pin native bits. Cross-project merge requests force
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

After `check`, materialize the CI provider's exact base/head diff and run the
explicit POC candidate:

```bash
git diff --name-only --diff-filter=ACMR BASE_SHA HEAD_SHA > .buildopt-changes
buildopt poc --changes-file .buildopt-changes
```

Commit the repository-owned qualified profile before using this command. It
enables only Build Impact and the exact standard `Jar` adapter for a selected
alternative; fallback is native full graph. Resolve `BASE_SHA` and `HEAD_SHA` from immutable provider revision fields; do
not infer a branch name or silently accept a shallow/empty diff. Keep every
Test-owned command as a separate unchanged workflow step. The command is an
owner-operated POC candidate and never substitutes for `BIA-002` production
promotion.

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
