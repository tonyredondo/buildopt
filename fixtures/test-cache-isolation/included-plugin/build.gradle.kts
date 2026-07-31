import org.gradle.api.tasks.compile.JavaCompile
import org.gradle.api.tasks.testing.Test

plugins {
    `java-gradle-plugin`
}

gradlePlugin {
    plugins {
        create("fixturePlugin") {
            id = "dev.buildopt.test-cache-isolation-fixture"
            implementationClass = "example.IncludedPlugin"
        }
    }
}

tasks.withType<JavaCompile>().configureEach {
    options.release = 17
}

tasks.withType<Test>().configureEach {
    failOnNoDiscoveredTests = false
}
