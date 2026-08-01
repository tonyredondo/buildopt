# Build Impact Analysis fixtures

`manifest.v1.json` is a repository-committed `C3-001` manifest bound to this
repository and its pull-request pipeline class. It enumerates one complete
customer-authorized alternative entrypoint set, required artifacts, the
Test Optimization-owned check boundary, global change paths, and the mandatory
`FULL_GRAPH` unknown-change fallback.

The fixture authorizes no omission by itself. C3-002 must still reconcile an
alternative against Gradle's declared graph, and C3-004 must keep selection
`INCONCLUSIVE` until the unchanged `BIA-002` evidence threshold passes.
