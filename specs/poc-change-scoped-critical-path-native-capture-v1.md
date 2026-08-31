# Change-Scoped Critical-Path Native Capture v1

Status: `CSCPN-001` contract frozen; native diagnostic capture is authorized.

This successor closes the trace-completeness gap left by the existing-evidence
route. It runs one fresh optimized-native Gradle diagnostic invocation for each
of the four incomplete public families. Micronaut retains its already-bound
operation/DAG evidence. Every invocation uses the exact target revision and
aggregate workflow already bound to the longitudinal change/output partition.

The invocations may emit task-operation traces, resolved hard-dependency graphs
and exact-output observations. They are diagnostic builds, not timing samples:
instrumented wall time cannot support a performance or product-value claim.
Public source remains unmodified, no candidate is applied, and no candidate
build is authorized.

A conclusive row requires a clean exact revision, successful native workflow,
source-bound structural evidence, a complete task-operation trace, a resolved
hard-dependency graph and reconstruction by the repository-owned analyzer. An
actionable row additionally requires at least 500 ms and 2% of the reconstructed
native critical path to belong to the structurally omitted output partition.
Cumulative task duration is never substituted for critical-path duration.

The discovery gate remains 5/5 conclusive families and at least 3/5 actionable
families with zero product failures. Failure stops candidate correctness and
timing. Passing authorizes only a separately frozen candidate-correctness block;
it does not itself authorize a candidate build or timing.

Capture is sequential, limited to four Gradle starts, 30 minutes per family and
12 workers, and requires at least 8 GiB free before each family. Temporary
checkouts and project outputs are removed after their bound evidence is retained.
