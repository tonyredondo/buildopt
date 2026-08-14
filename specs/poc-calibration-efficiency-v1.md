# Calibration efficiency

## Purpose

`POC-CALIBRATION-EFFICIENCY-001` reduces the time needed to evaluate a
Structural Build Impact candidate without changing the steady-state build
comparison, output-equivalence checks, drift rejection, or native Gradle
fallback. It is a proof-of-concept efficiency study, not an automatic profile
service or a production cache.

The experiment covers the same six qualified source-change cells as
`POC-CALIBRATION-ECONOMICS-001`. Two fresh captures per cell are required.

## Preregistered changes

### One Gradle discovery pass

The proposal must no longer observe the original workflow and then run a
second Gradle discovery for the original-plus-candidate workflow. After the
output contract identifies the candidate entrypoints, one combined discovery
must produce a complete snapshot for every original and candidate entrypoint.
The same snapshot must prove source ownership, unknown-relationship absence,
the reviewable graph, and the generated-state binding.

The output-contract preflight remains a real workflow execution and remains
inside the cold proposal clock.

### Exact proposal replay

Replay is opt-in through an explicit private cache directory. A hit is valid
only when the lookup binds all of the following:

- repository and pipeline identity;
- immutable base and target Git revisions;
- exact changed paths;
- original entrypoints, required outputs, global-change policy, Gradle command,
  Gradle options, timeout, and document paths;
- owner input and output-equivalence content digests;
- Gradle Wrapper script, properties, and JAR digests;
- installed BuildOpt executable digest; and
- the cached output contract, manifest, graph, generated binding, discovery
  snapshot, fallback input, and proposal document digests.

The snapshot must regenerate the cached graph and generated binding byte for
byte before replay. Missing, malformed, symlinked, permission-unsafe, digest-
mismatched, or semantically inconsistent state is a cache miss or a hard
configuration failure; it must never become omission authority. Changed,
ambiguous, global, or drifted inputs retain optimized native Gradle.

Cold proposal time and exact replay time are reported separately. Replay must
not be averaged into first-run calibration cost.

### Bounded adaptive candidate stabilization

The comparative control arm keeps the frozen three target-workload warm-ups.
The candidate arm may stop after two target warm-ups only when both exact task
outcome fingerprints are non-empty and equal. If they differ, a third warm-up
is mandatory and the last two fingerprints must match; otherwise measurement
fails closed. Cache seed and base-daemon stabilization remain mandatory.

The prior three-warm-up captures may be used to verify the stopping rule only
when the first two fingerprints match and the recorded third fingerprint
independently reconfirms the same task shape. The omitted third duration then
reduces calibration cost, but existing terminal measured pairs and savings are
not rewritten.

## Economics

For each of the six cells, the report must preserve the previous cold baseline
and calculate independently:

```text
optimized installed cost = fresh cold proposal mean
                         + prior candidate warm-up mean
                         - verified unnecessary third target warm-up mean

optimized POC cost = fresh cold proposal mean
                   + prior control warm-up mean
                   + prior candidate warm-up mean
                   - verified unnecessary third candidate warm-up mean

break-even builds = ceil(calibration cost / unchanged terminal mean saving)
```

The exact replay mean and replay break-even are a separate repeat-evaluation
view. Percentages, timings, and break-even counts are never averaged across
repositories, workflows, or change shapes.

## Acceptance

Every cell must:

1. produce two successful cold proposals and two exact replays;
2. keep proposal artifacts byte-identical between cold and replay paths;
3. reject at least one digest drift and preserve native fallback probes;
4. prove the adaptive rule against three previously recorded fingerprints;
5. reduce installed-workflow break-even by at least one qualifying build; and
6. retain the immutable terminal saving and correctness evidence.

## Boundaries

This block does not authorize automatic activation, production use,
repository-name rules, public-service SLAs, soak, design-partner operation, or
Test Optimization. It optimizes the cost of validating the idea; it does not
claim that calibration itself makes customer builds faster.

