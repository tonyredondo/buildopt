plugins {
    base
}

val fixtureOutputRoot = providers.gradleProperty("buildoptFixtureOutputDir")
    .map(::file)
    .orElse(layout.projectDirectory.dir("build").asFile)

layout.buildDirectory.set(layout.dir(fixtureOutputRoot.map { it.resolve("root") }))

subprojects {
    layout.buildDirectory.set(layout.dir(fixtureOutputRoot.map { it.resolve(project.name) }))
    pluginManager.apply("dev.buildopt.correlation-fixture")

    // The root clean task owns every fixture output directory. Keep both
    // subproject workers parallel with each other, but never let clean remove
    // their barrier markers after either worker has announced readiness.
    tasks.named("correlationFixture") {
        mustRunAfter(rootProject.tasks.named("clean"))
    }
}

tasks.named("clean") {
    dependsOn(subprojects.map { "${it.path}:clean" })
}

tasks.register("correlationFixture") {
    group = "verification"
    description = "Runs the equivalent cacheable tasks used by the correlation spike."
    dependsOn(subprojects.map { "${it.path}:correlationFixture" })
}

tasks.register("correlationMatrix") {
    group = "verification"
    description = "Runs every successful task shape used by the correlation spike."
    dependsOn(subprojects.map { "${it.path}:correlationFixture" })
    dependsOn(subprojects.map { "${it.path}:workerNoIsolationFixture" })
    dependsOn(subprojects.map { "${it.path}:workerProcessIsolationFixture" })
    dependsOn(subprojects.map { "${it.path}:childProcessFixture" })
}
