# Current installed longitudinal harness v1

## Purpose

This contract defines the `AF-014A` proof that later longitudinal measurements
exercise the current installed BuildOpt package rather than an in-tree binary,
historical release, or repository-specific integration. It validates the
measurement apparatus; it does not claim that BuildOpt is faster.

The authoritative machine contract is
[`poc-current-longitudinal-harness-v1.json`](./poc-current-longitudinal-harness-v1.json).

## Arms and state

The control is Gradle with its native local Build Cache enabled. The candidate
is one public `buildopt optimize ...` command from a package built from the
evaluated Git revision. Control and candidate use separate persistent
checkouts, Gradle user homes, daemon registries, dependency caches, native
Build Cache directories and BuildOpt state. Only the candidate may create
`.buildopt` state.

Every candidate learning invocation is externally timed and paired with a
control. Pair order starts control-first and alternates thereafter. There is no
untimed candidate warmup, calibration or profile publication. After the first
selected replay, the fixture advances to a later global-change commit; it is
never rewound.

The selected-path fixture adds eight seconds of deterministic, non-cacheable
work to the omitted subproject. This margin exists only to make the state
transition robust against cold JVM and runner variation. AF-014A is explicitly
not performance evidence; later frozen public-repository cohorts own all value
claims.

## Timing model

Each non-bypass candidate result contains four non-overlapping top-level
durations:

- `preExecutionNs`;
- `gradleExecutionNs`;
- `finalizationNs`; and
- `unattributedNs`.

Their sum must equal `totalNs`. Nested diagnostics attribute Gradle setup,
matching, local state, central state, output materialization, output
verification and discovery/learning. Nested values are explanatory and must
not be added to the top-level total. The independent harness also measures the
whole installed process; internal `totalNs` must fit inside that external wall
time.

Bypass deliberately emits no BuildOpt result. Its proof is external: it runs
the original Gradle workflow, removes `BUILDOPT_BYPASS` from the child,
produces the same required outputs and leaves the candidate checkpoint byte
for byte unchanged.

## Acceptance boundary

The controlled fixture must prove exact control, selected, native-retained and
bypass paths, a digest-bound package/source revision, isolated mutable state,
forward-only revisions, alternating order, complete timing reconciliation and
zero hidden learning. This is POC harness evidence only: no production
authorization, soak, design partner or Test Optimization claim follows.
