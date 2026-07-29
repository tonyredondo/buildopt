# Walking-skeleton overhead measurement v1

Status: implemented by `WS-009`.

This specification records the first product-overhead observation required by
RFC §29.4 criterion 6. It measures the optimization-off walking skeleton; it
does not claim savings, causality, statistical power, or readiness for
promotion.

## Measurement contract

One neutral process outside both arms owns the clock and required-output
validation:

```text
NEUTRAL_ENVELOPE_HANDOFF
  → execute native Gradle or the complete BuildOpt wrapper
  → observe the process exit
  → validate and hash neutral.properties
EXIT_CODE_AND_REQUIRED_DELIVERABLES_AVAILABLE
```

The envelope uses Go's monotonic clock for the stored duration. UTC timestamps
exist only for audit ordering. Both arms run the real Gradle 9.6.1 Wrapper with
`--no-daemon`, `--offline`, Configuration Cache enabled, the same Gradle user
home, the same copied fixture, and the same required `neutralProbe` output.
The output is removed before every observation.

- `NATIVE` executes the Wrapper without BuildOpt or its init script.
- `WRAPPER` executes `buildopt run --`, the neutral plugin handshake, local
  gateway, server ingest, and `BUILD_SESSION` export with every optimization
  disabled.
- Four pairs alternate `NATIVE → WRAPPER` and `WRAPPER → NATIVE`. The first
  pair is part of the result.
- Every arm must return `0`; every wrapper arm must complete exactly one
  authenticated handshake and server acceptance.
- Native and wrapper deliverable SHA-256 values and sizes must match for every
  pair.

The signed difference is:

```text
productSynchronousOverheadMs =
  wrapper customerVisibleBuildMs - native customerVisibleBuildMs

productSynchronousOverheadRatio =
  productSynchronousOverheadMs / native customerVisibleBuildMs
```

Positive values are regressions. Negative values are retained without
truncation. The report stores raw pairs plus the first-pair value, mean,
nearest-rank p50/p95, minimum, and maximum. These summaries are descriptive:
`promotionGateActive` remains `false`.

## Identity and qualification

The report binds the exact runner contract, METRICS-001 catalog, neutral
envelope, launcher, server, and plugin bytes by SHA-256. It also fixes the
successful local-fixture stratum, workload fingerprint, cache/workspace/daemon
state, and `PRODUCT_TOTAL` effect scope.

`HOST_SMOKE` exercises the same code but is explicitly non-qualifying.
Only `STRICT_GOLDEN_CONTAINER`, after the runner has verified the pinned
Linux AMD64 image and 4-vCPU/16-GiB cgroups, sets
`runnerClassQualified: true`. The checked-in report under
`benchmarks/results/` comes only from that strict path.

## Executable evidence

Run the host smoke:

```bash
./dev/check-walking-skeleton-overhead
```

Run or revalidate contractual evidence:

```bash
./dev/run-golden-lane-container --require-runner-class
```

The checker creates the strict report only when it is absent. Later runs
validate the immutable checked-in report while measuring into a temporary
file, so ordinary validation does not rewrite historical evidence.
