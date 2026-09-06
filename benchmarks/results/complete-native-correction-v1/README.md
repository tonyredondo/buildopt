# Complete native correction v1 evidence

Current boundary: `CNC-003` native harness verified locally; `CNC-004` next,
with no public execution.

The [plan](../../../docs/plans/complete-native-correction-poc.md) and
[tracker](../../../docs/plans/complete-native-correction-poc-tracker.md) define
the proposed experiment. The [source feasibility report](./selection/feasibility.md)
maps all five retained GraphQL Java blockers to exact source and consumers.

The CNC-001 advancement decision was `INCOMPLETE_EXPERIMENT_INPUT`: the owner-CI
workflow and the earlier local command exercise different version semantics.
A local correction is plausible, including a supported Bnd configuration fix,
but choosing that case is not proof of the owner-CI workflow. The report states
the scope decision needed before CNC-002. The owner has now accepted the local
non-CI scope. The [CNC-002 review](./contract/local-scope-and-budget-review.md)
selects exact Corretto package metadata and retains the original insufficient
fixture allocation. The owner subsequently approved its 60-start replacement.
The owner approved a two-hour execution ceiling with a review at 30 minutes,
including preparation, downloads and validation. No execution clock has
started. The [human contract](../../../specs/poc-complete-native-correction-v1.md),
[machine contract](../../../specs/poc-complete-native-correction-v1.json),
[subject manifest](../../../specs/poc-complete-native-correction-v1.subjects.json)
and `./dev/check-complete-native-correction contract` now define and independently
check the static protocol. CNC-003 now provides a local
[source/executable package snapshot](./contract/capture-package.json), generic
fake-child tests, read-only attempt reconstruction and a
[runbook](../../../docs/reference/complete-native-correction-capture.md).
Explicit `host-fixtures` prove detached-child ownership through transient user
cgroups, cancellation/expiry/client-loss cleanup and a complete fake P01-M02
consuming sequence. Generic/race tests also prove private Maven/home bindings,
fresh output state, exact output/producer reconstruction and malformed/drifted
evidence refusals. This closes the earlier `E-551` harness gaps, recorded in
`E-552`. The snapshot is still local/uncommitted, not real Gradle/runtime proof
or publication. Native capture requires committed package bytes, verified real
inputs, active ownership and all frozen order/budget gates. CNC-004 owns the
first fresh native execution; later recipe/candidate/value gates stay closed.

This directory contains no CNC diagnostic capture, patch,
candidate build, timing row, speedup, or product-qualification result. Historical
WCNCP reports are explicitly historical diagnostic inputs to source analysis;
they do not count as fresh CNC correctness, materiality, or value evidence.

Validation includes the static contract/negative suite, race checks, Go vet,
the separate host gate, ShellCheck, layout, tracker consistency, documentation and Base CI static
integration. Retained exact-commit bytes rehash to seven subject files and eight
spans; the Wrapper script, full archive and runtime binaries were not newly
verified. Independent native attempt/output checking exists, but no public CNC evidence has been
captured or checked. Nothing in this
evidence index authorizes a Gradle start or a substitute subject.
