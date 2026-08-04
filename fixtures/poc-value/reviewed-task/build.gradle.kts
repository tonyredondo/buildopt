import dev.buildopt.pilot.GeneratePilotManifest

val manifests = (1..8).map { index ->
    tasks.register<GeneratePilotManifest>("generateManifest$index") {
        entries.set((1..32).map { "entry-$index-$it" })
        outputFile.set(layout.buildDirectory.file("poc-value/manifest-$index.txt"))
    }
}

tasks.register("reviewedWorkload") {
    dependsOn(manifests)
}
