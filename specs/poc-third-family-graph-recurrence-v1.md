# Third-Family Graph Recurrence v1

Status: `TFGR-001` complete; `TFGR-002` is current.

This source-only successor searches Spring Framework, OpenTelemetry Java
Instrumentation and Micronaut Core for the third public value family. It
classifies up to 256 first-parent commits per exact SHA-256-bound reviewed graph
and groups only by source-derived owners plus change family. Repository and task
names are labels and cannot affect classification or selection.

A group is eligible only when at least five commits share the exact group and
its graph closure omits at least one project. Eligible groups are ranked by
commit count, then omitted-project count, then stable structural identity. The
route either selects one public observation for a separately frozen fresh graph
confirmation or stops. It may fetch public Git objects and inspect source; it
may not run Gradle, patch public source, execute a candidate, collect timing,
reuse predecessor summaries as evidence, or authorize production.
