# CI orchestration fixture

`github-validation.yml` is the inert GitHub Actions shape required by
`F0-030`. It fixes the protected-branch schedule, recovery dispatch,
repository concurrency, read-only permission, and single-lease adapter
boundary. It is validated as data and is not an active workflow in this
repository.

`private-beta-token-isolation.yml` is the inert `A1-G01` composition. Its fork
job has no BuildOpt token, same-repository pull requests use the stable read
secret, and only protected `main` uses the distinct stable read-write secret.
It deliberately excludes `pull_request_target` and is validated as data, not
activated in this repository.

Run `./dev/check-ci-orchestration` and
`./dev/check-private-beta-token-isolation` from the repository root.
