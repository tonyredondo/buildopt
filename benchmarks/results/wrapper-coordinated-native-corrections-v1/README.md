# Wrapper-coordinated native corrections v1 evidence

State: `FUNCTIONAL_FOUNDATION_READY`. Functional lanes `WCNCP-001` through
`WCNCP-007` are repaired and covered on standard CI. `WCNCP-008` is the next
block: freeze the neutral cohort and collect fresh ordinary wrapper
observations. No prospective row exists yet.

The previous `INCOMPLETE_PERFORMANCE_ENVIRONMENT` conclusion is
`RETRACTED_PREMATURE_CLOSURE`: it jumped from an
unexecuted WCNCP-008 to WCNCP-013. It is retracted. The retained
[`terminal-decision.json`](./terminal-decision.json),
[`report.json`](./report.json), and zero-row WCNCP-E008 through WCNCP-E012
files are audit records only and have no gate or terminal authority.

Current state:

- exact public cohort: `UNSELECTED_UNTIL_WCNCP_008`;
- prospective observations: 0;
- public source patches: 0;
- candidate builds: 0;
- timing samples: 0;
- owner reviews: 0;
- product failures: 0; and
- speedup or product-value claim: none.

The future diagnostic system lane may replay the known Micronaut, Spring, and
Elasticsearch recipes only as `HISTORICAL_RECIPE_SYSTEM_FIXTURE`. Historical
BuildOpt reports cannot supply prospective rows, breadth, timing, economics, or
terminal authority.

Expected future artifacts are listed in the
[tracker](../../../docs/plans/wrapper-coordinated-native-corrections-poc.md).
`WCNCP-003` is `DONE`: exact native passthrough including signals, schema-valid
observations, privacy redaction, bounded retrying outbox with acknowledged
deletion, real batch upload, and remotely verified status. Controlled overhead
remains pending. `WCNCP-004` is `DONE`: raw observation reconstruction,
repetition-aware compatibility grouping, detector v1, catalog adapters,
hosted materiality neutrality, and name-invariance negatives over synthetic
fixtures. None authorizes source mutation, candidate execution, or timing.

`WCNCP-E001` proof is
[`wcncp-e001-typed-state.json`](./wcncp-e001-typed-state.json),
`WCNCP-E002` proof is
[`wcncp-e002-authority.json`](./wcncp-e002-authority.json),
`WCNCP-E003` proof is
[`wcncp-e003-observation.json`](./wcncp-e003-observation.json), and
`WCNCP-E004` proof is
[`wcncp-e004-detector.json`](./wcncp-e004-detector.json), and
`WCNCP-E005` proof is
[`wcncp-e005-validation.json`](./wcncp-e005-validation.json), and
`WCNCP-E006` proof is
[`wcncp-e006-review.json`](./wcncp-e006-review.json), and
`WCNCP-E007` proof is
[`wcncp-e007-system.json`](./wcncp-e007-system.json): synthetic-only
converged observations, single lease ownership, outage preservation, restart
retry, isolation, fail-closed negatives, and durable decisions. Prospective
evidence remains zero. `WCNCP-008` is next: frozen cohort plus ordinary
wrapper observations only.

Run the frozen planning/schema contract with:

```bash
./dev/check-wrapper-coordinated-native-corrections-plan
```

Run the WCNCP-001 typed-state proof with:

```bash
./dev/check-wcncp-typed-state
```

Run the WCNCP-002 authority proof with:

```bash
./dev/check-wcncp-authority
```

Run the WCNCP-003 observation proof with:

```bash
./dev/check-wcncp-observation
```

Run the WCNCP-004 detector proof with:

```bash
./dev/check-wcncp-detector
```

Run the WCNCP-005 validator proof with:

```bash
./dev/check-wcncp-validation
```

Run the WCNCP-006 review proof with:

```bash
./dev/check-wcncp-review
```

Run the WCNCP-007 system proof with:

```bash
./dev/check-wcncp-system
```

Verify that the prospective lane remains open and that no retracted zero-row
artifact is treated as a terminal decision with:

```bash
./dev/check-wcncp-terminal
```
