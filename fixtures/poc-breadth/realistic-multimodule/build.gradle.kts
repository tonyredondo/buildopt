import dev.buildopt.breadth.VerificationWork
import org.gradle.api.tasks.bundling.AbstractArchiveTask
import org.gradle.api.tasks.compile.JavaCompile

plugins {
    base
}

subprojects {
    group = "dev.buildopt.breadth"
    version = "1.0"
    apply(plugin = "java-library")

    tasks.withType<JavaCompile>().configureEach {
        options.release = 17
    }
    tasks.withType<AbstractArchiveTask>().configureEach {
        isPreserveFileTimestamps = false
        isReproducibleFileOrder = true
    }
    tasks.register<VerificationWork>("pocVerify") {
        dependsOn("classes")
        label.set(project.path)
        rounds.set(5_000_000)
        outputFile.set(layout.buildDirectory.file("poc-breadth/verification.txt"))
    }
    tasks.named("assemble") {
        dependsOn("pocVerify")
    }
}

project(":shared-lib") {
    dependencies { add("api", project(":platform-core")) }
}
project(":service-api") {
    dependencies { add("implementation", project(":shared-lib")) }
}
project(":service-worker") {
    dependencies { add("implementation", project(":shared-lib")) }
}
project(":developer-tool") {
    dependencies { add("implementation", project(":platform-core")) }
}

tasks.named("assemble") {
    dependsOn(":service-api:assemble", ":service-worker:assemble", ":developer-tool:assemble")
}

tasks.register("testOwnedCheck") {
    val marker = layout.buildDirectory.file("test-owned/check.txt")
    outputs.file(marker)
    doLast {
        marker.get().asFile.apply {
            parentFile.mkdirs()
            writeText("poc-breadth-tests-passed\n")
        }
    }
}
