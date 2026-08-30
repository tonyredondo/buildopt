# Verified request hit shadow fixture

This fixture proves the shadow-only mechanics of the verified request hit POC
across Gradle 8/9 and Kotlin/Groovy DSL builds. One cacheable task writes a
deterministic declared output. The harness removes that output before every
native invocation, predicts from a complete `VRH-002` record, runs the exact
native command and compares the resulting present and absent states.

`BUILDOPT_SHADOW_MISMATCH` is used only by the negative fixture to simulate an
incorrect upstream safety claim. The first mismatch must quarantine the record
identity; the next invocation still runs native Gradle and makes no prediction.
