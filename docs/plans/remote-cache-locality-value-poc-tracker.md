# Remote Cache Locality Value POC Tracker

**Status:** closed — `STOP_REMOTE_CACHE_LOCALITY_VALUE_POC`<br>
**Current block:** none; all six blocks are resolved<br>
**Terminal outcomes:** `QUALIFY_REMOTE_CACHE_LOCALITY_PRODUCT` or
`STOP_REMOTE_CACHE_LOCALITY_VALUE_POC`

## Objective

Determine whether a verifying local Edge/L1 is a viable BuildOpt product unit
against optimized native Gradle reading the same remote cache directly. Only
cache read locality changes; all incremental costs must repay.

## Frozen decisions

- Prior locality and mixed-mechanism reports motivate design but supply no row.
- Exact five-family revisions, workflows and outputs are frozen before evidence.
- Both arms use the same graph, keys, immutable object bytes and outputs.
- Terminal evidence uses one preregistered unshaped owner-operated remote path.
- Prewarm/fill, verification, operation and fallback costs enter net value.
- A failed prerequisite closes dependent blocks as `NOT_AUTHORIZED`.

## Blocks

| Block | Deliverable | Entry gate | Exit gate | State |
| --- | --- | --- | --- | --- |
| `RCL-001` | Human/machine contract, cohort, costs, gates, checker and documentation ledger | User-authorized successor after EOF stop | Static contracts pass without public builds, cache seed or timing | `DONE` |
| `RCL-002` | Equal-opportunity harness and deterministic fixture proof | RCL-001 | Same graph/key/object/output manifests; read-only, corruption, outage and forbidden-mechanism negatives pass | `DONE` |
| `RCL-003` | Fresh five-family producer/consumer correctness and opportunity report | RCL-002 | 5/5 conclusive; at least 3/5 restore four objects and 8 MiB with exact outputs | `DONE_STOP` — 2/5 conclusive, 0/5 eligible, one required-output mismatch |
| `RCL-004` | Eight balanced warm-locality pairs per eligible family | RCL-003 | At least 3/5 pass 6/8 positive, 500 ms, 2%, positive lower bound, p95 and correctness gates | `NOT_AUTHORIZED` |
| `RCL-005` | Cold-fill and twenty-build installed economic ledger | RCL-004 | At least 3/5 net positive; portfolio positive; payback within ten consuming builds | `NOT_AUTHORIZED` |
| `RCL-006` | Installed explanation and terminal scorecard | RCL-005 or first failed prerequisite | Truthful terminal decision with unavailable fields preserved | `DONE` — terminal stop only |

## RCL-002 required proof

The harness must independently reconstruct cache-key/object manifests rather
than trust summary counts. It rejects graph drift, task-outcome drift, object
digest or length drift, measured writes, candidate warm origin reads, missing
telemetry, source drift and any enabled forbidden mechanism. Corrupt Edge bytes
must become a safe miss or native rebuild; origin outage must preserve exact
native output. This fixture block creates no public or wall-time claim.

## Measurement and authority

`RCL-001/002` authorize no public build or timing. `RCL-003` authorizes only
untimed public producer/consumer correctness after the harness passes.
`RCL-004` is the first timing block. Hosted CI validates deterministic
contracts and correctness only. No block patches public source or authorizes
production, automatic merge, soak, design partners or Test Optimization.

## Terminal result

Groovy restores 39 of 45 objects and reproduces its required JAR exactly, but
the committed object set is 6,904,026 bytes, below the frozen 8-MiB gate. Kafka
exposes 60 objects and 170,907,353 bytes and restores 58 tasks, but its required
`test-fixtures.jar` differs between producer and consumer while the main client
JAR remains identical. The frozen output glob cannot be narrowed after seeing
the result. The stop condition closes the remaining three rows as
`NOT_RUN_GATE_CLOSED`, leaves `RCL-004/005` unauthorized and records no timing
or speedup. This does not disprove cache locality; it rejects this frozen cohort
and output contract as a valid product-value experiment.

## Documentation ledger

Every block updates this tracker, human/machine contracts, specification and
benchmark indexes, validation reference, implementation tracker/evidence
ledger, generalization audit, performance findings and POC one-pager. Runtime
changes additionally update architecture, onboarding and troubleshooting.
