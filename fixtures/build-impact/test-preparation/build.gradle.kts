plugins {
    base
}

subprojects {
    apply(plugin = "java")
}

tasks.register("testClasses") {
    dependsOn(subprojects.map { "${it.path}:testClasses" })
    if (System.getenv("BUILDOPT_TEST_PREPARATION_UNSAFE") == "1") {
        dependsOn(":service-b:test")
    }
}
