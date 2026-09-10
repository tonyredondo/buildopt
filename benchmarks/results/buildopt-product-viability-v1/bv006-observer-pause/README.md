# Observer pause reproduction

Date: 2026-09-09. Program: `BUILDOPT-VIABILITY-V1`.
Decision: **`ORIGINAL_PAUSE_NOT_REPRODUCED`**. The bounded replay and diagnostic
controls are verified. The historical pause's cause is unproved; no production
correction, observation qualification, G0 pass or CPU-profile start follows.

The prior [isolation control](../bv006-cpu-isolation/README.md) rejected one
measured request for a 2,104.017-ms observation gap. Its adjacent scans took
about 3 ms. Static inspection now also confirms that the native owned service
accumulated **6,892.184 ms of CPU** across the gap: native work continued while
the driver emitted no sample. Adjacent payloads are about 27 KB, with no size
spike. [Original-gap scope](./analysis/original-pause-scope.json).

A task-local Go diagnostic replays all 69 retained snapshots through the same
pinned `json.Encoder.Encode` API at the original 100-ms cadence. It records
wake, record readiness, encoding, the underlying writer boundary and return,
with BOOTTIME, MONOTONIC, process CPU and UTC. An internal heartbeat and an
independent process heartbeat run every 10 ms. The external heartbeat uses
CPU 10; the probe retains the caller's allowed CPU mask. Only task processes
are affected. A final fsync is measured separately outside the periodic window;
the original sampler does not call fsync.

Two nominal cases replay 600 records each: a regular file on the original
filesystem and `io.Discard`. Two 80-record controls deliberately delay the writer
or SIGSTOP/SIGCONT the owned probe for about 2.1 seconds. Their classifications
must differ. Controlled delays are not reproductions of the historical cause.

All table times are maxima in milliseconds. A normal observation gap is about
100 ms because that is the registered cadence.

| Case | Records | Encoding | Writer boundary | Observation gap | Internal heartbeat gap | External heartbeat gap | Diagnostic result |
|---|---:|---:|---:|---:|---:|---:|---|
| file | 600 | 0.712 | 0.145 | 101.099 | 11.095 | 11.281 | NO_GAP_REPRODUCED |
| discard | 600 | 1.222 | 0.020 | 101.083 | 10.817 | 10.949 | NO_GAP_REPRODUCED |
| write-pause | 80 | 0.259 | 2104.466 | 2104.697 | 10.645 | 10.538 | WRITE_BOUNDARY_PAUSE |
| process-pause | 80 | 0.622 | 0.067 | 2191.061 | 2112.303 | 11.148 | PROCESS_OR_RUNTIME_PAUSE |

The file case's median encoding/writing costs are 0.167/0.055 ms. Neither nominal
case produces a gap over 500 ms. Both deliberately delayed cases are detected:
writer delay leaves both heartbeats responsive; whole-process pause interrupts
the internal heartbeat while the external one continues. All 1,360 payload
records agree with the frozen originals; written bytes and per-write hashes
are independently checked. [Complete result](./analysis/result.json).

This establishes useful diagnostic distinctions. It does **not** exclude an
intermittent filesystem delay or prove that the historical cause was scheduling.
Recorded input lookup replaces live `/proc` collection and its allocations;
the original driver's heap, concurrent work and host load are absent. Two
60-second cases cannot qualify rare-pause reliability. No speculative buffering,
CPU setting, timer change or threshold relaxation was applied.

Costs: **zero Gradle starts, zero helper JVMs**, four probe processes and four
heartbeat processes. One pinned Go build and one `go vet` pass; no dependencies
installed, no retries or additional cases. The four cases take about 141 seconds
wall in total. The phase has a 30-minute bound, 180-second active-probe bound,
512 MiB new-state ceiling and 40 GiB minimum free space. Current program Gradle
totals stay **644 reservations / 545 actual**. The CPU screen stays **0/36**;
all 80 validation transitions remain untouched.

[Closeout](./closeout.json), [independent audit](./independent-audit.json) and
[external seal](../bv006-observer-pause-seal-audit.json) bind source, raw output,
counts, process closure and prior evidence preservation. Resume from the
[next-step tracker](./next-step.md); D3 is verified as a bounded negative
reproduction, while a cause-based repair and the CPU screen remain blocked.
