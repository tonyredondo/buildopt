val leftOutput = layout.buildDirectory.file("left.bin")
val rightOutput = layout.buildDirectory.file("right-v2.bin")

val leftProducer = tasks.register("leftProducer") {
    inputs.file("inputs/left.txt").withPathSensitivity(PathSensitivity.RELATIVE)
    outputs.file(leftOutput)
    doLast {
        leftOutput.get().asFile.apply {
            parentFile.mkdirs()
            writeText(file("inputs/left.txt").readText())
        }
    }
}

val rightProducer = tasks.register("rightProducer") {
    inputs.file("inputs/right.txt").withPathSensitivity(PathSensitivity.RELATIVE)
    outputs.file(rightOutput)
    doLast {
        rightOutput.get().asFile.apply {
            parentFile.mkdirs()
            writeText(file("inputs/right.txt").readText())
        }
    }
}

tasks.register("bundle") {
    dependsOn(leftProducer, rightProducer)
    inputs.files(leftOutput, rightOutput).withPathSensitivity(PathSensitivity.RELATIVE)
    val bundleOutput = layout.buildDirectory.file("bundle.bin")
    outputs.file(bundleOutput)
    doLast {
        bundleOutput.get().asFile.writeText(
            leftOutput.get().asFile.readText() + rightOutput.get().asFile.readText(),
        )
    }
}
