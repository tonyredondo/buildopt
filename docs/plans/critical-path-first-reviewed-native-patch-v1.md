# Critical-Path-First Reviewed Native Patch v1 Tracker

| Block | Outcome | State |
|---|---|---|
| `CPFRNP-001` | Freeze ten unused families, owner workflows, gates, and budgets | `DONE` |
| `CPFRNP-002` | Capture one optimized-native diagnostic per family | `DONE_PARTIAL`: 10 rows; 4 complete, 6 typed incomplete |
| `CPFRNP-003` | Reconstruct critical-path economics and inspect only admitted task source | `DONE_STOP`: 19 material standard tasks, 0 repository-owned tasks; zero source inspection |
| `CPFRNP-004` | Apply the 10/10 conclusive and 3/10 proposal-family gate | `DONE_STOP`: 4/10 and 0/10 |
| `CPFRNP-005` | Prove correctness for at most two admitted proposals | `NOT_AUTHORIZED` |
| `CPFRNP-006` | Measure paired value, review, delivery, and combined economics | `NOT_AUTHORIZED` |
| `CPFRNP-007` | Issue the terminal decision and close documentation | `DONE` |

The route starts from repository-owned workflows and measured optimized-native
critical paths. Source is not inspected for correction opportunities until a
task passes the unchanged 500-ms, 2%, and hard-dependency critical-path gates.
Diagnostics are discovery evidence, not timing samples.

The terminal evidence is
[`critical-path-first-reviewed-native-patch-v1`](../../benchmarks/results/critical-path-first-reviewed-native-patch-v1/README.md).
The frozen campaign consumed all ten family rows. Four owner workflows completed;
six stopped on signing context, unavailable toolchains, wrapper/JDK compatibility,
owner test failures, or the fixed timeout. Independent trace replay finds 19
tasks above the 500-ms/2% thresholds in the four complete rows, all owned by
Gradle or Kotlin rather than the subject repositories. The breadth and proposal
gates therefore fail. `CPFRNP-005/006` never opened, and the route adds no patch,
candidate build, timing sample, product failure, or speedup claim.
