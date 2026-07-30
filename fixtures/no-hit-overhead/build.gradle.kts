import org.gradle.api.tasks.Exec
import org.gradle.api.tasks.WriteProperties
import org.gradle.api.tasks.compile.JavaCompile
import org.gradle.jvm.tasks.Jar

plugins {
    java
}

group = "dev.buildopt"
version = "1.0"

tasks.withType<JavaCompile>().configureEach {
    options.release = 17
}

tasks.named<Jar>("jar") {
    archiveBaseName = "no-hit-overhead"
    isPreserveFileTimestamps = false
    isReproducibleFileOrder = true
}

val longDelayMs = providers.gradleProperty("buildoptNoHitDelayMs").map(String::toLong)

tasks.register<Exec>("noHitLongProbe") {
    dependsOn(tasks.named("jar"))
    inputs.property("delayMs", longDelayMs)
    val marker = layout.buildDirectory.file("no-hit/long.txt")
    outputs.file(marker)
    environment("BUILDOPT_NO_HIT_DELAY_SECONDS", longDelayMs.get() / 1000)
    environment("BUILDOPT_NO_HIT_MARKER", marker.get().asFile.absolutePath)
    commandLine(
        "sh",
        "-c",
        """
        sleep "${'$'}BUILDOPT_NO_HIT_DELAY_SECONDS"
        mkdir -p "${'$'}(dirname -- "${'$'}BUILDOPT_NO_HIT_MARKER")"
        printf '%s\n' 'long-no-hit' >"${'$'}BUILDOPT_NO_HIT_MARKER"
        """.trimIndent(),
    )
}

tasks.register<WriteProperties>("noHitShortProbe") {
    dependsOn(tasks.named("jar"))
    destinationFile = layout.buildDirectory.file("no-hit/short.properties")
    property("marker", "short-l2-omitted")
}
