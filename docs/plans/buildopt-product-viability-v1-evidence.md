# BuildOpt Product Viability v1 Evidence Ledger

Date: 2026-09-08. Parent: [plan](./buildopt-product-viability-v1.md).
Progress: [tracker](./buildopt-product-viability-v1-tracker.md).
Repository-wide disposition: [research status register](../research-status.md).

This ledger records decision-relevant evidence, not a new benchmark run. Read
terminal receipts ahead of older prospective text in the same README. Some
historical documents still describe a then-active phase above its later stop.
Do not edit those records to make this program appear to have passed them.

## E01: Installed chronological profiles did not activate

[Current longitudinal attribution](../../benchmarks/results/current-longitudinal-attribution-v1.json)
records 100 comparable and exact-output pairs across five repository cohorts:
25 positive, 75 negative, zero selected profiles, and zero fragment activations.
The cumulative signed delta is -368.623 s. Recorded BuildOpt cost is 179.029 s;
the remaining -189.593 s is a Gradle/runner residual, not all attributable to
BuildOpt. No mechanism savings were attributed.

The [campaign contract](../../specs/poc-current-longitudinal-campaign-v1.md)
already preserves isolated chronological state and prior-only learning.
The [cohort contract](../../specs/poc-current-longitudinal-cohorts-v1.md)
uses 20 primary observations per family and ordered reserves. This was a real
bounded chronological study, not merely repeated builds at one revision.

**Decision:** Retire the tested profile/fragment product thesis. Reuse the
methodology; do not expect a longer history alone to create an executable action.

## E02: Selected wins did not survive immediate descendants

[Three-class chronological value](../../benchmarks/results/three-class-chronological-value-v1/README.md)
requalified Kafka on 8/8 pairs, then selected its profile on 0/3 immediate
descendants. Kafka's charged net was -58,055 ms; Groovy's was -2,995 ms.
Spring was not run because that experiment's terminal gate was already
impossible. Total consumed net was -61,050 ms with exact outputs.

**Limit:** Three descendants do not describe a year. The negative data and the
unrun subject remain real; the new protocol must not relabel them as long-run
results or copy the same early-value stopping rule into its fixed horizon.

## E03: Source recurrence was not runtime opportunity

[Economic opportunity first](../../benchmarks/results/economic-opportunity-first-v1/README.md)
contains 320 chronological source rows across five families, with no Gradle
starts or timings. Only Kafka met its recurrence threshold: 1/5 versus 3/5.

The [generalization audit](../findings/buildopt-generalization-audit.md)
traces the related producer-closure, request-aligned, portfolio, critical-path,
and history-admitted variants and their terminal boundaries.

**Decision:** No additional source-match search as a value experiment. An
observed native cause and complete workflow economics must precede candidates.

## E04: A second cache has not established incremental value

[Component findings](../findings/build-optimization-performance.md)
report Safe Cache versus native cache at +0.02% for Kotlin and -0.47% for
Groovy. Cache-off wins answer a different comparison.

[Remote locality v3](../../benchmarks/results/remote-cache-locality-value-v3/README.md)
completed 24 balanced pairs across three eligible families; none passed all
value criteria. Groovy missed the relative floor, OpenTelemetry regressed,
and Spring's lower confidence bound crossed zero. Real-path installed
economics were not demonstrated.

**Decision:** Preserve cache safety and existing infrastructure. Stop treating
a generic additional cache as the next acceleration thesis. A future measured
customer locality problem would require its own evidence and scope.

## E05: Runtime tuning and hot state remain retired

[Performance findings](../findings/build-optimization-performance.md)
record the bounded Spring 12-to-6-worker candidate at -191.5 ms / -2.00%,
with only 2/4 favorable pairs. A separate OpenTelemetry hot-state composition
regressed by 892 ms / 7.68%.

**Decision:** No new worker/heap search or hot-state investment in this program.

## E06: Native patches can help, within a selected scope

The [reviewed native patch portfolio](../../benchmarks/results/reviewed-native-patch-portfolio-v1/README.md)
and [performance findings](../findings/build-optimization-performance.md)
retain qualified Micronaut `PythonVfsBytecodeCompile` savings of 6,921.125 ms /
63.44%, and Spring `ArchitectureCheck` savings of 985.5 ms / 35.34%.

[Elasticsearch reviewed patch v2](../../benchmarks/results/economics-gated-reviewed-native-patch-v2/README.md)
has 8/8 positive pairs, 46,139 ms control versus 38,967.25 ms candidate:
7,171.75 ms / 15.54% saving. Its combined 233-build payback is projected from
compatible builds; it is not observed chronological repayment.

