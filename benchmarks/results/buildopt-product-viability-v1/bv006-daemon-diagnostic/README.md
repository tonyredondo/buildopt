# BV-006 owned-daemon and supervisor diagnostic

Date: 2026-09-09. Status: **verified diagnostic; BV-006 remains partial**.
Decision: `SUPERVISOR_SCAN_COST_CONFIRMED_VARIABILITY_CAUSE_UNRESOLVED`.

The supervisor spends approximately one CPU core repeatedly counting the run
tree while native builds execute. Two static calls to the unchanged production
scanner reproduce that cost with both owner daemons closed. This is a concrete
instrument defect to address; it does not establish how much native wall time
would improve or explain the earlier seconds-scale qualification deviations.
The [previous precision failure](../bv006-attribution/pilot-result.md) remains
unchanged. G0 timing and product value are unqualified.

## Fixed experiment and observed results

The fresh recovery batch completed all 32 plain owner requests: eight warmup
and eight measured requests per arm, with alternating N/I order by cycle.
All native exits succeeded. Each arm retained one daemon (N 853135, I 855399).
The frozen Elasticsearch anchor is `53a80bec683ad0b065ed9ebe6a57f984e6a91ed1`;
N retains the registered baseline and I adds the exact five-file C5 patch.
Pinned runtime/build JDKs, eight workers, CPU affinity 0-7, native cache and
the existing daemon policy were retained. The command was:

```text
./gradlew --offline --no-scan --max-workers=8 :server:precommit --continue --no-configuration-cache --build-cache --console=plain --daemon
```

All 16 measured requests have the same 1,326 task outcomes as the frozen plain
reference: 1,301 root tasks (1,256 UP-TO-DATE, 44 NO-SOURCE, one EXECUTED) and
25 included-build tasks (22 UP-TO-DATE, three NO-SOURCE). Plain requests do not
recapture graph edges. Source was checked before every request and after the
last request; this diagnostic does not replace the existing complete G2 output
proof. Both first cold requests have process/supervisor data but no JFR.
JFR starts between requests and covers the remaining warmups and measurements.

Ordinary medians over eight measured requests per arm, in milliseconds:

| Metric | N | I |
|---|---:|---:|
| Native wall | 5390.813 | 5296.579 |
| Daemon command wall | 5033.000 | 4931.000 |
| Supervisor CPU | 5372.796 | 5257.368 |
| Worker CPU accrued during disk spans | 5371.200 | 5255.876 |
| Longest scan per request | 2065.665 | 2059.273 |
| Sampled daemon CPU | 11640.000 | 10115.000 |
| GC pause union | 207.865 | 222.810 |
| JIT active interval union | 1366.650 | 1479.459 |
| Summed JIT thread duration | 1729.243 | 1976.317 |

Across the 16 measured requests, native wall totals 86,024.175 ms and
supervisor CPU totals 85,231.006 ms. Of that CPU, 85,204.424 ms accrued during
disk-scan spans. These are whole-worker CPU counters, including any concurrent
worker goroutines; they are not per-thread attribution. The independent static
reproduction isolates the scanner itself: two unchanged `treeBytesUntil` calls
each counted 10,955,491,091 bytes, taking 1,818.280/1,817.864 ms wall and
1,848.653/1,849.779 ms CPU. Already-cancelled traversal returned in 0.012180 ms;
one free-space probe took 0.012080 ms. No Gradle or Java process was started by
this reproduction. See the [raw result](./analysis/static-scan-reproduction.json).

CPU time is not recoverable wall time. The roughly 94-ms difference between
the N/I wall medians is neither a C5 saving nor an owner-readiness result.
The earlier large deviations were not reproduced in this fixed profiled batch.
GC pauses and JIT remain active; their overlapping intervals do not partition
the critical path. Some higher-CPU requests include sampled activity on
`gradle-enterprise-worker-*` threads. That observation establishes neither
causation nor an upload. Idle-thread waits and native samples cannot be counted
as blocked critical-path time. No JVM, heap, worker or plugin tuning is admitted.

## Observation quality and retained failures

