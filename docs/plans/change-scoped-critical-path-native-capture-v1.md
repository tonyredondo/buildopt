# Change-Scoped Critical-Path Native Capture v1 Tracker

**Status:** complete — discovery gate passed<br>
**Current block:** none

| Block | Deliverable | State |
|---|---|---|
| `CSCPN-001` | Freeze bounded native diagnostic contract | `DONE` |
| `CSCPN-002` | Capture four missing operation/DAG traces | `DONE` |
| `CSCPN-003` | Independently reconstruct and classify five families | `DONE` |
| `CSCPN-004` | Enforce the discovery gate | `DONE` |

The route is 5/5 conclusive and 4/5 actionable. Groovy, Kafka, OpenTelemetry
and Spring exceed the frozen 500-ms/2% critical-path gate; Micronaut remains a
conclusive no-action control. Four exact-revision native diagnostic builds
started successfully, with zero candidate builds, zero timing samples and zero
product failures.

The result authorizes only a separately frozen candidate-correctness contract.
It is not a speedup claim and does not authorize paired timing by itself.
