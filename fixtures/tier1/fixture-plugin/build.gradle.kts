import org.gradle.api.tasks.compile.JavaCompile

plugins {
    `java-gradle-plugin`
}

java {
    sourceCompatibility = JavaVersion.VERSION_17
    targetCompatibility = JavaVersion.VERSION_17
}

gradlePlugin {
    plugins {
        create("tierOneFixture") {
            id = "dev.buildopt.tier1-fixture"
            implementationClass = "dev.buildopt.fixtures.tierone.TierOneFixturePlugin"
        }
    }
}

tasks.withType<JavaCompile>().configureEach {
    options.release = 17
    options.encoding = "UTF-8"
    options.compilerArgs.addAll(listOf("-Xlint:all", "-Werror"))
}
