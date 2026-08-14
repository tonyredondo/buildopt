# Public installed qualified-profile replay

## Question

Can a user install one public BuildOpt package, place a previously reviewed
qualified profile in a clean Gradle checkout, and retain its exact selection,
output, value, drift, and fallback semantics through the public `buildopt poc`
command?

## Frozen method

The experiment uses the six terminal selective cells from the generic
change-breadth matrix. Each cell is reconstructed from its frozen public Git
revision and deterministic change. The checked terminal capture supplies the
reviewed profile, graph, manifest, generated state, semantic-output contract,
qualification, and expected output digest.

The runner installs `v0.3.1` through its immutable public installer. It does
not compile or execute the checkout's BuildOpt source. The preregistered
`v0.3.0` attempt stopped before Gradle because its launcher accepted the
original three structural bindings while the reviewed profile generator
emitted a fourth output-equivalence binding. No result from that invalid
attempt is retained. `v0.3.1` accepts both shapes and evaluates the optional
fourth file through the same fail-closed SHA-256 path. For every cell it:

1. runs the reviewed profile with `buildopt poc --changes-file ...`;
2. requires the exact selected entrypoint and embedded terminal qualification;
3. recomputes the owner-reviewed semantic output digest;
4. adds JSON whitespace to the digest-bound manifest without changing its
   semantics;
5. requires `NATIVE_FULL_GRAPH / PROFILE_PRECONDITION_FAILED`; and
6. recomputes the full-workflow output digest and requires it to equal the
   selective output from the same replay.

The whitespace probe distinguishes an exact profile-binding drift from an
invalid graph: the documents remain parseable, but the reviewed byte binding
no longer holds, so execution must remain safe and native.

The historical terminal digest is retained as diagnostic context but is not
an acceptance gate. The first corrected-package attempt showed that Groovy's
JAR embeds `BuildDate`: its reviewed contract removes `BuildTime`, so two
same-day qualification captures agree while a later-day rebuild legitimately
has a different cross-capture digest. A direct native Gradle diagnostic on the
same frozen revision produced the same later-day digest as BuildOpt. The
customer-relevant proof is therefore same-replay candidate versus native
fallback equivalence; changing the old timing qualification is outside this
non-timing block.

## Interpretation

This is adoption and correctness evidence, not a new performance experiment.
The terminal per-cell savings remain the only value claim. The experiment does
not average repository percentages, add mechanism effects, authorize automatic
or production activation, require a soak or design partner, or enter Test
Optimization scope.

The machine-readable contract is
[`poc-installed-profile-replay-v1.json`](./poc-installed-profile-replay-v1.json).
