plugins {
    java
}

tasks.withType<JavaCompile>().configureEach {
    options.release = 17
}
