# Product-Window Graph Recurrence v1 evidence

The fresh source-only result reconstructs all 192 rows: 64 each for Spring,
OpenTelemetry and Micronaut. It finds six eligible groups, all in Spring. The
frozen ranking selects `:spring-messaging / DEPENDENCY_SOURCE` with 11 exact
commits, a 12/27 affected closure and 15 omitted projects.

OpenTelemetry and Micronaut have no five-match group in the product window.
The earlier 256-row reports supplied no row. Gradle, candidates, timing and
public patches remain zero. This result authorizes only a separately frozen
fresh graph-confirmation contract for the selected exact commit.

Run `./dev/check-product-window-graph-recurrence` to reconstruct every group,
count, reason distribution, eligible ranking and raw SHA-256 from the 192 rows.
