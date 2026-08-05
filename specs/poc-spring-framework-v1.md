# Spring Framework value experiment

This contract opens a new public-repository hypothesis after the smaller
Spotless and Mockito workflows failed to provide stable material savings. It
uses Spring Framework because its pull-request build is large enough to expose
meaningful avoided work while remaining reproducible on the local 12-CPU POC
host.

The control is not weakened. The pinned Spring revision already enables
Gradle Build Cache and parallel execution, uses Gradle 9.6.1, and completed the
public `check antora` build step in 12 minutes 36 seconds. BuildOpt must create
value beyond those native settings.

## Three independent cells

The experiment keeps different optimization mechanisms separate:

- `BUILD_OUTPUTS` measures `assemble`. Build Impact may replace the full graph
  only with a reviewed alternative that covers the changed `spring-jms`
  project and every affected reverse dependent.
- `TEST_PREPARATION` measures `testClasses`. These are build-owned compilation
  and resource tasks; no Gradle `Test` task may be selected, omitted, cached,
  reordered, or retried.
- `FULL_VERIFICATION` measures `check`. Runtime Tuning may alter only resource
  settings. Every requested test task, test case, outcome, and required output
  must remain identical.

An online preflight may download dependencies and discover incompatibilities,
but it is not a timing sample. Accepted measurements run offline in isolated
control and candidate homes, use eight alternating pairs, and retain Spring's
native cache and parallelism in the control.

Each cell must save at least 500 milliseconds and 2%, have a positive lower
95% paired-bootstrap bound, preserve non-empty required outputs, preserve all
requested tests, and introduce no product-attributable failure. Thresholds do
not move and failed pairs are not discarded after observing results.

Only a generic mechanism that qualifies on Spring may be transferred unchanged
to `open-telemetry/opentelemetry-java-instrumentation`. The transfer may adapt
the repository manifest and declared graph, but not product behavior or the
gate.

Validate the frozen contract with:

```bash
./dev/check-poc-spring-framework
```
