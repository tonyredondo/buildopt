# Checkstyle regression: shared-host context and accepted disposition

Date: 2026-09-10. This addendum records the owner's decision after the
[regression investigation](../../benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-regression/README.md).
The experiment bundle, measurements and closeout audit remain unchanged.

The owner confirms that this workstation is shared: other agents sometimes run
CPU-intensive Rust and .NET builds, including Rust LLVM coverage work. The owner
accepts the investigation as sufficient to move on. Concurrent host activity is
therefore a plausible explanation for the transient original spike. Its overlap
with that particular request was not captured, so this is an accepted working
explanation, not a measured attribution to a particular process or a cache miss.

Close the retrospective spike investigation as accepted; do not spend additional
builds recovering its exact historical cause. Verified file reuse, subsequent
UP-TO-DATE behavior and the observed Gradle finalization wait retain their
original evidence scope. The whole-build timings remain measurements on a
shared host. Preserve both favorable and unfavorable samples and the original
negative historical result; this acceptance does not establish product value.

The next engineering step remains the
[safe finalization design and bounded whole-request comparison](../../benchmarks/results/buildopt-product-viability-v1/bv006-checkstyle-regression/next-step.md).
For that comparison, register a lightweight host-load/CPU-pressure observation
and a handling rule for external contention before collecting timings. Apply the
same rule to both arms, keep affected samples visible, and use the existing
small balanced budget. CPU affinity alone does not reserve those cores against
other workloads. Do not stop other agents' processes or add a broad CPU sweep.
Implementing this prospective measurement detail belongs to that next step;
this disposition starts no builds or watcher.

For the adaptive product, a single slow request on a shared host is insufficient
to invalidate reusable state or trigger a new optimization search. Evaluate
repeated comparable whole-request behavior alongside actual cache reuse and
environmental pressure. Exact policy thresholds still need prospective proof.
