# Owner precision pilot result

Status: **verified pilot; owner readiness remains blocked**.
Decision: **INSUFFICIENT_PRECISION_NO_CONFIRMATION**.

All 80 fixed requests succeeded: sixteen warmups and sixty-four measured
requests, with sixteen plain/lean pairs per arm. Both final source states
match the frozen anchor/baseline/C5 contract and both native units are closed.
All measured root workflows retain 1301 tasks: 1256 UP-TO-DATE, 44 NO-SOURCE
and one EXECUTED. All three Checkstyles are UP-TO-DATE.

| Metric | Result |
|---|---:|
| N lower-median capture extra | 181.448647 ms |
| I lower-median capture extra | 220.694225 ms |
| Absolute median difference; original limit 100 ms | 39.245578 ms |
| Wrapper p95; original limit 10 ms | 1.509480 ms |
| N paired-difference sample standard deviation | 2405.640 ms |
| I paired-difference sample standard deviation | 703.821 ms |
| Formula-required confirmation pairs per arm | 25656 |
| Registered maximum pairs per arm | 512 |
| Confirmation starts | 0 |

The original point estimates fit the envelope. This pilot cannot qualify
owner timing. The registered variance-only sizing rule exceeds its maximum
by a large margin; no confirmation or engineering prefix is admitted.
The sizing formula is a planning approximation with declared assumptions,
not a guarantee that this many future requests would establish stationarity
or precise confidence under heavy-tailed variation.

## All paired differences retained

Lean minus adjacent plain native duration, in milliseconds. No trimming,
timing-based exclusions, or extra pairs were applied.

| Pair | N | I |
|---:|---:|---:|
| 0 | -169.841962 | 611.717107 |
| 1 | -346.574482 | -366.952902 |
| 2 | 906.452565 | 1447.568285 |
| 3 | 1224.671881 | 154.964376 |
| 4 | -7607.053990 | 220.694225 |
| 5 | -4878.156404 | 89.125055 |
| 6 | 408.958230 | 948.684779 |
| 7 | 563.937141 | 643.485123 |
| 8 | 940.522539 | 322.939984 |
| 9 | 132.599691 | -70.282706 |
| 10 | 181.448647 | 410.054769 |
| 11 | 921.808662 | -101.643468 |
| 12 | 23.400716 | 275.899824 |
| 13 | 1498.888100 | -866.785220 |
| 14 | 120.883945 | 2134.605835 |
| 15 | 397.035988 | 172.613035 |

The largest N differences are -7607.054 and -4878.156 ms. The corresponding
daemon-command differences are -7941 and -5019 ms. Each arm retains one
daemon throughout all 40 of its requests. There is no daemon restart in these
receipts. The variations therefore also exist inside recorded daemon-command
intervals; a JVM/OS/supervisor cause has not been isolated.

## Proof and cost

The independently reconstructed result is `precision-pilot-result.json` in
the task state. It checks every native receipt/log/hash, fixed order, graph
contract, root outcomes, current final worktree identity, raw median/p95
arithmetic, statistical runtime and closed units. A separate frozen-CLI
validation reread all registered inputs after native execution and passed.

The main CLI is byte-identical to v5. Production collector, worker and owner
reader bytes remain unchanged. Replay unit/vet, nine statistical rejection
tests and the legacy 20-request native fixture passed. The confirmation native branch
and its actual-owner interval outcome remain unexecuted because pilot sizing
refused admission. No new positive owner qualification is claimed.

This phase consumed 144 Gradle reservations and 123 actual invocations:
three preflight, twenty attribution, twenty successful fixture, and eighty
pilot starts. The difference retains one unused preflight reservation and
twenty reservations from a fixture launch that failed before native execution.
Both setup errors and their corrected launches remain in the evidence.
There were no nested Gradle starts or standalone Java helpers this phase.

Previous program charges remain: 354 reservations/307 actual Gradle starts.
Combined totals are 498/430, including 29 prior nested commands. The older
unresolved possible metadata JVM launch remains explicitly unresolved.

No candidate engineering 0..20 or validation 21..100 request was executed.
The [next diagnostic](next-step.md) targets the source of within-command
variation, with explicit CPU/wait/GC/JIT/observer evidence before any correction.
