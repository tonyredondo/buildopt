# Build Impact Analysis fixtures

`manifest.v1.json` is a repository-committed `C3-001` manifest bound to this
repository and its pull-request pipeline class. It enumerates one complete
customer-authorized alternative entrypoint set, required artifacts, the
Test Optimization-owned check boundary, global change paths, and the mandatory
`FULL_GRAPH` unknown-change fallback.

`declared-graph.v1.json` binds a complete synthetic Gradle view to the canonical
manifest digest. A patcher source change predicts the customer-owned
`jvm-components` alternative while omitting only the unrelated golden-lane
fixture; build logic, unknown paths, missing relationships, insufficient
coverage, and Test-containing entrypoints use `FULL_GRAPH`.

`shadow-observation.v1.json` proves the full original entrypoints and required
outputs/checks were observed while only predicting `jvm-components`.
`paired-control-observation.v1.json` adds an isolated candidate whose projects,
artifacts, and Test-owned check exactly match the full baseline.

These fixtures authorize no active omission. C3-004 must keep selection
`INCONCLUSIVE` until the unchanged `BIA-002` evidence threshold passes.
