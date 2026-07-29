# BuildOpt Patcher

Customer-side Java artifact that validates and materializes a `PatchBundle` without executing bundle content.

It must be exact, idempotent, and path-safe. Its first prototype is bounded by `SPK-004`; workflow orchestration remains in Go.
