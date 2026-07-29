# F0-013 foundation contract fixtures

Synthetic golden and negative records for `EVIDENCE_RECORD`,
`OPTIMIZATION_POLICY`, and `RESOURCE_PROFILE` v1.

- `evidence/valid/qualified-adapter.json` shows complete observational evidence
  combined with independent repeatability and relocatability gates.
- `evidence/invalid/incomplete-trace-qualified.json` proves incomplete tracing
  cannot retain `QUARANTINE_VALIDATED`.
- `policy/valid/verified-policy.json` binds one exact catalog/profile and one
  qualified task. `policy/invalid/active-kill-switch.json` proves a kill switch
  cannot retain actions or cache access.
- `resource-profile/valid` is the exact four-arm golden-runner catalog required
  by `BANDIT-001`. Every field other than workers, Gradle heap, profile identity,
  and digest is held constant.
- `resource-profile/invalid/eligible-with-failed-memory.json` proves a failed
  startup, memory, or rollback gate cannot be eligible.

The cross-record test also checks policy/evidence identity and digest binding,
timestamp order, the four-arm catalog, fixed non-treatment fields, and cgroup
headroom. The records contain no production identifiers or secrets.
