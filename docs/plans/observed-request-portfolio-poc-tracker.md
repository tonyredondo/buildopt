# Observed recurrent request portfolio POC tracker

## Status

**Overall:** `ACTIVE`

**Progress:** two of eight blocks complete; exact observed-request portfolio lifecycle is next

**Current block:** `SWL-PORTFOLIO-002`

**Selected hypothesis:** `OBSERVED_RECURRENT_REQUEST_PORTFOLIO_V1`

**Predecessor:** the [request-aligned recurrent route](./request-aligned-learning-poc-tracker.md)
stopped before timing at 2/5 complete/action families. Its terminal result
authorized no successor; the user-authorized independent cause audit selected
this materially different route.

## Objective

Determine whether BuildOpt can learn safe partial-work actions from the
portfolio of exact Gradle commands a customer naturally repeats across commits,
then demonstrate cumulative wall-time value beyond optimized native Gradle in
at least three of five public repository families.

## Cause baseline

[`request-aligned-terminal-cause-audit-v1.json`](../../benchmarks/results/request-aligned-terminal-cause-audit-v1.json)
rebuilds the terminal population from raw capture rather than trusting its
summary:

| Cause | Rows | Request-relevant rows recoverable | Meaning |
| --- | ---: | ---: | --- |
| No changed path intersects request inputs | 44 | 0 | Correct native retention for that exact command |
| Request identity changed | 25 | 0 | All 25 have zero input intersections; precision can improve labels, not opportunity |
| Equivalent output producers | 30 | 2 | Kafka `:clients:jar` depends on `:clients:shadowJar`; both bind the same JAR bytes, and the two recoverable rows also require explicit absence bindings for `NO-SOURCE` outputs |
| Absent omitted output | 1 | 1 | Micronaut exposes one optional non-existent generated output that needs negative-state binding |

Evidence repair would change the action-bearing family count from 2/5 to 4/5,
but the relevant-row counts would remain **5/0/2/1/5**. Only Spring and Groovy
would still meet the five-row completeness requirement. Therefore
`SWL-PORTFOLIO-001` must repair correctness boundaries, while the new portfolio
model—not those repairs—must establish broader opportunity.

## Frozen invariants

1. Preserve the exact customer argv and working directory.
2. Observe only commands the customer actually invokes; never replace a leaf
   command with a broader command to improve benchmark relevance.
3. Keep repository names, task-name rules, path extensions and manual profiles
   out of product decisions.
4. Bind all evidence to portable Wrapper, Gradle, JDK, environment, graph,
   producer and output-state facts.
5. Treat present bytes and proven absence as distinct output states.
6. Retain optimized native Gradle on ambiguity, drift or incomplete evidence.
7. Count every irrelevant command, fallback and product failure.
8. Keep timing closed until the unchanged 5/5 completeness and 3/5 action gate
   passes.

## Ordered work

| Order | Block | Deliverable | State | Dependency |
| ---: | --- | --- | --- | --- |
| 0 | `SWL-PORTFOLIO-000` Terminal cause audit and route selection | Rebuild all 110 rows, quantify recoverability and preregister the materially different hypothesis | `DONE` | terminal request-aligned evidence |
| 1 | `SWL-PORTFOLIO-001` Evidence precision primitives | Dependency-ordered equivalent producer groups, explicit absent-output bindings and split compatibility/request-graph identity | `DONE` | SWL-PORTFOLIO-000 |
| 2 | `SWL-PORTFOLIO-002` Exact observed request portfolio | Persist exact recurrent argv identities, counts and lifecycle without extra builds or command substitution | `TODO` | SWL-PORTFOLIO-001 |
| 3 | `SWL-PORTFOLIO-003` Fresh public portfolio capture | Start empty and capture ordinary observed requests across the frozen five-family cohort | `WAITING` | SWL-PORTFOLIO-002 |
| 4 | `SWL-PORTFOLIO-004` Independent breadth gate | Rebuild every row; require 5/5 complete inputs, five relevant rows per family and exact actions in 3/5 | `WAITING` | SWL-PORTFOLIO-003 |
| 5 | `SWL-PORTFOLIO-005` Installed value | Eight balanced pairs per admitted action with exact outputs and complete BuildOpt costs | `WAITING` | passing breadth only |
| 6 | `SWL-PORTFOLIO-006` Chronological value | At least 15 comparable relevant transitions per admitted family with persistent isolated state | `WAITING` | passing installed value only |
| 7 | `SWL-PORTFOLIO-007` Terminal decision | Continue or stop from correctness, breadth, value, confidence, payback, overhead and failures | `WAITING` | conclusive preceding gate |

## Autonomous block contracts

### SWL-PORTFOLIO-001 — Evidence precision primitives

Change `internal/requestaligned` only through repository-independent evidence:

- represent one output as an equivalent producer group only when every owner is
  inside the requested graph, at least one owner is dependency-reachable from
  another, and every owner reports identical current kind, path and SHA-256;
- reject peers without dependency order, differing hashes, differing kinds,
  owners outside the requested graph, cycles and incomplete output evidence;
- represent an absent output with producer, kind, repository-relative path and
  `exists=false`; revalidate the same absence after a candidate rather than
  treating it as an empty checksum;
- retain strict compatibility identity for Wrapper, Gradle, JDK and safe
  environment facts, but compare finalized requested graph identity separately
  from the repository-wide build-logic inventory; and
