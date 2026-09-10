# Checkstyle native capability admission

Date: 2026-09-08. Program: `BUILDOPT-VIABILITY-V1`. Steps C1-C4 are verified.
Decision: **ADMIT_BOUNDED_CONTENT_AWARE_CHECKSTYLE_PROTOTYPE** at G1 only.
No incremental candidate, paired saving, adaptive result or product viability
has been established. The prior ForbiddenPatterns negative result is unchanged.

## What was resolved

The observed task cost is real, but enabling Checkstyle's existing `cacheFile`
setting does not preserve this owner's contract. The exact Checkstyle 13.11.0
engine and Gradle 9.7.1 task reproduce three counterexamples:

| Case | Native cache result | Uncached reference | Implication |
|---|---|---|---|
| Warm success in a new process | Zero XML file entries for two unchanged inputs | Two complete file entries | The report contract changes even without a violation |
| Changed bytes with a preserved modification time | Success, no reported violation | Failure, one `AvoidStarImport` violation | Content changes can be missed; the Gradle fixture changes file size and proves the task executes |
| Changed custom rule implementation, same configuration | Success, no reported violation | Failure, one `StringFormattingCheck` violation | Gradle reruns for the changed checking classpath, but the internal engine cache can still hide the new rule |

All eight Gradle invocations produced their expected outcomes: six successes
and two deliberately failing uncached references. Every selected task executed;
an up-to-date skip did not substitute for the cache counterexample. Twenty-four
direct engine cases also cover additions, ordinary edits, deletions, renames,
rule parameters, optional suppressions, parser failure, repair, empty sources
and restarted processes. Two earlier dependency preparation failures are retained.

The engine's [cache property](https://checkstyle.org/config.html#Checker),
[processing implementation](https://github.com/checkstyle/checkstyle/blob/checkstyle-13.11.0/src/main/java/com/puppycrawl/tools/checkstyle/Checker.java)
and [cache implementation](https://github.com/checkstyle/checkstyle/blob/checkstyle-13.11.0/src/main/java/com/puppycrawl/tools/checkstyle/PropertyCacheFile.java)
explain the observed behavior. Versioned source artifacts, executable hashes and
runtime receipts are pinned locally; this conclusion does not rely on current
documentation alone. The rejected setting is not promoted into the native baseline.

## Residual opportunity and its narrow margin

The analysis reuses all 20 already consumed engineering transitions. Checkstyle
actions execute at ordinals 7, 19 and 20. The other 17 remain in the denominator;
the setup anchor supplies no savings. No new chronological timing was run.

| Quantity | Result | Interpretation |
|---|---|---|
| Complete native `Run build` time | 502.069 s | Denominator over all 20 transitions |
| Sum of Checkstyle action spans | 315.887 s | Overlapping spans; not workflow saving |
| Sum of nested worker spans | 312.960 s | Engine, Ant and report work combined; not isolated parsing time |
| Union of Checkstyle action intervals | 184.290 s | Removes overlap, but competing tasks still exist |
| Fixed-dependency model benefit with all Checkstyle actions removed | 29.849 s | Optimistic model, retaining other task durations and task overhead |
| Model mean and percentage | 1.49245 s / 5.9452% | Clears the unchanged 1 s / 5% G1 admission floors |
| Margin above the 5% floor | 4.74555 s across 20 transitions | Mandatory work and product cost can consume this narrow margin |

The model gives zero at ordinal 7, 24.220 s at 19 and 5.629 s at 20. It is
not a measured counterfactual or an absolute bound on shared-resource effects.
In particular, removing Checkstyle could change CPU contention and other task
durations; that requires the later paired experiment.

An additional direct-engine diagnostic processed all 8,339 native source
inputs: 4,960 main, 2,825 test and 554 integration sources. The native exclusion
of `module-info.java` leaves 8,338 XML entries. All three reports are byte-identical
to the retained native reports after the single, explicit checkout-root mapping.
Native XML callback time totaled 0.175565432 s in these three unpaired probes.
That attributes a component; it does not measure a candidate's state management,
report reconstruction, Gradle scheduling or net saving.

The first component probe retained every file but used fingerprint order,
which differed from native report order. That mismatch is preserved. The
corrected diagnostic uses the original native enumeration order and checks
byte equality; the comparator was not weakened. Future candidate execution
must obtain order from its current input enumeration, not a control report.

## Source and baseline contract

The [source contract](./contract.json) pins Elasticsearch ordinal 20,
`22d6425e9a44ec9e1aedc2785f4478b8019c851a`, its plugin/configuration and custom
checks. The three active custom checks read current file text or its syntax
tree. Disabled custom checks are not admitted by this finding. Compiled-source
classpath avoidance is already native; the engine and custom-rule classpath,
configuration, suppressions and source collection remain complete inputs.

The future N arm retains ordinary Gradle up-to-date checks and the build cache,
the already qualified ForbiddenPatterns cacheability correction in both arms,
and the explicit no-configuration-cache baseline from the prior native Spotless
failure. This admission changes none of the original workflow or economic gates.

## Six-repository denominator

All six frozen endpoints now have 101 commits and 100 verified first-parent
edges, with tree availability and Git connectivity checked. Full histories,
exact anchors, endpoint hashes and fetch receipts are in
[replication histories](./replication-histories.json).

| Fixed order | Repository | History prerequisite | Buildability / opportunity / replication |
|---|---|---|---|
| 1 | Apache Groovy | verified | unmeasured |
| 2 | Apache Kafka | verified | unmeasured |
| 3 | Spring Framework | verified | unmeasured |
| 4 | OpenTelemetry Java | verified | unmeasured |
| 5 | Micronaut Core | verified | unmeasured |
| 6 | Hibernate ORM | verified | unmeasured |

The first two causally admitted engineering prefixes in this fixed order may
eventually replicate a seed-positive mechanism. No repository was substituted
or selected by these metadata checks. Validation source and workflows remain
unconsumed; history availability is not evidence of transfer.

## Costs, recovery and next work

This admission used eight Gradle starts, zero nested Gradle starts and 30 direct
engine JVM starts: 24 engine cases, two setup failures, one component ordering
mismatch and three corrected component probes. Four `javac` preparations are
recorded separately. The previous phase retains its 31 reservations / 30 actual
Gradle starts; program totals are 39 reservations / 38 actual starts. No failure
was erased and the successful cold Gradle invocation was reparsed without a rerun.

Allocation and machine costs are retained with the receipts. Customer-required
equivalents must be charged to adoption or maintenance in G3; a future automatic
implementation cannot make them free. Human customer effort and monetary value
remain unmeasured. No customer outreach or publication occurred.

Recovery locator:
`/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/buildopt-product-viability-v1/task-state.json`.
The new clean source worktree is `worktrees/elasticsearch-checkstyle` under that
program state root, branch `bv1-elasticsearch-checkstyle-admission`; the old
native proof worktree remains untouched. All admission processes have ended.

The next implementation is defined in the [candidate boundary](./candidate-boundary.md).
Implement and prove BV-003/BV-004 first, then qualify the paired runner. Keep
ordinals 21-100 untouched until the candidate and runner are frozen. A passed
G1 licenses this bounded prototype; it does not establish its correctness,
payback, recurrence, adaptive value or commercial viability.
