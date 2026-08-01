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

The pair authorizes no active omission. C3-003 must compare shadow predictions
with full execution, and C3-004 must keep selection `INCONCLUSIVE` until the
unchanged `BIA-002` evidence threshold passes.
