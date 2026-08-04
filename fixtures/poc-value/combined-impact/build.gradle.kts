import dev.buildopt.pocvalue.ExpensiveUnrelated
import org.gradle.api.tasks.bundling.AbstractArchiveTask
import org.gradle.api.tasks.compile.JavaCompile

plugins {
    base
}

subprojects {
    group = "dev.buildopt.pocvalue"
    version = "1.0"
    apply(plugin = "java-library")

    tasks.withType<JavaCompile>().configureEach {
        options.release = 17
    }
    tasks.withType<AbstractArchiveTask>().configureEach {
        isPreserveFileTimestamps = false
        isReproducibleFileOrder = true
    }
}

project(":service-a") {
    dependencies {
        add("implementation", project(":library-c"))
    }
}

project(":service-b") {
    tasks.register<ExpensiveUnrelated>("expensiveUnrelated") {
        rounds.set(8_000_000)
        outputFile.set(layout.buildDirectory.file("poc-value/unrelated.txt"))
    }
}

tasks.named("assemble") {
    dependsOn(":service-a:assemble", ":service-b:expensiveUnrelated")
}

tasks.register("testOwnedCheck") {
    val marker = layout.buildDirectory.file("test-owned/check.txt")
    outputs.file(marker)
    doLast {
        marker.get().asFile.apply {
            parentFile.mkdirs()
            writeText("poc-value-tests-passed\n")
        }
    }
}
