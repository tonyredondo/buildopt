# CI orchestration fixture

`github-validation.yml` is the inert GitHub Actions shape required by
`F0-030`. It fixes the protected-branch schedule, recovery dispatch,
repository concurrency, read-only permission, and single-lease adapter
boundary. It is validated as data and is not an active workflow in this
repository.

Run `./dev/check-ci-orchestration` from the repository root.
