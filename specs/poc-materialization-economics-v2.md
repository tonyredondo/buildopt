# Materialization economics V2

This proof-of-concept block asks whether verified output reuse can learn cheaply enough to repay on a substantial public Gradle repository. It retains the five frozen repositories, revisions, commands, correctness checks and 30-build payback gate from the automatic breadth V2 experiment.

The implementation replaces one durable file per captured output with one repository-bound sequential pack. The manifest still binds every relative path, mode, offset, size and SHA-256, while the pack itself has its own size and SHA-256. Candidate restoration verifies the complete pack and every entry before publishing any output. It creates each safe parent once and writes only absent, reconstructible workspace files directly; the durable pack and manifest remain the crash-recovery boundary. Private `.buildopt` state is excluded from Gradle output hashing because it can never be a customer build output. Output hashing and publication use a bounded worker pool while preserving sorted deterministic manifests.

Original workflow outputs and the configured Gradle task and project-dependency graphs are observed inside the useful baseline build. Candidate project-qualified lifecycle entrypoints are derived from the exact task dependency closure when present, with the complete project graph as a conservative fallback, eliminating configuration-only Gradle invocations. Missing tasks, projects, dependencies, conventional producer evidence or the original fallback entrypoint retain the native graph.

Each ordinary observation measures end-to-end customer wall time and attributes it to Gradle execution, discovery, verified materialization, required-output verification and remaining wrapper work. Recurring wrapper work is included in every paired duration; only initial discovery and capture overhead is amortized as learning cost. Attribution is observational and is not summed across repositories.

## Acceptance

- all five frozen subjects complete 17 useful invocations with no measurement-only workflow;
- every required-output digest and full-graph fallback remains exact;
- Spring Framework repays learning within 30 matching builds;
- OpenTelemetry, Kafka, Micronaut and Groovy remain qualified under the unchanged 8/8, positive-interval and 30-build gates;
- every baseline and paired observation has internally consistent phase attribution;
- no repository-name rule, production authority, soak, design-partner requirement or Test Optimization behavior is introduced.

Run the experiment one subject at a time with `./dev/run-materialization-economics-v2 /absolute/evidence/directory <repository-key>`. Validate the completed result with `./dev/check-materialization-economics-v2 /absolute/evidence/directory/summary.json`.
