import org.gradle.api.tasks.bundling.AbstractArchiveTask
import org.gradle.api.tasks.compile.JavaCompile
import org.gradle.api.tasks.JavaExec
import org.gradle.jvm.toolchain.JvmVendorSpec

plugins {
    `java-library`
}

group = "dev.buildopt"
version = providers.gradleProperty("buildoptVersion").getOrElse("0.1.0-SNAPSHOT")

base {
    archivesName = "buildopt-patcher"
}

java {
    toolchain {
        languageVersion = JavaLanguageVersion.of(21)
        vendor = JvmVendorSpec.ADOPTIUM
    }
}

val spike = sourceSets.create("spike")

dependencies {
    api(project(":jvm:generated-client"))
    add(spike.implementationConfigurationName, sourceSets.main.get().output)
    add(spike.implementationConfigurationName, project(":jvm:generated-client"))
}

tasks.withType<JavaCompile>().configureEach {
    options.release = 17
    options.encoding = "UTF-8"
    options.compilerArgs.addAll(listOf("-Xlint:all", "-Werror"))
}

tasks.withType<AbstractArchiveTask>().configureEach {
    isPreserveFileTimestamps = false
    isReproducibleFileOrder = true
}

val spikeReport = providers.gradleProperty("buildoptPatcherSpikeReport")
    .orElse(layout.buildDirectory.file("reports/patcher-spike.json").map { it.asFile.absolutePath })

tasks.register<JavaExec>("patcherSpike") {
    group = "verification"
    description = "Executes the SPK-004 PatchBundle matrix against real Git worktrees."
    dependsOn(tasks.named(spike.classesTaskName))
    classpath = spike.runtimeClasspath
    mainClass = "dev.buildopt.patcher.PatcherSpike"
    args(rootProject.layout.projectDirectory.asFile.absolutePath, spikeReport.get())
    inputs.file(rootProject.layout.projectDirectory.file("specs/patch-bundle-v1.json"))
    inputs.dir(rootProject.layout.projectDirectory.dir("contracts/jsonschema/testdata/patch-bundle.v1/blobs"))
    outputs.file(spikeReport)
}

tasks.jar {
    manifest {
        attributes(
            "Implementation-Title" to "BuildOpt PatchBundle Patcher",
            "Implementation-Version" to project.version,
        )
    }
}
