# One-input CI onboarding

## Decision

GitHub Actions and GitLab CI expose the same owner-invoked POC command:

```yaml
with:
  command: optimize build
```

The command is the only BuildOpt-specific input in the ordinary path. The
integration installs the verified native release, derives provider repository
and immutable revision facts, restores optional state, runs
`buildopt optimize`, and publishes a short review artifact. It requires no
BuildOpt service, hand-authored manifest, profile, graph, changes file, output
contract, evidence document, state path, or calibration flag.

Existing installation and review-only proposal modes remain compatibility
surfaces. They do not change the one-input path or grant selection authority.

## Portable exact state

Local optimize state remains tied to the canonical checkout path. In CI, the
repository-scope digest instead binds the provider, immutable numeric provider
repository ID, and provider repository path. The absolute runner checkout path
is never part of that CI digest or persisted report. Two runners for the same
provider repository can therefore present the same exact checkpoint.

A restored cache is always untrusted input. Before accepting it, the launcher
still compares the BuildOpt executable, Gradle Wrapper properties, complete
Gradle argv, provider repository scope, base and target revisions, discovery
context, and calibration budget. Drift starts a new native-authoritative
generation; malformed state is rejected and replaced. A cache miss or cache
service outage is only a performance miss. It cannot prevent the Gradle build.

The initial POC remains exact-revision only. State does not transfer between
commits, repositories, or CI providers. Cross-revision portfolio
applicability belongs to the separate centralized-state roadmap.

## Provider orchestration

GitHub uses the repository-scoped Actions cache, pinned to an immutable
`actions/cache` commit. Its key includes runner platform, provider repository
ID, checked-out SHA, resolved BuildOpt version, and Wrapper digest. GitLab uses
the project-scoped native job cache with project ID, checked-out SHA, and
requested BuildOpt version; the launcher remains the final compatibility gate
because GitLab cache entries can be replaced.

Both integrations reject user-supplied state/resume/calibration options. This
keeps one generated location and prevents a workflow from presenting an
arbitrary restored directory as qualified state.

## Review artifact and failure behavior

Both providers publish `.buildopt/ci-report/v1` with:

- `result.json`, when BuildOpt reaches final reporting;
- `value-report.md`, explaining graph reduction, measured value, economics and
  fallback in customer language;
- `value-report.json`, carrying the recomputable source metrics and formulas;
- `summary.md`, containing provider, repository, checked-out revision, and
  process result without raw Gradle arguments; and
- `SHA256SUMS` for every available result, summary, and value-report file.

Raw Gradle console output, credentials, absolute checkout paths, and the
command argument vector are not persisted. State stays in the provider cache;
it is not uploaded as a review artifact. Gradle or BuildOpt failure remains a
failed CI job, while the available report is uploaded for diagnosis.

Before the command starts, the helper removes only the previous derived result
and value-report files while preserving the learned state. An early launcher
failure therefore cannot publish stale evidence as the current job's result.

## POC boundary

Automatic profile use exists only inside the explicit `optimize` command and
keeps `productionAuthorized=false`. This block proves zero-configuration CI
orchestration and safe exact persistence. It does not claim cross-commit
learning, require a soak or design partner, introduce a central service, or
change Test Optimization.

The exact machine contract is
[`poc-magic-ci-onboarding-v1.json`](./poc-magic-ci-onboarding-v1.json).
