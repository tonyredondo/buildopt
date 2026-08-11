# Unseen Hibernate ORM holdout

This directory is the terminal evidence for `POC-GENERIC-HOLDOUT-001` under
the separately preregistered
[`poc-generic-holdout-v2`](../../../specs/poc-generic-holdout-v2.md)
contract. The generic installed `profile propose -> measure -> evaluate` path
ran unchanged on Hibernate ORM revision
`2b448a59d332326f0cd0691c868425124d55cbb5`. No repository-name rule,
post-result tuning, threshold change, failed-observation discard, or
production activation was allowed.

The proposal reduced root `assemble` from 29 projects to the single
`:hibernate-core:assemble` entrypoint for the frozen `Session.java` change.
All eight alternating pairs completed with three byte-identical required JARs
per arm and zero product-attributable failures:

| Metric | Optimized native Gradle | Generic BuildOpt candidate | Difference |
| --- | ---: | ---: | ---: |
| Mean wall time | 248,481.375 ms | 229,095.125 ms | **19,386.25 ms / 7.80% faster** |
| Positive pairs | — | 7/8 | One pair regressed by 1,118 ms |
| Paired 95% interval | — | — | **+9,718.5..+29,210.375 ms** |
| Required outputs | 3 JARs | Same 3 JARs | Byte-identical in every pair |

The frozen gate required 8/8 positive pairs, so the evidence state is
`INCONCLUSIVE` and the evaluator selected `NATIVE_FULL_GRAPH`. The independent
full-graph fallback completed successfully. The favorable mean is useful
holdout evidence for the structural hypothesis, but it does not authorize this
scope or a universal accelerator claim.

The immutable
[`v1 failed attempt`](../poc-generic-holdout-v1-attempt1/README.md) is retained
separately. It stopped before pair one because the owner-declared default
`build/libs` glob did not match Hibernate's repository-defined `target/libs`
directory. No v1 warm-up, proposal, or timing was reused here.

Validate the complete bundle without network access:

```bash
./dev/check-generic-holdout
```
