# Bounded sampler persistence: controlled proof

2026-09-09. Decision: **`BOUNDED_PERSISTENCE_SEPARATION_VERIFIED`** for R2/R3.
BV-006 remains partial; original observation, S1/G0 and product value remain
unqualified. CPU screen remains **0/36**. This phase used **zero Gradle/JVM starts**.

The live sampler now queues bounded payloads for a separate owned writer. Under
the same injected 2,104-ms third-record write delay, its maximum sampling gap
falls from 2,105.065581 to **100.774358 ms**, with all 40 records persisted in order.
This is a controlled sampling regression result, not a build-speed improvement.
The [previous real builds](../bv006-live-observer/README.md) supported separating
writes from collection; the cause of the historical 2,104-ms event is still unknown.

## Behavior and proof

The sampler keeps live process collection, JSON encoding, 100-ms cadence and both
10-ms heartbeats. Queue admission and actual file-write time have separate clocks.
The writer acknowledges exact order, size and hashes. Defaults bound pending data
to 128 records / 16 MiB, each frame to 512 KiB and cumulative payload to 512 MiB.
The in-flight frame remains pending until acknowledged. Saturation rejects the
attempt explicitly; it cannot silently drop an accepted record. The helper has
an additional bounded frame; queue payload is not a hard cap on total Go RSS.

| Case | Observed result |
|---|---|
| Normal | 40/40 persisted; maximum sampling gap 100.756 ms |
| 2,104-ms writer delay | Actual write 2,104.205 ms; maximum gap 100.774 ms; 40/40 persisted; peak queue 22 records / 77,237 bytes |
| Race detector with writer delay | 40/40 persisted; maximum gap 100.902 ms; no race finding |
| Writer error / premature exit | Two complete records, one pending; explicit error and closed helper |
| Partial write | Two complete records plus 1,750-byte partial tail; incomplete data retained and rejected |
| Record saturation | Capacity three includes in-flight data; five accepted records drain; sixth admission rejected |
| Byte / cumulative limit | One-byte ceilings reject the first payload explicitly |
| Cancellation before capture | Zero records; owned helpers close |
| Cancellation during capture / drain | All seven / three accepted records persist before close |
| Drain timeout | 200-ms deadline; stalled writer terminated and reaped in 201.103 ms; pending record retained in the report |
| Driver killed during blocked write | Two complete prefix records; final report absent, attempt unusable; helpers close within 10.083 ms |
| Collection error | Explicit failure before any payload admission |

The final 15 cases contain 145 complete payload records and one partial tail.
Expected failure cases are successful contract tests, not usable performance
attempts. Abrupt driver death leaves its exact accepted count unknown; no zero-loss
claim is made. The independent checker reconstructs raw bytes, acknowledgements,
phase clocks and closure; it rejects six altered/incomplete copies: dropped payload,
changed digest, reordered acknowledgement, excess queue, hidden pending data and
missing final report.

One earlier normal case persisted all 40 records but failed in the fixture reader:
strict decoding rejected `writeBoundary` because the test used a partial report
projection. The test reader alone was corrected. The failed case, original source
and binaries remain retained, with [exact relocation bindings](./inputs/v1-relocations.json).
Its service runtime was 4.294 s; the first controller's exact end stamp was not
retained. This is one fixture-reader correction, zero failed product corrections.

## Scope and accounting

Pinned Go 1.26.5 compilation passes for both source versions, normally and with
the race detector; final vet passes. There are five compiler/check commands and
16 case starts including the retained reader failure, 16 writer helpers and
16 heartbeat helpers. All owned processes are closed. The 15 final cases took
25.261 s. A prospective amendment added only the failed reader's replacement case
and recompilation; the 60-minute, 1-GiB and 40-GiB-free bounds remained unchanged.
No dependencies, host configuration, native worktree or production source changed.
Program Gradle accounting stays at **646 reservations / 547 actual starts**,
including the same 29 older nested starts and unresolved possible metadata JVM.

This is a task-owned diagnostic copy with fixture payloads of about 3.5 KiB.
It does not establish actual-owner overhead or chronological CLI integration.
Previous cold endpoint gaps of 503.588/603.431 ms still fail the original
endpoint-inclusive 500-ms bound. The 100-ms imbalance gate, 36-start CPU screen
ceiling and 80 untouched validation transitions are preserved.

## Evidence and continuation

- [Independent raw-data analysis](./analysis/qualification.json) and [auditor rejection proof](./analysis/checker-proof.json).
- [Source/binary freeze](./inputs/qualification-freeze.json), [complete source delta](./inputs/source-change.diff), and [implementation/proof notes](./implementation.md).
- [Allocation](./allocation.json), [execution receipts](./receipts/qualification.json), [closeout](./closeout.json), and [independent audit](./independent-audit.json).
- [Origin map](./origin-map.json), [evidence manifest](./evidence-manifest.json), and [seal audit](../bv006-buffered-observer-seal-audit.json).
- [Next: integrate and qualify the real consumer](./next-step.md). No native allocation is opened by this report.

Durable local locator: `.tools/state/buildopt-product-viability-v1/task-state.json`,
key `engineeringPrefix.bufferedObserver`. Source path:
`.tools/state/buildopt-product-viability-v1/bv006-buffered-observer/control/source`.
Repository `/home/tonyredondo/repos/github/tonyredondo/buildopt`, shared Git `.git`,
local `main` at `b76ded08c952ebb386576fafce4ae2d8fdcc09f1`; no publication.
