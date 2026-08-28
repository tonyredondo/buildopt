# Request-aligned recurrent learning POC tracker

## Status

**Overall:** `CLASSIFIER_COMPLETE`

**Progress:** three of eight blocks complete

**Current block:** `SWL-REQUEST-003`

**Predecessor:** the [change-aware producer route](./change-aware-producer-closure-poc-tracker.md)
stopped before timing at 1/5 action breadth. No predecessor timing or action is
imported.

## Objective

Determine whether BuildOpt can learn a generic partial-work action from the
exact Gradle command a customer repeatedly requests across commits, preserve
all required outputs and produce cumulative wall-time value beyond optimized
native Gradle in at least three of five public families.

## Why this route is different

The preregistered hypothesis is `REQUEST_ALIGNED_RECURRENT_CLOSURE_V1`.

The stopped route paired one fixed leaf workflow per repository with the first
five conclusive adjacent changes. Cause reconstruction shows that 23/24
negative transitions changed no declared input in that workflow. The new route
starts from ordinary wrapper requests, keys evidence by exact request identity
and waits for changes relevant to that requested graph. It never changes the
customer command and never counts an irrelevant commit as an optimization.

The remaining negative exposed a different defect: a frozen Groovy filename
did not match the versioned output currently produced by `:jar`. This route
discovers current outputs from producer tasks and binds their exact bytes.

## Non-goals

- no repository-specific paths, project names, task names or profiles;
- no synthetic command substituted for what the customer requested;
- no optimistic ownership for undeclared inputs;
- no timing until public action breadth reaches 3/5;
- no reuse of historical BuildOpt timings, actions or decisions;
- no Test Optimization; and
- no production soak, HA, RBAC, tenancy or design-partner work.

## Frozen invariants

1. The original Gradle argument vector remains authoritative.
2. Request identity includes the Wrapper, JDK, build-logic and relevant
   environment bindings required to compare graphs safely.
3. Only ordinary requested builds produce learning evidence.
4. `IRRELEVANT_TO_REQUEST` does not count toward relevant-transition or action
   breadth.
5. Output paths come from current producer evidence, never a filename guess.
6. Every omitted required output is byte-exact and bound to its producer.
7. Ambiguity, drift, missing evidence or product failure retains optimized
   native Gradle before mutation.
8. Every wrapper, observation, discovery, restore, validation and fallback
   cost counts if timing is later authorized.

## Ordered work

| Order | Block | Deliverable | State | Dependency |
| ---: | --- | --- | --- | --- |
| 0 | `SWL-REQUEST-000` Cause analysis and hypothesis selection | Recompute all predecessor negatives, reject non-material reruns and preregister the new route | `DONE` | terminal SWL-CHANGE evidence |
| 1 | `SWL-REQUEST-001` Request identity and current-output producer | Canonical request identity, current producer-output discovery and typed observation schema | `DONE` | SWL-REQUEST-000 |
| 2 | `SWL-REQUEST-002` Relevance classifier and fixtures | Classify relevant, irrelevant, global, ambiguous, unavailable and failed transitions across Gradle 8/9 and Kotlin/Groovy | `DONE` | SWL-REQUEST-001 |
| 3 | `SWL-REQUEST-003` Fresh public recurrent-request capture | Capture five relevant ordinary transitions in each frozen family without importing historical BuildOpt evidence | `TODO` | SWL-REQUEST-002 |
| 4 | `SWL-REQUEST-004` Independent breadth gate | Rebuild reports and require complete actions in at least 3/5 families | `WAITING` | SWL-REQUEST-003 |
| 5 | `SWL-REQUEST-005` Installed value | Eight balanced pairs per admitted action with exact outputs and complete costs | `WAITING` | breadth passes |
| 6 | `SWL-REQUEST-006` Chronological value | At least 15 comparable relevant transitions per admitted family with cumulative economics | `WAITING` | installed value passes |
| 7 | `SWL-REQUEST-007` Terminal decision | Continue or stop from correctness, breadth, value, confidence, payback, overhead and failures | `WAITING` | all authorized predecessors |

## Block contracts

### SWL-REQUEST-000 — Cause analysis and hypothesis selection

Rebuild every predecessor report from immutable capture evidence. Count causes
without using the aggregate summary. Distinguish declared-input disjointness
from producer-output drift. Compare at least these alternatives:

- relabel missing ownership as unaffected;
- continue sampling arbitrary commits against the fixed workflow;
- repair only the one stale output filename;
- use cache transport as the acceleration claim; and
- align learning to recurrent ordinary customer requests.

Only the final alternative is materially different and capable of producing
fresh evidence without weakening native semantics. This block authorizes the
route and nothing else.

### SWL-REQUEST-001 — Request identity and current-output producer

Define a deterministic request identity from the exact Gradle argv and all
graph-affecting bindings already available to the wrapper. Capture the complete
requested graph after inputs and outputs are finalized. Required outputs must
be selected from current producer tasks and stored as repository-relative path,
kind, existence and SHA-256 evidence.

Fixtures must prove stable identity across checkout paths and distinct identity
for task vector, Wrapper, JDK or build-logic changes. They must also reproduce
the Groovy versioned-JAR case and reject missing, ambiguous and outside-graph
producers. The producer is observation-only: no action, timing or mutation.

