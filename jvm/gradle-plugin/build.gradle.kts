import org.gradle.api.tasks.bundling.AbstractArchiveTask
import org.gradle.api.tasks.compile.JavaCompile
import org.gradle.jvm.toolchain.JvmVendorSpec

plugins {
    `java-library`
    `java-gradle-plugin`
}

group = "dev.buildopt"
version = providers.gradleProperty("buildoptVersion").getOrElse("0.1.0-SNAPSHOT")

base {
    archivesName = "buildopt-gradle-plugin"
}

java {
    toolchain {
        languageVersion = JavaLanguageVersion.of(21)
        vendor = JvmVendorSpec.ADOPTIUM
    }
}

dependencies {
    compileOnly(gradleApi())
}

gradlePlugin {
    plugins {
        create("buildOpt") {
            id = "dev.buildopt"
            implementationClass = "dev.buildopt.gradle.BuildOptProjectPlugin"
            displayName = "BuildOpt Gradle Plugin"
            description = "Authenticated neutral launcher rendezvous for BuildOpt"
        }
    }
}

tasks.withType<JavaCompile>().configureEach {
    options.release = 17
    options.encoding = "UTF-8"
    options.compilerArgs.addAll(listOf("-Xlint:all", "-Werror"))
}

tasks.withType<AbstractArchiveTask>().configureEach {
    isPreserveFileTimestamps = false
    isReproducibleFileOrder = true
}

tasks.jar {
    manifest {
        attributes(
            "Implementation-Title" to "BuildOpt Gradle Plugin",
            "Implementation-Version" to project.version,
        )
    }
}
