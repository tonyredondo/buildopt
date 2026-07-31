# Archive reproducibility Kotlin DSL recipe v1

This POC contract closes C4-003 with one production Java 17 recipe:
ARCHIVE_REPRODUCIBILITY_KOTLIN_DSL_V1 version 1.0.

The recipe accepts only a non-empty root build.gradle.kts of at most 1 MiB,
encoded as strict UTF-8 with LF endings and a final newline. It generates one
exact full-file replacement: the AbstractArchiveTask import, the unchanged
original source, and a configureEach block that enables reproducible file
order and disables preserved file timestamps.

It fails closed rather than guessing when the source has file annotations,
CR/NUL bytes, invalid UTF-8, or any existing archive-task import or property.
Only the exact envelope previously emitted by this version is idempotent. The
result owns defensive postimage bytes and exact SHA-256 preimage/postimage
digests, so the production signer and applier bind the generated replacement
without rereading or interpreting Gradle code.

The recipe does not evaluate a build, execute customer content, accept Groovy
DSL, rewrite nested build files, merge existing configuration, or activate
remote delivery.

Run ./dev/check-archive-reproducibility-recipe.

The checker verifies the machine-readable contract and executes the complete
PatchBundle runtime harness, including exact generation, repeat idempotency,
defensive output, strict negative inputs, signed bundle construction, and all
15 real-Git application and recovery cases.
