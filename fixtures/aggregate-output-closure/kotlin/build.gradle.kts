fun Project.registerPayload() = tasks.register("emitPayload") {
    val source = layout.projectDirectory.file("input.txt")
    val destination = layout.buildDirectory.file("custom-output/payload.bin")
    inputs.file(source)
    outputs.file(destination)
    doLast {
        val output = destination.get().asFile
        output.parentFile.mkdirs()
        output.writeText(source.asFile.readText().trim() + "\n")
    }
}

val stablePayload = project(":stable").registerPayload()
val changedPayload = project(":changed").registerPayload()

project(":changed") {
    tasks.register("bundleAll") {
        group = "build"
        dependsOn(changedPayload, stablePayload)
    }
}
