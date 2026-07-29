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
}

tasks.named("clean") {
    dependsOn(subprojects.map { "${it.path}:clean" })
}

tasks.register("correlationFixture") {
    group = "verification"
    description = "Runs the equivalent cacheable tasks used by the correlation spike."
    dependsOn(subprojects.map { "${it.path}:correlationFixture" })
}
