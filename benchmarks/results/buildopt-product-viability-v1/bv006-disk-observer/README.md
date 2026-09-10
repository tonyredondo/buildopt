# BV-006 disk-observer correction and fixed control

Date: 2026-09-09. Status: **verified correction and control; BV-006 remains partial**.
Decision: `SUPERVISOR_CPU_REDUCTION_NOT_ESTABLISHED`.

The live resource observer no longer blocks driver/free-space/process checks
while counting the run tree. The scanner itself uses about 24% less CPU per
complete traversal. The fixed owner control nevertheless establishes no
reduction in total supervisor CPU or native build latency. Both uncertainty
checks cross zero. Full polling still consumes approximately one core while
requests execute. This correction is retained for responsiveness and cheaper
scans; it has not resolved owner timing readiness or demonstrated C5 value.

## Change and contract proof

The [production watcher](../../../../dev/history-replay/watch_linux.go) runs
independent process and disk loops with the original 100-ms cadence and one
complete size scan at a time. Cancellation joins both loops and retains real
errors, including a disk failure racing with child completion. Native end is
still stamped before the observer join. The cheap free-space check now also
runs with the driver/process check. Exact preflight/postflight accounting and
all thresholds remain unchanged.

The [scanner](../../../../dev/history-replay/files.go) reads bounded name
batches and obtains metadata relative to the open directory, avoiding full-path
lookup and FileInfo allocation for every entry. It still counts every regular
file's logical size, counts hard links by pathname, ignores symlinks, tolerates
vanished temporary descendants, and fails on root loss or other I/O errors.
Sparse-file, exact-limit and cancellation tests cover those boundaries.
The [contract note](./analysis/contract.md) records the old and new behavior.

The actual consuming loop failed the slow-scan regression before the split and
passes afterward. Unit/vet/statistical checks and race checks pass. The complete
integration suite passes 104 actual non-Gradle fixture requests, including
native failure/fallback, deadline, detached descendants, driver death, lost
state, recovery and evidence rejection. Six actual Gradle fixture requests
verify EXECUTED, UP-TO-DATE and FROM-CACHE behavior with complete output/checker
proof. Two more Gradle requests verify old/new worker dispatch, identical
native output and lifecycle closure. The full Go build also passes.
No dependency, JVM configuration, worker-count or host-setting change was needed.

## Static scan versus complete requests

The fixed static order was old/new/new/old on the same closed 10,955,491,091-byte
run tree. Every call counted exactly those bytes. Mean CPU falls from
1938.897 to 1465.996 ms (24.390%); mean wall falls from 1901.638 to 1453.455 ms (23.568%).

These are two static repetitions per version, not native build savings.
The retained original CPU profile has 84.13% of samples in system calls.
A cheaper scan can simply run more often when its next tick is already pending.

The fresh owner control completes exactly 32 successful plain builds: eight
warmups and eight measured requests per supervisor version, alternating order
by cycle. N denotes the old supervisor; I denotes the corrected supervisor.
**Both run the same native baseline, with no C5 candidate applied.** Elasticsearch
is fixed at `53a80bec683ad0b065ed9ebe6a57f984e6a91ed1`, with the previously
registered ForbiddenPatterns baseline delta. Commands, runtime/build JDKs,
acquisition inputs, native cache, Configuration Cache setting and CPU affinity
0-7 are identical. Each arm has its own retained state and one daemon throughout
(N 968981; I 972291). No JFR or standalone JVM helper was started.

```text
./gradlew --offline --no-scan --max-workers=8 :server:precommit --continue --no-configuration-cache --build-cache --console=plain --daemon
```

Ordinary medians over eight measured requests per version, in milliseconds:

| Metric | Old | Corrected |
|---|---:|---:|
| Native wall | 5531.671 | 5396.445 |
| Daemon command wall | 5166.500 | 5038.500 |
| Supervisor CPU during request | 5430.815 | 5347.886 |
| Sampled daemon CPU | 12375.000 | 10520.000 |

