# Graph-Aware History Admission v1

Status: `GAH-001..003` complete as `VALIDATED_GRAPH_AWARE_ADMISSION`.

This source-only experiment replaces path-prefix recurrence with classification
against a retained, exact Gradle discovery graph. It does not run Gradle, patch
public source, create candidates, collect timing, or authorize production.

For each bounded first-parent commit, the detector must reject build-logic
changes, resolve every changed path to an unambiguous project owner, derive the
change family and transitive affected-project closure, and count the row only
when owner and family exactly match the current observation. Repository and
task names are labels and cannot affect any decision.

The retained snapshot is bound by raw SHA-256 and must be complete, cover the
requested entrypoints, and contain no unknown relationships. Five compatible
commits remain the frozen admission threshold. Missing ownership, ambiguity,
snapshot drift, structural changes and mismatched owners or families fail
closed with typed reasons.

The public validation reuses two exact observations solely to isolate detector
behavior: Kafka must reproduce admission and Spring must reproduce rejection.
The result is a detector correction, not new candidate correctness or value
evidence. A new public build requires a separately frozen experiment.
