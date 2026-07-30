import org.gradle.api.tasks.bundling.AbstractArchiveTask
import org.gradle.api.tasks.compile.JavaCompile
import org.gradle.api.tasks.JavaExec
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

val testKit = sourceSets.create("testKit")

dependencies {
    compileOnly(gradleApi())
    add(testKit.implementationConfigurationName, gradleTestKit())
}

gradlePlugin {
    plugins {
        create("buildOpt") {
            id = "dev.buildopt"
            implementationClass = "dev.buildopt.gradle.BuildOptProjectPlugin"
            displayName = "BuildOpt Gradle Plugin"
            description = "Authenticated neutral launcher rendezvous for BuildOpt"
        }
        create("buildOptTierOnePolicy") {
            id = "dev.buildopt.tier-one-policy"
            implementationClass = "dev.buildopt.gradle.BuildOptTierOnePolicyPlugin"
            displayName = "BuildOpt Tier 1 Cache Policy"
            description = "Default-deny task and transform policy for a managed BuildOpt cache"
        }
        create("buildOptManagedL1") {
            id = "dev.buildopt.managed-l1"
            implementationClass = "dev.buildopt.gradle.BuildOptManagedL1Plugin"
            displayName = "BuildOpt Managed L1"
            description = "Native generation-segmented DirectoryBuildCache configuration"
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

val tierOneRuntime = providers.gradleProperty("buildoptTierOneJava")
    .map(String::toInt)
    .orElse(21)
val tierOneGradleHome = providers.gradleProperty("buildoptTierOneGradleHome")
val tierOneFixtures = rootProject.layout.projectDirectory.dir("fixtures/tier1")

tasks.register<JavaExec>("tierOneTestKit") {
    group = "verification"
    description = "Runs the Tier 1 Kotlin/Groovy fixtures through Gradle TestKit."
    notCompatibleWithConfigurationCache("The task launches nested TestKit builds.")
    dependsOn(tasks.named(testKit.classesTaskName), tasks.named("jar"))
    classpath = testKit.runtimeClasspath
    mainClass = "dev.buildopt.gradle.TierOneTestKit"
    javaLauncher = javaToolchains.launcherFor {
        languageVersion = tierOneRuntime.map(JavaLanguageVersion::of)
    }
    argumentProviders.add(
        CommandLineArgumentProvider {
            listOf(
                tierOneFixtures.asFile.absolutePath,
                tierOneGradleHome.get(),
                tasks.jar.get().archiveFile.get().asFile.absolutePath,
                tierOneRuntime.get().toString(),
            )
        },
    )
    inputs.dir(tierOneFixtures)
    inputs.file(tasks.jar.flatMap { it.archiveFile })
    inputs.property("runtime", tierOneRuntime)
    inputs.property("gradleHome", tierOneGradleHome)
}

tasks.register<JavaExec>("tierOnePolicyTestKit") {
    group = "verification"
    description = "Runs the Tier 1 managed-cache default-deny conformance."
    notCompatibleWithConfigurationCache("The task launches nested TestKit builds.")
    dependsOn(tasks.named(testKit.classesTaskName), tasks.named("jar"))
    classpath = testKit.runtimeClasspath
    mainClass = "dev.buildopt.gradle.TierOnePolicyTestKit"
    javaLauncher = javaToolchains.launcherFor {
        languageVersion = tierOneRuntime.map(JavaLanguageVersion::of)
    }
    argumentProviders.add(
        CommandLineArgumentProvider {
            listOf(
                tierOneFixtures.asFile.absolutePath,
                tierOneGradleHome.get(),
                tasks.jar.get().archiveFile.get().asFile.absolutePath,
                tierOneRuntime.get().toString(),
            )
        },
    )
    inputs.dir(tierOneFixtures)
    inputs.file(tasks.jar.flatMap { it.archiveFile })
    inputs.property("runtime", tierOneRuntime)
    inputs.property("gradleHome", tierOneGradleHome)
}

tasks.register<JavaExec>("managedL1TestKit") {
    group = "verification"
    description = "Runs the generation-segmented native managed L1 conformance."
    notCompatibleWithConfigurationCache("The task launches nested TestKit builds.")
    dependsOn(tasks.named(testKit.classesTaskName), tasks.named("jar"))
    classpath = testKit.runtimeClasspath
    mainClass = "dev.buildopt.gradle.ManagedL1TestKit"
    javaLauncher = javaToolchains.launcherFor {
        languageVersion = tierOneRuntime.map(JavaLanguageVersion::of)
    }
    argumentProviders.add(
        CommandLineArgumentProvider {
            listOf(
                tierOneFixtures.asFile.absolutePath,
                tierOneGradleHome.get(),
                tasks.jar.get().archiveFile.get().asFile.absolutePath,
                tierOneRuntime.get().toString(),
            )
        },
    )
    inputs.dir(tierOneFixtures)
    inputs.file(tasks.jar.flatMap { it.archiveFile })
    inputs.property("runtime", tierOneRuntime)
    inputs.property("gradleHome", tierOneGradleHome)
}

tasks.jar {
    manifest {
        attributes(
            "Implementation-Title" to "BuildOpt Gradle Plugin",
            "Implementation-Version" to project.version,
        )
    }
}
