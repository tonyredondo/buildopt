# Change-aware producer closure POC tracker

## Status

**Overall:** `STOPPED_CURRENT_CHANGE_AWARE_DETECTOR`

**Progress:** four blocks executed and two downstream timing blocks were not
authorized

**Current block:** none; `SWL-CHANGE-005` issued the terminal decision

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
| 1 | `SWL-CHANGE-001` Complete change-aware evidence | Capture changed paths, complete task/input/output producers and required-output closure from fresh ordinary builds | `DONE` | SWL-CHANGE-000 |
| 2 | `SWL-CHANGE-002` Five-family breadth gate | Independently require complete input and actions in at least 3/5 families | `DONE` | SWL-CHANGE-001 |
| 3 | `SWL-CHANGE-003` Installed value | Eight balanced pairs for every admitted action, exact outputs and complete cost ledger | `NOT AUTHORIZED` | breadth failed at 1/5 |
| 4 | `SWL-CHANGE-004` Chronological value | Run at least 15 comparable transitions per admitted family with persistent per-arm state | `NOT AUTHORIZED` | installed value did not open |
| 5 | `SWL-CHANGE-005` Terminal decision | Recompute correctness, breadth, value, confidence, payback, overhead and failures | `DONE` | conclusive breadth gate |

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

Outcome: the producer and analyzer completed five adjacent first-parent
transitions in every frozen family. All 25 transitions are conclusive and all
five family inputs are complete. Spring exposes one exact testable action;
the other four Spring transitions and all 20 transitions in OpenTelemetry,
Kafka, Micronaut and Groovy return `NO_SAFE_ACTION`. The synthetic matrix also
covers all six required cases on Gradle 8.14.3 and 9.6.1 with Kotlin and
Groovy DSL. No wall time was measured and no action was activated. This block
establishes complete evidence, not passage or failure of the independent
breadth gate.

### SWL-CHANGE-002 — Five-family breadth gate

Require conclusive input in all five families. Independently derive affected
producer closure and exact omitted-output closure. At least three families
must expose a complete action. Incomplete evidence blocks; fewer than three
complete actions stops the route before timing.

Outcome: [`change-aware-breadth-gate-v1.json`](../../benchmarks/results/change-aware-breadth-gate-v1.json)
does not consume `summary.json`. It rebuilds all 25 reports from their captures,
checks every report and ledger digest, validates exact omitted-output bindings
and then recounts breadth. All five inputs are complete, but Spring is the only
family exposing a complete action: **1/5** versus the frozen **3/5** threshold.
The gate therefore returns `STOP_INSUFFICIENT_CHANGE_AWARE_BREADTH`, keeps
timing and activation false, and routes directly to the terminal decision.

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

Outcome: [`change-aware-terminal-decision-v1.json`](../../benchmarks/results/change-aware-terminal-decision-v1.json)
binds the frozen route, capture ledger and breadth result by SHA-256. Fresh
input completeness, 25/25 report reconstruction, the sole static exact-output
binding and zero capture/analyzer failures pass. Public action breadth fails at
**1/5** versus **3/5**. Installed value, chronological value, family
confidence, payback and native-retention overhead remain
`NOT_MEASURED_NOT_AUTHORIZED`; no timing row or action execution exists. The
terminal decision is
`STOP_CHANGE_AWARE_PRODUCER_CLOSURE_POC_FOR_CURRENT_DETECTOR`. It rejects this
detector as the generic value route, not every possible generic Gradle
optimization.

## Evidence ledger

| Evidence | Block | Required evidence | State |
| --- | --- | --- | --- |
| `SWL-CHANGE-E001` | SWL-CHANGE-000 | Deterministic discovery result, machine contract, tracker and executable checker | `DONE` |
| `SWL-CHANGE-E002` | SWL-CHANGE-001 | Fresh five-family producer/change/closure capture | `DONE` |
| `SWL-CHANGE-E003` | SWL-CHANGE-002 | Independently recomputed breadth decision | `DONE` — [`change-aware-breadth-gate-v1.json`](../../benchmarks/results/change-aware-breadth-gate-v1.json) |
| `SWL-CHANGE-E004` | SWL-CHANGE-003 | Installed paired evidence and full cost ledger | `NOT AUTHORIZED` — breadth failed before timing |
| `SWL-CHANGE-E005` | SWL-CHANGE-004 | Chronological rows, checkpoints and cumulative report | `NOT AUTHORIZED` — installed value did not open |
| `SWL-CHANGE-E006` | SWL-CHANGE-005 | Independent terminal scorecard | `DONE` — [`change-aware-terminal-decision-v1.json`](../../benchmarks/results/change-aware-terminal-decision-v1.json) |

## Immediate next action

No successor is authorized by this terminal result. Before any additional
timing, analyze the 24 `NO_SAFE_ACTION` causes and select a materially different
repository-independent hypothesis with capturable evidence and an unchanged
pre-timing breadth gate. Do not rerun this detector, benchmark the sole Spring
action or convert missing economics into zero.
