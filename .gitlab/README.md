# BuildOpt GitLab CI

For the copyable integration, version pinning and bypass behavior, see the
[CI integration guide](../docs/guides/ci-integration.md#gitlab-ci).

Include `buildopt-component.yml` at a reviewed full commit SHA. The normal path
only needs `gradle-tasks`; `version` can pin the native release. The component
is self-contained and does not require BuildOpt scripts to be copied into the
consumer repository.

It installs a checksum-verified package below `.buildopt/runtime`, invokes the
repository Gradle Wrapper through `buildopt gradle`, and retains normalized and
redacted artifacts for seven days. Cross-project merge requests force remote
behavior off. The component neither requests nor consumes a GitLab token,
deploy token, CI job token or BuildOpt remote credential.

Run `./dev/check-gitlab-ci` for the owner-controlled synthetic proof.
