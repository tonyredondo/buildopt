# Patch Autopilot recipes v1

PA-F2-002 adds two low-risk Gradle recipes to the closed registry:

- Groovy DSL archive reproducibility configures every
  `AbstractArchiveTask` for stable entry order and no preserved timestamps in
  the root `build.gradle`.
- Build-cache properties opts an existing root `gradle.properties` into
  `org.gradle.caching=true`.

Both recipes accept at most 1 MiB of strict UTF-8, LF-only source ending in a
newline. They produce a full-file `MODIFY` replacement with exact preimage and
postimage digests, return defensive bytes, and recognize only their own exact
generated suffix as idempotent. Existing, commented, partial, or ambiguous
configuration fails closed and leaves the repository unchanged.

The registry marks both recipes low risk and binds them to
`ARCHIVE_CONTENTS_V1`. A recipe can only progress through the existing six-run
candidate/control plus `FULL_RELEVANT_VALIDATION` path and draft-only
delivery. Exact signed inverse proof is deliberately completed by PA-F2-003,
not inferred from reversible source text.

Run `./dev/check-patch-autopilot-recipes`.
