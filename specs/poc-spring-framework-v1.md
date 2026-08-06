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

The post-diagnostic Runtime Tuning hypothesis is frozen before accepted paired
timing. Spring configures the global `checkstyleNohttp` task with a 1 GiB heap.
An isolated, non-gating full-scan probe took 35.29 seconds at 1 GiB, 31.88
seconds at 2 GiB, and 31.61 seconds at 4 GiB. The candidate therefore raises
only that standard Gradle `Checkstyle` task from exactly `1g` to `2g` when the
runner has at least 14 GiB of memory. It preserves repository-owned values other
than exactly `1g`, and it does not change task inputs, outputs, actions, cache
policy, requested tests, or worker count. The 4 GiB arm is rejected before
paired timing because another 2 GiB bought only 270 milliseconds in the
diagnostic probe.

An online preflight may download dependencies and discover incompatibilities,
but it is not a timing sample. Accepted measurements run offline in isolated
control and candidate homes, use eight alternating pairs, and retain Spring's
native cache and parallelism in the control. Before each measured arm, the
harness restores the same original-source native-cache seed, removes outputs
outside timing, and then applies the same fixed source mutation. This keeps
private daemons and dependency state warm while forcing the mutated
`checkstyleNohttp` key to execute in both arms instead of accepting a cached
result as Runtime Tuning evidence.

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
