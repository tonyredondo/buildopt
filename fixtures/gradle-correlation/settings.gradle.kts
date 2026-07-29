rootProject.name = "buildopt-gradle-correlation-fixture"

include(":alpha")
include(":beta")

val fixtureBuildCacheDirectory = providers.gradleProperty("buildoptFixtureCacheDir")
    .map(::file)
    .orElse(layout.settingsDirectory.dir(".gradle/build-cache").asFile)

buildCache {
    local {
        directory = fixtureBuildCacheDirectory.get()
    }
}
