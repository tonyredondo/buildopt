# Build Impact shadow validation v1

This contract closes `C3-003` with immutable, manifest/graph/adapter-bound
observations. In `SHADOW`, only the complete original entrypoints run; a valid
full execution can validate the model but cannot authorize selection. In
`PAIRED_CONTROL`, an isolated candidate must use the exact predicted
customer-owned entrypoints and match the successful baseline.

Successful observations require the exact declared project reach, complete
required artifact IDs, safe paths, byte digests and sizes, plus every required
check with its original owner and `PASSED` outcome. Test Optimization checks are
preserved and compared; Build Optimization never selects them.

Candidate build failure, project/entrypoint mismatch, missing or divergent
artifacts, or changed checks produce an explicit `FALSE_NEGATIVE` result.
Infrastructure failure, cancellation, invalid/incomplete baseline, or a
decision that already fell back to `FULL_GRAPH` is `INCONCLUSIVE`. History
never changes either classification.

Run:

```bash
./dev/check-build-impact-shadow-validation
```

The checker composes C3-001..003, verifies checked-in shadow and paired-control
observations through production parsers, and executes the full failure matrix.
All emitted results retain `selectionAuthorized=false`; C3-004 alone evaluates
the unchanged long-window `BIA-002` evidence threshold.
