# Central two-machine fixture

This deterministic two-project Gradle repository is used only by the isolated
central-service POC. The root `assemble` workflow builds both projects. A
source-only change in `:app` lets automatic Build Impact select
`:app:assemble`, while the unchanged `:unrelated` project contains explicit
non-cacheable deterministic work. That work makes graph reduction observable
without weakening the eight-pair qualification gate or depending on a native
Gradle cache miss.

Producer and consumer containers receive independent copies of the same Git
history. They never share a workspace, Gradle User Home or BuildOpt local
cache; only the owner-operated HTTPS service persists between them.
