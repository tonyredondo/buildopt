# Observed recurrent request portfolio POC contract

## Purpose

`OBSERVED_RECURRENT_REQUEST_PORTFOLIO_V1` tests whether BuildOpt can learn a
generic optimization from the exact distribution of Gradle commands a customer
actually repeats. It does not replace those commands with `build`, `check` or a
repository-specific task chosen because it benchmarks well.

The predecessor evaluated one fixed request in each public repository. Its
terminal cause audit reconstructs all 110 transitions and shows that evidence
repairs alone would yield relevant action rows of **5/0/2/1/5** for Spring,
OpenTelemetry, Kafka, Micronaut and Groovy. Only two families still meet the
five-relevant-transition input threshold. The route therefore needs a different
opportunity unit, not a weaker correctness gate.

## Invariants

1. Every request is an exact argv vector observed through the committed
   wrapper. BuildOpt may not substitute, widen or synthesize the command.
2. Product logic may not use repository names, task-name allowlists, path
   extensions or manual profiles.
3. Compatibility identity and request-graph identity are separate. Wrapper,
   Gradle, JDK or environment drift retains native; unrelated build-logic drift
   may only improve classification when the finalized requested graph remains
   exact.
4. Multiple producers may share one output only when they form a dependency-
   ordered group and the captured state proves the same kind, path and bytes.
   Otherwise ownership remains ambiguous.
5. A missing optional output is evidence, not an empty success. Its producer,
   kind, path and absent state must be bound and absence must be revalidated.
6. Irrelevant commits and native fallbacks remain in the ledger and never count
   as actions or savings.
7. No wall-time measurement opens until all five family inputs are complete,
   every family has five relevant transitions and at least three families
   expose an exact action.
8. Hosted CI validates contracts and correctness only. It never owns timing
   thresholds.

## Ordered proof

`SWL-PORTFOLIO-001` has implemented the three evidence-precision primitives
and tested their negative boundaries across Gradle 8/9 × Kotlin/Groovy. Its
checked result preserves the counterfactual at **5/0/2/1/5**, executes no
action and measures no wall time. `SWL-PORTFOLIO-002` adds a durable, private
and bounded portfolio of exact observed requests after the customer build
exits. It hashes argument boundaries, binds optional finalized evidence to the
same invocation, preserves typed incomplete/failed/cancelled/bypassed outcomes,
serializes concurrent updates and remains local when the server is unavailable.
It starts no Gradle build and grants no action authority. `SWL-PORTFOLIO-003`
then starts from zero portfolio state and captures the frozen five-family
cohort with one executable. Same-invocation evidence is collected from the
finalized task graph and completed tasks without an extra Gradle invocation.
Unavailable task inputs are typed separately from repository, Wrapper,
build-logic, graph and output unavailability; an incomplete typed capture is
not a product failure. Canonical JSON evidence is stored as deterministic gzip,
while content digests continue to address the uncompressed bytes.

The checked capture contains 128 observations and 113 transitions. Spring,
Kafka and Groovy are complete; Micronaut has two relevant/action transitions
and OpenTelemetry has none. The result is **3/5 complete families** and **4/5
families with at least one action**, with zero product or build failures.
Performance, selection and activation remain false. `SWL-PORTFOLIO-004` must
independently rebuild every report and apply the unchanged 5/5 completeness and
3/5 action-breadth gate; the capture summary itself cannot make that decision.

Only a passing breadth gate can open installed timing in
`SWL-PORTFOLIO-005`. Only passing installed value can open chronological value
in `SWL-PORTFOLIO-006`. `SWL-PORTFOLIO-007` issues the terminal continue/stop
decision. A failed gate routes directly to the terminal decision without
inventing zero-valued economics.

## Non-goals

- production soak, HA, tenancy, RBAC or design-partner qualification;
- Test Optimization;
- parsing a repository's CI files as product policy;
- executing extra customer builds solely to learn;
- importing historical BuildOpt timing, actions or decisions; or
- claiming that evidence repair itself improves wall time.
