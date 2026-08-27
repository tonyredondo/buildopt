# Sticky wrapper public opportunity gate v1

## Decision

`SWL-014C` is a proof-of-concept opportunity screen over the five frozen public
families. It asks whether the currently retained evidence contains a recurring,
exactly bound action whose projected critical-path saving can repay a bounded
trial within 30 compatible builds. It is not a performance benchmark and does
not authorize an optimization.

The detectors run in this fixed order:

1. `TASK_CONTRACT_JAVA_V1`;
2. `DECLARED_GRAPH_SCOPE_V1`.

The task-contract detector has no generic public input producer in the current
codebase. It therefore reports `INPUT_UNAVAILABLE`; another detector may not be
substituted for it. The graph detector consumes only evidence already captured
by the installed wrapper over the frozen current longitudinal cohort.

## Exact action requirements

A graph action is testable only when at least three observations carry the
same complete candidate-plan digest and binding digest, exact required-output
digests, no product failure and a positive omitted-critical-path contribution.
The action identifier is:

```text
SHA256(
  "buildopt-sticky-public-action-v1" NUL
  DETECTOR_ID NUL
  REPOSITORY_SCOPE_SHA256 NUL
  WORKFLOW_ARGUMENTS_SHA256 NUL
  CANDIDATE_PLAN_SHA256
)
```

Project counts are diagnostic evidence, not a substitute for the candidate
plan or critical-path trace. A result may include trial, validation and
publication costs only after an exact action exists; the frozen contract keeps
those values at zero because the retained source evidence has no such action.

## Gate

A family passes when at least one action has complete output ownership and
repays all stated costs within 30 compatible builds. At least three of the five
families must pass to advance to `SWL-014D`.

If fewer than three pass, the result is
`SWL_014C_WITH_STOP_EVIDENCE`. The tracker then skips `SWL-014D` and `SWL-015`
and advances to `SWL-016`, where the terminal decision is made. Missing input
remains visible and is never rewritten as a no-op, a different detector or an
estimated saving.

## Reproduction

```bash
./dev/run-sticky-wrapper-opportunity-gate
./dev/check-sticky-wrapper-opportunity-gate
```

The checker rebuilds the report from the SHA-256-bound cohort and raw evidence,
then rejects result drift and source tampering.
