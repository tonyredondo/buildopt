rootProject.name = "normalization-aware-cacheability-kotlin"

buildCache {
    local {
        directory = File(settingsDir, ".fixture-cache")
    }
}
