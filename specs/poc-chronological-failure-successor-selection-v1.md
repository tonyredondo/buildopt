# Chronological Failure Successor Selection v1

`CHRONOLOGICAL_FAILURE_SUCCESSOR_SELECTION_V1` determines whether the terminal
TCCV failures expose one materially different optimization mechanism worth a
new build experiment. It is source-only: checked TCCV captures, public commit
path facts and prior terminal mechanism evidence are inputs; Gradle, candidates,
timing and public-source writes are forbidden.

The audit must distinguish binding drift, changed-producer invalidation,
insufficient online evidence and negative candidate economics. A candidate is
actionable only if the frozen facts show a safe hit that an already tested
mechanism did not cover. Repository names and favorable commit selection may
not influence classification.

Content-addressed producer reuse is not new when it reduces to change-aware
producer closure or adaptive fragment materialization. It also cannot infer
cross-revision ABI or output equivalence. If every proposed hit either
invalidates on changed producer inputs, would omit a changed producer, or
duplicates a terminally tested mechanism, the only valid decision is
`NO_ACTIONABLE_MATERIALLY_DIFFERENT_SUCCESSOR`.

This decision closes experimentation for the current optimizer architecture.
It does not claim that all possible build-optimization products are impossible;
a future route requires a separately authorized architecture or product pivot.
