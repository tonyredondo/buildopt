# Owner engineering readiness

Status: **partial**, 2026-09-09. Decision: **INSUFFICIENT_READINESS**.
The actual owner overhead experiment completed and failed its fixed imbalance
gate. G0 readiness and lifecycle value are not established. Engineering
ordinals 0..20 and seed validation ordinals 21..100 remain unrun.

Read the [result and next action](prefix-report.md),
[machine-readable decision](readiness-decision.json) and
[command evidence](command-index.json). This closes the bounded qualification
attempt; it does not complete BV-006 or reject C5's existing correctness proof.

This step integrates the frozen C5 output contract into the historical runner,
qualifies capture overhead on Elasticsearch, runs the paired engineering
prefix and freezes the subsequent confirmation. It preserves the C5 candidate,
economic thresholds and all earlier negative decisions.

## Verified inputs and current dependencies

| Deliverable | Status | Evidence or remaining proof |
|---|---|---|
| C5 retained-output adapter | verified | [Frozen reader qualification](owner-readiness-v5/owner-qualification.json): 20 equivalence/rejection/reuse cases, complete retained C5 pair and two actual native pairs, including 268 causal origins |
| Affected runner qualification | verified | Current unit/vet, all integration cases, race, 18 native Gradle starts and full Go build pass; [43 archived source inputs](owner-readiness-v5/source-manifest.json) match their retained bytes |
| Actual owner capture overhead | partial | All 20 native requests pass; wrapper p95 1.352567 ms passes 10 ms, but arm imbalance 588.682498 ms fails 100 ms; [independent reconstruction](owner-overhead-reconstruction.json) |
| Paired engineering ordinals 0..20 | blocked | Exact candidate preconditions checked for all 21 revisions; owner instrumentation remains unqualified |
| Six-repository inventory | verified | [Frozen subjects](subjects.json); no new candidate timings or selection |
| Adaptive cohort registration | partial | [Registration](adaptive-cohort/README.md); seed and three possible replication owners registered, three explicit gaps |
| Confirmation allocation, fresh roots and immutable protocol | blocked | Requires qualified owner instrumentation and measured prefix sizing |

Three incomplete calibration runs are retained. The first copied JDK modes incorrectly
under the process umask, before any Gradle start. The second stopped after one
Gradle start because a live disk-size traversal treated deletion of a native
temporary file as fatal. Focused reproductions and corrections pass. The third
completed a cold native build successfully, reporting 222.842 seconds on the
old, subsequently unqualified clock, with 1,301 root
tasks and 44,802 output files, then exposed a sizing limit: serial durable
archival copied 24.723 files/second. Four captures alone projected beyond the
remaining frozen calibration allocation. Its exact driver was stopped, its
owned service is inactive with no remaining processes, and all partial evidence
remains retained.

A balanced 512-file diagnostic copies the same durable native outputs twice
per method. Serial copies take 17.653 and 18.934 seconds; eight bounded copies
take 2.919 and 2.761 seconds. All 2,048 copied files match their source hashes,
sizes and modes, with independent inodes. The conservative paired speedup is
6.047x for research archival only. The correction retains per-file durability
and adds an output-directory barrier; actual owner proof must run afresh.

Calibration 04 completes four native requests and two equivalent pairs,
including real native reuse. Each complete root graph contains 1,301 tasks;
the reused captures retain 50,221 output entries per arm. Research capture
takes 356.357 to 445.565 seconds per request. These measurements size the next
run; they do not show product savings.

An independent exact-PID observer found the recorded native end at least
5.234162394 seconds after the native CLI exited. The old worker waited for a
whole-tree resource scan before stamping completion. Calibration 04 timing is
therefore **unqualified**. Its immutable outputs remain available for separate
correctness/reuse qualification. The corrected worker stamps child completion
first, cancels a bounded live directory scan and retains a complete postflight
disk check outside customer timing. The new independent owner observation
bounds completion delay below 5.996661 ms for its fixed target request. The
separate capture-imbalance gate remains failed.

The first reuse comparison took 110.067 seconds because it reread unrelated
historical raw outputs. The new reader indexes executed origins and verifies
each actual donated file, while retaining complete current-output checking and
independent checking of every historical attempt. It rejects mutated donor
bytes in an added adversarial case. Complete reanalysis takes 46.800 seconds
for the retained C5 pair, 9.700 seconds for the cold native pair and 12.830
seconds for the reused pair, proving 268 prior executed origins. A separate
audit verifies permitted dates within the actual producer task-end boundary.
The old independent checker also verifies all raw calibration records in
120.623 seconds; none of this qualifies the old native timing.

One initial full-reader invocation reaches its 180-second limit without a
result. Its precise cause was not observed. Two diagnostic executions, including
the production environment, complete in 46.9 seconds, followed by the exact
three-pair invocation and stricter date audit in 86.623 seconds without a source
change. The timeout and its possible metadata JVM start remain charged and
retained. It supplies no positive proof.

The first overhead setup refuses a generated Python bytecode file inside the
frozen source package before any Gradle start or run-root creation. Removing
only that task-generated cache restores the exact package hash and all 39
source files. Diagnostic imports of frozen readers must use Python `-B`;
`-I` ignores `PYTHONDONTWRITEBYTECODE`. The fresh native run is then started
under the original fixed measurement plan and completes all 20 requests;
unused reservations remain charged.

All test timeouts remain retained. A combined integration batch exhausted its
160-second limit while synchronizing evidence. All remaining cases pass when
bounded individually; the supported check command now uses that schedule,
without omitting cases or weakening their original limits. None of these
qualification attempts supplies lifecycle value evidence.

## Recovery

BuildOpt: `/home/tonyredondo/repos/github/tonyredondo/buildopt`, local `main` at
`b76ded08c952ebb386576fafce4ae2d8fdcc09f1`, with pre-existing uncommitted work.
No commit or publication is part of this step.

State locator:
`/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/buildopt-product-viability-v1/task-state.json`.
Phase root: the adjacent `bv006` directory. It contains the prospective
allocation, immutable command receipts/logs, source/executable versions,
isolated run roots, original failures and qualification fixtures. Consult the
recorded identity before resuming. All task-owned native units are closed;
there is no active driver or background replay.

The next bounded work in BV-006 is to attribute collector materialization,
serialization, callback writing and daemon variability. Existing raw records
show identical task outcomes; an isolated file-write diagnostic does not
support changing only channel opening as the primary fix. Preserve the failed
sample set and qualify a new version against the unchanged gates before
running the engineering prefix. BV-007 remains dependent on a completed G0
freeze and is not part of this qualification closeout.
