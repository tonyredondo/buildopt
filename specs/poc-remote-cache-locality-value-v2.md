# Remote Cache Locality Value POC v2

Status: `RCL-001` complete. This block freezes the experiment before any
public build, cache seed or timing sample. `RCL-002` is next.

## Hypothesis

BuildOpt may be viable as a verifying local Edge/L1 for Gradle's native remote
Build Cache. The candidate must reduce complete wall time relative to optimized
native Gradle reading the same immutable cache objects directly from the same
remote origin. It may change only the read location; the requested Gradle
command, graph, task outcomes, cache keys, object bytes and required outputs
must be identical.

Historical locality, transfer and centralized-path results motivate this
route, but cannot satisfy an RCL evidence row. In particular, the v1 shaped-WAN
fixture and results that also reduced the graph are not product-value evidence.

## Frozen cohort and equal opportunity

The adjacent subjects file freezes five public repository families, exact
revisions, canonical workflows and required outputs before inspection. Names
are labels only and cannot affect eligibility or decisions. Every row is fresh.

The control is optimized native Gradle using `HttpBuildCache` directly against
the owner-operated Shared origin. The candidate is the same Gradle invocation
using the same protocol and committed objects through a verifying BuildOpt
Edge/L1 on the runner. Both measured arms are read-only and use independent,
equivalent project and Gradle homes with native local cache disabled. Build
Impact, task selection, source patches, output materialization, Runtime Tuning
and Test Optimization are forbidden.

Before timing, the checker must prove identical requested cache-key sets,
object SHA-256s and lengths, task-outcome sets, graph identity and required
outputs. A candidate measured run must make zero origin reads when classified
as warm. Missing or ambiguous telemetry is inconclusive, never a hit.

## Network and cost boundary

The terminal claim requires one frozen, unshaped owner-operated remote path.
Its origin endpoint, runner identity, route, RTT samples and transfer samples
are recorded before the first pair and cannot be selected after results. A
shaped link may be reported only as diagnostic sensitivity and cannot satisfy
the product gate.

Prewarming is not free and cannot use future state. Cache objects must have
been published by the frozen producer before either consumer arm. RCL reports
retain seed, Edge fill, verification, startup, storage and fallback costs as
signed milliseconds. Warm-pair value is a mechanism result only; product value
requires the later chronological ledger to repay all incremental cost.

## Ordered blocks

- `RCL-001` freezes this contract, cohort, isolation, costs, gates, authority,
  documentation ledger and executable checker without builds or timing.
- `RCL-002` proves the equal-opportunity harness with deterministic fixtures,
  identical key/object manifests, read-only behavior, corruption rejection,
  outage fallback and forbidden-mechanism negatives. It emits no public timing.
- `RCL-003` creates fresh public producer/consumer correctness evidence. All
  five families must be conclusive and at least three must restore sufficient
  identical remote objects and exact outputs before timing is authorized.
- `RCL-004` runs eight balanced warm-locality pairs per eligible family on the
  frozen unshaped path. At least three families must pass every value gate.
- `RCL-005` measures cold fill and at least twenty chronological ordinary
  consumer builds per passing family, retaining all costs, misses and fallbacks.
- `RCL-006` publishes the terminal installed scorecard and explanation.

A failed prerequisite closes dependent blocks as `NOT_AUTHORIZED`. Hosted CI
owns deterministic contracts and correctness, never wall-time gates.

## Gates

RCL-003 requires 5/5 conclusive families and at least 3/5 cache-opportunity
families. Each eligible family must restore at least four objects and 8 MiB.

RCL-004 retains all eight pairs. Qualification requires at least six positive
pairs, at least 500 ms and 2% mean saving, a positive deterministic paired-
bootstrap lower 95% bound, non-regressive p95, exact outputs and zero product
failures. Thresholds never move after evidence.

RCL-005 requires at least twenty ordinary builds per family, at least 3/5 net-
positive families, positive signed portfolio value, finite payback within ten
cache-consuming builds, exact outputs and zero product failures. Every seed,
fill, verification, Edge startup/operation and native-retention cost is included.

## Stop conditions and non-goals

Stop on unequal graphs, keys or bytes; incomplete 5/5 correctness; fewer than
3/5 eligible or warm-positive families; any output mismatch, cache poisoning
or product failure; non-positive installed portfolio; or payback beyond ten
consuming builds. Retain optimized native Gradle.

This POC does not authorize public-source patches, graph reduction, production,
automatic merge, soak, design partners, pricing, SLOs or Test Optimization.

## Verification

```bash
./dev/check-remote-cache-locality-value
```
