# Aggregate output closure fixture

The Groovy and Kotlin builds expose the same custom aggregate workflow. The
`:changed:bundleAll` lifecycle task has no outputs and depends on two custom
producers whose files live outside Gradle's conventional archive, classes,
distribution, and publication directories.

Changing `changed/input.txt` must let BuildOpt rebuild only
`:changed:emitPayload`, materialize the exact `:stable:emitPayload` output, and
preserve the complete two-file workflow output. The fixture intentionally does
not use Java, archive, distribution, publication, or repository-specific task
types so the product path must rely on the configured task and producer graph.
