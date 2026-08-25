# Structural profile rebinding result

This bounded POC result binds implementation commit
`7592fa57be4c84b69feca522a999849beb195e4a` to a canonical structural profile
fingerprint. The identity includes repository scope, requested and candidate
workflow, Gradle options, Wrapper, complete producer lineage, required and
candidate output contract, and change family. It excludes commit IDs and
absolute checkout paths.

## Result

- 2/2 structurally identical contexts produce the same fingerprint across
  different commits and Linux/Windows-style checkout roots.
- 5/5 compatibility drifts produce a different fingerprint: Wrapper,
  workflow, producer lineage, output contract and change family.
- 4/4 incomplete evidence cases are rejected: missing dependency, ambiguous
  output owner, missing output evidence and cyclic producer lineage.
- The real central integration test reuses a qualified profile after an
  unrelated/source commit, preserves evidence ancestry, refreshes exact output
  bytes through authoritative native Gradle when required, and rejects
  structural drift.
- Product failures are zero.

The result authorizes **structurally compatible profile rebinding** inside the
owner-operated POC. It does not authorize stale output reuse: evidence and
output revisions remain separate ancestor/revision bindings, and incomplete or
drifted state runs native Gradle.

No performance replay was run because the block changes compatibility and
correctness, not the already measured candidate wall time. It makes no
production, soak, design-partner, ABI or Test Optimization claim.

```bash
./dev/check-structural-profile-rebinding
```

The machine-readable evidence is [`summary.json`](./summary.json), and the
normative POC boundary is
[`specs/poc-structural-profile-rebinding-v1.md`](../../../specs/poc-structural-profile-rebinding-v1.md).
