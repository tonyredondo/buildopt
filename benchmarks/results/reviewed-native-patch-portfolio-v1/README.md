# Reviewed native patch portfolio v1 evidence

[`selection.json`](./selection.json) opens the architecture pivot with four
bounded proposals across three public families. It reconstructs selection from
the closed NAC public-correctness ledger and patch plan. Those inputs select
the cohort only: they provide no fresh correctness row and no timing sample.

[`result.json`](./result.json) contains the fresh correctness and eight-pair
measurements. Micronaut `PythonVfsBytecodeCompile` saves 6,921.125 ms (63.44%)
and Spring `ArchitectureCheck` saves 985.5 ms (35.34%); both have 8/8 positive
pairs, positive paired intervals, improved p95, exact cross-root outputs and
zero product failures. OpenTelemetry has only 6/8 positive pairs and regresses
p95; Spring `ShadowSource` saves only 137.875 ms, below the fixed 500-ms gate.

The two accepted proposals span two families and save a signed 7,906.625 ms
per compatible portfolio build. A deliberately conservative 2,340,000-ms
machine charge includes all four proposals' observed generation, diagnostics,
correctness, validation and delivery work and repays in 296 compatible builds.
Human review was not measured and is neither invented nor included. The result
qualifies the owner-operated reviewed-patch portfolio POC only; it does not
authorize production use, automatic application or automatic merge.

[`accepted-proposals.json`](./accepted-proposals.json) carries only the two
qualified digest-bound patches into the existing review-only Patch Autopilot
transaction. It requires owner review and exact apply/revert and grants no
automatic apply, merge or production authority.
