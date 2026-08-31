# Remote Cache Locality Value POC v3

Status: `RCL3-001/002` complete; `RCL3-003` is current. No v3 timing sample
exists yet.

## Correction from v2

V2 stopped after two families because a broad Kafka output glob included a
native-unstable test-fixtures JAR. V3 preserves that result and starts fresh.
It derives one primary semantic output from each public build definition,
checks native reproducibility separately from BuildOpt parity, and completes
all five correctness rows even when one fails. Names remain labels only.

## Single mechanism

The control is optimized native Gradle reading an immutable HTTP Build Cache
origin directly. The candidate runs the identical command, graph and cache-key
opportunity through the production verifying BuildOpt Edge/L1. Local Gradle
cache is disabled and measured arms are read-only. Build Impact, task
selection, source patches, output materialization, Runtime Tuning and Test
Optimization are forbidden.

## Frozen correctness sequence

Each family runs four fresh isolated builds: native producer A, native producer
B, direct-cache consumer and Edge consumer. The two producers prove the primary
output is natively reproducible. Both consumers must reproduce that digest and
the same task-outcome/key/object manifest. A native mismatch is typed
`NATIVE_OUTPUT_UNSTABLE`; an Edge-only mismatch is a product failure. Neither
can be silently excluded, but neither prevents completing the other families.

Primary outputs and their source bindings are frozen in the adjacent subjects
file. Auxiliary test fixtures, source archives and documentation archives are
excluded only where the public build definition exposes them as separate
classified artifacts. The rule cannot change after execution.

Execution preflight corrected Micronaut's logical project selector from
`:core:jar` to `:micronaut-core:jar`. Its settings include directory `core` but
the public settings plugin standardizes the Gradle project name. The invalid
selector failed before a producer build or evidence row existed; the primary
output, source binding, thresholds and classification rules did not change.

## Network profile

No live customer remote origin is available in the owner lab. V3 therefore
freezes one controlled public-workload envelope before timing: 30 ms per
successful origin GET, 100 MiB/s transfer, zero packet loss, and an unshaped
loopback Edge. This can qualify a controlled mechanism and calculate break-even;
it cannot by itself prove deployment value. A future real-path confirmation is
required before a product-viability claim.

## Blocks and gates

- `RCL3-001`: contract, five semantic outputs, network, cost model and checker.
- `RCL3-002`: production Edge harness, identical manifests, read-only behavior,
  corruption/outage fallback and latency-profile calibration.
- `RCL3-003`: all five four-build correctness rows. Timing requires 5/5 rows
  completed and at least 3/5 exact eligible families.
- `RCL3-004`: eight balanced direct/Edge pairs per eligible family. A family
  passes with 6/8 positive pairs, at least 500 ms and 2% mean saving, positive
  paired-bootstrap lower 95%, non-regressive p95 and zero product failures.
- `RCL3-005`: include seed, Edge fill, verification and operation costs over
  twenty cache-consuming builds per passing family. Product mechanism value
  requires 3/5 net-positive families, positive signed portfolio and payback
  within ten consuming builds.
- `RCL3-006`: terminal scorecard. It distinguishes controlled mechanism value
  from still-unverified real-path product viability.

Hosted CI owns deterministic contracts and correctness, never wall time.
Thresholds never move and failed pairs are retained. V3 authorizes no public
source patch, production, automatic merge, soak, design partner or Test
Optimization.

## Verification

```bash
./dev/check-remote-cache-locality-value-v3
```
