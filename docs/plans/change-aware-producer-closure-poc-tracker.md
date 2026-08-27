# Change-aware producer closure POC tracker

## Status

**Overall:** `HYPOTHESIS_PREREGISTERED`

**Progress:** 1/6 blocks complete

**Current block:** `SWL-CHANGE-001` — complete fresh change and producer evidence

**Predecessor:** the terminal
[`SWL-FRESH`](./fresh-generic-optimization-poc-tracker.md) route remains closed
at 1/5 action breadth. Its timings and action are not imported.

## Objective

Determine whether actual source changes provide the missing generic selection
dimension: BuildOpt should execute only the affected producer closure and
preserve every required unaffected output exactly. The POC continues to value
measurement only if this detector creates safe actions in at least three of
the five public Gradle families.

## Non-goals

- no repository-specific profiles, project names, task names or extensions;
- no Test Optimization behavior;
- no timing before the breadth gate;
- no weakening of byte-exact output equivalence;
- no production soak, design partner, HA, RBAC or multi-tenant work; and
- no reuse of historical BuildOpt state, decisions, profiles or measurements.

## Ordered work

| Order | Block | Deliverable | State | Dependency |
| ---: | --- | --- | --- | --- |
| 0 | `SWL-CHANGE-000` Hypothesis selection | Deterministic candidate analysis, selected hypothesis, machine contract and route | `DONE` | terminal SWL-FRESH evidence |
| 1 | `SWL-CHANGE-001` Complete change-aware evidence | Capture changed paths, complete task/input/output producers and required-output closure from fresh ordinary builds | `TODO` | SWL-CHANGE-000 |
| 2 | `SWL-CHANGE-002` Five-family breadth gate | Independently require complete input and actions in at least 3/5 families | `WAITING` | SWL-CHANGE-001 |
| 3 | `SWL-CHANGE-003` Installed value | Eight balanced pairs for every admitted action, exact outputs and complete cost ledger | `WAITING` | breadth at least 3/5 |
| 4 | `SWL-CHANGE-004` Chronological value | Run at least 15 comparable transitions per admitted family with persistent per-arm state | `WAITING` | installed value passes in at least 3/5 |
| 5 | `SWL-CHANGE-005` Terminal decision | Recompute correctness, breadth, value, confidence, payback, overhead and failures | `WAITING` | conclusive preceding gate |

## Block contracts

### SWL-CHANGE-000 — Hypothesis selection

Independently summarize the terminal fresh evidence and compare materially
different generic candidates. Select only a candidate whose missing inputs can
be captured across all five frozen public histories without repository rules.
The result must explicitly reject timing and activation.

Outcome: `CHANGE_AWARE_PRODUCER_CLOSURE_V1` is selected for implementation.
The fresh package already provides five complete requested graphs, five exact
output contracts and 100 primary plus 50 reserve chronological commits. What
is missing is the change-to-input-to-producer relationship; that is precisely
the new evidence `SWL-CHANGE-001` must generate. This outcome proves
implementability, not action breadth or value.

### SWL-CHANGE-001 — Complete change-aware evidence

Create a Gradle evidence producer that records all reachable tasks, dependency
edges, finalized file inputs, output producers and required-output lineage for
the requested workflow. Pair that evidence with adjacent first-parent Git
changes selected before looking at BuildOpt outcomes.

Every family must return one typed result:

- `TESTABLE_ACTIONS`;
- `NO_SAFE_ACTION`;
- `NOT_APPLICABLE`;
- `INPUT_UNAVAILABLE`; or
- `PRODUCER_FAILED`.

The first three are conclusive. Kotlin and Groovy fixtures must cover affected,
unaffected, ambiguous, global, missing-output and producer-failure cases.

### SWL-CHANGE-002 — Five-family breadth gate

Require conclusive input in all five families. Independently derive affected
producer closure and exact omitted-output closure. At least three families
must expose a complete action. Incomplete evidence blocks; fewer than three
complete actions stops the route before timing.

### SWL-CHANGE-003 — Installed value

Run eight balanced alternating candidate/native pairs per admitted family on
the controlled runner. Count discovery, hashing, restore, wrapper, Gradle,
fallback and validation costs. Require positive mean and conservative lower
bound, non-regressive candidate p95, byte-exact required outputs and zero
product failures in at least three families.

### SWL-CHANGE-004 — Chronological value

Use the preregistered 20 primary and ten reserve commits. Require at least 15
comparable adjacent transitions per admitted family. Preserve every result,
including negative and native-retained transitions, and report cumulative
portfolio value without averaging repository percentages.

### SWL-CHANGE-005 — Terminal decision

Continue only if correctness, breadth, installed value, chronological value,
family confidence, payback, native-retention overhead and zero-failure gates
all pass without threshold movement. Otherwise stop and name the failed
criteria. Unavailable values remain typed unavailable rather than zero.

## Evidence ledger

| Evidence | Block | Required evidence | State |
| --- | --- | --- | --- |
| `SWL-CHANGE-E001` | SWL-CHANGE-000 | Deterministic discovery result, machine contract, tracker and executable checker | `DONE` |
| `SWL-CHANGE-E002` | SWL-CHANGE-001 | Fresh five-family producer/change/closure capture | `TODO` |
| `SWL-CHANGE-E003` | SWL-CHANGE-002 | Independently recomputed breadth decision | `WAITING` |
| `SWL-CHANGE-E004` | SWL-CHANGE-003 | Installed paired evidence and full cost ledger | `WAITING` |
| `SWL-CHANGE-E005` | SWL-CHANGE-004 | Chronological rows, checkpoints and cumulative report | `WAITING` |
| `SWL-CHANGE-E006` | SWL-CHANGE-005 | Independent terminal scorecard | `WAITING` |

## Immediate next action

Implement `SWL-CHANGE-001`. Do not benchmark, activate the Spring action, or
reuse historical profiles while building the evidence producer.
