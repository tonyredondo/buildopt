# Verified request hit POC contract

## Purpose

`VERIFIED_REQUEST_HIT_V1` tests a materially different optimization: satisfy a
repeated exact Gradle request from previously verified outputs without starting
Gradle. This is the first route in the POC that targets the JVM/daemon,
configuration, task-graph and scheduling cost that Gradle's own build cache
still pays on a cache hit.

The hypothesis is not that an irrelevant source change is automatically safe.
It is that BuildOpt may eventually prove a complete request identity, complete
inputs, complete output state and a prior successful result strongly enough to
restore those outputs and return success. Any missing fact runs the unchanged
native command.

## Current evidence and limits

The closed observed-request portfolio contains 113 independently reconstructed
same-command transitions. Of those, 69 are `IRRELEVANT_TO_REQUEST`, 17 expose
an exact partial-graph action and 27 require native fallback because they are
global, ambiguous or input-unavailable. This gives a **theoretical** combined
action ceiling of 86/113 transitions (76.1%), not a safe eligibility result and
not a speedup.

Kafka, Micronaut, OpenTelemetry and Spring each contain at least five potential
whole-request-hit rows; Groovy contains none but has five partial-graph rows.
The preregistered eligibility threshold therefore passes at 4/5 families versus
3/5 required and opens only the safety-contract block. No timing, selection or
activation is authorized by this count.

Eight `testClasses` captures remain typed unavailable. The historical files
preserve only `TASK_INPUT_EVIDENCE_UNAVAILABLE`, so their exact exception cannot
be reconstructed after the fact. The current v2 producer now emits bounded
task paths and exception classes for future failures, never exception messages,
and keeps the same fail-closed status.

## Safety invariants

Before any shadow or real action, one record must prove all of the following:

1. exact length-framed argv and repository-relative working directory;
2. Wrapper, Gradle, JDK and safe environment compatibility;
3. the exact finalized request graph and task implementation/build-logic
   identity;
4. complete transitive repository and external inputs;
5. complete present and absent output states;
6. local state, destroyables, untracked writes and externally visible side
   effects are absent or explicitly covered;
7. every participating task is safely trackable/cacheable under the contract;
8. the source invocation completed successfully; and
9. output materialization and final verification are fail-closed.

`NATIVE_NOOP` remains the name for delegating to Gradle. A successful
`VERIFIED_REQUEST_HIT` is a distinct action and must prove that Gradle was not
started.

## Ordered route

1. **Eligibility audit.** Reconstruct the current portfolio into potential
   request hits, partial-graph actions and native fallbacks. Improve future
   capture diagnostics without reclassifying historical unavailable rows.
2. **Safety contract.** Implement every invariant above plus negative fixtures
   for drift, missing evidence, side effects, external inputs and failed prior
   outcomes.
3. **Shadow replay.** Predict a hit, still run native Gradle, and compare all
   required outputs and outcomes. A mismatch quarantines the identity.
4. **Gradle-free execution.** Restore outputs and return the prior success only
   after repeated exact shadow agreement; prove no Gradle process starts.
5. **Installed paired value.** On the controlled runner, compare the installed
   action with optimized native Gradle using balanced alternating pairs and
   identical outputs. Hosted CI owns correctness, not timing thresholds.
6. **Chronological combined value.** Replay the five public families with the
   frozen policy: irrelevant requests may hit, relevant requests may use the
   qualified partial graph, and global/ambiguous/unavailable requests run
   native. Report signed cumulative wall time; never add percentages from
   different workloads.
7. **Terminal decision.** Continue only with zero correctness failures,
   positive installed and cumulative value, acceptable native-retention
   overhead and stable family breadth. Otherwise stop this hypothesis.

## POC boundaries

- No production soak, HA, tenancy, RBAC or design-partner gate.
- No repository names, task-name allowlists or manual per-repository profiles
  in product logic.
- No timing gate in hosted CI.
- No Test Optimization.
- No reopening retired Runtime Tuning, Hot State, Copy or cache-parameter
  searches.
- Patch Autopilot remains separate future research after this route reaches a
  terminal decision.
