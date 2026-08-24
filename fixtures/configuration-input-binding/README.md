# Configuration-input binding fixture

This fixture asks whether Gradle's public configuration-input model exposes a
project owner for a repository file read through `ProviderFactory.fileContents`.
The Groovy and Kotlin builds intentionally have the same two-project shape:

- `versions.properties` is read during configuration;
- only `:service-a` consumes the resulting version;
- `:service-a:jar` and `:service-b:jar` make the output effect observable.

The validation stores Configuration Cache state, proves reuse without a
change, changes `versions.properties`, and observes Gradle's invalidation. The
fixture is evidence about Gradle's supported public model, not a repository
rule or a production compatibility promise.
