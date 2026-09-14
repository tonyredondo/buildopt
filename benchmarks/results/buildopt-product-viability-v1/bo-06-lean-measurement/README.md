# Checkstyle replay with lean recording

15 September 2026 · The identical-code control passed. The longer development
sequence could not start because its comparator qualification was missing.
BO-06 remains **partial**; this attempt produced no new optimization result.

Checkstyle checks Java code against coding rules. Our existing correction lets
it reuse work for unchanged files while preserving the complete reports. This
experiment was meant to check whether that correction saves time across the
first twenty code changes in the Elasticsearch development window.

## What ran

We first ran the same corrected code on both sides, using separate build state.
This checks whether the measurement produces a large apparent difference when
there is no optimization difference to measure. The control covered original
revisions 17–20; revision 17 was the cold start.

| Result | Observation |
| --- | --- |
| Builds | All eight succeeded |
| Output comparisons | Four live comparisons and four independent reconstructions passed |
| Total measured request time, excluding the cold start | 127.738 seconds versus 129.505 seconds |
| Difference | 1.767 seconds, or 1.38% of the faster side |
| Registered stopping rule | Stop if the difference is both at least three seconds and at least 5% |
| Decision | Control passed for this window |

The [complete results](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-lean-measurement/analysis/control-summary-v4.json)
retain every request, the cold start and recorded preparation costs. This small
control does not establish general measurement precision or a speedup from the
correction. The machine was busy during parts of the experiment. All observations
remain included; process samples are diagnostic and do not prove isolation.

The [approved method](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-lean-measurement/approved-method.md)
removes the Java diagnostic agent used in the earlier screen. Both sides still
record tasks and outputs. Any later saving measured this way would apply to
that recorded workflow. BO-09 still needs to compare the installed experience
with ordinary Gradle. The older precision and sampling failures remain unchanged.

## Why the next sequence did not start

The runner requires proof that the output comparator is qualified before it
allows a development replay. The policy supplied to this attempt had an empty
qualification field. The control phase permits that omission; the development
phase does not. Its preflight stopped with `lstat : no such file or directory`
before creating the run or starting a build.

This was a preparation error. Our local checks exercised the command paths
and the new admission rules, but missed the real Elasticsearch policy's
additional requirement. The older qualification names a different comparator,
so it cannot fill the gap. The [diagnosis](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-lean-measurement/analysis/prefix-admission-diagnosis.json)
records both versions and the source of the refusal.

Adding the missing proof would change the policy document that was fixed
before the control. That policy is part of the measurement identity. We kept
the measured inputs intact and closed the attempt as incomplete. The planned
42-build sequence and protected revisions 21–100 have **not run**. There is no
new evidence here that the correction saves time or makes builds slower.

## Checks and retained failures

The runner built with the pinned Go toolchain. Local tests covered output
differences, missing results, source changes, fallback, cancellation, process
closure and invalid readiness receipts. They used 94 small fixture requests,
within the limit of 104, with no Gradle or comparison JVM starts.

One fixture reached its time limit, then passed a separate bounded run. A
command-line test initially dispatched the test suite instead of `run`; the
test entry point was corrected and the affected checks passed. The original
failures remain recorded. All 93 fixture services and both owner sessions
were verified closed.

Offline reporting also needed corrections: the native CPU summary initially
included the other side's idle supervisor, accounting matched a fixture receipt
as though it were an owner receipt, and one derived seconds field contained
nanoseconds. The corrected report is explicitly versioned `v4`. These changes
did not alter the measured runner, request times, output comparisons or control
decision. The source and earlier reports are retained for inspection.

The attempt used eight owner builds and eight comparison JVMs, with no owner
retries. The [allocation closeout](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-lean-measurement/receipts/allocation-closeout-v2.json)
records disk use, limits and process closure.

## Next step

Complete the qualification for the current comparator and validate the actual
development and confirmation admission paths before another owner build.
The [continuation requirements](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-lean-measurement/continuation.md)
describe the missing proof and how to avoid repeating this preparation error.
The previous short-screen results remain exploratory. BO-07 is still deferred.

## Evidence

The [source archive](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-lean-measurement/runner-source.tar.gz)
contains the measured runner and fixture protocol; the
[patch](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-lean-measurement/runner-extension.patch)
shows the changes from the previous runner. This implementation is for research
on Linux AMD64. The supported Checkstyle candidate remains V2.

The [evidence index](https://github.com/tonyredondo/buildopt/blob/main/benchmarks/results/buildopt-product-viability-v1/bo-06-lean-measurement/evidence-manifest.json)
lists the exported records and their original locations. Full build outputs,
private caches and state inventories remain under
`.tools/state/buildopt-product-viability-v1/bo-06-lean-measurement-2026-09-14`.
The export verifies the reported arithmetic and provenance; independently
repeating every output comparison also requires those retained local files.
