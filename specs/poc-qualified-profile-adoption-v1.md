# Installed qualified-profile adoption contract

This contract verifies that the already qualified POC profile can be consumed
through an installed native package on the fixed OpenTelemetry and Kafka
revisions. It answers a usability question: can a repository commit reviewed
Build Impact state, run one public command, and retain a native full-graph
fallback?

It is deliberately not another performance experiment. The replay records no
durations, percentages, resource samples, or promotion decision. Existing
performance evidence remains authoritative for the exact measured workloads.

## Required replay

For each repository, a disposable checkout receives the checked manifest,
generated graph, generated binding, and qualified profile from
`fixtures/poc-qualified-profile-adoption/`. The replay must use the
`buildopt` binary from an installed Linux AMD64 package, not a source-tree Go
entrypoint.

The candidate change must:

1. emit the qualified-profile plan before Gradle output;
2. select the preregistered alternative;
3. enable only `STANDARD_JAR` in addition to Build Impact;
4. restore the named standard `Jar` producer from Gradle's cache on the second
   clean replay;
5. reproduce the historical required-output count and digest; and
6. avoid Gradle `Test`, publishing, managed runtime, hot state, and remote
   cache activity.

The global `gradle.properties` change must emit `FULL_GRAPH` with
`IMPACT_GLOBAL_CHANGE`, disable the candidate adapter, complete the repository's
native entrypoints, and reach a task outside the candidate graph.

## Boundary

Passing this contract proves installed-command adoption and conservative
fallback only. It does not broaden repository or change-shape support, prove a
production policy, or create fresh timing evidence.
