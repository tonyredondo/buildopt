# OpenTelemetry typed-producer graph reduction

This POC replaces the aggregate Spring autoconfigure `testClasses` candidate
with the four Gradle `AbstractCompile` tasks that own the declared
`build/classes/**` output. Discovery recognizes the Gradle task type, never a
repository-specific name: arbitrary tasks and every graph containing a
`Test` remain incomplete and retain the full graph.

The fixed OpenTelemetry `v2.30.0` revision proves that all four producers are
complete, contain no `Test`, have no unknown relationships and conservatively
reach the same 46 projects. A hot, output-reset comparison removes three task
nodes and two executed lifecycle/resource tasks while preserving all 125 files
byte for byte. This is a correctness/work-reduction result, not a material
build-time claim; the terminal performance gate remains unchanged.

Validate the checked evidence with:

```bash
./dev/check-poc-otel-graph-reduction
```

