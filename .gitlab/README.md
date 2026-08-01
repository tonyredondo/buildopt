# BuildOpt GitLab CI

Include `buildopt-component.yml` at a full 40-character BuildOpt commit SHA
and pass an exact release version, HTTPS archive URL, lowercase SHA-256, and
Gradle tasks.

The component supports Linux AMD64 runners, reuses the same verified Release
Bundle installer as GitHub Actions, invokes the repository Gradle wrapper, and
retains only normalized/redacted artifacts for seven days.

Cross-project merge requests force remote behavior off. The component neither
requests nor consumes a GitLab token, deploy token, CI job token, or BuildOpt
remote credential.

Run `./dev/check-gitlab-ci` for the owner-controlled synthetic proof.
