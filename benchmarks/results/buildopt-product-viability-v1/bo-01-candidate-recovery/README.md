# Candidate recovery before new measurements

BO-01 completed on 14 September 2026. The five candidates have recoverable
research records. Checkstyle's source, runner and required inputs also pass
their existing checks on this workstation. It can proceed to preparation of
the four-build measurement control in BO-02.

Micronaut and Spring have reproducible patches and consistent recorded
measurements. They need a complete workflow and a frequency assessment before
another timing experiment. Ktor and Beam remain evidence-review candidates for
BO-04. Their selected output contracts need particular attention.

No Gradle build or performance experiment ran in this step. The protected
validation changes were not used. This result verifies recovery, not product
value. Follow the [governing plan](../../../../docs/plans/buildopt-research-execution-plan-2026-09-14.md)
for the next decision.

## What is available

| Candidate | What was recovered and checked | What still needs proof |
|---|---|---|
| Elasticsearch Checkstyle | Five corrected source files, the existing baseline, four retained worktrees, nineteen frozen inputs and the complete manifest. Source prerequisites match at development changes 6, 7 and 17-20. | Whether the timing schedule produces a false improvement; then net saving against native Gradle across changing code. |
| Micronaut Python compilation | Original source and exact patch at `428ddeb3ad2acdabef2027cc06af3bf46865956a`. Eight recorded comparisons reproduce a 6.921-second saving. | How often a complete workflow benefits; its native dependencies and full output comparison on this workstation. |
| Spring architecture checks | Original source and exact patch at `91eb42645e26a7ef9382b4a655bcefe5c8682fee`. Eight recorded comparisons reproduce a 0.9855-second saving. | A credible opportunity under the current one-second net threshold, plus the same workflow and environment checks as Micronaut. |
| Ktor Build Impact | The original revision pair, selected plan description, output contract, package identity and eight comparisons reproduce the recorded 30.979-second saving. | A defensible requested output scope and an unanswered lifetime question. The original subject checkout and generated profile have not been recovered locally. |
| Apache Beam Build Impact | The original revision pair, selected plan description, output contract, package identity and eight comparisons reproduce the recorded 40.123-second saving. | The same scope and lifetime questions; original checkout, generated profile and runtime reconstruction if admitted. |

The [inventory](./inventory.json) records revisions, commands where retained,
patches, output requirements, costs, source locations and gaps for each case.
The [verification record](./verification.json) contains the fresh checks and
input hashes. Percentages from different repositories are not combined.

## Checkstyle can reach the measurement control

Checkstyle checks Java source against coding rules. The supported correction
keeps results for unchanged files and saves successful history at the end of
the build. We verified the exact source used by the
[completed finalization experiment](../bv006-checkstyle-finalization/README.md).

Its retained command is:

```text
{repo}/gradlew --offline --no-scan --max-workers=8 :server:precommit --continue
```

The manifest also binds the Gradle cache policy, observation agent, runtime,
dependency layers and output comparison. Gradle 9.7.1 runs with Configuration
Cache disabled in this workflow. Both sides retain the earlier file-check
baseline. The previous comparison used two optimized implementations; its mixed
timing result remains unchanged.

All nineteen frozen input bindings match. Each of the four worktrees has its
recorded revision and exact declared source changes. All 36 source checks for
the six development revisions match. The existing runner's `validate` command
passes against the retained manifest without starting a workflow. This covers
the bound inputs and preserves the correctness proof's original scope.

BO-02 needs a fresh run directory, manifest and allocation. The old deadline
and completed allocation stay closed. Both arms must contain the same supported
correction, as specified by the
[measurement-control protocol](../../../../docs/plans/buildopt-checkstyle-measurement-control-v1.md).
The control remains pending.

## Micronaut and Spring need a workflow-level question

The Micronaut correction lets Gradle reuse the output of
`:micronaut-context-python:compileVfsPythonBytecode`, which prepares Python
bytecode. The Spring correction does the same for
`:spring-core:checkArchitectureMain`, which checks rules about code structure.

Both original revisions exist in the registered local Git stores. Applying
the recorded edits in memory reproduces each patch's exact output hash.
The sources declare Gradle 9.7.1. No subject worktree was created or changed.

The portable result retains task names and paired durations, but not a complete
historical launch command, environment and set of raw timing logs. Those details
must be specified before fresh timing. The earlier output-digest and reversal
claims remain historical evidence; BO-01 did not rebuild those outputs.

The task-level setup estimates are 87 applicable builds for Micronaut and 366
for Spring. Neither was observed across that many changes. Spring's recorded
saving is also below the current one-second floor before adding costs. BO-03
must establish a credible opportunity; the old task-level pass does not grant
admission under the current criteria.

## Build Impact needs an explicit output boundary

Ktor requested `jvmJar --max-workers=12`; the candidate used
`:ktor-server-webjars:jvmJar`. The required output was one WebJars module JAR.
Beam requested `classes --max-workers=12`; its candidate used
`:examples:java:twitter:classes`. The required outputs were the compiled classes
under that example's output directory.

The eight pairs preserve those declared outputs. They do not establish equality
of every output that the original broad command might produce. BO-04 must decide
whether that narrower result is a legitimate workflow requirement and whether
the mechanism adds value beyond requesting the relevant module directly.
It cannot claim a full-build improvement by silently changing the required result.

The tested BuildOpt release was v0.6.1 at
`a354e3ee7fffc7abe83b8abe0a125956b159e01a`; that source commit exists locally.
The available records did not recover a registered original Ktor or Beam
checkout, installed release binary or generated profile directory. Their package
and profile hashes are retained, but current native buildability is unverified.
Do not substitute the current BuildOpt binary for the measured release.

Ktor also has a separate
[negative Jetty-profile lifetime result](../../poc-profile-lifetime-v1/README.md).
It lost time after discovery and fallback costs. That result must inform BO-04;
the WebJars result does not justify reopening generic plan reuse.

## Tooling and evidence checks

The repository doctor completed. Direct probes through `dev/run` confirm Go
1.26.5, Temurin 21.0.12+8 and Temurin 25.0.3+9. The global Java installation has a
different version, so subject work must continue using the declared runtimes.
Tool availability does not prove every candidate can build.

All six registered replication repositories still contain the required 101
commits and their recorded anchors: Groovy, Kafka, Spring, OpenTelemetry Java,
Micronaut and Hibernate. This was a metadata check. Their validation source and
timings were not inspected, and their existing unmeasured status is preserved.

The Build Impact evidence checker passes. The patch-portfolio checker exposed
a pre-existing `jq` precedence defect: it assigned `true` to the qualifying
proposal count instead of `2`. The correction computes the four aggregate
values before evaluating the existing assertions. No condition or data changed.

The corrected full checker passes. Seven altered inputs are rejected: wrong
schema, missing pair, changed timing, mismatched output digest, incorrect proposal
count, missing cost and incorrect qualification decision. Bash syntax and the
repository-pinned ShellCheck also pass.

## Recovery location and next action

Runtime receipts and the complete local task record are under:

```text
.tools/state/buildopt-product-viability-v1/bo-01-candidate-recovery-2026-09-14/task-state.json
```

BO-01 is verified. BO-02 is the next timing step. BO-03 and BO-04 may use the
recovered records for their separate admission decisions. No historical result
has been promoted to a sustained-saving or adaptive-product pass.
