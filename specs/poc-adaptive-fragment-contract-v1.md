# Adaptive fragment contract v1

Status: accepted POC contract for `AF-001`.

Machine policy: [`poc-adaptive-fragment-contract-v1.json`](./poc-adaptive-fragment-contract-v1.json).

## Purpose

This contract defines the smallest unit that the adaptive BuildOpt POC may
observe, qualify, suspend and eventually compose. It replaces complete
repository profiles as the unit of compatibility. It does not define durable
state, an online learner, an activation planner or a performance claim; those
belong to later adaptive-fragment blocks.

Gradle remains the execution engine and correctness fallback. A fragment is
eligible only when an explicit correctness authority and every binding that its
behavior depends on are current and unambiguous.

## Fragment classes

| Kind | Optimization represented | Required correctness authority | Minimum bindings |
|---|---|---|---|
| `SUBGRAPH` | A complete producer/project subgraph that may be omitted for a bounded workflow and change family. | Gradle's declared model or a reviewed adapter. | Workflow, Wrapper, producer lineage, output contract and change family. |
| `OUTPUT_MATERIALIZATION` | Exact restoration of an output set whose producer is independently verified. | Verified producer contract. | Wrapper, producer lineage and output contract. |
| `TASK_CONTRACT` | A cacheability or execution contract for a task implementation family. | Gradle-native guarantee, reviewed adapter or reviewed patch. | Wrapper, task implementation and output contract. |
| `PATCH` | A reviewed build-logic/source change with a bounded applicability base. | Reviewed patch. | Task implementation, output contract and patch base. |
| `CACHE_LOCALITY` | A bounded choice among already-correct Gradle cache locations. | Gradle-native cache semantics. | Wrapper, network class and cache namespace. |

Observation never creates authority. Repeated identical bytes may prioritize a
candidate, but they cannot create a `TASK_CONTRACT`, `PATCH` or subgraph
omission by themselves.

## Identity

Every fragment has two SHA-256 identities:

- `familyId` identifies the logical repository-scoped opportunity from the
  repository scope digest, fragment kind, canonical selector and authority
  type. It is stable across checkout roots, Git revisions and evidence refresh.
- `revisionId` identifies one exact evidence-bound revision from `familyId`,
  the authority digest, canonical declared bindings, dependencies and
  conflicts. Any change to those values creates a different revision.

Repository scope prevents one repository from activating another repository's
fragment. It is data isolation, not a repository-name product rule. Cross-
repository evidence may rank a structurally similar hypothesis, but local
authority and qualification are still mandatory.

Checkout paths, Git revisions and repository-name conditions are forbidden
identity inputs. Evidence-revision ancestry and output-revision ancestry remain
separate safety records; a stable family identity never makes stale bytes
current.

## Compatibility and selective invalidation

A fragment declares only the semantic bindings it consumes. Compatibility
compares exactly those bindings:

1. repository scope must match;
2. every declared binding must be present, valid and unambiguous; and
3. every declared digest must equal the fragment revision.

Missing or ambiguous relevant state retains native Gradle for that fragment.
Drift suspends that fragment revision. A context change that the fragment did
not declare cannot invalidate it.

Examples:

- change-family drift suspends a `SUBGRAPH` but leaves an independent
  `TASK_CONTRACT` eligible;
- task-implementation drift suspends that `TASK_CONTRACT` but not a subgraph
  that does not depend on it;
- output-contract drift suspends both when both declare the same output
  boundary; and
- platform drift does not suspend either unless the exact fragment declares a
  platform binding.

This is partial invalidation of a portfolio, not permission to activate the
remaining revisions. Activation also needs lifecycle, economics and conflict
checks from later blocks.

## Dependencies and conflicts

`requires` and `conflictsWith` contain sorted fragment family digests. A family
cannot require and conflict with the same family, or reference itself. They are
declarative in this block; `AF-009` owns conflict-aware composition and net-
value selection.

## Lifecycle and evidence inheritance

The lifecycle is:

```text
OBSERVED -> SHADOW -> QUALIFIED -> ACTIVE
    |          |          |          |
    +----------+----------+----------+--> SUSPENDED
                                      \
                                       -> EXPIRED
```

- evidence is inherited only by the same `revisionId` and only while its
  declared bindings remain compatible;
- a changed binding creates a new revision with no inherited qualification;
- evidence from another repository or family is a hypothesis prior only;
- `SUSPENDED` must return through `SHADOW` and requalification before it can be
  `ACTIVE` again; and
- expired evidence cannot authorize activation. Evidence horizons and economic
  decay are defined by `AF-005`, not retrofitted after results are known.

## Executable proof

The Go contract in [`internal/adaptivefragment`](../internal/adaptivefragment)
implements canonical identity, authority validation, declared-binding
compatibility and lifecycle transitions. Its tests prove path independence,
partial invalidation, native retention on missing or ambiguous state, authority
rejection and mandatory requalification.

Run:

```bash
./dev/check-adaptive-fragment-contract
```

The accepted outcome is `FRAGMENT_CONTRACT_ACCEPTED`. It authorizes `AF-002`
to define typed persistence schemas. It does not authorize fragment activation,
customer performance claims, production rollout, soak testing, design-partner
work or Test Optimization behavior.
