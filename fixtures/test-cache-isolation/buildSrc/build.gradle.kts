import org.gradle.api.tasks.compile.JavaCompile
import org.gradle.api.tasks.testing.Test

plugins {
    `java-gradle-plugin`
}

gradlePlugin {
    plugins {
        create("buildSrcFixture") {
            id = "dev.buildopt.test-cache-isolation-buildsrc"
            implementationClass = "example.BuildSrcPlugin"
        }
    }
}

tasks.withType<JavaCompile>().configureEach {
    options.release = 17
}

tasks.withType<Test>().configureEach {
    failOnNoDiscoveredTests = false
}
