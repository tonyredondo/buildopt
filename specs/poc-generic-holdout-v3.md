# Hibernate ORM holdout warm-up causality correction

The corrected v2 holdout completed all eight pairs and showed a positive mean
effect: BuildOpt saved 19.386 seconds (7.80%) with a positive paired 95%
interval and byte-identical required outputs. It did not qualify because pair
one was 1.118 seconds slower, leaving seven rather than eight positive pairs.

That result remains valid and immutable. It is not, however, a causal
explanation. Both arms became materially faster across the retained v2 run:
the first control was 31.470 seconds above the median of later controls and the
first candidate was 50.671 seconds above the median of later candidates. The
measurement implementation used one excluded invocation both to seed the
native cache and to warm a persistent daemon, then discarded the per-arm Gradle
logs. It therefore could not distinguish a product regression from incomplete
daemon/JIT/configuration stabilization.

Version 3 corrects that generic measurement limitation before any new timing:

- the existing cache-seed warm-up remains excluded;
- a second excluded invocation restores the same frozen seed and stabilizes
  each private daemon before pair one;
- every warm-up and measured arm records duration, a SHA-256 binding of its
  Gradle log, and bounded task-outcome counts;
- all eight alternating pairs are rerun from zero against fresh isolated arms;
- the Hibernate revision, mutation, workflow, output glob, Gradle options,
  candidate mechanism, fallback and qualification thresholds remain unchanged.

No v2 warm-up or timing is accepted by v3, and the 8/8 gate is not relaxed. If
the new complete run still contains a non-positive pair, native Gradle remains
the decision. If task shape differs or the diagnostic surface is incomplete,
the measurement fails closed rather than attributing the outlier by guesswork.

This is a POC measurement correction, not repository-specific product logic,
automatic activation, production hardening, Test Optimization, soak testing or
design-partner work.
