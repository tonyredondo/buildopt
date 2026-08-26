# Sticky-wrapper decision-store vectors

These fixtures exercise the closed JSON shapes. Cross-record ordering,
cryptographic signatures, equality of trial outputs, generation CAS, expiry,
revocation, corruption, scope and cache/state separation are tested by the Go
conformance suite in `internal/stickydecision`; JSON Schema alone cannot
compare those values.
