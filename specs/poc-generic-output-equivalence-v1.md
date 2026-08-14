# Generic output equivalence

## Purpose

`POC-GENERIC-OUTPUT-EQUIVALENCE-001` tests whether Structural Build Impact can
be evaluated on build-owned workflows whose outputs are semantically stable
but not byte-reproducible. Byte identity remains the default. An exception is
accepted only through a versioned, digest-bound, owner-reviewed contract.

The contract is repository-independent. It selects outputs by relative glob
and supports two narrow modes:

- `REPOSITORY_ROOT_TEXT` replaces only the absolute isolated repository root
  in bounded NUL-free UTF-8 text. All finding content and paths below the root
  remain exact.
- `CANONICAL_ZIP` compares sorted entry names, file modes, directory flags,
  uncompressed sizes, and payload digests. Container order, timestamps,
  compression encoding, comments, and extra fields are excluded. A rule may
  additionally declare exact Java-properties keys in exact entries as
  volatile; all other entry bytes remain bound.

Overlapping, unused, unsafe, malformed, encrypted, repeated-entry, missing-key,
or oversized contracts fail closed. The conformance suite changes undeclared
text, archive payloads, and property values to prove that semantic drift is
still rejected.

## Frozen public rerun

The machine contract freezes the same three public workflows that stopped on
output representation under `E-325`:

- Apache Groovy root `jar`, allowing only `BuildTime` in
  `META-INF/groovy-release-info.properties` to vary;
- Apache Kafka root `checkstyleMain`, retaining exact bytes for `main.html`
  and relocating only the isolated checkout prefix in `main.xml`; and
- Apache Kafka root `shadowJar`, comparing canonical ZIP contents.

Each workflow receives two fresh eight-pair captures on one BuildOpt revision.
Each arm runs one changed-target stabilization followed by two bounded shape
confirmations. The last two target fingerprints must match before pair 1; a
non-convergent arm stops without timing. The earlier target warmup remains in
the diagnostics but cannot veto a later, explicitly demonstrated steady state.
A convergence failure reports a bounded, sorted task-path and terminal-outcome
diff between the final confirmations; this is diagnostic only and cannot relax
or bypass the convergence precondition.
Eight reciprocal AB/BA blocks are evaluated with the current 500 ms/2%,
positive median/lower-bound, at-least-six-positive-block, non-regressive-p95,
stable-shape, complete-fallback, and zero-product-failure gates. Percentages
remain independent across workflows.

Gradle plain console can emit a task line before its terminal outcome. The
generic measurement layer preserves ordered outcome transitions as diagnostics,
uses the last emission as the terminal outcome, and fingerprints terminal task
outcomes only. It does not discard tasks or treat transient console rendering
as execution-shape drift. Malformed transitions still fail closed.

An equivalence contract is intentionally partial: required outputs that match
no rule retain the default `EXACT_BYTES` mode. A relocation rule must still
replace at least one checkout-root occurrence in every file it matches. This
keeps a misplaced or unnecessarily broad exception from silently behaving as
exact comparison.

## Boundaries

This is a review-only POC. The contract creates no automatic activation,
production authority, repository-name branch, Test Optimization behavior,
soak requirement, or design-partner dependency. A workflow that passes
semantic correctness but not wall-time value remains on optimized native
Gradle.

## Conformance

```bash
./dev/check-generic-output-equivalence
```

Terminal evidence is checked separately after both fresh captures exist.
