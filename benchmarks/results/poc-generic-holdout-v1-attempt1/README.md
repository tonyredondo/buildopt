# Hibernate ORM holdout attempt 1

This immutable attempt binds BuildOpt `f8dac70`, Hibernate ORM
`2b448a59d332326f0cd0691c868425124d55cbb5`, root `assemble`, one fixed
`hibernate-core/Session.java` comment change and the preregistered
`hibernate-core/build/libs/**` required output.

The unchanged generic proposal completed and selected
`:hibernate-core:assemble`, reducing 29 projects to one. Both isolated base
warm-ups completed. Before accepting pair one, output verification stopped the
experiment because the control produced no file matching the declared glob.
There are zero accepted timing observations and no performance conclusion.

The failure is an owner-contract error, not a product-attributable Gradle
failure. Hibernate's frozen `ModuleAspect.java` line 31 changes module build
directories to `target`, so the correct repository-owned output location is
`hibernate-core/target/libs/**`. The separately preregistered v2 correction
changes only that path and reuses no proposal, warm-up or timing from this
attempt.
