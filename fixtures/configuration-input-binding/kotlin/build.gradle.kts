val configuredVersion = providers.fileContents(
    layout.projectDirectory.file("versions.properties")
).asText.get().trim()

project(":service-a") {
    apply(plugin = "java")
    version = configuredVersion
}

project(":service-b") {
    apply(plugin = "java")
    version = "1.0.0"
}
