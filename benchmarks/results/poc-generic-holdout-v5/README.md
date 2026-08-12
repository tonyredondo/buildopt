# Hibernate ORM reciprocal crossover result

This immutable bundle is the terminal value evaluation for the unseen
Hibernate ORM holdout. It uses the version 5 protocol frozen before timing,
BuildOpt revision `00ad49754b276fc9bb5d199cfad0b36e73fb0743`, public Hibernate
revision `2b448a59d332326f0cd0691c868425124d55cbb5`, the root `assemble`
workflow, the exact `Session.java` comment change and the three required
`hibernate-core/target/libs` JARs.

The protocol collected two independent batches of eight alternating pairs.
Each adjacent control-first/candidate-first pair forms one reciprocal block,
so the qualification unit removes the first/second-period bias diagnosed in
version 4. Both arms performed four excluded warm-ups per batch, including two
observations of the exact target workload. Exact task paths and outcomes had
to remain stable before and during timing.

## Result

| Metric | Optimized native Gradle | BuildOpt | Result |
| --- | ---: | ---: | --- |
| Reciprocal-block mean | 216.724 s | 203.991 s | **12.733 s / 5.88% faster** |
| Positive blocks | - | 8/8 | Pass |
| Bootstrap 95% interval | - | - | **+6.808..+19.859 s** |
| Target workload | 300 tasks | 32 tasks | One stable fingerprint per arm |
| Required outputs | 3 JARs | Same 3 JARs | Byte-identical |
| Full-graph fallback | Pass | Pass | Both batches |

All eight reciprocal blocks were positive: +6.429, +8.209, +4.826, +2.080,
+22.272, +32.085, +15.239 and +10.730 seconds. The aggregate clears the
unchanged 500-ms, 2%, positive-lower-bound and eight-of-eight gates. The
decision is `REVIEW_STRUCTURAL_PROFILE`; it is not automatic or production
activation.

The two raw batches remain visible. Batch 1 had one negative raw pair and a
2.56% mean saving, while all four order-adjusted blocks were positive. Batch 2
had eight positive raw pairs and an 8.99% mean saving. This is why the accepted
claim is based on the preregistered crossover aggregation rather than selecting
the more favorable batch.

## Failure investigation retained by the implementation

Two earlier version 5 collection attempts were rejected before acceptance:

1. the first collector surfaced only a generic invalid-diagnostic error after
   the batch, so the runner was changed to stop immediately when structural
   evidence is unavailable;
2. that fail-fast path exposed duplicate identical `:hibernate-core:javadoc`
   console emissions. The collector now normalizes identical repeated task
   records and rejects repeated paths with conflicting outcomes.

The correction is repository-independent and covered by unit tests. It does
not relax task-shape stability, remove an observation or contain a Hibernate
task rule. The accepted bundle contains fresh measurements collected only
after both defects were fixed.

Validate the bundle without network access:

```bash
./dev/check-generic-holdout-crossover \
  benchmarks/results/poc-generic-holdout-v5
```
