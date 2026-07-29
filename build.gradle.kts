import org.gradle.api.tasks.wrapper.Wrapper

plugins {
    base
}

tasks.named("assemble") {
    dependsOn(":fixtures:golden-lane:assemble")
}

tasks.named("check") {
    dependsOn(":fixtures:golden-lane:check")
}

tasks.named("clean") {
    dependsOn(":fixtures:golden-lane:clean")
}

tasks.named<Wrapper>("wrapper") {
    gradleVersion = "9.6.1"
    distributionType = Wrapper.DistributionType.BIN
    distributionSha256Sum = "9c0f7faeeb306cb14e4279a3e084ca6b596894089a0638e68a07c945a32c9e14"
    networkTimeout = 30_000
    retries = 3
    retryBackOffMs = 1_000
    validateDistributionUrl = true
}
