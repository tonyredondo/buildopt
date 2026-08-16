# BuildOpt GitLab CI

For the copyable integration, version pinning and bypass behavior, see the
[CI integration guide](../docs/guides/ci-integration.md#gitlab-ci).

Include `buildopt-component.yml` at a reviewed full commit SHA. The normal path
only needs `command: optimize build`; `version` can pin the native release. The
component is self-contained and does not require BuildOpt scripts to be copied
into the consumer repository.

It installs a checksum-verified package below `.buildopt/runtime`, invokes the
repository Gradle Wrapper through `buildopt optimize`, restores only exact
provider-bound state from the project cache, and retains normalized and
redacted result artifacts for seven days. Cross-project merge requests force remote
behavior off. The component neither requests nor consumes a GitLab token,
deploy token, CI job token or BuildOpt remote credential.

The retained artifact includes the exact result, a customer-readable value
report, its recomputable JSON form and checksums. It excludes private state,
raw Gradle arguments, console logs and credentials.

Run `./dev/check-magic-ci-onboarding` and `./dev/check-gitlab-ci` for the
owner-controlled synthetic proofs.
