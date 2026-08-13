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
- Apache Kafka root `checkstyleMain`, relocating only the isolated checkout
  prefix in `main.html` and `main.xml`; and
- Apache Kafka root `shadowJar`, comparing canonical ZIP contents.

Each workflow receives two fresh eight-pair captures on one BuildOpt revision.
Eight reciprocal AB/BA blocks are evaluated with the current 500 ms/2%,
positive median/lower-bound, at-least-six-positive-block, non-regressive-p95,
stable-shape, complete-fallback, and zero-product-failure gates. Percentages
remain independent across workflows.

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
