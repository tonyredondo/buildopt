import org.gradle.api.tasks.CacheableTask
import org.gradle.api.tasks.InputFile
import org.gradle.api.tasks.OutputFile
import org.gradle.api.tasks.PathSensitive
import org.gradle.api.tasks.PathSensitivity

@CacheableTask
abstract class RequestHitOutput : DefaultTask() {
    @get:InputFile
    @get:PathSensitive(PathSensitivity.RELATIVE)
    abstract val payload: RegularFileProperty

    @get:OutputFile
    abstract val result: RegularFileProperty

    @TaskAction
    fun generate() {
        val suffix = System.getenv("BUILDOPT_SHADOW_MISMATCH") ?: ""
        result.get().asFile.apply {
            parentFile.mkdirs()
            writeText(payload.get().asFile.readText() + suffix)
        }
    }
}

tasks.register<RequestHitOutput>("requestHitOutput") {
    payload = layout.projectDirectory.file("inputs/payload.txt")
    result = layout.buildDirectory.file("request-hit/result.txt")
}
