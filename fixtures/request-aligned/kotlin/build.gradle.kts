plugins {
    java
}

group = "dev.buildopt.fixtures"
version = "6.0.0-SNAPSHOT"

tasks.jar {
    archiveClassifier = "raw"
    isPreserveFileTimestamps = false
    isReproducibleFileOrder = true
}
