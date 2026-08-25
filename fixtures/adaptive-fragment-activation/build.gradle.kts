import org.gradle.api.tasks.bundling.Zip

val packageAll by tasks.registering(Zip::class) {
    from(layout.projectDirectory.file("producer-a/build/fragment/value.txt")) {
        into("producer-a")
    }
    from(layout.projectDirectory.file("producer-b/build/fragment/value.txt")) {
        into("producer-b")
    }
    destinationDirectory.set(layout.buildDirectory.dir("distribution"))
    archiveFileName.set("adaptive-fragments.zip")
    isPreserveFileTimestamps = false
    isReproducibleFileOrder = true
    mustRunAfter(":producer-a:produce", ":producer-b:produce")
}

tasks.register("fullBuild") {
    dependsOn(":producer-a:produce", ":producer-b:produce", packageAll)
}
