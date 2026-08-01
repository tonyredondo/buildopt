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

`promotion-policy.v1.json` pins the unchanged `BIA-002` 30-day,
3,000-decision, 99%-coverage, 100-control-per-mandatory-class, and one-sided
95% false-negative limits consumed by C3-004.

These fixtures authorize no active omission. The two current observations are
honestly `INCONCLUSIVE`; even threshold-qualified synthetic evidence remains
non-authorizing until the separate C3-005 selection boundary accepts it.

`synthetic-repository/` is the C3-005 owner-controlled three-project Gradle
proof. Its canonical manifest permits one service-a alternative, its graph
maps a library-c change through service-a, and its separate Test-owned task
remains outside Build Impact selection. Isolated offline full/selected builds
must produce an identical service-a JAR and Test-owned marker while only the
full build materializes service-b.

`buildopt-impact generate` now discovers that repository through the real
Gradle model and writes the reviewable `buildopt-impact-graph.generated.json`
and `buildopt-impact.generated.json` files. `buildopt-impact check` regenerates
both files and rejects any byte-level drift. Discovery remains advisory: an
included build, unknown task shape, unsupported requested task, incomplete
model, or stale generated state preserves the original full entrypoint.
