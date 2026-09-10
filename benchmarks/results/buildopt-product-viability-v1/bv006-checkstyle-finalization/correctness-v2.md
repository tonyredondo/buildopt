# Supported finalization correctness

The candidate uses Gradle's public shared build service lifecycle to publish history after native tasks finish. Each task declares its service usage; preparation records a task-local commit operation. The finalizer requires successful executed work, matching context and request identity, and the exact closed native report digest. It imposes no serial service limit and adds no trailing task action or execution listener.

This is qualified for the pinned Gradle 9.7.1 owner with Configuration Cache disabled. References to runtime task objects have not been qualified for Configuration Cache or other Gradle versions.

| Check | Observed result |
| --- | --- |
| Final source compilation | Pass against pinned Gradle/Checkstyle classpath |
| Native formatter | Pass with strict warnings on all five Java files; history implementation reformatted |
| Cold execution | Native/adapted XML identical; nine independent eligible task histories published |
| Warm execution | Only four actual file checks: two native Main and two fallback inputs; adapted tasks reuse all files |
| Native violation | Both native and adapted tasks fail and adapted history is absent |
| Later task failure | Native task fails; no success is published |
| Changed/deleted report and changed context | All reject success publication |
| No source | Task is `NO-SOURCE`; previous history remains unchanged |
| Worker cancellation | Actual owned unit stopped at worker barrier; exit 143, previous success invalidated |
| Recovery | All eligible histories recover; their report hashes match and intermediate state is cleared |
| Up to date | Native, Main and Second remain `UP-TO-DATE` |
| All six final fixture invocations | `--warning-mode=fail`; no deprecation warning, expected success/failure outcomes |
| Phase observation | Actual Gradle worker option boundary and service-driven Commit/context spans observed by the unchanged agent |
| Portable source delivery | V2 patch and inverse reproduce/remove the exact five postimages against frozen original6 inputs |

The four existing core tests and 25 complete output reader cases remain valid for unchanged engine/state and comparison code. Full owner output equivalence is a separate acceptance step; this document alone does not claim it passed or that walltime improved.

The rejected V1 candidate used a deprecated task listener. Its small diagnostic fixture suppressed warnings for its own listeners and missed Elasticsearch's warning-as-error contract. The actual owner caught it: current build succeeded, V1 failed. Those two starts and all earlier setup failures are retained. The supported fixture removes the deprecated diagnostic listeners and checks state after the process completes.

There are 17 small fixture starts across the entire phase: nine original V1 cases, two preliminary supported cases before formatting, and six final supported lifecycle cases. Two formatter starts are separate. The six final cases run against the final formatted bytecode. An agent reader correction uses the same successful JVM log (`endNs` marks a completed span); it adds no JVM or benchmark run.

Raw evidence: `.tools/state/buildopt-product-viability-v1/bv006-checkstyle-finalization/analysis/correctness-v2.json`, `fixture-runs/supported-final-*`, `formatter-v2`, `receipts/agent-proof-v2-verified.json`, `receipts/portable-patch-v2-proof.json`. Candidate delivery: [supported patch](./candidate-v2.patch), [inverse](./candidate-v2.inverse.patch), [change from current optimization](./current-to-supported.patch). V1 artifacts remain rejected historical evidence.
