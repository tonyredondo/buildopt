# Ktor public installed qualified-profile replay

## Question

Can the public BuildOpt package replay all three reviewed Ktor structural
profiles through `buildopt poc`, preserve their exact plans and contemporary
outputs, and retain native Gradle before execution when the invocation's Gradle
options differ from the qualified profile?

## Frozen method

The replay uses the terminal dependency-source, JVM-resource and mixed-source
cells from `POC-NEW-FAMILY-CHANGE-BREADTH-001`. Each clean checkout starts from
the frozen public Ktor revision, reapplies the deterministic reviewed change
and copies only the terminal capture's profile, manifest, graph and generated
state.

The runner installs public `v0.3.2` and does not build or execute the checkout's
BuildOpt source. For every cell it:

1. runs the unchanged profile through `buildopt poc --changes-file ...`;
2. requires the exact candidate entrypoints and embedded historical
   qualification;
3. hashes every required JAR by exact bytes;
4. removes generated state and restores the same reviewed profile documents;
5. repeats the invocation with the complete qualified Gradle option list plus
   `--stacktrace`;
6. requires `NATIVE_FULL_GRAPH / PROFILE_GRADLE_OPTIONS_DRIFT` before Gradle;
   and
7. requires the contemporary candidate and native outputs to match exactly.

Normalized process logs are checked while the capture runs, and their
SHA-256 bindings are retained in each structured result. Raw logs are not
published because they are unnecessary for offline revalidation and may carry
host-specific diagnostic context.

The extra option is intentionally harmless to output bytes. It proves that a
profile cannot reuse a timing qualification under invocation options that were
not measured. The native fallback executes the caller-supplied option list and
the original `jvmJar` workflow.

## Interpretation

This is installed-path adoption and correctness evidence, not a new timing
experiment. The terminal per-cell savings and calibration economics remain
immutable. The block does not average percentages, add mechanism effects,
authorize automatic or production activation, require a soak or design
partner, or enter Test Optimization scope.

The machine-readable contract is
[`poc-new-family-installed-profile-replay-v1.json`](./poc-new-family-installed-profile-replay-v1.json).
