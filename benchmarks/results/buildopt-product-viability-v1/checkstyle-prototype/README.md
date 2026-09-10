# Checkstyle prototype and correctness proof

Date: 2026-09-08. Program: `BUILDOPT-VIABILITY-V1`.

**BV-003 and BV-004 are verified; G2 passes for the frozen Linux owner.**
The candidate avoids rechecking unchanged successful files while preserving
the complete native Checkstyle result. It survives the registered failure,
invalidation, cancellation and removal cases. There is no measured lifecycle
saving or product-viability result yet. C5 remains in progress; BV-005 is next.

The [decision](./correctness-decision.json), [closeout](./closeout.json) and
[evidence manifest](./evidence-manifest.json) bind this result. Follow the
[execution tracker](../../../../docs/plans/buildopt-product-viability-v1-tracker.md)
for subsequent work. Earlier negative ForbiddenPatterns and H2 decisions and
the [Checkstyle admission](../checkstyle-admission/README.md) remain unchanged.

## Candidate and actual inputs

The proof uses Elasticsearch engineering ordinal 20,
`22d6425e9a44ec9e1aedc2785f4478b8019c851a`, Gradle 9.7.1, Checkstyle 13.11.0
and Temurin 21.0.12+8 on Linux x86_64. The candidate changes five source files;
its separate native-baseline correction is identical in both worktrees.
Full source inventories differ only at these five paths.

The [contract](./candidate-contract.md) defines exact admission and fallback.
The [source manifest](./candidate-manifest.json), [patch](./candidate.patch),
[inverse](./inverse.patch) and [guarded application tool](./apply-candidate.py)
make the correction reviewable and removable. The native task type, source
collections, exclusions, rule engine, dependencies and reports remain in use.
Successful content hashes form local history; this is not another output cache.

Only the registered `:server` tasks opt in. Unknown rule/provider versions,
unsupported reporting/configuration or unavailable optional state use native
checking. Java 25 and Checkstyle 10.24.0 were exercised as native fallbacks;
other operating systems and incremental runtime versions are unverified.
The inherited native baseline disables Configuration Cache after its previously
recorded Spotless reuse failure. This phase does not qualify Configuration Cache.

## Observed correctness

| Proof | Result |
|---|---|
| Registered C01-C20 matrix | All 20 cases verified; 67 fixture observations, with passing and failing variants |
| Actual work avoided | A one-file edit processes 1 file in I versus 2 in N; a forced valid repeat processes 0 versus 2, with identical complete XML |
| Ordinary native avoidance | `UP-TO-DATE`, `NO-SOURCE` and cache restoration preserved; the first edit after restored output establishes fresh history |
| Owner regression tests | Native control 9/9; final candidate 13/13, zero skipped/failing/error methods; actual nested TestKit execution included |
| Complete owner success | `:server:precommit` passes; final candidate executes all 1,045 actionable tasks with `--rerun-tasks --no-build-cache` |
| Complete root graph/output contract | 1,301 tasks; 50,225 N and 50,226 I entries, all retained bytes rehashed; no unexplained differences |
| Required Checkstyle reports | All 8,338 ordered file entries retained: main 4,959, test 2,825, internal cluster tests 554 |
| Complete owner failure | The same controlled wildcard import produces the same Checkstyle diagnostic, exit 1 and failed tasks in both arms; source restored exactly |
| State and rule boundaries | Same-mtime changed bytes, changed checker bytecode, suppressions, missing/tampered output and corrupt/unavailable/transplanted history handled correctly |
| Process lifecycle | Same Gradle daemon and restarted processes verified; real native/candidate worker cancellations exit 143 and terminate owned workers; recovery fully checks input |
| Removal | Drift/partial source refused before writes; exact inverse and repeated removal verified; native checking works with adapter classes absent from rebuilt jars; reapplication verified |
| Source formatting | Native formatter selects and checks exactly the five final Java files |

The [matrix](./correctness-matrix.json) links each observation to its raw
result, source/runtime inputs, XML, processing events and before/after state.
The diagnostic observer was qualified against ordinary unobserved native
passing and failing results; it is not qualified for primary value timing.
[Reconstruction](./reconstruction.json) independently rehashes 131 fixture
reports and reconstructs test/start counts from retained evidence.

The final [full-output comparison](./owner-comparison/result.json) finds
50,020 exact entries. Its 205 other common entries consist of 64 previously
enumerated manifest-date cases, three lexical Checkstyle root mappings, one
RAT root/time mapping and 137 complete compiler-state comparisons. Date and
RAT rules require actual producer provenance. The sole additional output is
the exact generated adapter configuration. Seven original Go comparator tests
and 24 native-metadata checks qualify the permitted projections; unknown
fields and output differences are not ignored.

## Defects and evidence validity

Two compatibility defects were reproduced and corrected on their first attempt:
a checking provider with the original rules but no adapter classes, and caller
replacement of properties that removed the private state reference. Their
reproductions and corrected native fallbacks are retained in
[provider evidence](./adapter-provider-fix.json) and
[state-binding evidence](./state-binding-fix.json).

Earlier core tests retain their actual executable versions.
[Instruction comparison](./core-proof-validity.json) and
[source-change analysis](./state-binding-proof-validity.json) establish which
proof remains valid. The affected admission cases, owner tests, full workflow,
output comparison, inverse/reapplication and native format check ran again on
the final version. No failed result was replaced by a later passing receipt.

Initial formatter selections that matched no files were invalidated and
replaced by explicit five-file selection and guarded formatting. The retained
malformed-Java case produces an empty report in both native and candidate arms;
the reconstruction audit checks that actual failure output. A diagnostic
inverse driver originally assumed the wrong owner JAR location; its completed
native invocation was retained and inspected without a duplicate run.

## Cost, retention and next step

The [resource ledger](./resource-ledger.json) records 121 reservations and
121 actual Gradle invocations, including five nested TestKit starts. The
program total is 160 reservations and 159 actual invocations. This phase also
used five standalone javac preparations and three metadata-projection JVMs;
it started no separate direct-engine probe JVMs. These are research costs,
not customer savings. All owned native units have ended.

The allocation was prospectively extended from 120 to 124 Gradle starts under
the owner's standing budget authorization. The six-hour deadline, 120 GiB
new-state ceiling and 40 GiB minimum-free requirement stayed unchanged.
[Footprint evidence](./footprint.json) conservatively measures about 15.76 GiB,
including both worktrees and retained captures, with over 809 GiB free.
Native jobs enforce free-space, log-size and runtime bounds; whole-directory
footprint was checked at checkpoints, not continuously enforced.

Compact receipts, full output inventories, selected reports and tool sources
are portable here. The [raw-state index](./raw-state-index.json) binds larger
complete traces/output trees to their retained host-local paths. The recovery
locator remains
`/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/buildopt-product-viability-v1/task-state.json`.
The BuildOpt branch and existing working changes are preserved; nothing was
staged, committed, published or uploaded.

**Next: BV-005.** Implement and qualify the resumable historical replay
instrument: exact first-parent advancement, separate writable state for N/I,
complete outputs, process/start accounting, interrupted-pair recovery and an
independent checker that rejects forged or incomplete positive summaries.
Then BV-006 exercises the 20-transition engineering prefix and freezes the
confirmation. Validation ordinals 21-100 remain untouched. The two full
replications, 5% net-workflow and one-second-per-scheduled-transition floors,
payback/latency requirements, two additional owners, adaptive controller and
customer gates remain unproved and unchanged.
