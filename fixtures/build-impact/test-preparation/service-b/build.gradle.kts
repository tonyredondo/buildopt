plugins {
    java
}

if (System.getenv("BUILDOPT_TEST_PREPARATION_UNSAFE") == "1") {
    val unsafeVerification by tasks.registering(Test::class)
    tasks.named("testClasses") {
        dependsOn(unsafeVerification)
    }
}
