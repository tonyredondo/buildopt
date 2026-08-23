# Cross-Revision Producer Volatility Portfolio Protocol

## Purpose

This proof-of-concept block tests whether producer volatility learned from
authoritative optimized-native builds can make a later compatible structural
replay both byte-exact and economically useful. The portfolio is a diagnostic
memory of producer task paths, not a cache of historical outputs.

## Frozen public history

The subject is `micronaut-projects/micronaut-core` on public first-parent
history from branch `5.2.x`.

- The qualification revision remains
  `eb60c6c35f355750c6bced793e85c30629d27c4e`.
- The diagnostic learning revision is
  `6a11a05950f36193bd865d6b25c2bc17dfb4ff1c`. It taught the portfolio that
  `:micronaut-jackson-databind:jar` can produce different exact bytes across
  independent native roots. This revision cannot produce a performance claim.
- The later evaluation revision is the already-public first-parent commit
  `1be5f8cefb80b4dbd2b3e086705950ddc72f195f` (`Async Http Client`). It is
  measured only if automatic discovery proves that the current change,
  workflow and output contract remain compatible. Otherwise optimized native
  Gradle is retained and the experiment closes without a speed claim.

No revision may be replaced after timing starts.

## Portfolio contract

Each learning entry must come from two independent optimized-native output
observations with the same revision-bound binding, complete path universe and
producer attribution. The portfolio stores only the volatile producer task
paths and the digest of the authoritative result. It never stores or reuses
historical output bytes or hashes.

The portfolio is bound to digests of the repository scope, Gradle workflow,
Wrapper and owner-reviewed output contract. Drift in any field retains native
Gradle. At the evaluation revision, two new native observations recompute the
complete current output inventory. The effective quarantine is the union of
current volatile producers and compatible portfolio producers. Every other
output must retain its current-revision SHA-256; quarantined outputs must be
rebuilt locally by their attributed producer.

## Value gate

The evaluation uses eight alternating pairs and the existing automatic POC
gate: at least six positive pairs, a strictly positive deterministic saved-time
interval, non-regressive candidate p95, at least 500 ms and 2% mean saving,
exact required outputs and successful native fallback. Qualification,
diagnostic learning, publication and replay economics remain separate and
percentages are not added.

## Boundaries

This protocol adds no repository-name rule to product code, does not normalize
or ignore byte differences, does not authorize production use, does not
require soak or a design partner and does not include Test Optimization.
