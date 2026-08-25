# Active Build Impact fragments v1

Status: accepted POC contract for `AF-010`.

Machine policy:
[`poc-adaptive-fragment-activation-v1.json`](./poc-adaptive-fragment-activation-v1.json).

## Purpose

This contract performs the first real activation of the independently
invalidated fragment model. It activates only Build Impact producer pairs: a
`SUBGRAPH` fragment authorizes omission of one unaffected producer and its
required `OUTPUT_MATERIALIZATION` fragment authorizes restoring the exact
producer bytes. Neither fragment is usable alone.

This block proves correctness and execution behavior, not performance. The
predicted values used by the `AF-009` planner are synthetic. `AF-011` owns the
direct timed comparison of composed mechanisms against optimized native
Gradle.

## Independent producer state

Each producer owns two separate compatibility contexts because its structural
omission and materialized bytes consume different facts. A shared global
context would allow one producer's binding to overwrite another's and would
prevent fragment-specific invalidation.

Before Gradle starts, activation requires:

1. a canonical qualified or active `SUBGRAPH` fragment;
2. its exact required `OUTPUT_MATERIALIZATION` family;
3. current, unambiguous bindings for each fragment independently;
4. a stored output revision equal to the current producer output revision;
5. verified content digests for every repository-relative restored file; and
6. an exact positive composition selected by the `AF-009` planner.

An unaffected selected pair restores its output and omits its producer task.
A producer with declared binding drift, an expired/unqualified fragment or
stale bytes is excluded from the candidate set and rebuilt locally while
unrelated producer pairs remain eligible. Fragment families must also be
unique across producers so one planner selection cannot authorize two
execution boundaries accidentally. Selection remains input-order independent
and retains both original correctness authorities.

## Native fallback

The complete original workflow runs when the change is global or ambiguous,
state is missing or unsafe, an output path/digest is invalid, a selected pair
is incomplete, no output restoration remains, or the exact-composition planner
rejects every candidate. No partial execution begins before those checks pass.

Missing storage is not interpreted as a cache miss inside an already selected
plan. The POC returns to native Gradle before execution so stale or incomplete
bytes cannot enter the final artifact.

## Real Gradle proof

[`fixtures/adaptive-fragment-activation`](../fixtures/adaptive-fragment-activation)
contains two independent deterministic producers and one reproducible bundle.
The proof runs an isolated native control and candidate for six scenarios:

| Scenario | Required behavior |
|---|---|
| Unrelated change | Restore both producers; run only finalization. |
| Producer A change | Rebuild A; restore B. |
| Producer B change | Rebuild B; restore A. |
| Build-logic change | Run the complete original workflow. |
| Ambiguous change | Run the complete original workflow. |
| Missing stored output | Run the complete original workflow. |

Every candidate bundle and both producer outputs must match its native control
byte for byte. The proof also checks the producer tasks actually executed, so
an output comparison cannot hide accidental full-graph work.

Run:

```bash
./dev/check-adaptive-fragment-activation
```

The accepted outcome is `COMPOSABLE_BUILD_IMPACT_AVAILABLE`. It authorizes
`AF-011` to measure qualified multi-mechanism compositions directly. It does
not authorize production activation, a timing claim, soak work, design-partner
work or Test Optimization behavior.
