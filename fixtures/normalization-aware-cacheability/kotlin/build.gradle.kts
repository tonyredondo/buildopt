import org.gradle.api.file.DirectoryProperty
import org.gradle.api.file.RegularFileProperty
import org.gradle.api.tasks.CacheableTask
import org.gradle.api.tasks.InputDirectory
import org.gradle.api.tasks.OutputFile
import org.gradle.api.tasks.PathSensitive
import org.gradle.api.tasks.PathSensitivity
import org.gradle.api.tasks.TaskAction

plugins {
    base
}

@CacheableTask
abstract class PortableInputTask : DefaultTask() {
    @get:InputDirectory
    @get:PathSensitive(PathSensitivity.RELATIVE)
    abstract val inputDirectory: DirectoryProperty

    @get:OutputFile
    abstract val outputFile: RegularFileProperty

    @TaskAction
    fun generate() {
        outputFile.get().asFile.writeText(inputDirectory.file("value.txt").get().asFile.readText())
    }
}

tasks.register<PortableInputTask>("portableInput") {
    inputDirectory.set(layout.projectDirectory.dir("input"))
    outputFile.set(layout.buildDirectory.file("portable.txt"))
}
