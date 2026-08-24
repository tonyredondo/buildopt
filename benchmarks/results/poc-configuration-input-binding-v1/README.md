# Configuration-input binding result

Four independent fixtures prove that Gradle 8.14.3 and 9.6.1, with Groovy and
Kotlin DSL, detect a root file read through `ProviderFactory.fileContents` as
a Configuration Cache input.

The executable fixture and contract are frozen at BuildOpt revision
`cee4f6c2d0897008097e424225b231dd53a74eec`.

All four cases store the configuration, reuse it before the change, invalidate
it after `versions.properties` changes, change only `:service-a`'s versioned
JAR and leave `:service-b` stable. The supported Configuration Cache report
records the changed file and the build-logic source location. It does not
provide a complete semantic mapping to the affected project.

BuildOpt therefore retains native Gradle for the complete requested workflow.
It does not use internal Gradle fingerprint files or the build script location
as product authority. No performance replay was run because the safety
precondition for narrower selection was not established.
