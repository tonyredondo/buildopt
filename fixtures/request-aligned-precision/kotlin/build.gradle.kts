val leftOutput = layout.buildDirectory.file("left.bin")
val rightOutput = layout.buildDirectory.file("right.bin")
val optionalOutput = layout.buildDirectory.file("optional.bin")
val bundleOutput = layout.buildDirectory.file("bundle.bin")

val leftProducer = tasks.register("leftProducer") {
    inputs.file("inputs/left.txt")
    outputs.file(leftOutput)
    doLast { leftOutput.get().asFile.writeText(file("inputs/left.txt").readText()) }
}

val rightProducer = tasks.register("rightProducer") {
    inputs.file("inputs/right.txt")
    outputs.file(rightOutput)
    doLast { rightOutput.get().asFile.writeText(file("inputs/right.txt").readText()) }
}

val rightAlias = tasks.register("rightAlias") {
    dependsOn(rightProducer)
    outputs.file(rightOutput)
}

val optionalProducer = tasks.register("optionalProducer") {
    outputs.file(optionalOutput)
    onlyIf { false }
}

tasks.register("bundle") {
    dependsOn(leftProducer, rightAlias, optionalProducer)
    inputs.files(leftOutput, rightOutput)
    outputs.file(bundleOutput)
    doLast {
        bundleOutput.get().asFile.writeText(leftOutput.get().asFile.readText() + rightOutput.get().asFile.readText())
    }
}
