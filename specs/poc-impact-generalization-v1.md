# Build Impact generalization experiment

This preregistered proof-of-concept experiment tests whether the installed
Build Impact mechanism continues to reduce real Spring Framework work across a
broader change and output matrix. It compares the packaged BuildOpt command
with optimized native Gradle; it does not add Spring-specific product logic or
widen production authority.

The matrix covers:

- leaf-source compilation through `classes`;
- shared-source test preparation through `testClasses`;
- leaf-source build-owned verification through `checkstyleMain`;
- leaf-source binary packaging through `jar`; and
- leaf-source distribution through `sourcesJar`.

Structural discovery found complete declared relationships for compilation,
test preparation, and packaging. Those three cells use the repository's broad
task-name selector as control and one repository-authorized project entrypoint
as candidate. Four paired observations alternate order. Both arms use the same
fixed source mutation, offline dependencies, restored native-cache seed, all
12 available CPUs, and clean outputs. The installed launcher, manifest
validation, graph validation, and selection overhead are included.

Spring's `checkstyleMain` and `sourcesJar` graphs report unknown relationships,
which makes their generated graph state incomplete.
Those verification and distribution cells therefore make no performance claim:
BuildOpt must retain the broad original selector and produce the same required
output as native Gradle. The protocol freezes this limitation before any timing.

Each performance cell must independently save at least 500 ms and 2%, have a
positive deterministic paired-bootstrap lower bound, improve all four pairs,
produce non-empty byte-identical declared outputs, and introduce no
product-attributable failure. A failed or unfavorable observation is retained;
thresholds and cells cannot move after measurement.

Two non-performance cells change `buildSrc` and `gradle.properties`. Both must
classify as global and restore the complete original `classes` entrypoint. An
unknown or ambiguous change retains the same fail-closed behavior.

The first complete-run attempt was discarded after eight performance
observations because valid but incomplete generated state returned an error
before the first capability-fallback cell could execute. The planner was fixed
to distinguish invalid bindings, which still fail, from valid incompleteness,
which now retains the original full graph. No observation from that interrupted
execution is reused, and neither the measurement method nor the value gates
changed.

The next attempt correctly retained the full graph, but the harness expected
the narrower `IMPACT_UNKNOWN_RELATIONSHIP` reason. Evaluation checks the graph's
global completeness first and therefore reports `IMPACT_GRAPH_INCOMPLETE`.
That attempt was also discarded, the exact expected reason was corrected, and
the measurement method, cells, thresholds, and product remained unchanged.

The following complete execution emitted a result, but validation rejected it
because one mandatory inter-arm clean and cache restore took 6.066 seconds,
exceeding the preregistered five-second setup budget. That result and all twelve
observations were discarded. The operational setup budget was revised to ten
seconds and is now enforced before a result can be emitted. The build-time value
gates, alternation, workloads, product, and all other measurement rules remain
unchanged.

This experiment does not execute root-build Gradle `Test` tasks or change test
selection, retries, sharding, prioritization, or execution. Hot State, Runtime
Tuning, Safe Cache, Shared Cache, and Edge Cache remain disabled. The output is
POC evidence, not a production, soak, design-partner, or universal savings
claim.
