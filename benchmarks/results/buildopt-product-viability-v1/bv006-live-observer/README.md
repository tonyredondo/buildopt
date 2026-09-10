# Live sampler diagnostic: real write delays localized

Date: 2026-09-09. Program: BUILDOPT-VIABILITY-V1, BV-006 partial.

The actual sampler now records its live `/proc` scan, JSON encoding, synchronous
file write and return phases, with process CPU, monotonic/boottime clocks,
internal/external heartbeats and Go runtime pause histograms. Changes exist only
in the [task-bound source](./control/source/live_observer_test.go). Production and
all earlier sealed results retain their bytes.

**The two real builds expose synchronous write delays of 323/341 ms.** Those
writes account for 98.78%/98.94% of their 327/344 ms sampling intervals. Both
heartbeats continue at approximately 10 ms during each event. This localizes the
delay to the writer boundary; it does not distinguish filesystem internals from
writer-thread scheduling, or establish the cause of the older 2,104 ms rejection.

| Arm | Cold native wall | Payload records | Maximum live scan | Maximum write | Maximum inter-sample gap | Write share of largest gap |
|---|---:|---:|---:|---:|---:|---:|
| N | 211.804 s | 2121 | 6.378 ms | 323.423 ms | 327.407 ms | 98.78% |
| I | 220.229 s | 2201 | 11.374 ms | 340.541 ms | 344.203 ms | 98.94% |

N shares native CPUs 0–7 with its supervisor; I places the supervisor on CPU 8.
Both use eight native workers/CPUs, the same source/baseline and plain offline
`:server:precommit`. No Checkstyle candidate is installed in either row. External
heartbeat CPU 10 is research instrumentation. Both native builds succeed with
the same 1,326 task outcomes (1,045 actionable tasks each); 813,911 native/observer
thread masks are checked. No snapshot-exit gap or missing payload is observed.
The two cold wall times are diagnostic observations, not an optimization effect.

**The original observation gate is not passed.** Neither sampler has an
inter-sample gap over 500 ms, hence the scoped result
`LIVE_GAP_OVER_500MS_NOT_REPRODUCED`. However, the endpoint-inclusive daemon
metric is 503.587931/603.431035 ms from native start to its first daemon snapshot.
These were prospectively registered cold rows, not warm measured qualification;
both exceed that original 500 ms criterion. Keep both rows and all thresholds.
The historical 2,104 ms rejection remains unexplained and rejected. S1/G0, product
timing and lifecycle value remain unqualified; the CPU screen is still 0/36.

## Focused behavior proof

Nine fixed non-Gradle cases exercise the same live sampler in real owned cgroups:
normal collection, 2,104 ms writer delay, process suspension, writer failure,
cancellation before sampling and during a write, scan failure, exited-process
recovery, and bounded record exhaustion. All pass independent reconstruction:
128 complete payloads and 130 phase rows, with the two deliberate failures
retained. The original synchronous writer finishes its active write before
cancellation completes. Ticker coalescing during a deliberate pause is visible
as a gap; exact payload accounting does not claim every scheduled tick ran.

Normal fixture scan/encoding/write maxima are 0.605/0.057/0.036 ms. Deliberate
writer delay is correctly separated from whole-process suspension. Every
heartbeat exits, including on failure/cancellation. Source-bound compilation
and `go vet` pass using pinned Go 1.26.5; no dependency installation was needed.
An initial compile setup error from an unused import is retained; it started
no probe or Gradle and was corrected before qualification.

## Cost and limits

Exactly two Gradle reservations and actual starts, nine Go fixture processes,
one owner driver, two owned Go workers and eleven external heartbeat processes.
No standalone helper JVM, nested build, JFR, extra warmup or replacement run.
Three compiler/check commands include the failed initial compilation.
The two-start owner envelope is 1,168.165 s including fresh state preparation and
verification, versus 432.033 s summed native wall; the remainder is research work,
not an inferred product overhead. Driver CPU during the two sampling intervals
is 6.766/7.237 s; external heartbeat CPU is 0.424/0.409 s. Supervisor CPU remains
188.201/216.601 s, roughly 0.89/0.98 of one core over native wall. This diagnostic
has not established a reduction in that separate observer cost.

The fixed 90-minute/36-GiB phase bounds and 40-GiB minimum free space apply.
The two-start native allocation is exhausted and closed. Program Gradle totals:
646 reservations / 547 actual, retaining the older 29 nested starts and unresolved
possible metadata JVM. All 80 validation transitions remain untouched.

## Evidence and next work

- [Registered two-start design](./inputs/owner-design.json) and [frozen manifest](./inputs/owner-manifest.json).
- [Nine-case proof](./analysis/qualification.json) and [complete owner reconstruction](./analysis/owner-result.json).
- [Write-boundary diagnosis](./analysis/write-diagnosis.json) and [cold daemon boundaries](./analysis/daemon-boundaries.json).
- [Source diff](./inputs/sampler-change.diff) and [implementation notes](./implementation.md), [raw owner records](./owner/run/attempts/diagnostic-01-N-plain/sample.json), and [second row](./owner/run/attempts/diagnostic-02-I-plain/sample.json).
- [Closeout](./closeout.json), [independent audit](./independent-audit.json) and [seal audit](../bv006-live-observer-seal-audit.json).
- [Next: bounded separation of sampling and file writes](./next-step.md).

Native worktrees and private caches remain host-local, identified in the raw
receipts and durable task state. Portable evidence contains source, records,
bindings, logs and audits; it does not duplicate the full native cache trees.
