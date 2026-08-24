# Configuration-input binding POC

## Question

Can BuildOpt use supported Gradle evidence to map a configuration-time file
input completely to the projects whose requested outputs it affects?

The answer determines whether a changed configuration input may enter a
project-scoped candidate or must retain the complete requested workflow.

## Supported evidence

The fixture reads `versions.properties` with
`ProviderFactory.fileContents`, queries the provider during configuration and
uses the result only to set `:service-a`'s version. `:service-b` is present to
make an unaffected project observable.

Gradle documents that querying this provider during configuration makes the
file a Configuration Cache input. Gradle also reports the file and the build
logic location that performed the read. That location is provenance for the
read; it is not a semantic contract that identifies every affected project.
A root script, settings script or convention plugin can configure any number
of projects.

The validation therefore requires four independent cases:

- Gradle 8.14.3 with Groovy DSL;
- Gradle 8.14.3 with Kotlin DSL;
- Gradle 9.6.1 with Groovy DSL;
- Gradle 9.6.1 with Kotlin DSL.

Every case must store and reuse Configuration Cache state, invalidate it when
the file changes, report the changed input, change `:service-a`'s JAR version
and preserve `:service-b`'s JAR version.

## Decision

The supported report proves build-model relevance but does not expose a
complete input-to-project ownership relation. BuildOpt must therefore retain
the full requested workflow for this input class. It must not interpret a
build-logic source location or Gradle's internal fingerprint files as project
ownership.

This is a terminal negative result for the bounded hypothesis, not permission
to add a filename, repository or DSL rule. A future supported Gradle API may
reopen it.

References:

- [Gradle Configuration Cache requirements](https://docs.gradle.org/current/userguide/configuration_cache_requirements.html)
- [Gradle `ProviderFactory.fileContents`](https://docs.gradle.org/current/dsl/org.gradle.api.provider.ProviderFactory.html#org.gradle.api.provider.ProviderFactory:fileContents(org.gradle.api.file.RegularFile))
- [Gradle `ValueSource`](https://docs.gradle.org/current/userguide/value_providers.html)

## POC boundaries

This result changes neither Test Optimization nor production authority. It
requires no soak or design partner. It protects exactness while the POC moves
to the next generic hypothesis: aggregate output closure.
