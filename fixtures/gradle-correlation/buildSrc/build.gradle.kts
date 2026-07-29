import org.gradle.api.tasks.compile.JavaCompile

plugins {
    `java-gradle-plugin`
}

java {
    toolchain {
        languageVersion = JavaLanguageVersion.of(21)
    }
}

gradlePlugin {
    plugins {
        create("correlationFixture") {
            id = "dev.buildopt.correlation-fixture"
            implementationClass = "dev.buildopt.fixtures.correlation.CorrelationFixturePlugin"
        }
    }
}

tasks.withType<JavaCompile>().configureEach {
    options.release = 17
    options.encoding = "UTF-8"
    options.compilerArgs.addAll(listOf("-Xlint:all", "-Werror"))
}
