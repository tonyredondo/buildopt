# Wrapper-coordinated native corrections v1 evidence

State: `STOP_INSUFFICIENT_PROSPECTIVE_OPPORTUNITY_BREADTH`. Functional lanes `WCNCP-001` through
`WCNCP-007` are covered and `WCNCP-008` completed the frozen ten-family capture.
`WCNCP-009` is complete at 10/10 conclusive and 1/3 actionable material
families, so candidate, value, and review blocks never open. The prospective
[`WCNCP-009A` successor](../../../specs/poc-wcncp-controlled-materiality-v1.md)
preserved the exhausted local rows but its first attempt failed the stability
gate at 5.1795 after setup and before every controlled diagnostic. The checked
failure is retained under [`wcncp-e009a/attempt-1`](./wcncp-e009a/attempt-1/).
The separately frozen [`WCNCP-009B`](../../../specs/poc-wcncp-controlled-materiality-v2.md)
added only a fixed 120-second post-prefetch quiescence interval before the
unchanged gate. It passed at 1.028603 and completed all six controlled rows.

The pre-capture operational audit also added the missing administrative
`wcncp-actor grant` command: transport token issuance alone remains
untrusted, while an explicit grant enables the frozen actor authority without
printing the credential. A local real-process readiness reproduction preserved
the native child result, retained an upload that exceeded the 100-ms deadline,
and accepted its later exact batch publication; those synthetic facts are not
prospective cohort or value evidence.

The previous `INCOMPLETE_PERFORMANCE_ENVIRONMENT` conclusion is
`RETRACTED_PREMATURE_CLOSURE`: it jumped from an
unexecuted WCNCP-008 to WCNCP-013. It is retracted. The retained
[`terminal-decision.json`](./terminal-decision.json) and zero-row WCNCP-E009
through WCNCP-E012 files are audit records only and have no gate or terminal
authority. WCNCP-E008 has been replaced by the completed prospective record.

Current state:

- exact public cohort: `FROZEN_10_PRIMARY_PLUS_20_RESERVES`;
- prospective observations: 30 (3 per family);
- selected additional native diagnostics: 16 (maximum 20);
- conclusive opportunity families: 10/10;
- actionable material families: 1/3 required (GraphQL Java);
- WCNCP-009A controlled diagnostic starts: 0 (environment gate failed);
- WCNCP-009B controlled diagnostic starts: 6/6;
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
evidence for WCNCP-001 through WCNCP-007 remains synthetic. The
[`cohort-freeze.json`](./cohort-freeze.json) record
binds ten primary revisions, twenty chronological reserves, owner workflows,
required outputs, wrapper/workflow/source hashes, typed exclusions, and ten
successful native admissibility preflights. The completed
[`WCNCP-E008 capture`](./wcncp-e008-capture.json) and
[`observations`](./observations/) reconstruct 10/10 conclusive families, 30/30
successful native children and required-output manifests, ten acknowledged TLS
batches and verified snapshots, plus three separately excluded runner or
infrastructure attempts. No duration is performance evidence. The checked
[`WCNCP-E009 breadth report`](./wcncp-e009-breadth.json), [fresh evidence](./wcncp-e009/),
and [diagnostic selection](./wcncp-e009-diagnostic-selection.json) reconstruct
16 selected starts. The [controlled successor](./wcncp-e009b/README.md) adds six
rows and the [final report](./wcncp-e009-final.json) reconstructs 10/10
conclusive but only 1/3 actionable material families. The authoritative
[terminal decision](./wcncp-e013-final.json) therefore closes source mutation,
candidates, paired timing, and review for this cohort.

Reconstruct the checked freeze from the manifest and, when the ten public
checkouts are available, their Git archives with:

```bash
./dev/check-wcncp-cohort-freeze [family=/absolute/checkout ...]
```

Run the frozen planning/schema contract with:

```bash
./dev/check-wrapper-coordinated-native-corrections-plan
```

Reconstruct the prospective capture and its fail-closed negatives with:

```bash
./dev/check-wcncp-prospective-capture
./dev/check-wcncp-prospective-capture-negatives
```

Reconstruct the WCNCP-009 source, diagnostic, reachability and family counts
and exercise summary-falsification rejection with:

```bash
./dev/check-wcncp-opportunity-breadth [/absolute/source-root]
./dev/check-wcncp-opportunity-breadth-negatives
./dev/check-wcncp-controlled-materiality-plan
./dev/check-wcncp-controlled-materiality /absolute/evidence
./dev/check-wcncp-controlled-materiality-v2 "$PWD/benchmarks/results/wrapper-coordinated-native-corrections-v1/wcncp-e009b"
./dev/check-wcncp-opportunity-breadth-final
./dev/check-wcncp-terminal
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
