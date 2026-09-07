# Complete native correction: rejected native admission

Date: 2026-09-07. Evidence: `E-556`. Decision: `STOP_NATIVE_ADMISSION`.
The fourth authorized two-hour attempt completed P01/P02, D01/D02 and M01/M02.
Native admission fails: warmed configuration is below the unchanged 500-ms
floor, and the two native output inventories differ. CNC-005..007 were not
executed. This is a conclusive admission stop, not a failed candidate, measured
speedup, or a claim that every native correction is unviable.

## Identity and authority

The owner explicitly approved another corrected-package attempt, capped at
7,200 elapsed seconds, with all earlier results and costs retained. Execution
used BuildOpt `cd164da42266d7b9de5f923d770fb5d5d74f0c95`, after successful
[Base CI](https://github.com/tonyredondo/buildopt/actions/runs/34038365351) and
[Native Platform CI](https://github.com/tonyredondo/buildopt/actions/runs/34038365261)
on that actual checkout. The [package](./package.json) digest is
`07618fd835064dfa97eeea4f96d8ee3deafb32fd00b31e196fdcc20af0d1f628`;
its executable digest is
`2d6e119813e45f2fc0d306fe49e56c0efedcc2c6b62020333e7542aade321a32`.
Every package file matched its committed bytes before capture.

The immutable clock started at boot time `3532637.98`, before new worktrees
and runtime preparation; its deadline is `3539837.98`, with review at
`3534437.98`. Three fresh registered detached worktrees share the original
recovered Git repository. The exact GraphQL revision/archive, source files and
spans, and both copied Corretto archive/install trees passed verification.
No source clone/copy, runtime substitution or owner-home reuse occurred.
P01/P02 began with empty private homes and generated-source state. Their
dependency-only seed contains 1,521 files, excluding execution/output state.
All subsequent starts were offline. The original three campaigns remain intact.

## Six retained starts

| Slot | Native process seconds | Reconstructed result |
| --- | ---: | --- |
| P01 | 122.459808782 | Native preparation and post-checks passed. |
| P02 | 175.870604048 | Separate native preparation and post-checks passed. |
| D01 | 33.605023969 | Expected strict exit 1; complete root report captured. |
| D02 | 35.286764557 | Expected strict exit 1; independent root report captured. |
| M01 | 107.682036551 | Instrumented native `assemble` passed; all five JARs retained. |
| M02 | 6.771103050 | Same native home/worktree reused; all five JARs retained. |

The [capture-check output](./capture-check.json) was reconstructed by the frozen
runner against the original campaign before documentation changes. Every row
has an immutable reservation, start marker, raw process/stream records and
equal before/after verified-input bindings. Both strict failures are expected
native diagnostics, not additional product-attributable failures. No recipe,
candidate, owner-test-suite execution or value pair exists.

The six native processes cost **481.675340957 seconds** in total. Preparation,
verification, seed copying, quiescence, analysis, validation and publication
add elapsed cost; process sums are not total operational cost. There are nine
real starts across all retained windows (0 + 1 + 2 + 6). Earlier research was
not continuously timed and remains unknown, not zero. No payback is calculated.

## Complete diagnostic binding

Each report contains five unique problems, zero overflow and the requested
`assemble` profile. All diagnostics, including non-problem input observations,
remain in the raw HTML; no expected-problem subset replaces the full report.

| Binding | Fresh report site | Existing owner/consumer obligation |
| --- | --- | --- |
| B1 | `build.gradle:144` | Git worktree detection and version fallback. |
| B2 | `build.gradle:169` | Git branch result consumed by the local version. |
| B3 | `build.gradle:186` | Build-result completion summary. |
| B4 | `build.gradle:938` | Test-state completion summary and ordering. |
| B5 | `BundleTaskExtension.java:407`, task `:jar` | Bnd reads `Task.project`; public configuration and both JAR consumers remain relevant. |

Both reports match the source-bound owners in the
[feasibility ledger](../selection/feasibility.md). No additional reported blocker
was found. That does not prove the proposed lifecycle or Bnd corrections safe;
the unexecuted real fixtures remain mandatory before any candidate.

## Controlled materiality and native output drift

The frozen 120-second quiescence and seven-sample check passed at
**1.0816372922167297**, below 1.15. M01/M02 ran on the same local non-CI host,
CPUs 0-3, four workers and the temporary performance/EPP profile. This short
probe is not certification of complete workload stability.

| Row | Root workflow ms | Configuration interval-union ms | Configuration share |
| --- | ---: | ---: | ---: |
| [M01](./M01-materiality.json) | 105,425 | 12,869 | 12.206782% |
| [M02](./M02-materiality.json) | 6,091 | 362 | 5.943195% |

The existing `CONFIGURATION_CACHE_UNLOCK` analyzer reconstructs the union of
root load, configure and task-graph intervals, without summing nested work.
Only these fresh traces enter the analysis; the reused analyzer/schema does
not reuse WCNCP evidence. The conservative result is **362 ms**, below 500 ms,
despite passing the 2% screen. The fresh-home M01 result cannot substitute for
the warmed native result. Neither row measures attainable saving; both retain
diagnostic instrumentation and neither is a candidate/value comparison.

Three JARs are byte-identical between M01 and M02. The shadow intermediate and
final unclassified JAR are each 784 bytes smaller in M02 and have different
SHA-256 digests. Their manifests fall from **2,357 to 65 bytes**: Bnd/OSGi headers,
including `Bundle-Version`, `Bundle-SymbolicName` and `Import-Package`, disappear.
The plain JAR stays unchanged. M02 reports `:jar UP-TO-DATE`, while `:shadowJar`
and the three downstream processing/packaging tasks execute again.

This is an observed native cold-to-warm output difference, not a candidate
regression or an allowed ZIP normalization. The source and task outcomes suggest
an execution-dependent manifest handoff; the exact root cause is not yet proved.
M01/M02 are not the two fresh-root C01/C02 correctness rows, so no cross-root
correctness claim follows. The observed output instability independently prevents
assuming a stable exact-output comparator. No manifest repair or extra Gradle
probe was performed.

## Retention, replay and closure

The [raw archive](./native-evidence.tar.gz) is 24,766,160 bytes, SHA-256
`652bde5148757e2b495986fbe813349ae9fa50ff4815e669e2acd8cca3e7ca78`.
It contains original clock/ownership/environment records, the seed inventory,
all six complete attempt records/logs, both HTML reports, operation traces,
task graphs, inventories and all ten retained JAR byte streams. Archive
comparison against the original campaign passed. Large prior-state caches,
dependency/runtime binaries and registered source worktrees remain local;
they were not removed or claimed to be inside the archive.

The exact guardian and its bound profile hold were stopped after M02. Both
were absent and `balanced` was observed at boot time `3533926.18`, 1,288.20
seconds after initialization. This is a shutdown observation, not a reset or
the final elapsed campaign cost. Later checking/documentation/CI time remains
inside the same two-hour ceiling.

The separate [30-minute review](./review.json) records stop at boot time
`3534483.70`. It was written after the raw capture archive and is retained
alongside it; the capture archive is not rewritten to hide that sequence.

From the BuildOpt worktree, replay the portable terminal evidence and negatives:

```bash
./dev/check-complete-native-correction-terminal
./dev/check-complete-native-correction-terminal --self-test
```

This checker verifies package sources against the execution commit, rejects
unsafe archive members, rehashes raw streams/artifacts/output bytes, reconstructs
six starts and cost, checks both complete report bindings, reruns interval-union
analysis, and compares native inventories. It does not launch Gradle or recreate
the original host/runtime verification. It rejects forged admission, hidden
starts/cost, substitution of cold materiality, and invented output equality.
Base CI runs this replay without new public builds or wall-time gating.

`CNC-004` is verified **as rejected admission**. `CNC-008` closes Phase A and
`CNC-014` records this bounded terminal synthesis. CNC-005..007 and CNC-009..013
remain blocked/unexecuted, not successful zero-cost phases. The wrapper/backend
implementation and earlier qualified reviewed-native corrections are unchanged.
This directed result does not establish installed value, persistence, breadth
or commercial viability. A native manifest-lifecycle investigation is a distinct
possible follow-up; it must not silently expand this stopped correction recipe.

## Validation record

| Check | Observed result |
| --- | --- |
| Frozen live capture checker | All six original rows, reports, verified-input bindings and retained outputs reconstructed before runbook edits. |
| Portable terminal replay and five negatives | Passed; forged admission/count/cost/materiality/output-equality summaries rejected. |
| Static Phase A contract | Passed; 60 slots, 7,200-second ceiling and 1,800-second review unchanged. |
| Materiality and strict-selector race suites | Passed in 1.021 and 1.017 seconds. |
| ShellCheck, layout, tracker and Base CI static | Passed; 53 accepted RFC decisions retained. |
| Documentation | Passed: 622 Markdown, 3,066 JSON, 3,126 local links, 2,143 repository commands and 51 Go packages; English check passed. |
| Original source/history retention | All twelve registered source worktrees clean at the frozen SHA; four immutable state hashes unchanged; raw archive comparison passed. |

The documentation checker initially rejected an internal tracker ID at the
start of this README heading; the descriptive heading above fixes that issue.
No experiment threshold or runtime behavior changed. Publication/hosted-CI
completion is tracked separately against its actual final commit, not inferred
from these local results or the execution baseline's older green checks.
