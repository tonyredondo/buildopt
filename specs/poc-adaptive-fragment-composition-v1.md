# Adaptive fragment composition POC protocol

This protocol closes `AF-011` by measuring compatible adaptive fragments on
one Gradle workflow instead of adding percentages from separate experiments.
The fixture combines an owner-reviewed task-contract patch with Build Impact
subgraph reduction. Both mechanisms are measured independently against the
same optimized native Gradle control and then executed together in a third,
directly timed arm.

For each Kotlin and Groovy DSL variant, preparation and one warm-up per arm are
unmeasured. Eight measured pairs alternate arm order. Gradle 9.6.1 runs with
the build cache, Configuration Cache and parallel execution enabled. Every
comparison removes produced outputs before each arm while retaining only the
arm's warmed Gradle user home. The control runs the complete workflow with the
original non-cacheable task contract.

An independent controlled HTTP-cache experiment revalidates cache locality on
its own exact output contract. Locality is not inserted into the final
Build-Impact-plus-patch arm because the fixture does not share the committed
remote-cache object contract. It may remain a separately qualified fragment;
the report must not imply that it contributed to the directly measured
composition.

Each direct mechanism and the final composition must save at least 500 ms and
2% on average, have at least seven positive pairs, preserve byte-identical
required outputs, produce no BuildOpt-attributable failure and have a positive
paired-bootstrap 95% lower bound. The independent locality evidence retains
its frozen four-pair gate. Thresholds are fixed before measurement.

Passing yields `COMPOSED_VALUE_QUALIFIED`. Failure yields
`RETAIN_BEST_SINGLE_FRAGMENT`; it never authorizes production rollout. Test
Optimization, soak testing and design-partner validation are outside this POC
block.

```bash
./dev/check-adaptive-fragment-composition
./dev/run-adaptive-fragment-composition \
  /absolute/path/to/result.json \
  /absolute/path/to/fresh-locality-evidence.json
```
