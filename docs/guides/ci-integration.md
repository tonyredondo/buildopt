# CI integration

The POC has one ordinary CI input: the Gradle workflow to optimize. BuildOpt
installs itself, derives provider revisions, restores only exact compatible
state, runs the repository Wrapper, and publishes a review artifact. The
target repository keeps its own tasks and exit semantics.

## GitHub Actions

Pin the Action source to a reviewed full commit SHA:

```yaml
- name: Optimize the existing Gradle build
  uses: tonyredondo/buildopt@<40-character-commit-sha>
  with:
    command: optimize build
```

The Action resolves the latest published native release, verifies its SHA-256
and internal manifest, installs below `runner.temp`, derives the immutable
GitHub base/head context, and caches `.buildopt/optimize/v1` under the provider
repository ID, exact SHA, Wrapper and resolved BuildOpt version. Restored files
are untrusted until the launcher validates every binding. The
`buildopt-optimize-result` artifact contains `result.json`, a readable CI
summary, the customer-level `value-report.md`, recomputable
`value-report.json`, and checksums for every included report. It never contains
the state portfolio, raw command, console log, credential or checkout path. To
keep native bits fixed while updating workflow code:

```yaml
with:
  version: <published-version>
```

Run the Action after the repository checkout and make the immutable comparison
base available (for `actions/checkout`, use `fetch-depth: 0`). This is provider
Git history, not a BuildOpt profile or configuration file. An unavailable base
retains native Gradle with an explicit reason.

`archive-url` and `archive-sha256` remain a paired compatibility override for
the older signed Release Bundle v1 installer. Ordinary users should not need
them.

Do not expose cache/server credentials to untrusted forks. The Action does not
request them and its default local path works without secrets.

Omit `command` to retain the historical install-only surface. The Action also
keeps the explicit review-only proposal mode below for existing consumers.

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

BuildOpt's separate manual workflow-breadth fixture applies that same confirmed
owner-input contract to packaging, typed verification, distribution, and
build-owned test preparation. It uploads review evidence with exact output
digests and verifies native fallback for an unsupported executable task. It
does not time or activate any candidate; real workflow families need their own
paired installed-path value evidence after review.
The preserved [hosted result](../../benchmarks/results/poc-generic-workflow-breadth-v1/README.md)
contains the exact decisions and output digests.

## GitLab CI

Include the component from an immutable BuildOpt revision:

```yaml
include:
  - project: tonyredondo/buildopt
    ref: <40-character-commit-sha>
    file: /.gitlab/buildopt-component.yml
    inputs:
      command: optimize build
```

The component is self-contained: it resolves and verifies the native package,
installs below `.buildopt/runtime`, derives GitLab project/base/head metadata,
runs `buildopt optimize`, restores and saves exact state through the native job
cache, and publishes a normalized job event plus the same result bundle. Set
`version` to pin native bits. Cross-project merge requests force remote
behavior off, and no GitLab or BuildOpt credential is requested.

GitHub and GitLab cache hits are performance hints, never authority. Missing,
corrupt, cross-repository, cross-revision, Wrapper-drifted or executable-drifted
state starts a new native-authoritative generation. This POC does not infer
profile applicability across commits and requires no central service. See the
[one-input CI contract](../../specs/poc-magic-ci-onboarding-v1.md).

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
