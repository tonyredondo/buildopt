# BuildOpt Product Viability v1 evidence

Program: `BUILDOPT-VIABILITY-V1`. Started: 2026-09-08.
Follow the [current execution tracker](../../../docs/plans/buildopt-product-viability-v1-tracker.md)
for gate status and next work. Historical research remains classified by the
[research status register](../../../docs/research-status.md).

The [latest walltime attempt](./bv006-walltime-decision/README.md) closed as
partial/inconclusive: one diagnostic and two screen warmups; the supervisor
aborts the candidate on ESRCH. Zero measured comparable pairs. Profiles 4/2 and
independent history did not start. The frozen 12/profile, 36-total screen retains
its failure, without automatic replacement. Follow the [bounded next prerequisite](./bv006-walltime-decision/next-step.md).
Older “next” links below retain their original campaign context.

| Step | Evidence | Scope |
|---|---|---|
| BV-001 | [Executable inputs and build proof](./bv001/inputs.md) | Actual dirty-source identity, required toolchains, offline native workflow, 100 consecutive transitions, and preparation costs |
| BV-002 | [Native opportunity and negative G1](./bv002/opportunity.md) | Anchor plus all 20 transitions; 20 owner tests; full-task ceiling below both economic floors |
| BV-009 | [Conditional H2 admission](./bv009/h2-admission.md) | Two source-bound input checks, no admitted repair and no additional native starts |
| BV-013 | [Scoped technical decision](./viability-decision.md) | Product evidence remains incomplete; selected seed intervention closed, other gates explicitly unmeasured |
| Checkstyle C1-C4 | [Native capability and residual admission](./checkstyle-admission/README.md) | Native-cache counterexamples reproduced; narrow optimistic G1 admits a prototype; six histories verified |
| Checkstyle BV-003/BV-004 | [Prototype and correctness](./checkstyle-prototype/README.md) | Five-file patch/inverse; 20 cases, 67 observations, complete owner outputs and native removal verified; G2 passes within the frozen scope |
| BV-005 | [Qualified replay instrument](./bv005/README.md) | Strict chronology/state, actual native lifecycle and nested starts, recovery, independent rejection and symmetric fixture overhead; no public value run |
| BV-006 | [Partial owner readiness](./bv006/prefix-report.md) | C5 output/reuse integration verified; fixed 20-start owner overhead fails the 100-ms imbalance gate. Engineering replay and confirmation freeze blocked |
| BV-006 continuation | [Attribution and precision pilot](./bv006-attribution/README.md) | 20 diagnostic and 80 pilot requests verified; variance sizing rejects confirmation despite passing pilot point estimates. Owner readiness/value remain unqualified |
| BV-006 daemon diagnostic | [Supervisor cost and native activity](./bv006-daemon-diagnostic/README.md) | 32 fresh requests verified; full-tree scans reproduce nearly one core of supervisor CPU cost. Native latency cause/value remain unresolved; [observer control completed below](./bv006-disk-observer/README.md) |
| BV-006 disk observer | [Correction and fixed control](./bv006-disk-observer/README.md) | Behavior, 104 non-Gradle fixtures, eight Gradle fixtures and 32 owner builds verified. Static scans use 24% less CPU; complete-request CPU/wall reduction remains unestablished. [Isolation attempt closed below](./bv006-cpu-isolation/README.md) |
| BV-006 CPU isolation | [Successful builds, rejected observation quality](./bv006-cpu-isolation/README.md) | 40 successful Gradle starts; 2,104-ms snapshot gap fails the 500-ms bound. CPU screen blocked at zero starts; diagnose unmeasured sampler phases next |
| BV-006 observer pause | [Bounded replay and diagnostic controls](./bv006-observer-pause/README.md) | Zero Gradle/JVM; four cases distinguish writer/process pauses, but the original event is not reproduced. Cause and CPU screen remain blocked |
| BV-006 live sampler | [Write delays localized in actual builds](./bv006-live-observer/README.md) | Nine non-Gradle cases and two cold builds verified; 323/341 ms writes account for over 98% of 327/344 ms gaps. Old cause and S1/G0 unresolved; CPU screen 0/36. [Bounded writer separation next](./bv006-live-observer/next-step.md) |
| BV-006 buffered writer | [Bounded persistence proof](./bv006-buffered-observer/README.md) | R2/R3 verified in 15 final cases; one fixture-reader failure retained. Controlled 2,104-ms write leaves 100.774-ms sampling gap, all 40 records persisted. Zero Gradle/JVM; S1/G0 and CPU screen blocked. [Real consumer integration next](./bv006-buffered-observer/next-step.md) |
| BV-006 observed replay CLI | [Actual consumer and non-Gradle proof](./bv006-observer-integration/README.md) | R4.1–R4.3 verified: 29 final fixture workflows, C5/legacy/race checks and six release-checker rejections. Zero Gradle/JVM; 81 fixture charges retain five unresolved early launches. S1/G0 and screen 0/36 remain blocked. [Actual-owner cold coverage and cost next](./bv006-observer-integration/next-step.md) |

The [original follow-up proposal](./next-investigation.md) is retained as written
before approval. Its admission and the subsequent bounded correctness proof
are complete. BV-005 now qualifies the chronological replay instrument on
local fixtures. BV-006 integrates and qualifies the frozen owner output rules,
but its initial owner capture gate fails at 588.682498 ms. The continuation
retains that failure, completes attribution and a separate 80-request pilot,
and closes as `INSUFFICIENT_PRECISION_NO_CONFIRMATION`: the sizing rule requires
25,656 pairs per arm, above the registered maximum of 512. The
[next diagnostic](./bv006-attribution/next-step.md) must isolate variation
inside daemon-command intervals and the supervisor's contribution before fresh
owner qualification, engineering replay or confirmation.
There is no
paired chronological saving, adaptive-controller result or product-viability
claim yet. The BV-005 and BV-006 evidence bundles extend the
[prototype evidence index](./checkstyle-prototype-evidence-index.json) without
rewriting earlier decisions.
Negative prerequisite attempts remain included alongside successful checks.
