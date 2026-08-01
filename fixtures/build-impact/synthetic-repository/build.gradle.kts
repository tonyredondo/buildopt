import org.gradle.api.tasks.bundling.AbstractArchiveTask

subprojects {
    group = "dev.buildopt.synthetic"
    version = "1.0"

    tasks.withType<AbstractArchiveTask>().configureEach {
        isPreserveFileTimestamps = false
        isReproducibleFileOrder = true
    }
}

tasks.register("assemble") {
    group = "build"
    dependsOn(
        ":library-c:assemble",
        ":service-a:assemble",
        ":service-b:assemble",
    )
}

tasks.register("testOwnedCheck") {
    val marker = layout.buildDirectory.file("test-owned/check.txt")
    outputs.file(marker)
    doLast {
        marker.get().asFile.apply {
            parentFile.mkdirs()
            writeText("synthetic-tests-passed\n")
        }
    }
}
