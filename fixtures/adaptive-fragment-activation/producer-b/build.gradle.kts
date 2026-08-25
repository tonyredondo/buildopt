val produce by tasks.registering {
    val source = layout.projectDirectory.file("src/value.txt")
    val output = layout.buildDirectory.file("fragment/value.txt")
    inputs.file(source)
    outputs.file(output)
    doLast {
        output.get().asFile.apply {
            parentFile.mkdirs()
            writeText(source.asFile.readText())
        }
    }
}