Implemented outcome: the canonical identity binds the exact argument and
requested-task vectors, Gradle version, Wrapper, portable JDK facts, safe
environment aggregate, build logic and finalized task graph. The producer
captures repository-relative current outputs and unique ownership across the
Gradle 8.14.3/9.6.1 Kotlin/Groovy matrix. Two checkout roots yield the same
identity; every compatibility mutation changes it. Reproducible archive
settings make the versioned Groovy JAR stable across independent reruns, and
missing, ambiguous and outside-graph ownership fails typed. No timing or
activation occurred.

### SWL-REQUEST-002 — Relevance classifier and fixtures

Classify each adjacent request transition as exactly one of:

- `RELEVANT_COMPLETE`;
- `IRRELEVANT_TO_REQUEST`;
- `GLOBAL_OR_AMBIGUOUS`;
- `INPUT_UNAVAILABLE`; or
- `PRODUCER_FAILED`.

Only `RELEVANT_COMPLETE` can derive an action. The classifier must preserve the
original request, prove at least one changed path intersects a finalized
declared input, construct direct/transitive producer closure and bind every
omitted required output exactly. The fixture matrix covers both DSLs and both
supported Gradle lines with affected, irrelevant, global, ambiguous, renamed
output, missing output and producer-failure cases.

Implemented outcome: all 32 cases pass across Gradle 8.14.3/9.6.1 and
Kotlin/Groovy. Relevant changes derive the exact affected producer closure and
bind the current `build/right-v2.bin` output from the omitted producer;
irrelevant changes emit no action. Identity drift and ambiguous ownership are
global or ambiguous, missing current outputs are unavailable, and producer
failure remains typed. A relevant full-graph request emits no invented action.
Every result preserves exact argv and keeps performance and activation false.

### SWL-REQUEST-003 — Fresh public recurrent-request capture

Start with zero BuildOpt observations. For each frozen family, process ordinary
request identities in chronological first-parent order until five
`RELEVANT_COMPLETE` transitions exist or a preregistered 30-transition budget
is exhausted. Irrelevant transitions remain in the ledger but do not satisfy
the five-transition requirement. No repository-specific selection or hidden
reserve replacement is allowed.

### SWL-REQUEST-004 — Independent breadth gate

Ignore any aggregate summary and regenerate every report from its capture.
Require five complete family inputs, five relevant transitions per family and
at least three families with one exact action. Incomplete relevant evidence
blocks; fewer than three action families stops the route before timing.

### SWL-REQUEST-005 — Installed value

Run eight alternating candidate/native pairs for every admitted family on the
controlled runner. Both arms use the same request semantics, dependency state
and native-cache opportunity. Count all BuildOpt costs. Require byte-exact
outputs, zero product failures, positive mean and conservative lower bound, and
non-regressive candidate p95 in at least three families.

### SWL-REQUEST-006 — Chronological value

Run at least 15 comparable `RELEVANT_COMPLETE` transitions per admitted family
with persistent but isolated per-arm state. Record every negative, native
retention and fallback. Report cumulative signed wall-time value and payback;
never average repository percentages.

### SWL-REQUEST-007 — Terminal decision

Continue only if correctness, breadth, installed value, chronological value,
family confidence, payback, native-retention overhead and zero-failure gates
all pass without threshold movement. Unavailable values remain typed rather
than zero.

## Evidence ledger

| Evidence | Block | Required evidence | State |
| --- | --- | --- | --- |
| `SWL-REQUEST-E001` | SWL-REQUEST-000 | Recomputed cause analysis, rejected alternatives, selected hypothesis and frozen route | `DONE` — [`request-aligned-successor-selection-v1.json`](../../benchmarks/results/request-aligned-successor-selection-v1.json) |
| `SWL-REQUEST-E002` | SWL-REQUEST-001 | Request identity, current-output producer and negative fixtures | `DONE` — [`request-aligned-producer-fixtures-v1.json`](../../benchmarks/results/request-aligned-producer-fixtures-v1.json) |
| `SWL-REQUEST-E003` | SWL-REQUEST-002 | Relevance classifier and Gradle/DSL fixture matrix | `DONE` — [`request-aligned-classifier-fixtures-v1.json`](../../benchmarks/results/request-aligned-classifier-fixtures-v1.json) |
| `SWL-REQUEST-E004` | SWL-REQUEST-003 | Fresh five-family recurrent-request ledger | `TODO` |
| `SWL-REQUEST-E005` | SWL-REQUEST-004 | Independent breadth decision | `WAITING` |
| `SWL-REQUEST-E006` | SWL-REQUEST-005 | Installed paired value and cost ledger | `WAITING` |
| `SWL-REQUEST-E007` | SWL-REQUEST-006 | Chronological cumulative value | `WAITING` |
| `SWL-REQUEST-E008` | SWL-REQUEST-007 | Terminal scorecard | `WAITING` |

## Documentation contract

| Document | Required content |
| --- | --- |
| POC one-pager | Current hypothesis, predecessor cause data, absence of new timings and next block |
| Generalization audit | Why request alignment is generic and what remains unproven |
| Architecture overview and repository map | Wrapper observation, request identity, producer evidence and fail-open boundary |
| Product workflow guide | Explain that ordinary requested commands are preserved; no manual profile is required |
| Benchmark index | Link every checked result and distinguish selection evidence from performance evidence |
| Implementation tracker | Current phase, block states and evidence IDs |

## Immediate next action

Implement `SWL-REQUEST-003` exactly as specified: start from zero BuildOpt
observations and capture chronological ordinary requests until each frozen
public family reaches five `RELEVANT_COMPLETE` transitions or its 30-transition
budget is exhausted. Preserve every irrelevant and unsafe result in the ledger.
Do not time or activate a candidate in this block.

No successor timing is authorized by `SWL-REQUEST-002`.
