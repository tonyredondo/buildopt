# Direct source ownership inside conservative Gradle components

`POC-SOURCE-OWNERSHIP-001` tests one generic correction exposed by the frozen
Micronaut structural-transfer attempt. Gradle project dependencies can contain
strongly connected components. BuildOpt must keep the union of their source
and external dependency boundaries for conservative production impact closure,
but that union is not evidence that every member directly owns every source.

Generated component members therefore retain their original project source
roots as optional `ownedSourcePaths`. The field is a strict subset of the
expanded `sourcePaths`. Production impact evaluation continues to use only the
expanded boundary. The explicit owner-operated POC evaluator may use owned
paths for direct attribution, resolving nested directories by the existing
most-specific rule. Equal-specificity matches, malformed ownership, unknown
paths, incomplete state, or insufficient candidate coverage retain the full
graph.

The correction is repository-independent: it contains no project names,
Micronaut paths, task names, or repository-specific exceptions. Unit and real
Gradle fixtures must prove the conservative and direct boundaries separately.
Only after those checks pass may the existing Micronaut workload be replayed;
its repository revision, mutation, control/candidate tasks, outputs, cache
state, pair order, eight-pair requirement, and value thresholds remain
unchanged.

The unchanged replay passed. Direct ownership resolved only to
`:micronaut-http-client-jdk`, inside the 22-project candidate. Against the
optimized native 75-project control, the installed candidate saved
17,561.625 ms/72.97% on average with 8/8 positive pairs, a positive paired
interval, exact required JARs and verified global full-graph fallback. This
closes the generic ownership hypothesis; it does not generalize the percentage
to another repository or authorize automatic activation.

Run the bounded contract with:

```bash
./dev/run --toolchain go -- ./dev/check-poc-source-ownership
```

This is POC evidence only. It does not alter production selection, Test
Optimization, release behavior, or operational readiness.
