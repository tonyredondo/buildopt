# POC overhead attribution v1

This diagnostic contract decomposes the installed candidate and optimized
native Gradle control before changing performance-sensitive code. It exists to
remove measured POC overhead, not to create a production profiler.

Gradle 9.6.1 build-operation traces provide one non-overlapping external
timeline: launcher and Gradle-client startup, configuration before the main
task interval, the temporal task interval, Gradle finalization, and client or
launcher teardown. Init/plugin setup is reported separately and is zero when
the measured candidate uses the native-only fast path. Positive deltas mean the
candidate is slower. Trace timings are diagnostic because tracing perturbs the
build; they never replace the uninstrumented breadth result.

The focused correction may remove only a difference demonstrated by code and
phase evidence. Task selection, Gradle arguments, isolated Gradle homes,
required outputs, fixture mutations, pair order, sample count, and every
`POC-BREADTH-001` threshold remain unchanged. The breadth matrix is then rerun
without tracing on the same strict 4-CPU/16-GiB runner.

Validate checked evidence with:

```bash
./dev/check-poc-overhead
./dev/check-poc-breadth benchmarks/results/poc-breadth-v2.json
```

This is owner-controlled POC evidence. It does not require a soak, design
partner, production deployment, or Test Optimization.
