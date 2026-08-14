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
3. recomputes the owner-reviewed semantic output digest and requires the
   terminal digest;
4. adds JSON whitespace to the digest-bound manifest without changing its
   semantics;
5. requires `NATIVE_FULL_GRAPH / PROFILE_PRECONDITION_FAILED`; and
6. recomputes the full-workflow output digest and requires the same terminal
   digest.

The whitespace probe distinguishes an exact profile-binding drift from an
invalid graph: the documents remain parseable, but the reviewed byte binding
no longer holds, so execution must remain safe and native.

## Interpretation

This is adoption and correctness evidence, not a new performance experiment.
The terminal per-cell savings remain the only value claim. The experiment does
not average repository percentages, add mechanism effects, authorize automatic
or production activation, require a soak or design partner, or enter Test
Optimization scope.

The machine-readable contract is
[`poc-installed-profile-replay-v1.json`](./poc-installed-profile-replay-v1.json).