**Critical scope limit:** The Elasticsearch value runner invoked
`:server:forbiddenPatterns` and removed its output marker before invocations.
The [installed experiment plan](./installed-elasticsearch-native-correction-v1.md)
documents that distinction. These are selected task/workflow experiments,
not percentages for arbitrary CI pipelines or ordinary evolving workspaces.

**Decision:** Preserve the ability to deliver verified native corrections.
Require a new mechanism, an ordinary workflow, and actual lifecycle accounting.

## E07: More cacheability recipes did not establish breadth

[Durable native optimization](../../benchmarks/results/durable-native-optimization-v1/README.md)
produced eight source candidates but failed a required semantic validation
when a directory input lacked normalization. Later phases were unrun.

[Customer economics replication](../../benchmarks/results/reviewed-native-patch-customer-economics-v1/README.md)
found one actionable family among five. Hibernate's correct cache-restoring
candidate regressed by 190.875 ms / 1.26% over eight pairs. Review/delivery did
not run after its value gate failed.

**Decision:** Do not equate source safety, a cache hit, or another recipe with
a scalable paid product. An annotation-only successor is not H1.

## E08: Previous build-logic searches were narrower than all invalidation

The [generalization audit](../findings/buildopt-generalization-audit.md)
reports four replayable workflows among ten selected workflows, with 19
material tasks, all standard Gradle/Kotlin work. Six incomplete workflows
cannot support no-opportunity conclusions.

A subsequent source scan covered 833 files for 61 material tasks. Its six
explicit cache/state opt-outs belonged to non-material tasks: no proposal
family passed. That rules out the tested explicit-opt-out detector, not every
possible cause of unnecessary native invalidation.

**Decision:** H2 may investigate an observed causal signature. It may not
repeat the broad opt-out inventory or treat missing native history as a defect.

## E09: Installed Elasticsearch delivery closed negatively

The [2026-09-08 terminal finding](../findings/buildopt-elasticsearch-installed-experiment-2026-09-08.md)
records verified C12, M24, and 52 owner methods with twelve nested TestKit
builds. Persistent transport acknowledged 0/20 observations within the unchanged
100-ms deadline; none was present after reopening storage. V/L/H/O were unrun.

**Decision:** Keep `STOP_INSTALLED_PERSISTENT_UPLOAD_DEADLINE` closed. Repairing
storage latency would close an infrastructure defect, not establish product
value. The new native-correction workflow must prove its own delivery contract.

Host-local evidence root: `.tools/state/eic-correctness-v3`.
Recovery locator: `.tools/state/eic-qualified-correction-v2/task-state.json`.
The following pins were rechecked on 2026-09-08:

| Artifact | SHA-256 |
|---|---|
| `experiment-decision.json` | `fe800627b6e91807f927dae8d1f1dacd332e21b89458d0506dda425744fdf524` |
| `qualification/experiment-closeout.json` | `a5a4a9fe2449cefa6823e64da1d17565c5972d548c4571963b17b5282294d87d` |

## E10: A new incremental-work hypothesis has a concrete seed

At Elasticsearch `16bd5bc5355ac7c6ad736f8a6f93281b24a05ab7`, source
`build-tools-internal/src/main/java/org/elasticsearch/gradle/internal/precommit/ForbiddenPatternsTask.java`
has SHA-256
`61fe2eaa06ff463c2a49cea656b147889060855acfa094288ebb8b31567e11b6`.
Its task action loops over every file. Its input getter constructs a new file
collection, which matters for native incremental implementation.

Earlier read-only inspection of retained EIC operation records identified:

| Diagnostic | Task duration | `checkInvalidPatterns` action | Meaning |
|---|---:|---:|---|
| C007, native benign single-file mutation | 6,015 ms | 5,864 ms | Full scanning work after a real changed input in the mutation fixture |
| C008, cacheability correction with the same mutation shape | 8,003 ms | 7,651 ms | A whole-task cache miss still executes the scanning action |

Raw files are `attempts/C007/operations-log.txt` and
`attempts/C008/operations-log.txt` under the E09 root. Task/action operation
IDs are `138884`/`138926` and `138150`/`138339`, respectively. These durations
are retained diagnostic observations; BV-002 must reconstruct their provenance
and obtain its new causal proof. They are not an accepted speedup pair and
cannot promise 6-8 seconds of end-to-end saving.