Paired old-minus-corrected means and 95% circular moving-block bootstrap
intervals (10,000 draws, fixed seed 20260909). Positive means a lower corrected
cost; both declared block lengths are retained. With only eight chronological
pairs, these are exploratory intervals, not G0 or population-level validation.

| Metric | Mean difference, ms | Point fraction | Block 2 interval, ms | Block 4 interval, ms |
|---|---:|---:|---|---|
| Native wall | 63.062 | 1.132% | -39.725 to 173.434 | -10.747 to 136.872 |
| Supervisor CPU | 76.004 | 1.379% | -24.787 to 185.323 | -7.712 to 159.720 |

The 1.132% native point estimate is not an established speedup. CPU medians
remain around 5.4 seconds for requests lasting around 5.4 seconds. This control
cannot establish the cause of the previous seconds-scale deviations, and
must not be accumulated with old qualification samples until a gate passes.

## Fidelity, failures and costs

All 16 measured requests preserve the same 1,326 task outcomes as the frozen
plain reference: 1,301 root tasks (1,256 UP-TO-DATE, 44 NO-SOURCE, one EXECUTED)
and 25 included-build tasks (22 UP-TO-DATE, three NO-SOURCE). Source was checked
before every request and after the last one. Plain requests do not recapture
graph edges or replace the earlier complete G2 output proof.

Measured daemon CPU coverage is at least 96.1150%; maximum snapshot gap is 163.430993 ms. There are 0 snapshot gaps. Maximum within-request UTC/BOOTTIME spread is 312,791 ns.

The registered 90%/500-ms quality bounds pass. Missing-middle, missing-tail and
invalid CPU-counter cases are rejected by the independent checker. The first
positive rejection-test input was a cold fixture whose daemon starts after the
CLI; it correctly failed the measured-owner coverage rule. The final rejection
proof uses an already-verified warm raw record, mapping only the exact CPU
counter field names. No new native run or threshold change was used for that
checker proof.

An invalid generated experiment identifier was rejected before native launch
and corrected to the existing `FIXED_NI` protocol value; the separate subject
and design identify this supervisor control. A summary-note syntax error did
not change execution: the launcher independently verified all prerequisite
receipts before starting the owner, and the summary was saved before its first
native request. All preparation/checker failures and the expected red test
remain evidence. There was no failed production correction and no owner restart.

This phase uses 40/40 Gradle reservations and actual starts (eight fixture plus
32 owner), 104 non-Gradle fixture requests, five standalone static scans, zero
standalone JVM helpers and zero new nested Gradle starts. The combined program
ledger is 604 reservations / 505 actual Gradle starts, including the same 29
older nested commands. The older unresolved possible metadata JVM stays
unresolved. The three-hour / 60-GiB / 40-GiB-free allocation is closed; all owned
processes are closed and worktrees retained. No commit, push or cleanup occurred.

## Evidence and recovery

- [Registered control design](./inputs/control-design.json), [execution freeze](./inputs/control-freeze.json),
  [actual qualification](./analysis/qualification.json), [checker rejection proof](./analysis/checker-rejection-proof.json).
- [Complete raw reconstruction and intervals](./analysis/owner-result.json),
  [static comparison](./analysis/static-comparison.json), [summary](./analysis/summary.json).
- [Closeout](./closeout.json), [origin map](./origin-map.json),
  [independent audit](./independent-audit.json), [manifest](./evidence-manifest.json),
  [external seal](../bv006-disk-observer-seal-audit.json).

BuildOpt remains at `main`, `b76ded08c952ebb386576fafce4ae2d8fdcc09f1`, with
unrelated entry work preserved. The task state is
`/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/buildopt-product-viability-v1/task-state.json`,
key `engineeringPrefix.diskObserver`. Its `stateRoot` is the sibling
`bv006-disk-observer` directory; both exact subject worktree identities are
registered there. The current production code and old/new experimental packages
are separately retained in this bundle.

The [next bounded step](./next-step.md) isolates research-observer CPU from the
native CPU allocation while preserving live guards. C5 engineering 0..20,
validation 21..100, H3 and other-repository native value tests remain unrun.
