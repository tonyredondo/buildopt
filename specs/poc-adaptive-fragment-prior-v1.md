# Cross-repository adaptive-fragment hypothesis prior v1

Status: accepted POC ranking contract for `AF-007`.

Machine policy: [`poc-adaptive-fragment-prior-v1.json`](./poc-adaptive-fragment-prior-v1.json).

## Purpose

The prior reduces the order in which BuildOpt explores already-observed
hypotheses in a new repository. It does not transfer a correctness decision, a
value claim or permission to activate anything. Every returned candidate still
requires fresh local correctness and value evidence under the target
repository's exact bindings.

Repository identity is provenance only. The typed input requires every source
scope to differ from the target scope and rejects duplicate evidence from one
source/hypothesis pair. Names, paths, remote URLs and repository hashes do not
enter the structural fingerprint, score or tie-break.

## Generic feature vector

The target and each source observation use the same bounded feature vector:

| Feature | Weight | Reason |
|---|---:|---|
| Task implementation SHA-256 | 35 | Identifies the executable task implementation independently of repository path or name. |
| Plugin-version SHA-256 | 25 | Identifies the producing plugin version without embedding coordinates or a repository rule. |
| Gradle major | 10 | Avoids assuming equivalent behavior across incompatible Gradle generations. |
| Task class | 10 | Separates compile, archive, code-generation and custom task shapes. |
| Graph shape | 10 | Distinguishes single-project, multi-project and included-build topology. |
| Output shape | 10 | Distinguishes single, multiple and aggregate output contracts. |

An exact task-implementation or plugin-version match is mandatory. The other
features refine priority. A query with neither match returns no candidate. The
score is an exploration ordering, not predicted milliseconds or an activation
threshold.

## Evidence and authority

Only source observations that already passed exact-output and local correctness
checks are accepted. A positive observation must also have passed its local
value gate. Product-attributable failure, inconsistent value state, duplicate
evidence, duplicate repository/hypothesis input and target-local evidence fail
closed.

One source repository contributes at most one observation per hypothesis. This
prevents repeated evidence from one checkout dominating the prior. At least two
independent source repositories are required.

The ranked output always carries:

- `localCorrectnessRequired=true`;
- `localValueRequired=true`; and
- `activationAuthorized=false`.

## Checked proof

The deterministic vector uses six source observations from four opaque
repository scopes and ranks three generic hypothesis classes for two holdouts.
Replacing every source and holdout repository identity, then reversing input
order, returns the same fingerprint and ranking. Reversing positive and
non-positive source outcomes changes the top exploration priority while all
authority flags remain false. Six unsafe mutations are rejected and an
unmatched task/plugin query returns zero candidates.

Run:

```bash
./dev/check-adaptive-fragment-prior
```

The checked values are synthetic ranking vectors. They run no Gradle build,
make no timing claim and add no production, soak, design-partner or Test
Optimization scope.
