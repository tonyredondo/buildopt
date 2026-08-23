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
  `6a11a05950f36193bd865d6b25c2bc17dfb4ff1c`. A prior diagnostic run at this
  revision exposed volatile `:micronaut-jackson-databind:jar` bytes. That
  observation motivated the experiment but did not preregister an expected
  producer list: the fresh independent pair must learn only what it observes.
  This revision cannot produce a performance claim.
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

Learning does not require a structural candidate or captured transport pack.
Protocol v4 explicitly opts into a diagnostic observation of the native Gradle
output contract before source-ownership selection. The observation is stored
privately, bound by digest and compared with a second independent native root.
This diagnostic path never mutates optimize state, authorizes transport or
claims performance; a later revision must still prove its own complete
materialization, current bytes, correctness and value before any replay.

The portfolio is bound to digests of the canonical repository identity,
Gradle workflow, Wrapper and owner-reviewed output contract. The repository
scope is derived from the canonical repository ID rather than the checkout
path, so two independent roots of the same repository remain comparable.
Wrapper, workflow or output-contract drift remains incompatible. Such drift
produces a structured `NATIVE_RETAINED` result naming every changed binding and
stops the remaining timing pairs because they cannot support a portfolio value
claim. At a compatible evaluation revision, two new native observations
recompute the complete current output inventory. The effective quarantine is
the union of current volatile producers and compatible portfolio producers.
Every other output must retain its current-revision SHA-256; quarantined
outputs must be rebuilt locally by their attributed producer.

## Value gate

The evaluation uses eight alternating pairs and the existing automatic POC
gate: at least six positive pairs, a strictly positive deterministic saved-time
interval, non-regressive candidate p95, at least 500 ms and 2% mean saving,
exact required outputs and successful native fallback. Qualification,
diagnostic learning, publication and replay economics remain separate and
percentages are not added.

## Observed outcome

The fresh learning pair compared 11,187 outputs. Five Kotlin task-state files
differed, so producer-atomic quarantine excluded 476 outputs from five
producers and left 10,711 exact outputs transportable. The later revision kept
the same canonical repository and workflow bindings, but its Wrapper and
output contract changed. BuildOpt returned structured `NATIVE_RETAINED`, named
both drifted bindings and stopped before timing. Its current-revision native
pair independently observed two volatile JAR producers among 186 outputs.

The changing producer sets confirm that one native pair is bounded evidence,
not a universal volatility list. Portfolio safety and path-independent
repository identity are proven; cross-revision replay value is not. The next
block must reject incompatible contexts before paying for the second native
observation, then preregister a fully compatible public revision before any
new timing claim.

## Boundaries

This protocol adds no repository-name rule to product code, does not normalize
or ignore byte differences, does not authorize production use, does not
require soak or a design partner and does not include Test Optimization.
