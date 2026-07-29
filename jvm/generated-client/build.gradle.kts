import org.gradle.api.tasks.compile.JavaCompile
import org.gradle.jvm.toolchain.JvmVendorSpec

plugins {
    `java-library`
}

group = "dev.buildopt"
version = providers.gradleProperty("buildoptVersion").getOrElse("0.1.0-SNAPSHOT")

base {
    archivesName = "buildopt-generated-client"
}

java {
    toolchain {
        languageVersion = JavaLanguageVersion.of(21)
        vendor = JvmVendorSpec.ADOPTIUM
    }
}

tasks.withType<JavaCompile>().configureEach {
    options.release = 17
    options.encoding = "UTF-8"
    options.compilerArgs.addAll(listOf("-Xlint:all", "-Werror"))
}
