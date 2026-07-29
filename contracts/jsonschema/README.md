# JSON Schema contracts

Versioned JSON Schema 2020-12 contracts for exportable records and signed commands.

Schema implementation begins with `build-session.v1.schema.json` in `F0-011`. Every schema must define its identifier, versioning and compatibility behavior, required fields, unknown-field policy, formats, bounds, and valid and invalid fixtures. Signed commands fail closed on unknown fields.

`F0-010` creates this namespace only; it does not materialize empty schemas.
