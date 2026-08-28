# Request-aligned recurrent learning POC contract

Status: `PUBLIC_CAPTURE_COMPLETE`.

## Question

Can BuildOpt discover a safe partial-work action from the Gradle command a
customer actually repeats across commits, instead of pairing arbitrary changes
with one fixed leaf workflow, and then deliver cumulative wall-time value over
optimized native Gradle?

The route now has an observation producer, adjacent-transition classifier and
a fresh five-family public capture. It does not authorize timing, execution of
a candidate or reuse of historical BuildOpt timings and actions.

## Implemented producer boundary

`SWL-REQUEST-001` now materializes a checkout-independent identity over the
exact Gradle arguments, requested tasks, Gradle and Wrapper bindings, portable
JDK facts, relevant environment digest, build logic and finalized task graph.
It also discovers existing outputs from their current unique producer rather
than guessing a version-sensitive filename. The checked Gradle 8.14.3/9.6.1
Kotlin/Groovy matrix reproduces the versioned Groovy JAR and returns typed
`UNAVAILABLE` outcomes for missing, ambiguous and outside-graph producers.

This is observation evidence only. It does not execute a reduced graph,
activate an action or measure wall time.

## Implemented relevance boundary

`SWL-REQUEST-002` consumes two complete observations plus the changed paths
between their revisions. It preserves the exact request and classifies every
transition as one of `RELEVANT_COMPLETE`, `IRRELEVANT_TO_REQUEST`,
`GLOBAL_OR_AMBIGUOUS`, `INPUT_UNAVAILABLE` or `PRODUCER_FAILED`.

Only a relevant and complete transition can describe a partial-work action.
That action is the affected producer closure within the requested graph, while
every omitted required output is bound to its current unique producer and
byte evidence. A complete request graph remains `RELEVANT_COMPLETE` but emits
no invented reduction. Identity drift, ambiguous ownership, missing outputs
and producer failure emit no action and retain the exact native request.

The checked Gradle 8.14.3/9.6.1 Kotlin/Groovy matrix covers eight scenarios per
row (32 total), including a renamed current output. It proves all five status
classes, exact argument preservation and zero timing or activation authority.

## Implemented public-capture boundary

`SWL-REQUEST-003` starts from zero BuildOpt observations and processes the
frozen first-parent histories with the exact preregistered ordinary request.
It keeps every relevant, irrelevant and unsafe transition in the ledger and
stops per family only after five relevant transitions or the fixed
30-transition budget.

The checked capture contains 110 transitions. Groovy reaches five relevant
actions in seven attempts and Spring reaches five in 13. Kafka exhausts 30
transitions as global or ambiguous. Micronaut exhausts 30 with 16 irrelevant
and 14 unsafe outcomes; OpenTelemetry exhausts 30 with 19 irrelevant and 11
global or ambiguous outcomes. Therefore only two of five family inputs are
complete and only those two families expose actions.

This is capture evidence, not the independent breadth decision. The next block
must reconstruct every report from raw captures and apply the frozen gate
without trusting the aggregate summary. No candidate ran and no wall-time
measurement opened.

## Why the predecessor stopped

The terminal change-aware capture contains 25 adjacent transitions. One Spring
transition exposes a complete action. Of the 24 `NO_SAFE_ACTION` results:

- 23 have no changed path consumed by any declared input in the fixed requested
  graph; and
- one Groovy transition reaches `:compileJava`, but the frozen required output
  is `build/libs/groovy-raw.jar` while the current `:jar` task produces
  `build/libs/groovy-raw-6.0.0-SNAPSHOT-raw.jar`.

The first group is workload/change misalignment, not 23 independent
optimization failures. Relabelling those commits as actions would claim work
that optimized native Gradle was never asked to execute. The second group is a
current-output discovery defect. Repairing that single row cannot satisfy the
three-family breadth gate by itself.

## Materially different hypothesis

`REQUEST_ALIGNED_RECURRENT_CLOSURE_V1` observes ordinary invocations of the
committed BuildOpt Wrapper. An exact request identity comprises the Gradle
argument vector and the execution bindings that can affect its graph. Evidence
is accumulated only across repeated compatible request identities.

For each adjacent revision BuildOpt records:

1. the exact customer-requested command without substituting another task;
2. the complete requested task graph and finalized declared inputs;
3. the changed paths between the observed revisions;
4. current output paths discovered from their Gradle producer tasks rather
   than a version-sensitive filename template;
5. direct and transitive output-producer lineage; and
6. byte-exact required-output evidence.

A transition is `RELEVANT_COMPLETE` only when at least one changed path
intersects a declared input in the requested graph and all producer/output
evidence is complete. `IRRELEVANT_TO_REQUEST` is useful compatibility evidence
but cannot count as an action or positive breadth. Global, ambiguous,
unavailable and producer-failed cases retain the original native command.

## Frozen gates

- all five public families must expose complete producer input;
- each family must provide five `RELEVANT_COMPLETE` ordinary transitions;
- at least three of five families must expose a complete action before timing;
- control and candidate must execute the same customer-requested semantics;
- required outputs remain byte exact and product-attributable failures remain
  zero;
- no repository name, task name, path extension or manual profile may select
  an action; and
- hosted CI validates contracts and frozen evidence only. Timing, if opened,
  runs on the controlled runner with balanced native/candidate order.

The exact machine authority is
[`poc-request-aligned-learning-v1.json`](./poc-request-aligned-learning-v1.json).

## Non-goals

- no Test Optimization;
- no command substitution or invented customer workload;
- no claim that a path absent from Gradle's declared inputs is semantically
  safe;
- no cache-only claim against a control with weaker cache opportunity;
- no production hardening, soak or design-partner requirement; and
- no threshold changes after public evidence is observed.
