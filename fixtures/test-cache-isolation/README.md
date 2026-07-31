# A0-G08 test cache isolation fixture

The root Java build composes an actual `buildSrc` plugin build and an included
plugin build. Each scope has a cacheable Gradle `Test` task with deterministic
empty-test output on Gradle 9.6.1. The fixture has no external repositories or
dependencies.

`compositeTest` exposes the root/included boundary. The checker removes only
the declared test-output directories between observations and invokes the
actual `buildSrc` project separately so its `Test` task is also observed.
