# Bounded external persistence for the live sampler

Identity: BuildOpt local main at b76ded08c952ebb386576fafce4ae2d8fdcc09f1,
Linux x86_64. Durable locator: `../task-state.json`, key
`engineeringPrefix.bufferedObserver`. Source is a new copy of the sealed live
diagnostic; no native worktree, production edit, host setting or publication.

Supported cause: in two actual builds, a synchronous file write occupied over
98% of the largest sampling interval. Controlled 2,104-ms write delay previously
caused a 2,105-ms sampling gap while both heartbeats continued. That retained
source-bound case is the failing regression baseline; no extra build is needed.
The historical 2,104-ms event itself remains causally unresolved.

| Step | Status | Acceptance |
|---|---|---|
| R1 | verified | Preserve source-bound write delays and all old rejection/gate outcomes |
| R2 | verified | Nonblocking admission into a byte/record-bounded queue; one owned external writer, exact acknowledgements, explicit errors and bounded drain/termination |
| R3 | verified | Live collection continues during 2,104-ms writes; normal and failure paths retain exact order/hash/count or explicit incomplete state, with no surviving helper |
| R4/R5 | blocked | C5/screen integration and original S1/G0 proof remain separate; zero native allocation |

The sampler retains the existing JSON encoder, live collection, 100-ms cadence,
phase clocks and 10-ms heartbeats. Its write boundary now measures queue admission,
not disk time; disk-write clocks and hashes arrive in independent acknowledgements.
Pending data includes the frame in flight until its acknowledgement. Defaults:
128 pending records, 16 MiB pending payload, 512 KiB per frame and 512 MiB total.
An extra helper frame of at most 512 KiB and the existing bounded phase/heartbeat
arrays are accounted separately. Reject saturation immediately; never overwrite
or discard an accepted frame silently. Framing is bounded and order checked.

An external process is justified by cancellation: the driver can close its
pollable transport pipes and terminate/reap a stalled writer. A blocked regular
file write in an in-process goroutine cannot be assumed cancellable. The helper
stays on the driver's CPU mask and inherits the same owned cgroup; it observes a
separate lifetime pipe, whose EOF rejects parent death even while a write sleeps.
The original native affinity, full disk guard and ownership code remain unchanged.

Ordinary stop ends capture, then drains accepted frames for up to 5 s; final
acknowledged count/bytes and EOF are required for success. Errors/timeouts stop
and reap the helper and retain accepted/acknowledged/pending counts. Abrupt driver
death may lose its in-memory queue: missing final receipts must make that attempt
unusable, never a zero-loss success. Whole capture/drain/close clocks and helper
CPU are research cost. This is not a product optimization or S1/G0 claim.

Fixed tests (15 cases including one race run): normal, writer-delay, writer-error,
short-write, record-saturation, byte-saturation, total-output-limit, cancel-before,
cancel-capture, cancel-drain, drain-timeout, writer-exit, driver-exit, scan-error,
and race writer-delay. Each is an owned non-Gradle unit, at most 20 s; 180 s total
proof wall, 1 GiB phase output and 40 GiB minimum free. Qualification must inspect
raw payloads, acknowledgements, phase records, metrics, errors and process closure.

## Verified proof and retained correction

`analysis/qualification.json` reconstructs all 15 final cases from raw bytes,
acknowledgements, phase clocks and owned process identities. The controlled write
delay remains 2,104.204626 ms, but the largest sampling gap is 100.774358 ms;
all 40 records persist in order. Peak queue occupancy is 22 records / 77,237 bytes.
The earlier synchronous control reached 2,105.065581 ms. This comparison establishes
sampling continuity under the injected delay, not product/build acceleration.
The race case also retains all 40 records and has no race detector finding.

Record saturation includes the in-flight frame: capacity three accepts and drains
five total records, then explicitly rejects the next admission. Byte and cumulative
quota cases use a one-byte ceiling and reject the first payload; they do not measure
near-limit throughput. Error, short-write, process exit and drain timeout retain
their pending counts. Short-write retains its 1,750-byte partial tail. The 200-ms
drain deadline terminates and reaps the stalled helper in 201.102646 ms. Killing
the driver leaves no final report and is correctly classified incomplete; both
helpers close, with no claim about the lost in-memory queue's exact size.

The first normal test completed 40/40 persisted records but its fixture reader
failed on `unknown $.writeBoundary`: strict decoding into a partial projection was
incorrect. Only that test reader changed to ordinary JSON projection. A prospective
amendment added one case and one compiler command; all original time/disk bounds
and zero native starts remained. The failed run, logs and source/binaries are
retained in `runs/normal-reader-v1`, `control/v1`, and their receipts. The initial
controller end stamp was not retained; its service runtime is 4.294 s in the log.
`inputs/v1-relocations.json` resolves the original source/executable bindings to
the byte-identical retained v1 artifacts. Product correction failures: zero;
fixture-reader corrections: one. Final proof binds the v2 source and both binaries
through `inputs/qualification-freeze.json`.

All five pinned Go commands pass: v1 and v2 normal/race compilation, then final
vet. Final case wall time is 25.261212383 s, plus the retained v1 service runtime.
Totals: 16 case drivers, 16 writer helpers and 16 heartbeat helpers; zero Gradle,
JVM or screen starts. `analysis/checker-proof.json` verifies literal outcomes and
rejects six altered/incomplete artifact sets. `inputs/source-change.diff` records
the complete task-copy delta. Only two existing task-copy files and two new test
files differ; production and other frozen source files remain unchanged.

## Proof boundary and next dependency

The payloads here are about 3.5 KiB from live fixture processes, not the actual
owner's larger population. Queue payload bounds are not a hard bound on total
Go RSS. The configured unit memory ceiling is 512 MiB. Exact owner overhead,
endpoint coverage, chronological CLI integration and C5 cost/output consumption
remain unqualified. The old 2,104-ms event remains unexplained and the previous
cold 504/603-ms endpoint gaps remain rejected. No old proof is silently promoted.

R4 must bind this observer to the actual replay consumer, qualify complete and
failed output/lifecycle/cost paths without Gradle, then establish original S1/G0
owner criteria under a separately declared native allocation. CPU screen remains
0/36; no automatic extension and all 80 validation transitions remain untouched.
