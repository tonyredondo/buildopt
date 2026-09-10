# Collector cost attribution and owner requalification

Status: **partial; owner readiness blocked**, 2026-09-09. This continues BV-006 after its
[sealed readiness failure](../bv006/prefix-report.md). The original 20 samples,
source package and failed 588.682498-ms imbalance gate remain immutable.
No seed validation or adaptive-controller work is part of this phase.

The [fixed diagnostic](attribution.md) is **verified**: all 20 native requests succeeded, all
four build contexts have complete graph/phase coverage, and all 16 measured
root workflows retain the same 1301-task outcomes. Root collector CPU medians are
541.334 ms for N and 552.130 ms for I. The inherited native comparison remains
negative: 2011.844440-ms arm imbalance; wrapper p95 is 1.609903 ms. Native
differences change sign and span -22.697 to +6.052 s. These diagnostic results
do not qualify G0 or establish a causal JIT/GC explanation.

The selected continuation is the registered
[two-stage precision design](precision-design.md). It keeps production capture
and worker bytes unchanged and adds block-interval checks to the unchanged
10/100-ms point gates. The [80-request pilot](pilot-result.md) is now verified:
all native requests succeed, the point imbalance is 39.245578 ms and wrapper
p95 is 1.509480 ms. Its variance requires 25,656 confirmation pairs per arm,
above the registered maximum of 512. The decision is
**INSUFFICIENT_PRECISION_NO_CONFIRMATION**. No confirmation or engineering
request is admitted. The legacy fixture, unit/vet and statistical rejection
checks pass; the actual confirmation branch remains unexecuted.

The [next bounded diagnostic](next-step.md) will need to isolate variation
inside daemon-command intervals and the owned supervisor's contribution.
It is pending; all native units from this attempt are closed. The raw point
estimate alone cannot supply a positive owner qualification.

## Registered diagnostic

Use the same frozen Elasticsearch anchor, native baseline, five-file C5
candidate, offline `:server:precommit` command, JDKs, worker count and private
state model. Run exactly four warmups and four plain/instrumented pairs per
arm, in the original alternating order: 20 native starts. New roots must not
inherit task caches or outputs. Diagnostic capture preserves the complete
primary graph contract and writes its own separate phase counters.

The existing complete reproduction is the failed owner-overhead command in
BV-006. It takes approximately 27 minutes and does not isolate the cause.
A seconds-long, deterministic reproduction of the arm imbalance has **not**
been established. For that reason the diagnostic retains the actual owner
workload; a smaller synthetic fixture only checks the diagnostic's behavior.
Neither replaying the failed arithmetic nor passing that fixture repairs G0.

Ranked hypotheses and discriminating observations:

| Hypothesis | Prediction | Measurement |
|---|---|---|
| Graph construction or JSON serialization dominates | Large direct wall/CPU cost is concentrated before the first task; a supported change to that phase should reduce it | Graph construction, Checkstyle enumeration, dependencies, output enumeration, JSON and UTF-8 encoding |
| Per-task callback work dominates | Summed callback construction, serialization, synchronization and writes account for the extra native duration | Before/after callback totals and nested append/write timers, with counts |
| Daemon warmup/JIT/GC or other Gradle work dominates variation | Direct collector costs are relatively similar between arms while native duration differs outside those phases | Native/daemon boundaries, process CPU, compilation time and GC deltas; unexplained residual remains explicit |

Record monotonic wall time and current-thread CPU time separately. Phase timers
are nested: do not add inclusive totals to their subcomponents. The observer
adds work and may affect scheduling; these results cannot qualify owner timing
or supply primary performance claims, regardless of the inherited harness's
numerical verdict. JIT/GC counters cover the instrumented build interval, not
events before init-script execution. Association does not establish causality.

First run two bounded synthetic Gradle requests: unchanged capture, then
diagnostic capture. Require identical normalized graph/task coverage, valid
phase counts, nonnegative wall/CPU data and a successful native exit. A
preflight failure preserves its exact source and consumes its allocation.

## Decision and subsequent proof

1. Independently bind every diagnostic sample, primary graph, sidecar, native
   receipt, daemon identity and source/state pin. Retain all outcomes.
2. Attribute inclusive collector totals and their nonoverlapping components;
   report the residual rather than assign it to the collector by subtraction
   without supporting evidence. Select at most one supported correction.
3. If variability dominates, freeze one adequately sized qualification design
   before new native starts. Preserve the 10-ms wrapper and 100-ms imbalance
   limits; do not add samples until a point estimate passes.
4. Requalify changed output, failure and lifecycle contracts. A fresh actual
   owner measurement must pass before candidate engineering ordinals 0..20.
   Confirmation sizing/freeze remains dependent on that prefix; ordinals
   21..100 stay untouched.

The initial allocation of eight hours/176 Gradle starts is preserved. Before
precision implementation/native starts, the owner's standing budget instruction
supports one prospective extension to sixteen hours/2400 possible Gradle
reservations, ending 2026-09-09 20:36:54 UTC. The 400 qualification workflow/tool
reservations, 120-GiB new-state cap and 40-GiB free-space floor remain unchanged.
The whole continuation consumed 144 reservations/123 actual starts, including
both retained fixture setup failures. The unused difference is one preflight
reservation and twenty reservations from a fixture launch that failed before
native execution. Combined program totals are 498 reservations/430 actual
Gradle starts, including 29 older nested commands; the older possible metadata
JVM launch remains unresolved. Later work needs a separate prospective
allocation. This stop follows the registered precision rule, not a claim that
C5 or BuildOpt has been proved ineffective.

The task locator is
`/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/buildopt-product-viability-v1/task-state.json`,
under `engineeringPrefix.collectorAttribution`. Raw state and receipts are in
the adjacent `bv006-attribution` directory. Work remains on local BuildOpt
`main` at `b76ded08c952ebb386576fafce4ae2d8fdcc09f1`; existing changes and old
subject worktrees are preserved. No commit, publication or global toolchain
configuration change is part of this phase.
