# Adaptive fragment ordinary-build learner v1

Status: accepted POC online-learning contract for `AF-006`.

Machine policy: [`poc-adaptive-fragment-online-v1.json`](./poc-adaptive-fragment-online-v1.json).

## Purpose

The learner advances fragment evidence only when the owner requested the
underlying build. It cannot create a measurement-only invocation. Every update
is bound to the exact repository scope, context bindings, fragment revision,
comparable cohort and exact required outputs. A product-attributable failure or
binding/output drift rejects the update without changing the prior checkpoint.

This block consumes signed observations; it does not run Gradle, choose a
candidate or grant activation. `QUALIFIED` means that a later planner may
consider the fragment after its independent correctness and value gates. Native
Gradle remains authoritative until that later composition work exists.

## Immutable checkpoint and restart

Each accepted requested build creates a new canonical checkpoint generation.
The checkpoint is RFC 8785 JCS JSON; its external SHA-256 and `supersedes`
digest prevent partial writes or implicit repair. Resume requires all three of:

1. the exact checkpoint digest;
2. the exact repository scope; and
3. the exact context-bindings digest.

Unknown fields, multiple JSON documents, digest mismatch and binding mismatch
fail closed. Because updates return a new value, an interruption leaves the
previous generation byte-for-byte usable.

## Learning transitions

The bounded synthetic state-machine vector uses three fragment families: a
base fragment, one dependent fragment and one unrelated fragment.

| Requested builds | State | Meaning |
|---:|---|---|
| 1 | `OBSERVED` | Evidence exists but is insufficient. |
| 2 | `SHADOW` | Comparable ordinary evidence can continue; no activation. |
| 4 | `QUALIFIED` | Four exact positive candidate-value observations satisfy this POC state gate. |
| 5 | base/dependent `SUSPENDED`; unrelated `QUALIFIED` | Base cumulative value falls to -250 ms. Only it and its transitive dependent suspend; the unrelated fragment remains +200 ms and qualified. |

The four-build threshold exercises lifecycle and restart semantics; it is not a
universal statistical threshold or a performance claim. Later active planning
must still apply the repository's correctness and robust value gates.

## Evidence and boundaries

The checked report accepts five ordinary builds and 15 comparable fragment
samples, with zero measurement-only builds and zero mutation of a prior
checkpoint. It rejects measurement-only input, context drift, cohort drift,
inexact outputs and product failure. Exact resume succeeds; digest, repository,
bindings and unknown-field mutations fail.

Run:

```bash
./dev/check-adaptive-fragment-online
```

The fixture uses synthetic signed values solely to prove state behavior. It
makes no Gradle timing claim, runs no build, activates no fragment and adds no
production, soak, design-partner or Test Optimization scope.
