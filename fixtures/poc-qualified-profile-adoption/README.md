# Qualified POC profile adoption fixtures

These fixtures are the repository-owned state consumed by the installed
`buildopt poc` command during the fixed OpenTelemetry and Kafka adoption
replay. They are not universal recommendations and they do not authorize a
production rollout.

Each repository directory contains the reviewed Build Impact manifest, the
generated graph and discovery state, and the exact qualified profile. The
generated files remain bound to the immutable public revision recorded in
`specs/poc-qualified-profile-adoption-v1.json`.

Run `./dev/check-poc-qualified-profile-adoption` to validate the checked state.
The real-repository replay is intentionally manual because it downloads and
hydrates both fixed public builds; it records decisions and output digests but
does not measure performance.