Gradle provides incremental task actions through `InputChanges`; unavailable
history and changed non-file inputs require full processing. Its documentation
also requires a stable instance for the queried file property. These API facts
support implementation feasibility, not correctness of our proposed patch.
[InputChanges](https://docs.gradle.org/current/dsl/org.gradle.work.InputChanges.html),
[incremental task implementation](https://docs.gradle.org/current/userguide/custom_tasks.html).
Sources checked 2026-09-08; pin documentation to the subject Gradle version
when implementing.

## E11: Product and commercial evidence are still missing

[Develocity MCP documentation](https://docs.gradle.com/develocity/current/integrations/drv-mcp/)
already describes build analysis and input comparison for cache misses.
[Native cache guidance](https://docs.gradle.org/current/userguide/build_cache_performance.html)
describes comparison in CI and remote-cache performance. Official sources
checked 2026-09-08; they establish capabilities, not our competitive win rate.

There is no verified customer sample, paid-pilot conversion, recurring revenue,
retention measurement, or representative build-frequency dataset in the evidence
used for this plan. Prior owner review of a controlled patch is not a paying
customer cohort. Public Git replay can prove technical behavior under a defined
workload; it cannot supply these missing commercial facts.

**Decision:** Keep technical and commercial gates separate. Test whether
delivering and maintaining a verified correction earns payment before building
an extensive hosted product.

## E12: Adaptive policy can be evaluated while it continues learning

The user clarified that the final product should detect when a correction no
longer applies or pays, then investigate a replacement. This is a product
requirement, not a new positive experiment result. E01/E02 still close their
tested optimization mechanisms; they do not establish that all adaptive
management of different native corrections is unviable.

[Progressive validation](https://riverml.xyz/dev/api/evaluate/progressive-val-score/)
evaluates a decision before incorporating its outcome into later learning.
This methodological reference was reviewed on 2026-09-08. It supports the
separation between a frozen learning policy and evolving state; it does not
prove BuildOpt has profitable actions or a correct controller.

**Decision:** Preserve the fixed-correction mechanism test and require BV-011
to compare native, fixed and adaptive strategies. Freeze controller code,
thresholds, allowed transformations and costs before its held-out outcomes;
permit causal learning and replacement during evaluation. No adaptive package,
threshold calibration, N/F/A replay or self-managing value is verified today.

## E13: Checkstyle has residual work, but its raw native cache fails the contract

The owner-approved [C1-C4 admission](../../benchmarks/results/buildopt-product-viability-v1/checkstyle-admission/README.md)
pins Checkstyle 13.11.0, Gradle 9.7.1 and Elasticsearch ordinal 20. Twenty-four
real-engine cases and eight native Gradle fixture requests reproduce missing
XML entries and hidden violations after preserved-mtime source edits or custom
rule bytecode changes. Configuration and suppression invalidation cases are
also retained. Direct enablement of the native engine cache is incompatible.

The complete engineering-prefix analysis has a 29.849-second optimistic
fixed-dependency-model saving ceiling: 1.49245 s per transition and 5.9452%.
It includes all 20 transitions and does not treat overlapping task spans as
workflow saving. Three component probes preserve all 8,338 ordered XML entries,
with byte equality after explicit root mapping. Their component attribution
is not an implementation benchmark or a measured report-reconstruction cost.

**Decision:** Admit a bounded content-aware Checkstyle prototype at G1 only,
with full checking inputs, report semantics and fail-closed native behavior.
Its small margin requires actual correctness, cost and chronological proof.
All six fixed replication histories are verified; their buildability,
opportunities and transfer remain unmeasured. The original ForbiddenPatterns
negative result, 80 unconsumed validation transitions, retired mechanisms and
fixed/adaptive/customer gates remain unchanged.

## E14: Content-aware Checkstyle preserves the frozen owner contract

The [prototype evidence](../../benchmarks/results/buildopt-product-viability-v1/checkstyle-prototype/README.md)
verifies BV-003/BV-004 at engineering ordinal 20 on Linux, Gradle 9.7.1,
Checkstyle 13.11.0 and Temurin 21.0.12+8. Its exact five-file patch retains
native checking and complete ordered XML while reusing content-bound successes.
All 20 registered correctness cases are covered by 67 verified fixture
observations and the actual cancellation/inverse proofs;
the final candidate passes 13 owner methods and the native control passes nine.

Both complete `:server:precommit` workflows pass. The final candidate executes
all 1,045 actionable tasks without build-cache reuse. The 1,301-task root graph
and all 50,225 N / 50,226 I output entries are compared: 50,020 exact entries,
64 bounded manifest-date cases, three Checkstyle root mappings, one RAT mapping,
137 complete native compiler-state comparisons and one exact generated config.
Comparator qualification preserves unknown fields, producer provenance and all
8,338 required Checkstyle file entries. Controlled failures, real cancellation,
cache/state loss, same-daemon operation and exact native restoration also pass.

Two compatibility defects were reproduced and fixed on their first attempt:
missing adapter classes in a rules provider and replaced private state binding.
Earlier receipts retain their actual code versions; changed admission paths and
final owner/inverse proof were rerun. Invalid formatter selections and diagnostic
setup failures remain recorded. This phase charges 121 actual Gradle starts,
including five nested TestKit starts; all are research cost, not measured saving.

**Decision:** G2 passes only for this source/runtime/output boundary. Proceed
to BV-005 instrument qualification and then BV-006 engineering replay. The
optimistic 5.9452% G1 ceiling is still not a measured saving. All 80 validation
transitions, two additional owner replications, adoption/payback/latency,
adaptive management and commercial evidence remain unproved. No retired route,
threshold or historical decision changes.

## E15: The historical replay instrument is qualified on local fixtures

The [BV-005 report](../../benchmarks/results/buildopt-product-viability-v1/bv005/README.md)
records an executable fixed N/I protocol, strict source/runtime/patch/state
bindings and an independent raw-evidence checker. Fifteen integration/native
case families cover chronology, isolation, typed failures, exact restoration,
actual process death, preserved partial costs and rejection of forged results.
Two fresh fixture replications reverse their initial N/I order. The complete
Go build, unit/vet checks and selected unchanged historical contracts pass.

Fifty actual Gradle commands include eight nested TestKit requests. Native
EXECUTED, UP-TO-DATE and FROM-CACHE behavior passes in both arms. A separate
20-command symmetric no-action experiment measures lean-capture median extra
23.645480 ms for N and 12.670142 ms for I; the 10.975338 ms imbalance and
1.486684 ms wrapper p95 pass its frozen small-fixture limits. These are
instrument overhead measurements, not a build-optimization saving estimate.

An unavailable optional cgroup I/O diagnostic and an asynchronous test assertion
were corrected with their failures retained. The latter now waits for both
empty process group and vanished process identity within the original deadline;
five repeated actual driver deaths pass. Expired standalone manifests correctly
refused work; their diagnostic setup failures remain in the evidence. The
current production binary validates, executes, independently checks and refuses
a duplicate completed replay without changing its evidence.

**Decision:** BV-005 passes within Linux AMD64, cgroup v2, user systemd and
fixed N/I fixture scope. Proceed to BV-006 owner output/overhead integration and
paired engineering replay. The full C5 owner-specific comparison rules must be
integrated and requalified before confirmation. G3/G4 value, the adaptive
controller and customer viability remain unproved; all 80 validation transitions
and retired-route decisions remain unchanged.

## E16: Actual owner instrumentation fails readiness before C5 history replay

The [BV-006 report](../../benchmarks/results/buildopt-product-viability-v1/bv006/prefix-report.md)
and [independent reconstruction](../../benchmarks/results/buildopt-product-viability-v1/bv006/owner-overhead-reconstruction.json)
retain all 20 requests from the fixed plain/lean owner experiment. All native
requests succeed. Wrapper p95 is 1.352567 ms, within 10 ms. Median incremental
capture cost is 139.333503 ms for N and 728.016001 ms for I. Their absolute
difference, 588.682498 ms, fails the unchanged 100-ms gate. No samples are
removed or added to obtain a passing result.

The repaired native-end boundary is independently observed within 5.996661 ms
for the fixed target request. Earlier calibration timing is unqualified after
an observed delay of at least 5.234 seconds; its complete outputs remain valid
for output/reuse proof. The frozen v5 reader passes 20 adversarial cases, the
full retained C5 comparison and two native pairs, including 268 causal origins.
Affected unit, integration, race, native lifecycle and full Go build checks pass.

All 16 measured overhead requests have identical outcomes across 1,301 root
tasks, including three `UP-TO-DATE` Checkstyle tasks. Existing records place most
paired variation inside Gradle but do not isolate its cause. An exact-byte
Java sink diagnostic measures only 14–32 ms for per-record file opening,
locking and writing; file-open-only optimization is discarded as the primary
repair. Graph construction, serialization and daemon variability require
attribution before choosing a correction or a new fixed measurement design.

**Decision:** `INSUFFICIENT_READINESS`; BV-006 remains partial. C5 engineering
ordinals 0..20 and all seed validation ordinals remain unrun. G2 stays verified
within its existing boundary; G0 confirmation readiness and G3 value remain
unproved. Requalify owner instrumentation against unchanged limits before
history replay. No retired route is reopened.

## Evidence update rules

Append new evidence IDs and link their immutable result and tested input hashes.
Retain contradictory and negative records. A later success may close a new
scope; it does not rewrite an older failure. Record observed facts, causal
inferences, proposed thresholds, and missing data separately. Raw host-local
artifacts need portable receipts and a verified recovery path before cleanup.
