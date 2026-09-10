# Actual observer phase diagnostic

Task identity and live process receipts: `../task-state.json`, key
`engineeringPrefix.liveObserver`. Local main at b76ded08c952ebb386576fafce4ae2d8fdcc09f1.
The frozen CPU isolation source is copied into `control/source`; production and
prior bundles remain unchanged. Native owner: shared Elasticsearch Git history,
anchor 53a80bec683ad0b065ed9ebe6a57f984e6a91ed1, no candidate correction.

| Step | State | Acceptance |
|---|---|---|
| P2 | verified | Instrument original live collection and synchronous JSON/file writes; preserve 100-ms cadence, errors, affinity and native lifecycle guards |
| P3 | verified | Nine fixed non-Gradle cases: normal, writer delay, process suspension, writer failure, immediate cancellation, cancellation during write, scan error, exited-process recovery, bounded records. Verify raw records independently, errors and helper termination |
| P4 | verified diagnostic | After P3, freeze source/binary/manifest and reserve two total Gradle starts, fresh N/I state, one cold build each; identical native flags and baseline, external heartbeat CPU10 |
| P5 | blocked | Diagnose only an observed event. A negative reproduction does not qualify precision, G0, lifecycle value or the CPU screen |

Phase stamps contain BOOTTIME, MONOTONIC and process CPU. An internal 10-ms
heartbeat distinguishes sampler blockage from whole-process delay; an independent
Go process on CPU10 has its own heartbeat. It starts before sampling and exits on
control-pipe EOF or a fixed deadline; its complete records and termination are
required. Runtime pause histograms are captured without forcing a GC or using
ReadMemStats in the hot path. Collection, encoding, write, and post-write costs
remain separate; raw payload digests/counts connect phase rows to actual writes.

Bounded memory: 10,000 phase rows, 100,000 heartbeat rows; raw payload ceiling
512 MiB per request. Limits fail explicitly, never discard samples. The original
blocking writer remains blocking: cancellation during a write completes after
that write returns. Owner invocation already bounds the request and owns its
service tree; diagnostics introduce no asynchronous writer or scheduler fix.

Allocation fixes 90 minutes, 36 GiB, 40 GiB minimum free, 120 seconds of probe
work and at most two owner Gradle starts (including failures/preparation). No
extra Gradle fixtures, retries, metadata JVMs or nested build starts. Both owner
rows are cold diagnostics, not steady-state comparisons. Preserve any failed
row and all original 500-ms criteria. Reuse unchanged affinity/watchdog proof.

P2/P3 evidence: `analysis/qualification.json`, `receipts/qualification.json`,
`receipts/build-v2.json`, `receipts/vet.json`, source/binary bindings in
`inputs/qualification-freeze.json`. Nine live cgroup cases produced 128 complete
payloads and 130 phase rows; two additional rows retain the deliberately failed
writer/scan attempts. All helpers exited. The two owner starts are reserved in
`inputs/owner-design.json` and `inputs/owner-manifest.json`.

One initial compile setup failure retained an unused import after extraction;
it launched no probe/native child. Removing that import produced the tested
binary; no failed diagnostic/production correction or experimental retry.

Ticker coalescing during the deliberately blocked interval is visible as a gap.
Exact payload accounting does not claim that a blocked sampler emits every
scheduled tick. CPU counters cover their named sampler/heartbeat intervals;
final serialization is outside sampling, retained in whole-driver wall cost.

P4 evidence: `analysis/owner-result.json` and `analysis/write-diagnosis.json`.
Both cold builds succeeded; all 4,322 records and 1,326 equal native outcomes
reconstructed, 813,911 thread masks verified, both services/daemons closed.
`analysis/daemon-boundaries.json` preserves the cold 503.588/603.431-ms
endpoint gaps: no original 500-ms criterion pass. Real 323/341-ms synchronous
writes explain over 98% of their 327/344-ms inter-sample gaps, with continuing
heartbeats. Original 2,104-ms cause remains unresolved; P5/S1 remain blocked.
Actual source diff: `inputs/sampler-change.diff`. Native allocation exhausted: 2/2.