- prove that unrelated build-logic drift can become `IRRELEVANT_TO_REQUEST`
  only when the requested task graph is unchanged and no changed path intersects
  its inputs. Wrapper/Gradle drift and graph drift remain global.

Required fixtures: Gradle 8.14.3 and 9.6.1, Kotlin and Groovy, positive
dependency-ordered aliases, positive absent-state binding, and every negative
listed above. Re-run the frozen 110-row audit and require the counterfactual
counts **5/0/2/1/5**. Do not execute an action or measure wall time.

Implemented outcome: [`request-aligned-evidence-precision-v1.json`](../../benchmarks/results/request-aligned-evidence-precision-v1.json)
passes Gradle 8.14.3/9.6.1 × Kotlin/Groovy. The v2 evidence model accepts only
dependency-ordered identical producer aliases, binds present and absent output
states, revalidates absence after the candidate boundary, and separates strict
compatibility, exact requested graph and repository-wide build-logic drift.
All unordered, hash/kind mismatch, outside-graph, cycle, malformed and
post-candidate appearance negatives fail closed. The independently regenerated
historical counterfactual remains **5/0/2/1/5**. No action ran and no wall time
was measured.

### SWL-PORTFOLIO-002 — Exact observed request portfolio

The committed wrapper records only invocations it already received. The record
must bind exact argv, requested tasks, portable compatibility identity,
requested graph identity, repository identity, first/last observation and
frequency. Observation must be asynchronous or piggyback on existing evidence;
it may not start another Gradle build. Failed, interrupted and bypassed commands
remain typed and cannot become candidates.

Selection may rank recurrent identities only by their own observed frequency,
relevance and complete evidence. It may not parse CI YAML as authority, invent
commands or merge different argv vectors. A cache/server outage runs the exact
native command and leaves a bounded local observation for later synchronization.

### SWL-PORTFOLIO-003 — Fresh public portfolio capture

Use one exact BuildOpt executable and empty portfolio/action/timing state. The
benchmark configuration may identify public repositories and their ordinary
commands, but product code receives only actual wrapper invocations. Preserve
every command, transition and typed negative in chronological order. Continue
within the preregistered budget until every family has five relevant transitions
or the budget is exhausted. Do not replace a sparse request with an unobserved
broader command.

### SWL-PORTFOLIO-004 — Independent breadth gate

Ignore aggregate summaries. Verify capture/report digests, regenerate every
classification and output-state binding, and apply the unchanged thresholds:
all five inputs complete, five relevant transitions per family, exact actions
in at least three families and zero product failures. Failure routes to
`SWL-PORTFOLIO-007`; it cannot open timing.

### SWL-PORTFOLIO-005 — Installed value

Run eight balanced alternating candidate/native pairs for each admitted action
on the controlled runner. Both arms receive the same native cache opportunity,
dependency state and exact requested command. Count wrapper, selection,
restore, validation and fallback costs. Require byte-exact required outputs,
positive mean and lower bound, non-regressive p95 and zero product failures in
at least three families.

### SWL-PORTFOLIO-006 — Chronological value

Replay at least 15 comparable relevant transitions per admitted family with
persistent but isolated state per arm. Report signed cumulative wall-time value,
native-retention overhead, payback and selected/fallback counts. Never average
repository percentages or omit negative transitions.

### SWL-PORTFOLIO-007 — Terminal decision

Continue only if correctness, breadth, installed value, chronological value,
positive-family confidence, payback, native-retention overhead and zero-failure
gates pass without moving thresholds. Any unavailable value stays typed rather
than becoming zero. A stopped route authorizes no successor automatically.

## Evidence ledger

| Evidence | Block | Required evidence | State |
| --- | --- | --- | --- |
| `SWL-PORTFOLIO-E001` | SWL-PORTFOLIO-000 | Rebuilt terminal cause audit and selected hypothesis | `DONE` — [`request-aligned-terminal-cause-audit-v1.json`](../../benchmarks/results/request-aligned-terminal-cause-audit-v1.json) |
| `SWL-PORTFOLIO-E002` | SWL-PORTFOLIO-001 | Cross-Gradle/DSL evidence precision fixtures and frozen-ledger counterfactual | `DONE` — [`request-aligned-evidence-precision-v1.json`](../../benchmarks/results/request-aligned-evidence-precision-v1.json) |
| `SWL-PORTFOLIO-E003` | SWL-PORTFOLIO-002 | Exact observed request portfolio lifecycle | `TODO` |
| `SWL-PORTFOLIO-E004` | SWL-PORTFOLIO-003 | Fresh five-family portfolio ledger | `WAITING` |
| `SWL-PORTFOLIO-E005` | SWL-PORTFOLIO-004 | Independent breadth decision | `WAITING` |
| `SWL-PORTFOLIO-E006` | SWL-PORTFOLIO-005 | Installed paired value and cost ledger | `WAITING` |
| `SWL-PORTFOLIO-E007` | SWL-PORTFOLIO-006 | Chronological cumulative value | `WAITING` |
| `SWL-PORTFOLIO-E008` | SWL-PORTFOLIO-007 | Terminal scorecard | `WAITING` |

## Documentation contract

Every block updates this tracker, the machine contract, benchmark index,
validation reference, generalization audit, implementation tracker and POC
one-pager. Architecture and workflow docs change whenever runtime behavior
changes. No document may present a counterfactual action row as measured wall-
time value.
