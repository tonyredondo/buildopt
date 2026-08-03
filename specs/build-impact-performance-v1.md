# Build Impact performance

This contract measures the time saved when Build Impact omits a project that a
repository-owned manifest declares unaffected by a change. It compares the
full `assemble` graph with the declared `:service-a:assemble` alternative for a
change in `library-c`; both arms still run the Test Optimization-owned
`testOwnedCheck` separately.

The benchmark copies the repository's three-project Gradle fixture into
isolated workspaces and generates deterministic compilation load: 96 sources
in `library-c`, 96 in `service-a`, and 2,048 in the omitted `service-b`.
Caching, the daemon, and Configuration Cache are disabled so the comparison
measures avoided build work. One warmup precedes four measured pairs with
alternating order.

Valid evidence requires:

- the full graph to compile and materialize `service-b`;
- the selected graph to execute no `service-b` task and create no service-b
  artifact;
- byte-identical service-a JARs and Test-owned marker outputs in both arms;
- positive savings in every measured pair and in the mean.

The alternative comes from the checked-in customer manifest. The production
`buildimpact.PlanSelection` boundary, qualification requirements, and
fail-closed fallbacks remain covered by the separate active-selection proof.
This benchmark measures the benefit available after that gate authorizes the
alternative; it does not bypass or authorize promotion and does not change any
Test Optimization behavior.

Run and validate the bounded experiment with:

```bash
./dev/run-build-impact-performance /tmp/build-impact-performance.json
./dev/check-build-impact-performance /tmp/build-impact-performance.json
```

The result is descriptive POC evidence for this workload. It does not imply
the same reduction for changes that require the full graph and does not run
the deferred soak.

