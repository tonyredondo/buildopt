# Generic POC Functional-Coverage Decision v1

## Purpose

This contract converts the preregistered functional-coverage gate into one
terminal decision for the generic, one-command BuildOpt POC. It consumes the
frozen lifetime breadth V3 evidence and does not collect new timings or change
any threshold after observing that evidence.

The decision is deliberately narrower than a product verdict. It answers one
question: has the current generic structural-profile hypothesis demonstrated
repeatable, net-positive wall-time value across ordinary Gradle commit
sequences?

## Frozen gate

Every criterion must pass:

1. all five preregistered repository families are observed;
2. selected and native-retained executions preserve exact declared outputs and
   have zero product failures;
3. selection uses no repository-name, filename or extension rule;
4. every qualified target has at least 6/8 positive pairs, a positive paired
   lower bound and a non-regressive candidate p95;
5. at least three repository families select a compatible descendant and end
   net positive;
6. at least half of structurally eligible non-global descendants select;
7. projected and observed payback occur within five matching builds; and
8. decisions available before Gradle have median wrapper overhead below 500 ms
   and p95 below 1,000 ms.

`CONTINUE_GENERIC_POC` requires all eight criteria. Any failure yields
`STOP_GENERIC_POC`.

## Meaning of stop

`STOP_GENERIC_POC` withdraws the broad claim for the current hypothesis. It
does not delete evidence or imply that every BuildOpt mechanism is useless.
Bounded accelerations, correctness/fail-open controls, safe cache infrastructure
and the Kafka result remain valid within their measured scope. The decision
does not authorize repository-specific product rules, production hardening,
soak work, design-partner work or Test Optimization.

## Verification

```bash
./dev/check-functional-coverage-decision
```

The checker first validates lifetime breadth V3, regenerates the decision from
the pinned evidence, compares it byte-for-byte with the retained result and
then exercises negative decision and threshold-drift cases.
