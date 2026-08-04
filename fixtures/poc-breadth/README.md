# POC breadth fixtures

`realistic-multimodule` is an owner-controlled five-project Gradle graph used to
exercise no-change, leaf-change, shared-change, and global build-logic behavior.
It supports Kotlin and Groovy DSL from the same sources and intentionally
contains deterministic non-cacheable verification work so Build Impact can be
compared with a native Gradle control that already uses its first-party caches.
