import org.gradle.api.tasks.compile.JavaCompile
import org.gradle.api.tasks.testing.Test

plugins {
    java
}

tasks.withType<JavaCompile>().configureEach {
    options.release = 17
}

tasks.withType<Test>().configureEach {
    failOnNoDiscoveredTests = false
}

tasks.register("compositeTest") {
    dependsOn(tasks.named("test"))
    dependsOn(gradle.includedBuild("included-plugin").task(":test"))
}
