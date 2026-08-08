# Qualified-profile matrix fixtures

The Spring fixture is the reviewed repository-owned profile for the fixed
Spring Framework matrix cell. It enables Build Impact only because the direct
standard-Jar adapter did not qualify on this scope.

OpenTelemetry and Kafka reuse the profiles under
`fixtures/poc-qualified-profile-adoption/` and `fixtures/poc-kafka-packaging/`.
All three profiles remain POC fixtures bound to immutable public revisions;
they are not universal recommendations or production policy.
