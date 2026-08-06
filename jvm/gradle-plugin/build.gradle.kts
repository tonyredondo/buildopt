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
        create("buildOptStandardJarCache") {
            id = "dev.buildopt.standard-jar-cache"
            implementationClass = "dev.buildopt.gradle.BuildOptStandardJarCachePlugin"
            displayName = "BuildOpt POC Standard Jar Cache"
            description = "Explicit cache eligibility for unmodified standard Gradle Jar producers"
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
val betaFixtureResult = providers.gradleProperty("buildoptBetaFixtureResult")
val betaFixtureProfiles = providers.gradleProperty("buildoptBetaFixtureProfiles")
val betaFixtureBenchmarkDigest = providers.gradleProperty("buildoptBetaFixtureBenchmarkDigest")

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

tasks.register<JavaExec>("testCacheIsolationTestKit") {
    group = "verification"
    description = "Runs A0-G08 no-grant Test cache isolation."
    notCompatibleWithConfigurationCache("The task launches nested TestKit builds.")
    dependsOn(tasks.named(testKit.classesTaskName), tasks.named("jar"))
    classpath = testKit.runtimeClasspath
    mainClass = "dev.buildopt.gradle.TestCacheIsolationTestKit"
    javaLauncher = javaToolchains.launcherFor {
        languageVersion = JavaLanguageVersion.of(21)
    }
    argumentProviders.add(
        CommandLineArgumentProvider {
            listOf(
                rootProject.layout.projectDirectory
                    .dir("fixtures/test-cache-isolation")
                    .asFile.absolutePath,
                tierOneGradleHome.get(),
                tasks.jar.get().archiveFile.get().asFile.absolutePath,
                "21",
            )
        },
    )
    inputs.dir(rootProject.layout.projectDirectory.dir("fixtures/test-cache-isolation"))
    inputs.file(tasks.jar.flatMap { it.archiveFile })
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

tasks.register<JavaExec>("standardJarCacheTestKit") {
    group = "verification"
    description = "Proves the explicit POC cache adapter for standard Jar producers."
    notCompatibleWithConfigurationCache("The task launches a nested TestKit build.")
    dependsOn(tasks.named(testKit.classesTaskName), tasks.named("jar"))
    classpath = testKit.runtimeClasspath
    mainClass = "dev.buildopt.gradle.StandardJarCacheTestKit"
    javaLauncher = javaToolchains.launcherFor {
        languageVersion = JavaLanguageVersion.of(21)
    }
    argumentProviders.add(
        CommandLineArgumentProvider {
            listOf(
                tierOneGradleHome.get(),
                tasks.jar.get().archiveFile.get().asFile.absolutePath,
            )
        },
    )
    inputs.file(tasks.jar.flatMap { it.archiveFile })
    inputs.property("gradleHome", tierOneGradleHome)
}

tasks.register<JavaExec>("managedSharedTestKit") {
    group = "verification"
    description = "Runs the locally authenticated HttpBuildCache conformance."
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
                "shared-only",
            )
        },
    )
    inputs.dir(tierOneFixtures)
    inputs.file(tasks.jar.flatMap { it.archiveFile })
    inputs.property("runtime", tierOneRuntime)
    inputs.property("gradleHome", tierOneGradleHome)
}

tasks.register<JavaExec>("l1L2LifecycleTestKit") {
    group = "verification"
    description = "Runs the L2-to-L1 revocation and aborted-writer lifecycle."
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
                "lifecycle-only",
            )
        },
    )
    inputs.dir(tierOneFixtures)
    inputs.file(tasks.jar.flatMap { it.archiveFile })
    inputs.property("runtime", tierOneRuntime)
    inputs.property("gradleHome", tierOneGradleHome)
}

tasks.register<JavaExec>("circuitBreakerTestKit") {
    group = "verification"
    description = "Proves Gradle preservation while the managed L2 circuit is open."
    notCompatibleWithConfigurationCache("The task launches nested TestKit builds.")
    dependsOn(tasks.named(testKit.classesTaskName), tasks.named("jar"))
    classpath = testKit.runtimeClasspath
    mainClass = "dev.buildopt.gradle.ManagedL1TestKit"
    javaLauncher = javaToolchains.launcherFor {
        languageVersion = JavaLanguageVersion.of(21)
    }
    argumentProviders.add(
        CommandLineArgumentProvider {
            listOf(
                tierOneFixtures.asFile.absolutePath,
                tierOneGradleHome.get(),
                tasks.jar.get().archiveFile.get().asFile.absolutePath,
                "21",
                "circuit-only",
            )
        },
    )
    inputs.dir(tierOneFixtures)
    inputs.file(tasks.jar.flatMap { it.archiveFile })
    inputs.property("gradleHome", tierOneGradleHome)
}

tasks.register<JavaExec>("betaFixtureSizeMatrixTestKit") {
    group = "verification"
    description = "Runs the private-beta small/medium/large Gradle fixture matrix."
    notCompatibleWithConfigurationCache("The task launches nested TestKit builds.")
    dependsOn(tasks.named(testKit.classesTaskName), tasks.named("jar"))
    classpath = testKit.runtimeClasspath
    mainClass = "dev.buildopt.gradle.BetaFixtureSizeMatrixTestKit"
    javaLauncher = javaToolchains.launcherFor {
        languageVersion = JavaLanguageVersion.of(21)
    }
    argumentProviders.add(
        CommandLineArgumentProvider {
            listOf(
                tierOneGradleHome.get(),
                tasks.jar.get().archiveFile.get().asFile.absolutePath,
                "21",
                betaFixtureResult.get(),
                betaFixtureBenchmarkDigest.get(),
                betaFixtureProfiles.get(),
            )
        },
    )
    inputs.file(tasks.jar.flatMap { it.archiveFile })
    inputs.property("gradleHome", tierOneGradleHome)
    inputs.property("benchmarkDigest", betaFixtureBenchmarkDigest)
    inputs.property("profiles", betaFixtureProfiles)
    outputs.file(betaFixtureResult)
}

tasks.register<JavaExec>("gatewayRotationTestKit") {
    group = "verification"
    description = "Runs stable gateway restart and complete-rotation conformance."
    notCompatibleWithConfigurationCache("The task launches nested TestKit builds.")
    dependsOn(tasks.named(testKit.classesTaskName), tasks.named("jar"))
    classpath = testKit.runtimeClasspath
    mainClass = "dev.buildopt.gradle.GatewayRotationTestKit"
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

tasks.register<JavaExec>("tierOneCacheConformanceTestKit") {
    group = "verification"
    description = "Runs Tier 1 Gradle HTTP cache conformance."
    notCompatibleWithConfigurationCache("The task launches nested TestKit builds.")
    dependsOn(tasks.named(testKit.classesTaskName), tasks.named("jar"))
    classpath = testKit.runtimeClasspath
    mainClass = "dev.buildopt.gradle.TierOneCacheConformanceTestKit"
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
