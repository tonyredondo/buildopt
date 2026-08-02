# GitLab CI component v1

Consumers include `.gitlab/buildopt-component.yml` at a full BuildOpt commit
SHA and supply an exact release version, HTTPS archive URL, and lowercase
SHA-256. The component reuses the same strict Release Bundle v1 installer as
the GitHub Action, so download, layout, modes, checksums, atomic installation,
and idempotent reuse have one implementation.

The generated environment file is private and contains only verified local
paths. The job invokes the repository Gradle wrapper through `buildopt run`
and the packaged init script, then retains the redacted event/export directory
for seven days even when the job fails.

Merge requests from another GitLab project force remote behavior off. The
template does not request or consume a GitLab token, deploy token, CI job
token, or BuildOpt remote credential.

Run the composed component, event, and synthetic integration proof with
`./dev/check-gitlab-ci`.
