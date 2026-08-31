# Change-Scoped Critical-Path Discovery v1

Status: terminal `STOP_CHANGE_SCOPED_CRITICAL_PATH_DISCOVERY_EXISTING_EVIDENCE`.

This evidence-only route classifies the exact five public families from
repository-owned structural records and existing native operation traces. It
does not modify public source, run Gradle, execute a candidate or collect new
timing. Historical performance results motivate the route but supply no fresh
row.

A conclusive row requires an exact change/output binding plus task durations,
hard-dependency edges and critical-path membership from the same attributed
window. An actionable row additionally requires at least 500 ms and 2% of
causally avoidable native wall time, exact required outputs and zero product
failures. Cumulative task duration is never treated as critical-path time.

The gate requires 5/5 conclusive families and 3/5 actionable families before
any candidate build or timing. Repository/task names cannot affect
classification. Missing operation/DAG evidence is
`INCOMPLETE_NO_CRITICAL_PATH_TRACE`, not a zero-opportunity row.

The existing corpus is conclusive only for Micronaut. Its attributed candidate
eliminates 110 tasks and 4,731 ms of cumulative task work, but none of those
tasks belongs to the critical path and the critical path grows by 178.875 ms.
The other four families lack retained task-DAG critical-path traces. The route
therefore stops at 1/5 conclusive and 0/5 actionable without builds or timing.
