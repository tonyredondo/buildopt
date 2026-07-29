import org.gradle.caching.http.HttpBuildCache

rootProject.name = "buildopt-gradle-correlation-fixture"

include(":alpha")
include(":beta")

val fixtureBuildCacheDirectory = providers.gradleProperty("buildoptFixtureCacheDir")
    .map(::file)
    .orElse(layout.settingsDirectory.dir(".gradle/build-cache").asFile)
val fixtureBuildCacheUrl = providers.gradleProperty("buildoptFixtureCacheUrl")

buildCache {
    if (fixtureBuildCacheUrl.isPresent) {
        local {
            isEnabled = false
        }
        remote<HttpBuildCache> {
            url = uri(fixtureBuildCacheUrl.get())
            isPush = true
            isAllowInsecureProtocol = true
        }
    } else {
        local {
            directory = fixtureBuildCacheDirectory.get()
        }
    }
}
