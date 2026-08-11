# Review-only generic profile proposal in CI

## Purpose

`POC-GENERIC-PROFILE-CI-001` exposes the unchanged generic structural proposal
command through the repository-root GitHub Action. A repository owner supplies
one checked-in configuration plus exact base and target commits. CI derives the
change set and uploads a review artifact. It never measures, writes an active
profile, changes the Gradle workflow, or authorizes production use.

This block answers an adoption question: can a clean CI checkout produce the
same bounded proposal or explicit native decision without hand-authored BuildOpt
graphs and without repository-name logic?

## Owner configuration

The default path is `.buildopt/profile-ci.json`:

```json
{
  "schemaVersion": "buildopt.poc/profile-ci-input/v1",
  "repositoryId": "owner/repository",
  "pipelineClass": "pull-request",
  "entrypoints": ["assemble"],
  "requiredOutputs": ["module/build/libs/*.jar"],
  "globalChanges": [],
  "gradleCommand": "./gradlew",
  "gradleOptions": [],
  "timeoutMinutes": 10
}
```

The original Gradle entrypoints and required outputs are repository-owned
semantics. An empty `globalChanges` array keeps BuildOpt's conservative default
for settings, root build logic, `buildSrc`, `build-logic`, Gradle properties and
Wrapper changes. The Action accepts only the strict v1 keys, bounded unique
arrays, a clean relative Gradle command and a timeout from one to 30 minutes.
The configuration contains no credentials.

## Immutable CI invocation

Consumers pin the Action to a reviewed full commit and check out the exact
target with enough history to resolve its base:

```yaml
- name: Check out the exact target
  uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1
  with:
    ref: ${{ github.event.pull_request.head.sha }}
    fetch-depth: 0
    persist-credentials: false

- name: Publish the BuildOpt proposal
  id: buildopt-proposal
  uses: tonyredondo/buildopt@<40-character-commit-sha>
  with:
    mode: profile-proposal
    profile-base-revision: ${{ github.event.pull_request.base.sha }}
    profile-target-revision: ${{ github.event.pull_request.head.sha }}
```

The Action requires Linux, read-only repository permission, a full immutable
BuildOpt Action reference and an exact checked-out target. It compiles the CLI
from that pinned source rather than resolving an older release. Its internal
`setup-go` and `upload-artifact` references are full commits.

## Review artifact

The uploaded `buildopt-profile-proposal` bundle contains:

- normalized owner input and the exact sorted base-to-target change set;
- `review.json`, a short Markdown summary and SHA-256 checksums;
- the complete `buildopt-profile-proposal.json` decision;
- candidate manifest, graph, generated-state binding and fallback input only
  when discovery produces `MEASURE_STRUCTURAL_CANDIDATE`; and
- the exact `profile measure` handoff bound to the BuildOpt source revision.

`NATIVE_FULL_GRAPH` is a successful, reviewable outcome for global, ambiguous,
incomplete, Test-bearing, unsupported or non-reducing workflows. It creates no
candidate documents. An invalid configuration, missing revision or Gradle
command failure uploads diagnostics when possible and then fails the job; it
cannot appear as a successful proposal.

Every complete artifact requires `reviewRequired=true`,
`activationAutomatic=false`, `productionAuthorized=false` and
`testOptimization=OUT_OF_SCOPE`.

## Conformance

Local conformance creates independent clean Git repositories and proves a
candidate, byte-identical replay, native fallback and invalid-target rejection:

```bash
./dev/check-generic-profile-ci
```

The manual read-only hosted fixture exercises the composite Action and uploads
the real artifact:

```bash
gh workflow run profile-proposal-fixture.yml
```

The hosted run is adoption evidence only. It creates no wall-time percentage;
performance remains owned by `profile measure` and `profile evaluate` after a
human reviews the proposal.
