# Sticky-wrapper no-op and observation overhead v1

Status: accepted POC measurement contract for the sticky-wrapper retention
path.

The committed wrapper must not make a native Gradle build pay for BuildOpt
infrastructure that has no consumer. When the repository has no active server
credential and no explicit BuildOpt integration, the launcher takes a
conservative native no-op path:

- it validates the wrapper boundary and committed configuration;
- it does not start a gateway, plugin handshake, managed L1, central cache
  probe or Gradle bootstrap state;
- it scrubs BuildOpt-only variables before starting the repository Wrapper; and
- the no-observation case still uses the lightweight process supervisor so the
  Wrapper remains in an isolated process group and descendant signals retain
  their existing contract; only optional BuildOpt infrastructure is removed.

This is not an optimization decision. It only avoids unnecessary BuildOpt work;
Gradle remains the executor and its own cache/Configuration Cache remain the
source of build reuse. A configured server credential or explicit integration
keeps the established instrumented path. A malformed committed configuration
also keeps that path so its diagnostics are not hidden.

## Observation modes

`BUILDOPT_STICKY_OBSERVATION` controls the optional data-only observer:

| Value | Behavior |
| --- | --- |
| unset, `1`, `light` | Record bounded provenance and timings without spawning Git; the executable digest is computed concurrently when possible and the recorder directory is created only after the child exits. |
| `full` | Include the best-effort Git source revision lookup for diagnostic campaigns. |
| `0` | Do not create observation state; use the lightweight process-supervisor path that preserves the native process-group and signal contract. |

Unknown values retain the existing full launcher path rather than silently
changing the requested diagnostic mode. Observation failures remain diagnostic
only and never change the child result.

The observer does not run speculative Gradle work in parallel with the build.
Only the optional executable digest runs concurrently with Gradle; it is never
required for correctness and a not-yet-ready result remains unavailable. The
post-build append is still synchronous so a completed record is not silently
discarded when the launcher exits.

## Measurement

[`run-sticky-wrapper-noop-overhead`](../dev/run-sticky-wrapper-noop-overhead)
uses a tiny Gradle-Wrapper-shaped process and interleaves direct, native no-op
and light-observation invocations. It warms the executable, records at least
20 samples per mode, reports raw samples plus nearest-rank p50/p95/max, and
stores paired candidate-minus-direct overhead. The light observer's exact
pre-child decision time is reported separately from its post-child recorder
cost; its best-effort digest is not allowed to delay either boundary. This
microbenchmark measures wrapper overhead, not Gradle build speed;
the real Wrapper observation check remains the proof of Gradle equivalence.

The POC guardrails are:

| Measurement | Limit |
| --- | ---: |
| Native no-op p95 overhead over direct | 100 ms |
| Light observation p95 overhead over direct | 250 ms |
| Light observation p95 pre-child decision | 100 ms |

Run the generator with a built binary:

```bash
./dev/run-sticky-wrapper-noop-overhead /absolute/result.json /absolute/buildopt 20
```

Validate the checked result with:

```bash
./dev/check-sticky-wrapper-noop-overhead
```

The checked result is
[`sticky-wrapper-noop-overhead-v1.json`](../benchmarks/results/sticky-wrapper-noop-overhead-v1.json).
It is local POC evidence for retention cost. It makes no claim that the
wrapper improves a repository's Gradle wall time; that claim requires the
paired build-time experiments and exact output checks in the performance
scorecards.
