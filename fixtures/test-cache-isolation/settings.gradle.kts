import org.gradle.caching.http.HttpBuildCache

rootProject.name = "test-cache-isolation-root"
includeBuild("included-plugin")

buildCache {
    local {
        isEnabled = false
    }
    remote<HttpBuildCache> {
        url = uri(providers.gradleProperty("buildoptTestCacheUrl").get())
        isPush = true
        isAllowUntrustedServer = true
        credentials {
            username = "buildopt"
            password = providers.gradleProperty("buildoptTestCachePassword").get()
        }
    }
}
