# Current longitudinal campaign v1

Status: accepted POC measurement contract for `AF-014C`.

## Purpose

This protocol measures the current installed BuildOpt package over the five
public first-parent cohorts frozen by `AF-014B`. It compares the exact requested
workflow against optimized native Gradle while preserving chronological
candidate learning and every negative observation. It does not authorize a
production rollout or decide whether the adaptive-fragment POC should continue.

The authoritative machine contract is
[`poc-current-longitudinal-campaign-v1.json`](./poc-current-longitudinal-campaign-v1.json).

## Execution model

Repositories run sequentially. Each family has persistent but isolated control
and candidate checkouts, Gradle homes, build caches and daemon registries. Only
the candidate owns BuildOpt state. Observation `N` may consume only candidate
state committed by observations `1..N-1`; there is no future-state replay and
no untimed candidate warmup.

The first pair is control-first and order alternates thereafter. Both arms run
the frozen workflow on the same public revision. The runner records independent
monotonic wall time, the candidate's non-overlapping internal phases, cache,
daemon and state fingerprints, required-output hashes and the complete installed
decision. Learning and calibration work remains inside candidate wall time.

## Exclusions and reserves

A primary commit may be excluded only for native build failure, unavailable
dependencies after preparation, runner-environment failure or native-output
nondeterminism. The next frozen reserve is consumed in its recorded order. A
slow candidate, native retention, a negative delta or an unhelpful change shape
is never excluded. All exclusions remain in the raw evidence.

## Fragment boundary

The installed `buildopt optimize` command currently selects whole structural
profiles. The storage-neutral adaptive-fragment planner and Build Impact
activator are not yet wired into that public command. Therefore the campaign
must report `NO_FRAGMENT_RUNTIME` and an empty activated-fragment set unless an
installed result carries real fragment identities. A selected whole profile
must never be relabelled as a fragment. `AF-014D` owns the consequence for
mechanism attribution.

## Acceptance

Each repository must provide at least 15 comparable pairs or be explicitly
`INSUFFICIENT_COHORT`. Required outputs must be exact, all adverse deltas and
exclusions must remain visible, and measurement-only executions may not update
fragment authority. A positive or negative row requires both the corresponding
cumulative sign and at least 60% of comparable pairs in that direction;
otherwise the row is inconclusive. The deterministic checker recomputes signed
and cumulative net value from immutable observations. These are current POC measurements, not
a production, soak, design-partner or Test Optimization claim.