The measured daemon CPU coverage minimum is 96.3182%, above the preregistered
90% floor; the maximum snapshot gap is 100.895229 ms, below 500 ms. There are
zero process-snapshot gaps in the completed batch. Maximum within-request
UTC/BOOTTIME mapping spread is 376,171 ns. Both JFR files contain required
event families and no DataLoss events. Events below the 10-ms blocking/I/O
threshold are not observable through those event types; compilation has a
1-ms threshold. Process CPU is tick-quantized and omits sampled boundaries;
`/proc/PID/schedstat` describes only the main thread.

The initial owner attempt performed one successful cold native build, then
the new diagnostic sampler failed reading a process that exited during the
snapshot (`ESRCH`). Its artifacts and all 32 reservations remain charged.
A real kernel reproduction failed against the original consumer and passed
after a diagnostic-only correction. Only typed ESRCH/ENOENT at the three
registered process files becomes an explicit gap; ownership, permission and
I/O errors remain fatal. Unit, vet and race checks passed. One fresh full batch
was declared before execution, with no row reuse or further owner restart.
Production `dev/history-replay` source was not changed.

The initial fixture completed two actual Gradle requests with identical
outputs/outcomes and representative CPU/JIT/GC/wait events. Coverage rejection
checks reject a missing middle and missing tail without altering raw evidence.
The standard N JFR JSON expansion exceeded the wrapper's 1-GiB bound: the JDK
exited zero but the wrapper failed. This is preserved explicitly. A compact
export of the same binary recordings was qualified against all 1,449 fixture
events and 5,028 shared frames. Two representation corrections preserve raw
thread IDs and class names; all initial exports remain evidence. Checker
corrections use the exact existing cancellation sentinel and separate root
from included-build task IDs. No gate or sample selection was changed.

## Evidence, costs and recovery

- [Decision and complete descriptive statistics](./analysis/decision.json),
  [independent reconstruction](./analysis/owner-result.json),
  [native background activity](./analysis/native-background-activity.json).
- [Registered recovery design](./inputs/design-v2.json),
  [frozen inputs](./inputs/freeze-v2.json),
  [sampler qualification](./analysis/sampler-qualification.json),
  [compact export proof](./analysis/compact-export-proof.json).
- Raw recordings `raw/owner/profile-N.jfr` and `raw/owner/profile-I.jfr`,
  requests, receipts and both source versions remain in the local evidence bundle.
  The recordings are not included in Git; their sizes and checksums are in the
  [publication record](../publication.md). The redundant oversized standard JSON
  remains local, with its digest in [the failure record](./analysis/oversize-export.json).
- [Closeout](./closeout.json), [origin map](./origin-map.json),
  [manifest](./evidence-manifest.json) and
  [external seal audit](../bv006-daemon-diagnostic-seal-audit.json).

This phase charged 66 Gradle reservations and 35 actual starts (two fixture,
one failed-owner diagnostic, 32 fresh owner), plus 16 standalone diagnostic
helper JVMs and three non-JVM reproduction children. The first batch's 31
unused reservations remain charged. No new nested Gradle starts occurred.
Combined program totals are 564 reservations / 465 actual Gradle starts,
including the same 29 older nested commands. The older unresolved possible
metadata JVM remains unresolved. The three-hour / 60-GiB / 40-GiB-free bounds
were retained; helper capacity is exhausted and no further native run belongs
to this phase. All owned services and children are closed; worktrees remain.

Recovery locator: `engineeringPrefix.daemonDiagnostic` in
`/home/tonyredondo/repos/github/tonyredondo/buildopt/.tools/state/buildopt-product-viability-v1/task-state.json`.
The raw phase is the sibling `bv006-daemon-diagnostic` directory. BuildOpt
remains on `main` at `b76ded08c952ebb386576fafce4ae2d8fdcc09f1` with the user's
pre-existing dirty work preserved. No publication or cleanup was performed.

Continue with the [bounded supervisor correction and control](./next-step.md).
C5 engineering ordinals 0..20, all validation ordinals 21..100, H3 and the
other repositories' native builds remain unrun.
