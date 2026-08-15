# Rejected pre-measurement attempt: configuration-on-demand entrypoint

The third fresh attempt configured Ktor with the reviewed JVM-only properties
but failed before executing any owner `assemble` task. Ktor enables Gradle
configuration-on-demand. BuildOpt registered its synthetic output-contract task
inside `projectsEvaluated`, after Gradle had already resolved the requested root
task name, so Gradle reported `Task 'buildoptOutputContract' not found`.

The generic correction registers the synthetic task as soon as the root project
exists and attaches the resolved owner-task dependencies after project
evaluation. Both untimed model-observation phases own one explicit
`--no-configure-on-demand` value so all producer tasks and relationships can be
inspected; reviewed project properties remain present. Timed control and
candidate invocations keep the exact preregistered Ktor settings. No proposal,
warm-up, output or duration from this failed attempt is reused.
