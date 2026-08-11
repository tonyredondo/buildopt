# Hibernate ORM target-workload warm-up diagnosis

This immutable bundle records the first causal follow-up to the Hibernate v2
7/8 holdout result. It uses BuildOpt revision `dba128a109b90e58eebbd635baf2ac3b976fc48f`
and preserves the same public Hibernate revision, exact `Session.java` change,
root `assemble` workflow, `target/libs` outputs, 12-worker optimized-native
control, Build-Impact-only candidate, eight alternating pairs and unchanged
8/8 qualification gate.

The generic measurement added a second excluded base-revision invocation per
private daemon and recorded bounded task outcomes plus log SHA-256 bindings.
The excluded observations confirmed that v2 had not reached a stable daemon
state:

| Arm | Cache seed | Base daemon stabilization | Reduction |
| --- | ---: | ---: | ---: |
| Native control | 259.140 s | 32.914 s | 226.226 s / 87.30% |
| BuildOpt candidate | 176.163 s | 25.023 s | 151.140 s / 85.80% |

The first measured pair changed from v2's −1.118 seconds to +11.883 seconds,
so the original negative was recoverable and must not be treated as proof that
the candidate is structurally slower. The complete run did not stabilize the
target workload, however:

| Metric | Optimized native | BuildOpt candidate | Result |
| --- | ---: | ---: | --- |
| Mean wall time | 251.652 s | 245.372 s | **6.280 s / 2.50% faster** |
| Positive pairs | — | 4/8 | Retain native under the unchanged gate |
| Paired 95% interval | — | — | −6.604..+20.190 s |
| Required outputs | 3 JARs | Same 3 JARs | Byte-identical in all pairs |

Pairs 1–4 were positive; pairs 5–8 were negative while candidate task outcomes
remained exactly 32 tasks. The control retained 301 tasks through pair 7 and
reported 300 in pair 8. Both arms continued becoming faster across the measured
series because the added stabilization invocation ran the base revision, where
the changed `hibernate-core` compilation could be restored from the frozen
cache seed. It therefore did not warm the expensive target-revision workload.

An exploratory host sample during the negative window showed high Linux IO PSI
(`some` 73.39% over 60 seconds) with negligible CPU and memory pressure. That
sample is diagnostic only and is not part of the qualification result because
it was not captured at each arm boundary.

The full-graph fallback succeeded with byte-identical outputs. Version 3
correctly retains native Gradle and authorizes only a generic measurement
correction: warm the exact target workload from the same frozen seed, bind a
normalized task-outcome fingerprint, and record interval-scoped host PSI before
another complete run. No v2/v3 timing may be reused.

Validate the bundle without network access:

```bash
./dev/check-generic-holdout \
  benchmarks/results/poc-generic-holdout-v3 \
  specs/poc-generic-holdout-v3.json
```
